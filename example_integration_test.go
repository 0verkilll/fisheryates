// example_integration_test.go demonstrates how an external package that owns
// Set / Record / Derivation should call this package.
//
// The non-test functions (DerivationFromPRNG, ReadHeader, ReadInEmbedOrder,
// EmbedPositionOf, WalkDigits) are the ones to copy into your package.
// The types below mirror the caller's real definitions — replace with yours.
package fisheryates_test

import (
	"testing"

	fy "github.com/0verkilll/fisheryates"
	"github.com/0verkilll/f5prng"
)

// ── mirror of the caller's types ─────────────────────────────────────────────

type Record struct {
	K      uint8
	MsgLen uint32
}

// Derivation holds the per-password derived state.
// Add InverseOrder if you need O(1) "which embed pos carries coeff j?" lookups.
type Derivation struct {
	K, MsgLen       uint8
	XORMask         [4]byte
	Order           []int // σ:   Order[i]        = original coeff index at embed pos i
	InverseOrder    []int // σ⁻¹: InverseOrder[j] = embed position of original coeff j
	HeaderPositions []int // Order[0:numHeaderCoeffs]
}

type Set struct {
	Coefficients      []int16
	TotalCoefficients int
	Usable            int        // N = non-zero AC DCT coefficient count = shuffle size
	Derivation        *Derivation
	Records           []Record
}

// ── integration helpers ───────────────────────────────────────────────────────

// numHeaderCoeffs is the number of DCT coefficients that carry the F5 header.
// Adjust to match the header width in your F5 implementation.
const numHeaderCoeffs = 4

// DerivationFromPRNG builds a Derivation for one password candidate.
//
//   - src must be freshly seeded with the candidate password.
//   - If s.Derivation.HeaderPositions is already populated (from a prior partial
//     decode), it is used as the known-head oracle: src is consumed and the
//     function returns (nil, nil) immediately if the head does not match,
//     letting the caller skip to the next candidate without further cost.
//   - On success, src has advanced exactly s.Usable+1 NextInt-equivalents:
//     N calls for the shuffle, then 4 bytes for the XOR mask.
//
// Returned Derivation fields:
//
//	Order         = σ  (pair.Forward)  — iterate to read/write embed positions
//	InverseOrder  = σ⁻¹ (pair.Inverse) — map original coeff index → embed pos
//	HeaderPositions                     — Order[0:numHeaderCoeffs]
//	XORMask                             — 4 post-shuffle PRNG bytes
func DerivationFromPRNG(s *Set, src f5prng.RandomSource) (*Derivation, error) {
	N := s.Usable

	var knownHead []int
	if s.Derivation != nil {
		knownHead = s.Derivation.HeaderPositions
	}

	pair, err := fy.CompleteFromKnownHead(N, src, knownHead)
	if err != nil {
		return nil, err
	}
	if len(knownHead) > 0 && !pair.HeadMatch {
		return nil, nil // wrong password — caller tries next candidate
	}

	// PRNG has advanced exactly N calls. The next 4 bytes are the XOR mask
	// (same byte-stream position as F5.jar's post-shuffle header key derivation).
	maskBytes := src.NextBytes(4)
	var xorMask [4]byte
	copy(xorMask[:], maskBytes)

	headerPos := make([]int, numHeaderCoeffs)
	copy(headerPos, pair.Forward[:numHeaderCoeffs])

	return &Derivation{
		XORMask:         xorMask,
		Order:           pair.Forward,
		InverseOrder:    pair.Inverse,
		HeaderPositions: headerPos,
	}, nil
}

// ReadHeader extracts and XOR-unmasks the F5 header bytes.
func ReadHeader(s *Set, d *Derivation) [numHeaderCoeffs]byte {
	var hdr [numHeaderCoeffs]byte
	for i, origIdx := range d.HeaderPositions {
		hdr[i] = byte(s.Coefficients[origIdx]) ^ d.XORMask[i]
	}
	return hdr
}

// ReadInEmbedOrder returns all stego coefficients in F5 embed order.
// out[i] = s.Coefficients[d.Order[i]]: the coefficient at embed position i.
func ReadInEmbedOrder(s *Set, d *Derivation) []int16 {
	out := make([]int16, len(d.Order))
	for i, origIdx := range d.Order {
		out[i] = s.Coefficients[origIdx]
	}
	return out
}

// EmbedPositionOf returns the embed position for original coefficient j. O(1).
func EmbedPositionOf(d *Derivation, j int) int {
	return d.InverseOrder[j]
}

// WalkDigits returns the FY digit sequence that produced d.Order.
// digit[i] = pool index selected at step i = int(PRNG.NextInt()) % (N - i).
// Useful for PRNG analysis; only call after the seed is confirmed.
func WalkDigits(d *Derivation) ([]int, error) {
	return fy.RecoverDigits(d.Order)
}

// ── tests ─────────────────────────────────────────────────────────────────────

// newSrc returns a deterministic RandomSource using SplitMix64.
// In your package, replace with: prng := factory.NewPRNG(); prng.Seed([]byte(password))
func newSrc(seed uint64) f5prng.RandomSource {
	return &splitMix64RS{state: seed}
}

func TestIntegration_DerivationRoundTrip(t *testing.T) {
	const N = 512
	s := &Set{
		Usable:            N,
		TotalCoefficients: N + 100,
		Coefficients:      make([]int16, N+100),
	}
	for i := range s.Coefficients {
		s.Coefficients[i] = int16(i + 1)
	}

	seed := uint64(0xF5_CAFE_DEAD_BEEF)

	// First pass: no prior knowledge, derive everything fresh.
	d, err := DerivationFromPRNG(s, newSrc(seed))
	if err != nil || d == nil {
		t.Fatalf("first DerivationFromPRNG: err=%v d=%v", err, d)
	}

	// Order must be a valid permutation of [0, N).
	seen := make([]bool, N)
	for i, v := range d.Order {
		if v < 0 || v >= N || seen[v] {
			t.Fatalf("Order[%d]=%d invalid", i, v)
		}
		seen[v] = true
	}

	// σ ∘ σ⁻¹ = identity: Order[InverseOrder[j]] == j.
	for j := range N {
		if d.Order[d.InverseOrder[j]] != j {
			t.Errorf("Order[InverseOrder[%d]] = %d ≠ %d", j, d.Order[d.InverseOrder[j]], j)
		}
	}

	// Second pass: with HeaderPositions as the known-head oracle.
	s.Derivation = &Derivation{HeaderPositions: d.HeaderPositions}
	d2, err := DerivationFromPRNG(s, newSrc(seed))
	if err != nil || d2 == nil {
		t.Fatalf("second pass with correct seed: err=%v d=%v", err, d2)
	}
	for i := range N {
		if d.Order[i] != d2.Order[i] {
			t.Errorf("Order[%d] mismatch", i)
		}
	}
	if d.XORMask != d2.XORMask {
		t.Errorf("XORMask mismatch: %v vs %v", d.XORMask, d2.XORMask)
	}
}

func TestIntegration_WrongSeedRejected(t *testing.T) {
	const N = 256
	s := &Set{Usable: N, Coefficients: make([]int16, N)}

	d, _ := DerivationFromPRNG(s, newSrc(0xAAAA))
	s.Derivation = &Derivation{HeaderPositions: d.HeaderPositions}

	got, err := DerivationFromPRNG(s, newSrc(0xBBBB))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("wrong seed passed the known-head check — false positive")
	}
}

func TestIntegration_EmbedExtractRoundTrip(t *testing.T) {
	const N = 128
	original := make([]int16, N)
	for i := range N {
		original[i] = int16(i * 3)
	}
	s := &Set{Usable: N, Coefficients: original}

	d, err := DerivationFromPRNG(s, newSrc(0x1234567890ABCDEF))
	if err != nil || d == nil {
		t.Fatalf("DerivationFromPRNG: %v %v", err, d)
	}

	// Embed: 1 bit per coefficient via LSB substitution.
	msg := make([]byte, N/8)
	for i := range msg {
		msg[i] = byte(i * 7)
	}
	stego := make([]int16, N)
	copy(stego, original)
	for pos, origIdx := range d.Order {
		bit := (msg[pos/8] >> (pos % 8)) & 1
		stego[origIdx] = (original[origIdx] &^ 1) | int16(bit)
	}

	// Extract: read bits in embed order.
	s.Coefficients = stego
	extracted := make([]byte, N/8)
	for pos, coeff := range ReadInEmbedOrder(s, d) {
		extracted[pos/8] |= (byte(coeff) & 1) << (pos % 8)
	}

	for i := range msg {
		if extracted[i] != msg[i] {
			t.Errorf("byte %d: embedded 0x%02x, extracted 0x%02x", i, msg[i], extracted[i])
		}
	}
}

func TestIntegration_WalkDigitsRoundTrip(t *testing.T) {
	const N = 64
	s := &Set{Usable: N, Coefficients: make([]int16, N)}
	d, _ := DerivationFromPRNG(s, newSrc(0xDECAFBAD))

	digits, err := WalkDigits(d)
	if err != nil {
		t.Fatalf("WalkDigits: %v", err)
	}

	replay := make([]int, N)
	for i := range N {
		replay[i] = i
	}
	for i, dig := range digits {
		last := N - 1 - i
		replay[dig], replay[last] = replay[last], replay[dig]
	}
	for i := range N {
		if replay[i] != d.Order[i] {
			t.Errorf("replay[%d]=%d, Order[%d]=%d", i, replay[i], i, d.Order[i])
		}
	}
}

func TestIntegration_EmbedPositionOf(t *testing.T) {
	const N = 32
	s := &Set{Usable: N, Coefficients: make([]int16, N)}
	d, _ := DerivationFromPRNG(s, newSrc(0xABCD))

	// EmbedPositionOf(j) must satisfy Order[EmbedPositionOf(j)] == j.
	for j := range N {
		pos := EmbedPositionOf(d, j)
		if d.Order[pos] != j {
			t.Errorf("coeff %d: EmbedPositionOf=%d, Order[%d]=%d", j, pos, pos, d.Order[pos])
		}
	}
}

// ── minimal f5prng.RandomSource adapter (SplitMix64) ─────────────────────────
// In your package replace with factory.NewPRNG() + prng.Seed([]byte(password)).

type splitMix64RS struct{ state uint64 }

func (s *splitMix64RS) Seed(_ []byte) error { return nil }
func (s *splitMix64RS) Clear()              {}
func (s *splitMix64RS) NextInt() int32 {
	s.state += 0x9E3779B97F4A7C15
	z := s.state
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	z ^= z >> 31
	return int32(z)
}
func (s *splitMix64RS) NextBytes(n int) []byte {
	out := make([]byte, n)
	for i := 0; i < n; i += 4 {
		v := s.NextInt()
		for j := 0; j < 4 && i+j < n; j++ {
			out[i+j] = byte(v >> (j * 8))
		}
	}
	return out
}

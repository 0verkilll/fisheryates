package fisheryates

import (
	"slices"
	"testing"

	"github.com/0verkilll/f5prng"
	"github.com/0verkilll/securerandom"
	"github.com/0verkilll/sha1"
)

// nonBulkSource wraps an f5prng.RandomSource and deliberately does NOT expose
// NextBytesInto. Passing an instance of this type to Generate forces the
// serial shuffle path even when the underlying source supports bulk fill.
//
// Used by TestBatchedVsSerial_ByteStreamEquivalence to prove that the batched
// path consumes the PRNG state byte-for-byte identically to the serial path
// for any (seed, size) pair.
type nonBulkSource struct {
	inner f5prng.RandomSource
}

func (w *nonBulkSource) Seed(s []byte) error   { return w.inner.Seed(s) }
func (w *nonBulkSource) NextInt() int32        { return w.inner.NextInt() }
func (w *nonBulkSource) NextBytes(n int) []byte { return w.inner.NextBytes(n) }
func (w *nonBulkSource) Clear()                { w.inner.Clear() }

// Compile-time assertions: nonBulkSource must satisfy RandomSource but MUST
// NOT accidentally satisfy RandomSourceWithBytesInto (otherwise the test
// would silently measure the wrong thing).
var (
	_ f5prng.RandomSource = (*nonBulkSource)(nil)
)

// TestBatchedVsSerial_ByteStreamEquivalence locks in the fix for the
// third-pass bug where shuffleBatched over-read PRNG state on the final
// refill and produced a different post-shuffle PRNG state than shuffleSerial.
//
// For the same (seed, size), the two paths must now produce:
//  1. Identical permutations, AND
//  2. (Implicitly) identical downstream PRNG state — if they disagree on the
//     number of bytes consumed, the permutation itself diverges because the
//     unbiasedMod rejection retry count depends on exactly which 4-byte
//     chunks are drawn.
//
// Sizes chosen to cover:
//   - below batchMinSize (serial-serial control)
//   - right at the threshold (off-by-one trap)
//   - a small-medium batched size where one refill is enough
//   - a size that crosses multiple refills
//   - a size whose maxRandom*4 is greater than batchBytes (forces the
//     min(batchBytes, need) cap to kick in)
func TestBatchedVsSerial_ByteStreamEquivalence(t *testing.T) {
	sizes := []int{1, 16, 64, 100, 255, 256, 257, 500, 1024, 4096, 10000, 50000}
	seeds := [][]byte{
		[]byte("determinism-v1"),
		[]byte("another-seed-with-different-bytes"),
		[]byte{0x00, 0x01, 0x02, 0x03},
		[]byte{0xFF, 0xFF, 0xFF, 0xFF},
	}

	for _, seed := range seeds {
		for _, size := range sizes {
			t.Run("", func(t *testing.T) {
				// Batched path: raw SecureRandom, which satisfies
				// RandomSourceWithBytesInto.
				hasher1 := sha1.NewSHA1(sha1.NewBigEndian())
				sr1 := securerandom.NewSecureRandom(hasher1)
				if err := sr1.Seed(seed); err != nil {
					t.Fatalf("seed batched: %v", err)
				}
				fy1 := NewFisherYates()
				batched, err := fy1.Generate(size, sr1)
				if err != nil {
					t.Fatalf("batched Generate(size=%d): %v", size, err)
				}

				// Serial path: same SecureRandom wrapped to hide
				// NextBytesInto, forcing shuffleSerial.
				hasher2 := sha1.NewSHA1(sha1.NewBigEndian())
				sr2 := securerandom.NewSecureRandom(hasher2)
				if err := sr2.Seed(seed); err != nil {
					t.Fatalf("seed serial: %v", err)
				}
				wrapped := &nonBulkSource{inner: sr2}
				fy2 := NewFisherYates()
				serial, err := fy2.Generate(size, wrapped)
				if err != nil {
					t.Fatalf("serial Generate(size=%d): %v", size, err)
				}

				if !slices.Equal(batched, serial) {
					// Show the first divergence for a useful error.
					for i := 0; i < len(batched) && i < len(serial); i++ {
						if batched[i] != serial[i] {
							t.Fatalf("size=%d seed=%q: batched != serial at idx %d (batched=%d, serial=%d); first 16: batched=%v serial=%v",
								size, seed, i, batched[i], serial[i],
								batched[:min(16, len(batched))], serial[:min(16, len(serial))])
						}
					}
					t.Fatalf("size=%d seed=%q: length mismatch batched=%d serial=%d",
						size, seed, len(batched), len(serial))
				}
			})
		}
	}
}

// TestBatchedVsSerial_ConsumesSamePRNGState proves the stronger invariant:
// after the shuffle completes, the underlying PRNG is in the same state on
// both paths. This is the property that makes embed/extract round-trips
// work even when one side uses a RandomSource view that doesn't forward
// NextBytesInto — if the two paths drain the PRNG differently, subsequent
// NextBytes calls (header XOR in the F5 stack) return different values and
// the round-trip fails silently with corrupted bytes.
//
// Check: after Generate(size) via each path, read 16 bytes from the same
// seeded PRNG and compare.
func TestBatchedVsSerial_ConsumesSamePRNGState(t *testing.T) {
	sizes := []int{1, 256, 1024, 4096, 100_000}
	seed := []byte("post-shuffle-state-check")

	for _, size := range sizes {
		t.Run("", func(t *testing.T) {
			// Batched run.
			hasher1 := sha1.NewSHA1(sha1.NewBigEndian())
			sr1 := securerandom.NewSecureRandom(hasher1)
			if err := sr1.Seed(seed); err != nil {
				t.Fatalf("seed: %v", err)
			}
			fy1 := NewFisherYates()
			if _, err := fy1.Generate(size, sr1); err != nil {
				t.Fatalf("batched Generate: %v", err)
			}
			afterBatched := sr1.NextBytes(16)

			// Serial run via hiding wrapper.
			hasher2 := sha1.NewSHA1(sha1.NewBigEndian())
			sr2 := securerandom.NewSecureRandom(hasher2)
			if err := sr2.Seed(seed); err != nil {
				t.Fatalf("seed: %v", err)
			}
			fy2 := NewFisherYates()
			wrapped := &nonBulkSource{inner: sr2}
			if _, err := fy2.Generate(size, wrapped); err != nil {
				t.Fatalf("serial Generate: %v", err)
			}
			afterSerial := sr2.NextBytes(16)

			if !slices.Equal(afterBatched, afterSerial) {
				t.Fatalf("size=%d: post-shuffle PRNG state diverges\n  after batched: % x\n  after serial:  % x",
					size, afterBatched, afterSerial)
			}
		})
	}
}

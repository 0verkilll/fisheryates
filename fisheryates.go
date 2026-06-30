package fisheryates

import (
	"errors"
	"fmt"

	"github.com/0verkilll/f5prng"
)

// Sentinel errors for validation failures.
// Use errors.Is() to check error types.
var (
	// ErrSizeNegative is returned when size is negative.
	ErrSizeNegative = errors.New("fisheryates: size must be non-negative")

	// ErrSizeExceedsMax is returned when size exceeds MaxPermutationSize.
	ErrSizeExceedsMax = errors.New("fisheryates: size exceeds MaxPermutationSize")
)

// newSizeNegativeError returns a translated error for negative size.
func newSizeNegativeError() error {
	msg := translate("error.size_negative", "size must be non-negative")
	return fmt.Errorf("%s: %w", msg, ErrSizeNegative)
}

// newSizeExceedsMaxError returns a translated error for size exceeding max.
func newSizeExceedsMaxError() error {
	msg := translate("error.size_exceeds_max", "size exceeds MaxPermutationSize (100M elements, ~800MB)")
	return fmt.Errorf("%s: %w", msg, ErrSizeExceedsMax)
}

// rejectionMask limits a uint32 to its low 31 bits. Used by the [Unbiased]
// permutator's rejection-sampling loop. Defined here so [unbiased.go] and
// any future variants can share the constant.
const rejectionMask uint32 = 0x7FFFFFFF

// rejectionRange is 2^31, the size of the positive int32 domain produced by
// f5prng.RandomSource.NextInt() after masking the sign bit.
const rejectionRange uint64 = 1 << 31

// batchBytes is the size of the pre-fetched PRNG byte buffer used by the
// [Unbiased] hot shuffle loop when the RandomSource supports NextBytesInto.
// Each NextInt-equivalent draw consumes 4 bytes, so one refill serves up to
// batchBytes/4 = 4096 draws. Sized to:
//   - amortise the interface call overhead across ~4K iterations;
//   - fit comfortably in L1D (16 KiB here) alongside the permutation working
//     set;
//   - match typical L1 cache line granularity multiples.
const batchBytes = 16 * 1024

// batchMinSize is the smallest permutation size at which the batched shuffle
// path beats the serial NextInt path on the securerandom + SHA-1 reference
// PRNG. Below this, the fixed cost of allocating and initially filling the
// batch buffer outweighs the amortised per-draw savings.
const batchMinSize = 256

// FisherYates implements the Fisher-Yates shuffle using the Java reference
// algorithm — the same biased-modulo reduction used by the F5.jar / Android
// PixelKnot embed pipeline.
//
// # Default semantics (since fisheryates v2)
//
// This permutator is byte-compatible with the reference Java F5 implementation:
// the embed and extract sides agree on the shuffle order on every artifact
// produced by F5.jar, the original Java F5 standalone, and the Android
// PixelKnot app (e.g. sample.jpg). If you are decoding any of those, the
// default constructor [NewFisherYates] is what you want.
//
// # Algorithm
//
//	Initialize perm = [0, 1, 2, ..., size-1]
//	For maxRandom = size; maxRandom > 0; maxRandom-- :
//	    var2 = int(NextInt())                  // 4 PRNG bytes, sign-extended
//	    idx  = var2 % maxRandom
//	    if idx < 0 { idx += maxRandom }        // Java's signed-mod fixup
//	    Swap perm[idx] with perm[maxRandom-1]
//
// The plain `int(NextInt()) % maxRandom` reduction carries a small modulo
// bias toward low indices — about maxRandom / 2^31, which is statistical
// noise for typical F5 sizes (< 0.005% at maxRandom = 1e5). For new code
// that wants strictly-uniform draws, use [NewUnbiased] instead.
//
// # Determinism and concurrency
//
// The struct is stateless. Generate / GenerateInto calls on the same
// instance are safe to run concurrently as long as each call uses its own
// f5prng.RandomSource (sources are documented as not thread-safe).
//
// Same seed ⇒ same permutation, always.
//
// # Migration from earlier versions
//
// In versions prior to v2, [NewFisherYates] returned a strictly-uniform
// (rejection-sampled) shuffle. Code that relied on that behaviour should
// switch to [NewUnbiased]:
//
//	// before v2:
//	p := fisheryates.NewFisherYates()
//
//	// v2+ (preserve unbiased behaviour):
//	p := fisheryates.NewUnbiased()
//
// Code that decodes PixelKnot / F5.jar artifacts should use the default
// (it now works out of the box; previously it required the now-deprecated
// [NewJavaCompat]).
type FisherYates struct{}

// NewFisherYates returns a Permutator that reproduces Java's reference
// biased Fisher-Yates shuffle (the F5/PixelKnot wire format).
//
// See [FisherYates] for the algorithm, determinism guarantees, and the
// migration note for callers upgrading from versions where this constructor
// returned the rejection-sampled variant.
func NewFisherYates() Permutator {
	return &FisherYates{}
}

// shuffle runs the in-place Java reference shuffle on perm using random.
// Consumes 4 bytes (one NextInt) per swap; total PRNG bytes consumed equals
// 4 * len(perm). This consumption rate is part of the byte-compatible
// contract — downstream PRNG users (e.g. the F5 header XOR) rely on the
// PRNG state having advanced exactly this far before they read.
//
// # Why we use NextInt and not NextBytesInto
//
// Java's reference assembles the int from four GetNextByte() calls in
// little-endian byte order with sign-extending int8 casts. f5prng.NextInt
// implements that exact assembly (see f5prng/securerandom.go: NextInt is
// documented as load-bearing for Java byte-stream parity). Calling
// NextBytesInto here would still consume the same 4 bytes, but we'd have
// to re-do the assembly inside this function, duplicating logic that
// f5prng already pins via determinism tests. NextInt is the contract.
func (fy *FisherYates) shuffle(perm []int, random f5prng.RandomSource) {
	maxRandom := len(perm)
	for range len(perm) {
		idx := int(random.NextInt()) % maxRandom
		if idx < 0 {
			idx += maxRandom
		}
		maxRandom--
		perm[idx], perm[maxRandom] = perm[maxRandom], perm[idx]
	}
}

// Generate produces a Java-bias-compatible permutation of [0, size).
// Returns an empty slice for size == 0; an error for size < 0 or
// size > MaxPermutationSize.
func (fy *FisherYates) Generate(size int, random f5prng.RandomSource) ([]int, error) {
	log().Debug("Generate called", "size", size)

	if size < 0 {
		return nil, newSizeNegativeError()
	}
	if size == 0 {
		log().Debug("Generate completed", "size", 0)
		return []int{}, nil
	}
	if size > MaxPermutationSize {
		return nil, newSizeExceedsMaxError()
	}

	perm := make([]int, size)
	for i := range size {
		perm[i] = i
	}
	fy.shuffle(perm, random)
	log().Debug("Generate completed", "size", size)
	return perm, nil
}

// GenerateInto produces a Java-bias-compatible permutation of [0, size) into
// the supplied buffer. The buffer is grown if it cannot hold size elements;
// the returned slice may share storage with buf or be a fresh allocation.
//
// This is the zero-allocation variant for hot paths (e.g. password
// brute-force) where the caller pools permutation buffers across calls.
func (fy *FisherYates) GenerateInto(buf []int, size int, random f5prng.RandomSource) ([]int, error) {
	log().Debug("GenerateInto called", "size", size, "buf_cap", cap(buf))

	if size < 0 {
		return nil, newSizeNegativeError()
	}
	if size == 0 {
		log().Debug("GenerateInto completed", "size", 0)
		return buf[:0], nil
	}
	if size > MaxPermutationSize {
		return nil, newSizeExceedsMaxError()
	}

	if cap(buf) < size {
		buf = make([]int, size)
	} else {
		buf = buf[:size]
	}
	for i := range size {
		buf[i] = i
	}
	fy.shuffle(buf, random)
	log().Debug("GenerateInto completed", "size", size)
	return buf, nil
}

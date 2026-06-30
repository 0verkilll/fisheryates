package fisheryates

import (
	"sync/atomic"

	"github.com/0verkilll/f5prng"
)

// Unbiased implements the Fisher-Yates shuffle with strictly uniform draws
// via rejection sampling over the 31-bit positive int32 range.
//
// # When to use
//
//   - You are creating new artifacts and want statistically unbiased
//     permutations. The default [FisherYates] (which matches Java's biased
//     reference) carries a tiny modulo bias toward low indices — about
//     maxRandom / 2^31 (≈0.005% at size 1e5, ≈4.7% at size 1e8). Bias is
//     irrelevant for steganographic shuffling, but real for cryptographic
//     or simulation use.
//
//   - You need the [Unbiased.Rejections] counter for diagnostics.
//
// For decoding any artifact embedded by the reference Java F5 implementation
// (F5.jar, PixelKnot, Android port), use the default [FisherYates] which
// reproduces Java's biased modulo byte-for-byte.
//
// # Algorithm
//
//	Initialize perm = [0, 1, 2, ..., size-1]
//	For maxRandom = size; maxRandom > 0; maxRandom-- :
//	    Repeat: u = uint32(NextInt()) & 0x7FFFFFFF
//	            until u < (2^31 - 2^31 % maxRandom)
//	    idx = u % maxRandom
//	    Swap perm[idx] with perm[maxRandom-1]
//
// Determinism per seed is preserved (same seed ⇒ same permutation).
// Rejected draws are counted in [Unbiased.Rejections].
type Unbiased struct {
	// rejections counts the number of NextInt() draws that were discarded
	// by the rejection-sampling loop. Useful for observability when comparing
	// against the biased Java reference. Accessed atomically so concurrent
	// Generate/GenerateInto calls on the same instance do not race on it.
	rejections uint64
}

// NewUnbiased returns a Permutator that uses rejection sampling for strictly
// uniform shuffles. See [Unbiased] for trade-offs vs. the default
// [FisherYates].
func NewUnbiased() Permutator {
	return &Unbiased{}
}

// Rejections returns the cumulative number of uniform-rejection retries
// performed by this permutator across all Generate/GenerateInto calls.
//
// This is intended for diagnostics: if the value grows faster than roughly
// size * (maxRandom / 2^31) per call, something is likely wrong with the
// RandomSource. In typical F5 usage (maxRandom <= MaxPermutationSize, i.e.
// <= 1e8) the expected rate is well under 5% of draws.
func (u *Unbiased) Rejections() uint64 {
	return atomic.LoadUint64(&u.rejections)
}

// unbiasedMod draws a uniform integer in [0, maxRandom) from r.NextInt()
// using rejection sampling over the 31-bit positive range of int32.
//
// Rationale: int(r.NextInt()) % maxRandom is biased because 2^31 is not in
// general divisible by maxRandom. The low (2^31 mod maxRandom) residues
// receive one extra draw compared to the rest, producing a modulo bias of
// roughly maxRandom / 2^31. At maxRandom = 100M this is ~4.7%; at 1e5 it is
// ~0.005%. We eliminate the bias by rejecting draws that fall in the tail
// above the largest multiple of maxRandom that fits in 2^31.
//
// Precondition: maxRandom > 0 and maxRandom <= MaxPermutationSize (1e8), so
// 2 * maxRandom < 2^31 and the expected number of retries per call is < 1.
//
// Note: Generate/GenerateInto inline this logic into their hot loops for
// performance. This method is retained for direct testing and for callers
// that want the helper as a public primitive.
func (u *Unbiased) unbiasedMod(r f5prng.RandomSource, maxRandom int) int {
	m := uint32(maxRandom) //nolint:gosec // bounded by MaxPermutationSize (1e8) < 2^31
	threshold := uint32(rejectionRange - (rejectionRange % uint64(m))) //nolint:gosec // result fits in uint32
	var rejected uint64
	for {
		v := uint32(r.NextInt()) & rejectionMask
		if v < threshold {
			if rejected != 0 {
				atomic.AddUint64(&u.rejections, rejected)
			}
			return int(v % m)
		}
		rejected++
	}
}

// shuffleBatched is the bulk-fill rejection-sampled hot path. See
// [batchBytes], [batchMinSize], and the original FisherYates implementation
// for the rationale on batch sizing and the sign-extending byte assembly.
func (u *Unbiased) shuffleBatched(perm []int, bulk f5prng.RandomSourceWithBytesInto) {
	n := len(perm)
	_ = perm[n-1]

	var bufArr [batchBytes]byte
	var buf []byte
	pos := 0

	maxRandom := uint32(n) //nolint:gosec // bounded by MaxPermutationSize < 2^31
	var rejected uint64
	for maxRandom > 0 {
		threshold := uint32(rejectionRange - (rejectionRange % uint64(maxRandom))) //nolint:gosec // fits in uint32
		var randomIndex uint32
		for {
			if pos+4 > len(buf) {
				want := batchBytes
				if need := int(maxRandom) * 4; need < want {
					want = need
				}
				buf = bufArr[:want]
				_ = bulk.NextBytesInto(buf)
				pos = 0
			}
			b := buf[pos : pos+4 : pos+4]
			pos += 4
			s0 := int(int8(b[0]))
			s1 := int(int8(b[1]))
			s2 := int(int8(b[2]))
			s3 := int(int8(b[3]))
			val := s0 | (s1 << 8) | (s2 << 16) | (s3 << 24)
			v := uint32(int32(val)) & rejectionMask //nolint:gosec // intentional bit pattern
			if v < threshold {
				randomIndex = v % maxRandom
				break
			}
			rejected++
		}

		maxRandom--
		perm[randomIndex], perm[maxRandom] = perm[maxRandom], perm[randomIndex]
	}
	if rejected != 0 {
		atomic.AddUint64(&u.rejections, rejected)
	}
}

// shuffleSerial is the per-draw rejection-sampled fallback for callers that
// don't supply NextBytesInto, and for permutations smaller than [batchMinSize].
func (u *Unbiased) shuffleSerial(perm []int, random f5prng.RandomSource) {
	n := len(perm)
	_ = perm[n-1]

	maxRandom := uint32(n) //nolint:gosec // bounded by MaxPermutationSize < 2^31
	var rejected uint64
	for maxRandom > 0 {
		threshold := uint32(rejectionRange - (rejectionRange % uint64(maxRandom))) //nolint:gosec // fits in uint32
		var randomIndex uint32
		for {
			v := uint32(random.NextInt()) & rejectionMask //nolint:gosec // intentional
			if v < threshold {
				randomIndex = v % maxRandom
				break
			}
			rejected++
		}
		maxRandom--
		perm[randomIndex], perm[maxRandom] = perm[maxRandom], perm[randomIndex]
	}
	if rejected != 0 {
		atomic.AddUint64(&u.rejections, rejected)
	}
}

func (u *Unbiased) shuffleInto(perm []int, random f5prng.RandomSource) {
	if len(perm) == 0 {
		return
	}
	if len(perm) >= batchMinSize {
		if bulk, ok := random.(f5prng.RandomSourceWithBytesInto); ok {
			u.shuffleBatched(perm, bulk)
			return
		}
	}
	u.shuffleSerial(perm, random)
}

// Generate produces a strictly-uniform permutation of [0, size) using
// rejection sampling. See [Unbiased] for the algorithm and trade-offs vs.
// the Java-biased default.
func (u *Unbiased) Generate(size int, random f5prng.RandomSource) ([]int, error) {
	log().Debug("Unbiased.Generate called", "size", size)

	if size < 0 {
		return nil, newSizeNegativeError()
	}
	if size == 0 {
		return []int{}, nil
	}
	if size > MaxPermutationSize {
		return nil, newSizeExceedsMaxError()
	}

	perm := make([]int, size)
	for i := range size {
		perm[i] = i
	}
	u.shuffleInto(perm, random)
	return perm, nil
}

// GenerateInto is the buffer-reusing variant of [Unbiased.Generate].
func (u *Unbiased) GenerateInto(buf []int, size int, random f5prng.RandomSource) ([]int, error) {
	log().Debug("Unbiased.GenerateInto called", "size", size, "buf_cap", cap(buf))

	if size < 0 {
		return nil, newSizeNegativeError()
	}
	if size == 0 {
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
	u.shuffleInto(buf, random)
	return buf, nil
}

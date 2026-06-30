package fisheryates

import (
	"errors"
	"fmt"

	"github.com/0verkilll/f5prng"
)

// ErrKnownHeadTooLong is returned when knownHead is longer than size.
var ErrKnownHeadTooLong = errors.New("fisheryates: knownHead length exceeds size")

// defaultFY is the shared stateless instance used by package-level functions.
// FisherYates carries no mutable state, so it is safe for concurrent use.
var defaultFY FisherYates

// CompleteFromKnownHead is a package-level shorthand that avoids a type
// assertion when callers only need PermPair and do not hold a FisherYates
// instance.  Semantics are identical to [FisherYates.CompleteFromKnownHead].
func CompleteFromKnownHead(size int, random f5prng.RandomSource, knownHead []int) (*PermPair, error) {
	return defaultFY.CompleteFromKnownHead(size, random, knownHead)
}

// GenerateWithInverse is a package-level shorthand for
// [FisherYates.GenerateWithInverse].
func GenerateWithInverse(size int, random f5prng.RandomSource) (*PermPair, error) {
	return defaultFY.GenerateWithInverse(size, random)
}

// TailOf returns the last k elements of sigma — a convenience for extracting
// the sigmaTail argument to [EarlyRejectFromTail] once you have a known-good
// permutation from a prior [CompleteFromKnownHead] call.
func TailOf(sigma []int, k int) []int {
	if k <= 0 || len(sigma) == 0 {
		return nil
	}
	if k > len(sigma) {
		k = len(sigma)
	}
	out := make([]int, k)
	copy(out, sigma[len(sigma)-k:])
	return out
}

// PermPair holds both σ and σ⁻¹ of a Fisher-Yates shuffle produced in a single
// PRNG pass via the reverse-digit theorem. Obtain one from
// [FisherYates.GenerateWithInverse] or [FisherYates.CompleteFromKnownHead].
type PermPair struct {
	// Forward is σ: the shuffled permutation.
	// Forward[i] is the original index of the element placed at shuffled position i.
	// Apply it to map original-order data into shuffled order:
	//   shuffled[i] = original[Forward[i]]
	Forward []int

	// Inverse is σ⁻¹: the extraction map.
	// Inverse[i] is the shuffled position of original element i.
	// Apply it to recover original order from shuffled data:
	//   original[i] = shuffled[Inverse[i]]
	Inverse []int

	// HeadMatch reports whether every element of the knownHead argument to
	// [FisherYates.CompleteFromKnownHead] matched the corresponding element
	// of Forward. Always true when knownHead was nil or empty.
	HeadMatch bool

	// MatchCount is the number of knownHead positions that matched Forward.
	// Equals len(knownHead) when HeadMatch is true.
	MatchCount int
}

// GenerateWithInverse produces the complete σ and σ⁻¹ in a single PRNG pass.
//
// Compared to [FisherYates.Generate] followed by a separate O(N) inversion loop,
// this eliminates one allocation and one full scan when both directions are needed
// (e.g. an F5 embed pass followed by an extract pass over the same permutation).
//
// The PRNG is consumed byte-for-byte identically to [FisherYates.Generate], so the
// returned Forward slice is always equal to what Generate would return for the same
// seed and size.
func (fy *FisherYates) GenerateWithInverse(size int, random f5prng.RandomSource) (*PermPair, error) {
	return fy.CompleteFromKnownHead(size, random, nil)
}

// CompleteFromKnownHead generates the full σ and σ⁻¹ and optionally validates a
// known prefix of σ.
//
// Parameters:
//   - size:      total permutation length N (>= 0, <= [MaxPermutationSize])
//   - random:    PRNG source pre-seeded by the caller
//   - knownHead: the caller's known values of σ[0..len(knownHead)-1]; pass nil
//     (or an empty slice) to skip validation
//
// Typical F5 usage: the caller knows the first K shuffled coefficient positions
// from a partially decoded stego image, passes them as knownHead, and uses
// PermPair.HeadMatch to confirm the candidate PRNG seed before trusting the rest
// of the permutation. On a match, PermPair.Forward completes the full sequence and
// PermPair.Inverse provides the extraction map.
//
// The reverse-digit theorem guarantees correctness of Inverse:
//
//	FY = T_{N-1} ∘ ⋯ ∘ T_0   (each T_i is a transposition, self-inverse)
//	FY⁻¹ = T_0 ∘ ⋯ ∘ T_{N-1}  (same swaps, reversed order)
//
// Memory cost: 3 × size × 8 bytes (digits buffer + Forward + Inverse).
func (fy *FisherYates) CompleteFromKnownHead(size int, random f5prng.RandomSource, knownHead []int) (*PermPair, error) {
	log().Debug("CompleteFromKnownHead called", "size", size, "head_len", len(knownHead))

	if size < 0 {
		return nil, newSizeNegativeError()
	}
	if size > MaxPermutationSize {
		return nil, newSizeExceedsMaxError()
	}
	if len(knownHead) > size {
		return nil, ErrKnownHeadTooLong
	}
	if size == 0 {
		return &PermPair{
			Forward:    []int{},
			Inverse:    []int{},
			HeadMatch:  true,
			MatchCount: 0,
		}, nil
	}

	// Phase 1: consume all N PRNG values, recording each step's pool index.
	// digit[i] is the swap index when maxRandom = size−i, matching the byte
	// consumption order of FisherYates.shuffle() exactly.
	digits := make([]int, size)
	maxRandom := size
	for i := range size {
		v := int(random.NextInt()) % maxRandom
		if v < 0 {
			v += maxRandom
		}
		digits[i] = v
		maxRandom--
	}

	// Phase 2: forward pass — apply swaps 0..N−1 to produce σ.
	fwd := make([]int, size)
	for i := range size {
		fwd[i] = i
	}
	for i, idx := range digits {
		last := size - 1 - i
		fwd[idx], fwd[last] = fwd[last], fwd[idx]
	}

	// Phase 3: validate the known head.
	headMatch := true
	matchCount := 0
	for i, want := range knownHead {
		if fwd[i] == want {
			matchCount++
		} else {
			headMatch = false
		}
	}

	// Phase 4: reverse pass — apply swaps N−1..0 to produce σ⁻¹.
	inv := make([]int, size)
	for i := range size {
		inv[i] = i
	}
	for i := size - 1; i >= 0; i-- {
		last := size - 1 - i
		inv[digits[i]], inv[last] = inv[last], inv[digits[i]]
	}

	log().Debug("CompleteFromKnownHead completed",
		"size", size, "head_match", headMatch, "match_count", matchCount)

	return &PermPair{
		Forward:    fwd,
		Inverse:    inv,
		HeadMatch:  headMatch,
		MatchCount: matchCount,
	}, nil
}

// EarlyRejectResult is returned by [EarlyRejectFromTail].
type EarlyRejectResult struct {
	// Rejected is true if the candidate PRNG stream cannot have produced
	// the expected tail positions.  A rejected candidate can be discarded
	// immediately — the rest of the permutation is guaranteed wrong.
	Rejected bool

	// StepsConsumed is the number of PRNG calls that were read.
	// On rejection this is the step where the mismatch occurred (1-based).
	// On acceptance it equals len(sigmaTail).
	StepsConsumed int
}

// EarlyRejectFromTail is the fast-rejection oracle for seed brute-forcing.
//
// # Background
//
// The Fisher-Yates shuffle settles positions from RIGHT to LEFT:
//
//	PRNG call 0 → digit[0] → settles σ[N-1]   (1 call to check)
//	PRNG call 1 → digit[1] → settles σ[N-2]   (2 calls to check)
//	…
//	PRNG call K → digit[K] → settles σ[N-1-K]
//	PRNG call N → digit[N] → settles σ[0]     (ALL N calls to check)
//
// If you know the TAIL of σ (σ[N-K..N-1]), you can verify or reject a
// candidate PRNG stream after just K calls — ~N/K times faster than a full
// run.  For N=240 000 and K=32, that is a 7 500× speed-up for rejected
// candidates.  Since the probability that a wrong candidate passes K
// independent checks is ≈ 1/N^K ≈ 0, the survivors are almost certainly
// correct.
//
// # Contrast with CompleteFromKnownHead
//
//   - [CompleteFromKnownHead] checks σ[0..K-1] (the HEAD).
//     The head is settled LAST, so all N PRNG calls must be read first.
//     Use it when you need the full permutation or when the head is all
//     you have.
//
//   - EarlyRejectFromTail checks σ[N-K..N-1] (the TAIL).
//     The tail is settled FIRST; each mismatch aborts after i+1 calls.
//     Use it as a cheap pre-filter before the full CompleteFromKnownHead.
//
// # Parameters
//
//   - sigmaTail: the last len(sigmaTail) elements of σ in order,
//     i.e. sigmaTail[0]=σ[N-K], …, sigmaTail[K-1]=σ[N-1].
//     Typically K=1 is enough for a 240 000-element shuffle (false-positive
//     rate ≈ 1/240 000 per candidate).  K=8 gives ≈ 1/240 000^8.
//   - size: the total permutation length N.
//   - random: a freshly seeded PRNG for the candidate password.
//     On rejection random has been read StepsConsumed times.
//     On acceptance random has been read len(sigmaTail) times and is
//     positioned for a subsequent CompleteFromKnownHead call (re-seed first)
//     or further PRNG reads.
//
// EarlyRejectFromTail does not validate that sigmaTail is a valid subset of
// [0,N); the caller is responsible for that.
func EarlyRejectFromTail(sigmaTail []int, size int, random f5prng.RandomSource) EarlyRejectResult {
	K := len(sigmaTail)
	if K == 0 || size == 0 {
		return EarlyRejectResult{Rejected: false, StepsConsumed: 0}
	}

	// Mirror the position-tracking state from RecoverDigits:
	// posOf[v] = current position of element v.
	// valAt[p] = element at position p.
	// Both start as the identity.
	posOf := make([]int, size)
	valAt := make([]int, size)
	for i := range size {
		posOf[i] = i
		valAt[i] = i
	}

	for i := 0; i < K; i++ {
		// Step i settles position N-1-i with the element sigmaTail[K-1-i].
		target := sigmaTail[K-1-i]
		expectedDigit := posOf[target] // what the digit MUST be

		maxRandom := size - i
		v := int(random.NextInt()) % maxRandom
		if v < 0 {
			v += maxRandom
		}

		if v != expectedDigit {
			return EarlyRejectResult{Rejected: true, StepsConsumed: i + 1}
		}

		// Simulate the swap so subsequent steps see the right pool state.
		p1, p2 := expectedDigit, maxRandom-1
		if p1 != p2 {
			displaced := valAt[p2]
			valAt[p2] = target
			valAt[p1] = displaced
			posOf[target] = p2
			posOf[displaced] = p1
		}
	}

	return EarlyRejectResult{Rejected: false, StepsConsumed: K}
}

// RecoverDigits recovers the complete Fisher-Yates digit sequence from a
// permutation σ that was produced by [FisherYates.Generate] (or the Forward
// field of a [PermPair]).
//
// digit[i] is the pool index selected at FY step i — the value that would be
// computed as:
//
//	int(PRNG.NextInt()) % (N - i)   (with Java signed-mod fixup)
//
// This is the full "walk": the concrete sequence of N choices that drove the
// shuffle. Knowing the walk lets you:
//
//   - Reconstruct the exact PRNG outputs at each step (PRNG_int32[i] ≡ digit[i]
//     (mod N−i); the residue narrows the int32 domain ~N−i-fold per step).
//   - Verify that two separately-seeded PRNG streams produced the same shuffle.
//   - Implement custom embedding / extraction schemes indexed by step index
//     rather than permutation position.
//
// To recover the walk from partial knowledge (only σ[0..K-1] known): you need
// the seed first, because the last K digits depend on the pool arrangement at
// step N−K, which itself depends on the first N−K digits.  Use
// [FisherYates.CompleteFromKnownHead] to verify a candidate seed with your
// known head, then call RecoverDigits on the resulting Forward slice.
//
// Time: O(N).  Space: O(N).
func RecoverDigits(sigma []int) ([]int, error) {
	N := len(sigma)
	if N == 0 {
		return []int{}, nil
	}

	seen := make([]bool, N)
	for _, v := range sigma {
		if v < 0 || v >= N {
			return nil, fmt.Errorf("fisheryates: element %d out of range [0,%d)", v, N)
		}
		if seen[v] {
			return nil, fmt.Errorf("fisheryates: duplicate element %d in permutation", v)
		}
		seen[v] = true
	}

	// posOf[v] = current position of element v in the simulated perm array.
	// valAt[p] = element currently at position p.
	// Both start as the identity (matching FY's initial perm = [0..N-1]).
	posOf := make([]int, N)
	valAt := make([]int, N)
	for i := range N {
		posOf[i] = i
		valAt[i] = i
	}

	digits := make([]int, N)
	for i := range N {
		// FY step i settles position N-1-i with the element currently there.
		// The digit used at step i is the current position of sigma[N-1-i] —
		// i.e., which pool slot was selected before the swap.
		target := sigma[N-1-i]
		p1 := posOf[target] // ← this IS digit[i]
		p2 := N - 1 - i    // position being settled

		digits[i] = p1

		if p1 != p2 {
			displaced := valAt[p2]
			valAt[p2] = target
			valAt[p1] = displaced
			posOf[target] = p2
			posOf[displaced] = p1
		}
	}

	return digits, nil
}

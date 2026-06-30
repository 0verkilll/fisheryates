package fisheryates

import (
	"context"
	"sync"

	"github.com/0verkilll/f5prng"
)

// ScanResult is returned by [ScanCandidates] when a match is found.
type ScanResult struct {
	// Index is the position of the matching candidate in the input slice.
	Index int

	// Pair holds the complete σ (Forward) and σ⁻¹ (Inverse) for the
	// matching candidate, produced in a single PRNG pass via the
	// Reverse-Digit Inverse Theorem.
	//
	//   Pair.Forward[i]  = original coefficient index at embed position i
	//   Pair.Inverse[i]  = embed position of original coefficient i
	//
	// These are all N elements — the complete walk.
	Pair *PermPair
}

// ScanCandidates runs CompleteFromKnownHead against each candidate PRNG in
// parallel and returns the first (and in practice only) candidate whose first
// len(knownHead) positions of σ match knownHead exactly.
//
// # Typical F5 usage
//
// You have N = number of non-zero AC DCT coefficients, knownHead = the first
// K embed positions of σ (σ[0..K-1]) observed from the stego image, and a
// list of candidate PRNG sources — one per candidate password, each freshly
// seeded.
//
// ScanCandidates tries all candidates concurrently.  The match is
// cryptographically certain: for K=32 and N=240 000, the probability that a
// wrong candidate passes is ≈ (1/N)^32 ≈ 0.  The matching ScanResult gives
// you the complete walk (Pair.Forward, all N elements) and the extraction map
// (Pair.Inverse) without any further PRNG work.
//
// # Parameters
//
//   - ctx: cancellation; cancelled after the first match to stop remaining workers.
//   - size: total permutation length N.
//   - knownHead: σ[0..len(knownHead)-1] that the correct candidate must reproduce.
//   - candidates: one RandomSource per candidate, freshly seeded.
//     Each source is consumed and cleared by this call; do not reuse them.
//   - workers: goroutine concurrency (0 = len(candidates), i.e. all at once).
//     For 1340 candidates on a modern CPU, workers=8 is a good default.
//
// Returns (result, true) on match, (zero, false) if no candidate matches.
func ScanCandidates(
	ctx context.Context,
	size int,
	knownHead []int,
	candidates []f5prng.RandomSource,
	workers int,
) (ScanResult, bool) {
	if len(candidates) == 0 {
		return ScanResult{}, false
	}
	if workers <= 0 || workers > len(candidates) {
		workers = len(candidates)
	}

	type indexedSource struct {
		idx int
		src f5prng.RandomSource
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	work := make(chan indexedSource, workers)
	resultCh := make(chan ScanResult, 1)
	var wg sync.WaitGroup

	// Feed candidates into the work channel.
	go func() {
		defer close(work)
		for i, src := range candidates {
			select {
			case <-ctx.Done():
				return
			case work <- indexedSource{i, src}:
			}
		}
	}()

	// Worker pool: each worker calls CompleteFromKnownHead and checks HeadMatch.
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range work {
				select {
				case <-ctx.Done():
					return
				default:
				}

				pair, err := defaultFY.CompleteFromKnownHead(size, item.src, knownHead)
				item.src.Clear()
				if err != nil || !pair.HeadMatch {
					continue
				}

				// First match wins — cancel all other workers.
				select {
				case resultCh <- ScanResult{Index: item.idx, Pair: pair}:
					cancel()
				default:
				}
				return
			}
		}()
	}

	// Close resultCh once all workers finish so the receive below unblocks.
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	result, found := <-resultCh
	return result, found
}

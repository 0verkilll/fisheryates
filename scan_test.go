package fisheryates

import (
	"context"
	"testing"

	"github.com/0verkilll/f5prng"
)

// makeSrcs builds a slice of RandomSources: all decoys except correctIdx
// which uses correctSeed.
func makeSrcs(total, correctIdx int, correctSeed uint64) []f5prng.RandomSource {
	srcs := make([]f5prng.RandomSource, total)
	for i := range total {
		if i == correctIdx {
			srcs[i] = newSplitMix64Source(correctSeed)
		} else {
			srcs[i] = newSplitMix64Source(uint64(i)*0x9E3779B97F4A7C15 + 1)
		}
	}
	return srcs
}

// TestScanCandidates_FindsMatch verifies the correct candidate is found among
// many wrong ones and that the returned walk is complete and correct.
func TestScanCandidates_FindsMatch(t *testing.T) {
	fy := NewFisherYates().(*FisherYates)

	const (
		size       = 50_000
		headLen    = 32
		total      = 1340
		correctIdx = 701
		correctSeed = uint64(0xF5CAFEBEEFDEAD00)
	)

	// Build ground-truth permutation.
	truth, _ := fy.Generate(size, newSplitMix64Source(correctSeed))
	knownHead := make([]int, headLen)
	copy(knownHead, truth[:headLen])

	result, found := ScanCandidates(
		context.Background(),
		size,
		knownHead,
		makeSrcs(total, correctIdx, correctSeed),
		8,
	)
	if !found {
		t.Fatal("ScanCandidates: no match found")
	}
	if result.Index != correctIdx {
		t.Errorf("matched index=%d, want %d", result.Index, correctIdx)
	}

	// Complete walk must match ground truth exactly.
	if len(result.Pair.Forward) != size {
		t.Fatalf("Forward len=%d, want %d", len(result.Pair.Forward), size)
	}
	for i := range size {
		if result.Pair.Forward[i] != truth[i] {
			t.Errorf("Forward[%d]=%d, want %d", i, result.Pair.Forward[i], truth[i])
		}
	}

	// σ⁻¹ correct (Reverse-Digit Theorem): σ(σ⁻¹(i)) = i for all i.
	for i := range size {
		if result.Pair.Forward[result.Pair.Inverse[i]] != i {
			t.Errorf("σ(σ⁻¹(%d)) ≠ %d", i, i)
		}
	}
}

// TestScanCandidates_NoMatch verifies clean behaviour when nothing matches.
func TestScanCandidates_NoMatch(t *testing.T) {
	const size = 1000
	knownHead := []int{999, 998, 997} // no random seed will produce this head

	srcs := make([]f5prng.RandomSource, 50)
	for i := range srcs {
		srcs[i] = newSplitMix64Source(uint64(i) + 1)
	}
	_, found := ScanCandidates(context.Background(), size, knownHead, srcs, 4)
	if found {
		t.Error("expected no match; got one")
	}
}

// TestScanCandidates_ContextCancel verifies early cancellation works cleanly.
func TestScanCandidates_ContextCancel(t *testing.T) {
	const size = 10_000
	srcs := make([]f5prng.RandomSource, 500)
	for i := range srcs {
		srcs[i] = newSplitMix64Source(uint64(i) + 1)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel
	_, found := ScanCandidates(ctx, size, nil, srcs, 8)
	// may or may not find (depends on timing) — just must not panic or deadlock
	_ = found
}

// BenchmarkScanCandidates_1340 mirrors the real scenario:
// 1340 candidates, N=240K, 32 known head positions, 8 workers.
func BenchmarkScanCandidates_1340(b *testing.B) {
	fy := NewFisherYates().(*FisherYates)
	const (
		size        = 240_000
		headLen     = 32
		total       = 1340
		correctIdx  = 1200
		correctSeed = uint64(0xBEEFCAFE12345678)
	)

	truth, _ := fy.Generate(size, newSplitMix64Source(correctSeed))
	knownHead := make([]int, headLen)
	copy(knownHead, truth[:headLen])

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		srcs := make([]f5prng.RandomSource, total)
		for i := range total {
			if i == correctIdx {
				srcs[i] = newSplitMix64Source(correctSeed)
			} else {
				srcs[i] = newSplitMix64Source(uint64(n*total+i) + 1)
			}
		}
		result, found := ScanCandidates(context.Background(), size, knownHead, srcs, 8)
		if !found {
			b.Fatal("no match")
		}
		_ = result
	}
}

package fisheryates

import (
	"errors"
	"testing"
)

// TestPermPair_Correctness verifies that PermPair.Forward and PermPair.Inverse are
// true inverses of each other: σ(σ⁻¹(i)) = i and σ⁻¹(σ(i)) = i for all i.
func TestPermPair_Correctness(t *testing.T) {
	fy := NewFisherYates().(*FisherYates)

	for _, size := range []int{1, 2, 7, 8, 32, 100, 1000, 10000} {
		for seed := uint64(0); seed < 5; seed++ {
			src := newSplitMix64Source(seed*0x9E3779B97F4A7C15 + 1)
			pair, err := fy.GenerateWithInverse(size, src)
			if err != nil {
				t.Fatalf("size=%d seed=%d: GenerateWithInverse error: %v", size, seed, err)
			}

			if len(pair.Forward) != size {
				t.Fatalf("size=%d: Forward len=%d", size, len(pair.Forward))
			}
			if len(pair.Inverse) != size {
				t.Fatalf("size=%d: Inverse len=%d", size, len(pair.Inverse))
			}

			// σ(σ⁻¹(i)) == i
			for i := range size {
				if pair.Forward[pair.Inverse[i]] != i {
					t.Errorf("size=%d seed=%d: σ(σ⁻¹(%d)) = σ(%d) = %d ≠ %d",
						size, seed, i, pair.Inverse[i], pair.Forward[pair.Inverse[i]], i)
				}
			}

			// σ⁻¹(σ(i)) == i
			for i := range size {
				if pair.Inverse[pair.Forward[i]] != i {
					t.Errorf("size=%d seed=%d: σ⁻¹(σ(%d)) = σ⁻¹(%d) = %d ≠ %d",
						size, seed, i, pair.Forward[i], pair.Inverse[pair.Forward[i]], i)
				}
			}
		}
	}
}

// TestPermPair_ForwardMatchesGenerate verifies that PermPair.Forward is byte-for-byte
// identical to what FisherYates.Generate produces for the same seed and size.
func TestPermPair_ForwardMatchesGenerate(t *testing.T) {
	fy := NewFisherYates().(*FisherYates)

	for _, size := range []int{1, 8, 32, 256, 1000} {
		for seed := uint64(0); seed < 8; seed++ {
			src1 := newSplitMix64Source(seed*0xDEADBEEF + 3)
			src2 := newSplitMix64Source(seed*0xDEADBEEF + 3)

			ref, err := fy.Generate(size, src1)
			if err != nil {
				t.Fatalf("Generate error: %v", err)
			}
			pair, err := fy.GenerateWithInverse(size, src2)
			if err != nil {
				t.Fatalf("GenerateWithInverse error: %v", err)
			}

			for i := range size {
				if pair.Forward[i] != ref[i] {
					t.Errorf("size=%d seed=%d: Forward[%d]=%d, Generate[%d]=%d",
						size, seed, i, pair.Forward[i], i, ref[i])
				}
			}
		}
	}
}

// TestCompleteFromKnownHead_MatchingHead verifies that passing the actual first K
// elements of σ as knownHead yields HeadMatch=true and MatchCount=K.
func TestCompleteFromKnownHead_MatchingHead(t *testing.T) {
	fy := NewFisherYates().(*FisherYates)
	const (
		size    = 500
		headLen = 32
	)

	for seed := uint64(0); seed < 10; seed++ {
		// Generate the ground-truth permutation.
		src1 := newSplitMix64Source(seed * 0x1111111111111111)
		ref, _ := fy.Generate(size, src1)

		// Extract the first 32 elements as the known head.
		knownHead := make([]int, headLen)
		copy(knownHead, ref[:headLen])

		// Call CompleteFromKnownHead with the same seed — must match.
		src2 := newSplitMix64Source(seed * 0x1111111111111111)
		pair, err := fy.CompleteFromKnownHead(size, src2, knownHead)
		if err != nil {
			t.Fatalf("seed=%d: CompleteFromKnownHead error: %v", seed, err)
		}

		if !pair.HeadMatch {
			t.Errorf("seed=%d: HeadMatch=false with correct seed", seed)
		}
		if pair.MatchCount != headLen {
			t.Errorf("seed=%d: MatchCount=%d, want %d", seed, pair.MatchCount, headLen)
		}
		// Forward must equal the ground-truth permutation.
		for i := range size {
			if pair.Forward[i] != ref[i] {
				t.Errorf("seed=%d: Forward[%d]=%d, want %d", seed, i, pair.Forward[i], ref[i])
			}
		}
	}
}

// TestCompleteFromKnownHead_WrongSeed verifies that a mismatched seed yields
// HeadMatch=false and a low MatchCount.
func TestCompleteFromKnownHead_WrongSeed(t *testing.T) {
	fy := NewFisherYates().(*FisherYates)
	const (
		size    = 500
		headLen = 32
	)

	// Ground truth: seed A.
	srcA := newSplitMix64Source(0xAAAAAAAAAAAAAAAA)
	ref, _ := fy.Generate(size, srcA)
	knownHead := make([]int, headLen)
	copy(knownHead, ref[:headLen])

	// Attempt with a different seed — should not match.
	srcB := newSplitMix64Source(0xBBBBBBBBBBBBBBBB)
	pair, err := fy.CompleteFromKnownHead(size, srcB, knownHead)
	if err != nil {
		t.Fatalf("CompleteFromKnownHead error: %v", err)
	}

	if pair.HeadMatch {
		t.Error("HeadMatch=true with wrong seed — false positive")
	}
	// For independent random permutations, expected matches ≈ headLen/size = 32/500 ≈ 0.
	// A match count >= headLen/2 would be astronomically unlikely.
	if pair.MatchCount >= headLen/2 {
		t.Errorf("MatchCount=%d suspiciously high for wrong seed", pair.MatchCount)
	}
}

// TestCompleteFromKnownHead_NilHead verifies that nil knownHead behaves identically
// to GenerateWithInverse (no validation, HeadMatch=true).
func TestCompleteFromKnownHead_NilHead(t *testing.T) {
	fy := NewFisherYates().(*FisherYates)

	src1 := newSplitMix64Source(0xCAFE)
	src2 := newSplitMix64Source(0xCAFE)

	pairA, _ := fy.GenerateWithInverse(256, src1)
	pairB, _ := fy.CompleteFromKnownHead(256, src2, nil)

	if !pairB.HeadMatch {
		t.Error("HeadMatch should be true when knownHead is nil")
	}
	if pairB.MatchCount != 0 {
		t.Errorf("MatchCount=%d, want 0 when knownHead is nil", pairB.MatchCount)
	}
	for i := range 256 {
		if pairA.Forward[i] != pairB.Forward[i] {
			t.Errorf("Forward[%d]: GenerateWithInverse=%d, CompleteFromKnownHead(nil)=%d",
				i, pairA.Forward[i], pairB.Forward[i])
		}
	}
}

// TestCompleteFromKnownHead_EdgeCases covers size=0 and error paths.
func TestCompleteFromKnownHead_EdgeCases(t *testing.T) {
	fy := NewFisherYates().(*FisherYates)
	src := newSplitMix64Source(0)

	// size=0 must succeed with empty slices.
	pair, err := fy.CompleteFromKnownHead(0, src, nil)
	if err != nil {
		t.Fatalf("size=0: unexpected error: %v", err)
	}
	if len(pair.Forward) != 0 || len(pair.Inverse) != 0 {
		t.Error("size=0: expected empty Forward and Inverse")
	}

	// knownHead longer than size must return ErrKnownHeadTooLong.
	_, err = fy.CompleteFromKnownHead(4, src, []int{0, 1, 2, 3, 4})
	if !errors.Is(err, ErrKnownHeadTooLong) {
		t.Errorf("expected ErrKnownHeadTooLong, got %v", err)
	}

	// Negative size must return ErrSizeNegative.
	_, err = fy.CompleteFromKnownHead(-1, src, nil)
	if !errors.Is(err, ErrSizeNegative) {
		t.Errorf("expected ErrSizeNegative, got %v", err)
	}

	// size > MaxPermutationSize must return ErrSizeExceedsMax.
	_, err = fy.CompleteFromKnownHead(MaxPermutationSize+1, src, nil)
	if !errors.Is(err, ErrSizeExceedsMax) {
		t.Errorf("expected ErrSizeExceedsMax, got %v", err)
	}
}

// TestPermPair_ApplyToCoefficients demonstrates the intended F5 usage:
// embed using Forward, extract using Inverse.
func TestPermPair_ApplyToCoefficients(t *testing.T) {
	fy := NewFisherYates().(*FisherYates)
	const N = 128

	// Simulate N DCT coefficients (original order).
	original := make([]int, N)
	for i := range N {
		original[i] = i * 100 // arbitrary values
	}

	src := newSplitMix64Source(0xF5DECAF)
	pair, err := fy.GenerateWithInverse(N, src)
	if err != nil {
		t.Fatalf("GenerateWithInverse error: %v", err)
	}

	// Embed: apply σ to get shuffled coefficient order.
	// shuffled[i] = original[Forward[i]]
	shuffled := make([]int, N)
	for i := range N {
		shuffled[i] = original[pair.Forward[i]]
	}

	// Extract: apply σ⁻¹ to recover original order.
	// recovered[i] = shuffled[Inverse[i]]
	recovered := make([]int, N)
	for i := range N {
		recovered[i] = shuffled[pair.Inverse[i]]
	}

	// Must round-trip exactly.
	for i := range N {
		if recovered[i] != original[i] {
			t.Errorf("coefficient[%d]: original=%d, recovered=%d", i, original[i], recovered[i])
		}
	}
}

// TestEarlyRejectFromTail_Accept verifies that the correct PRNG seed is never
// rejected by EarlyRejectFromTail.
func TestEarlyRejectFromTail_Accept(t *testing.T) {
	fy := NewFisherYates().(*FisherYates)

	for _, size := range []int{8, 32, 100, 1000, 10000} {
		for seed := uint64(0); seed < 5; seed++ {
			src1 := newSplitMix64Source(seed * 0xFEDCBA9876543210)
			sigma, _ := fy.Generate(size, src1)

			for k := 1; k <= 32 && k <= size; k++ {
				tail := TailOf(sigma, k)
				src2 := newSplitMix64Source(seed * 0xFEDCBA9876543210)
				res := EarlyRejectFromTail(tail, size, src2)
				if res.Rejected {
					t.Errorf("size=%d seed=%d k=%d: correct seed rejected at step %d",
						size, seed, k, res.StepsConsumed)
				}
				if res.StepsConsumed != k {
					t.Errorf("size=%d k=%d: StepsConsumed=%d, want %d",
						size, k, res.StepsConsumed, k)
				}
			}
		}
	}
}

// TestEarlyRejectFromTail_Reject verifies that a wrong PRNG seed is rejected.
// With K=1 and N=1000, expected rejection rate ≈ 999/1000 per wrong candidate.
func TestEarlyRejectFromTail_Reject(t *testing.T) {
	fy := NewFisherYates().(*FisherYates)
	const (
		size = 1000
		k    = 4
	)

	// Generate ground truth with seed A.
	srcA := newSplitMix64Source(0xAAAAAAAAAAAAAAAA)
	sigma, _ := fy.Generate(size, srcA)
	tail := TailOf(sigma, k)

	// Try 100 different wrong seeds — nearly all should be rejected at step 1.
	rejected := 0
	for seed := uint64(1); seed <= 100; seed++ {
		srcB := newSplitMix64Source(seed * 0xBBBBBBBBBBBBBBBB)
		res := EarlyRejectFromTail(tail, size, srcB)
		if res.Rejected {
			rejected++
		}
	}
	// Expect ≥ 90% rejection (true rate is ≈ (N-1)/N ≈ 99.9%).
	if rejected < 90 {
		t.Errorf("only %d/100 wrong seeds rejected; expected ≥ 90", rejected)
	}
}

// TestEarlyRejectFromTail_EarlyAbort confirms that rejection happens early
// (before all K steps) on wrong seeds, saving PRNG calls.
func TestEarlyRejectFromTail_EarlyAbort(t *testing.T) {
	fy := NewFisherYates().(*FisherYates)
	const (
		size = 500
		k    = 16
	)

	srcA := newSplitMix64Source(0xCAFE)
	sigma, _ := fy.Generate(size, srcA)
	tail := TailOf(sigma, k)

	totalSteps := 0
	trials := 100
	rejected := 0
	for seed := uint64(1); seed <= uint64(trials); seed++ {
		srcB := newSplitMix64Source(seed * 0x1234)
		res := EarlyRejectFromTail(tail, size, srcB)
		if res.Rejected {
			rejected++
			totalSteps += res.StepsConsumed
		}
	}
	if rejected == 0 {
		t.Skip("no rejections (unlikely but possible)")
	}
	avgSteps := float64(totalSteps) / float64(rejected)
	// Wrong candidates should abort well before step k on average.
	if avgSteps >= float64(k) {
		t.Errorf("avg steps on rejection = %.1f, want < %d", avgSteps, k)
	}
}

// BenchmarkEarlyRejectFromTail_K1 measures rejection cost with K=1 (1 PRNG call).
func BenchmarkEarlyRejectFromTail_K1(b *testing.B) {
	fy := NewFisherYates().(*FisherYates)
	src := newSplitMix64Source(1)
	const size = 240_000
	sigma, _ := fy.Generate(size, src)
	tail := TailOf(sigma, 1)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := newSplitMix64Source(uint64(i) + 1)
		EarlyRejectFromTail(tail, size, s)
	}
}

// BenchmarkEarlyRejectFromTail_K32 measures rejection cost with K=32.
func BenchmarkEarlyRejectFromTail_K32(b *testing.B) {
	fy := NewFisherYates().(*FisherYates)
	src := newSplitMix64Source(1)
	const size = 240_000
	sigma, _ := fy.Generate(size, src)
	tail := TailOf(sigma, 32)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := newSplitMix64Source(uint64(i) + 1)
		EarlyRejectFromTail(tail, size, s)
	}
}

// TestRecoverDigits verifies that RecoverDigits extracts the exact digit
// sequence that FY used, and that replaying those digits from the identity
// produces the original σ.
func TestRecoverDigits(t *testing.T) {
	fy := NewFisherYates().(*FisherYates)

	for _, size := range []int{1, 2, 8, 32, 100, 1000, 10000} {
		for seed := uint64(0); seed < 5; seed++ {
			src := newSplitMix64Source(seed*0x9E3779B97F4A7C15 + 7)
			sigma, err := fy.Generate(size, src)
			if err != nil {
				t.Fatalf("Generate error: %v", err)
			}

			digits, err := RecoverDigits(sigma)
			if err != nil {
				t.Fatalf("size=%d seed=%d: RecoverDigits error: %v", size, seed, err)
			}
			if len(digits) != size {
				t.Fatalf("size=%d: got %d digits", size, len(digits))
			}

			// Replay: apply recovered digits to identity → must reproduce sigma.
			replay := make([]int, size)
			for i := range size {
				replay[i] = i
			}
			for i, d := range digits {
				last := size - 1 - i
				replay[d], replay[last] = replay[last], replay[d]
			}
			for i := range size {
				if replay[i] != sigma[i] {
					t.Errorf("size=%d seed=%d: replay[%d]=%d, sigma[%d]=%d",
						size, seed, i, replay[i], i, sigma[i])
				}
			}

			// Digit range invariant: digit[i] must be in [0, size-i).
			for i, d := range digits {
				maxRandom := size - i
				if d < 0 || d >= maxRandom {
					t.Errorf("size=%d seed=%d: digit[%d]=%d out of [0,%d)",
						size, seed, i, d, maxRandom)
				}
			}
		}
	}
}

// TestRecoverDigits_KnownSmall pins the recovery result for a hand-traced example.
func TestRecoverDigits_KnownSmall(t *testing.T) {
	// FY on [0..7] with digits [6,2,4,3,2,1,1,0] produces [0,5,1,7,3,4,2,6].
	sigma := []int{0, 5, 1, 7, 3, 4, 2, 6}
	want := []int{6, 2, 4, 3, 2, 1, 1, 0}

	got, err := RecoverDigits(sigma)
	if err != nil {
		t.Fatalf("RecoverDigits error: %v", err)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("digit[%d]: got %d, want %d", i, got[i], want[i])
		}
	}
}

// TestRecoverDigits_RoundTripViaPermPair confirms that RecoverDigits(pair.Forward)
// gives digits whose replay matches pair.Forward exactly.
func TestRecoverDigits_RoundTripViaPermPair(t *testing.T) {
	fy := NewFisherYates().(*FisherYates)

	src := newSplitMix64Source(0xF5CAFE)
	pair, err := fy.GenerateWithInverse(500, src)
	if err != nil {
		t.Fatalf("GenerateWithInverse error: %v", err)
	}

	digits, err := RecoverDigits(pair.Forward)
	if err != nil {
		t.Fatalf("RecoverDigits error: %v", err)
	}

	replay := make([]int, 500)
	for i := range 500 {
		replay[i] = i
	}
	for i, d := range digits {
		last := 499 - i
		replay[d], replay[last] = replay[last], replay[d]
	}
	for i := range 500 {
		if replay[i] != pair.Forward[i] {
			t.Errorf("position %d: replay=%d, Forward=%d", i, replay[i], pair.Forward[i])
		}
	}
}

// TestRecoverDigits_ErrorCases checks invalid input handling.
func TestRecoverDigits_ErrorCases(t *testing.T) {
	_, err := RecoverDigits([]int{0, 0, 2}) // duplicate
	if err == nil {
		t.Error("expected error for duplicate element")
	}

	_, err = RecoverDigits([]int{0, 5, 2}) // out of range
	if err == nil {
		t.Error("expected error for out-of-range element")
	}

	got, err := RecoverDigits([]int{}) // empty
	if err != nil || len(got) != 0 {
		t.Errorf("empty: got err=%v, len=%d", err, len(got))
	}
}

// BenchmarkRecoverDigits measures walk recovery from the complete permutation.
func BenchmarkRecoverDigits(b *testing.B) {
	fy := NewFisherYates().(*FisherYates)
	src := newSplitMix64Source(0)
	const size = 100_000

	src.state = 42
	sigma, _ := fy.Generate(size, src)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := RecoverDigits(sigma); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGenerateWithInverse measures the cost of producing both σ and σ⁻¹.
func BenchmarkGenerateWithInverse(b *testing.B) {
	fy := NewFisherYates().(*FisherYates)
	src := newSplitMix64Source(0)
	const size = 100_000

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src.state = uint64(i) + 1
		if _, err := fy.GenerateWithInverse(size, src); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGeneratePlusSeparateInverse measures the naive alternative:
// Generate then invert separately, for comparison with BenchmarkGenerateWithInverse.
func BenchmarkGeneratePlusSeparateInverse(b *testing.B) {
	fy := NewFisherYates().(*FisherYates)
	src := newSplitMix64Source(0)
	const size = 100_000
	inv := make([]int, size)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src.state = uint64(i) + 1
		perm, err := fy.Generate(size, src)
		if err != nil {
			b.Fatal(err)
		}
		for j, v := range perm {
			inv[v] = j
		}
		_ = inv
	}
}

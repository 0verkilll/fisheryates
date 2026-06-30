package testing_test

import (
	"testing"

	"github.com/0verkilll/fisheryates"
	testhelpers "github.com/0verkilll/fisheryates/testing"
	"github.com/0verkilll/securerandom"
	"github.com/0verkilll/sha1"
)

// mockReporter implements TestReporter for testing error paths
type mockReporter struct {
	errorfCalls int
	fatalfCalls int
	errorCalls  int
}

func (m *mockReporter) Helper() {}
func (m *mockReporter) Errorf(_ string, _ ...interface{}) {
	m.errorfCalls++
}
func (m *mockReporter) Fatalf(_ string, _ ...interface{}) {
	m.fatalfCalls++
	panic("Fatalf called") // Stop execution like real Fatalf
}
func (m *mockReporter) Error(_ ...interface{}) { m.errorCalls++ }

// TestAssertValidPermutation demonstrates AssertValidPermutation helper
func TestAssertValidPermutation(t *testing.T) {
	fy := fisheryates.NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	_ = random.Seed([]byte("test"))

	perm, err := fy.Generate(10, random)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// This should not fail
	testhelpers.AssertValidPermutation(t, perm)
}

// TestAssertPermutationsEqual demonstrates AssertPermutationsEqual helper
func TestAssertPermutationsEqual(t *testing.T) {
	fy := fisheryates.NewFisherYates()

	// Generate same permutation twice with same seed
	hasher1 := sha1.NewSHA1(sha1.NewBigEndian())
	random1 := securerandom.NewSecureRandom(hasher1)
	_ = random1.Seed([]byte("test"))
	perm1, err := fy.Generate(10, random1)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	hasher2 := sha1.NewSHA1(sha1.NewBigEndian())
	random2 := securerandom.NewSecureRandom(hasher2)
	_ = random2.Seed([]byte("test"))
	perm2, err := fy.Generate(10, random2)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// This should not fail - same seed produces same permutation
	testhelpers.AssertPermutationsEqual(t, perm1, perm2)
}

// TestAssertPermutationsDifferent demonstrates AssertPermutationsDifferent helper
func TestAssertPermutationsDifferent(t *testing.T) {
	fy := fisheryates.NewFisherYates()

	// Generate different permutations with different seeds
	hasher1 := sha1.NewSHA1(sha1.NewBigEndian())
	random1 := securerandom.NewSecureRandom(hasher1)
	_ = random1.Seed([]byte("test1"))
	perm1, err := fy.Generate(10, random1)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	hasher2 := sha1.NewSHA1(sha1.NewBigEndian())
	random2 := securerandom.NewSecureRandom(hasher2)
	_ = random2.Seed([]byte("test2"))
	perm2, err := fy.Generate(10, random2)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// This should not fail - different seeds produce different permutations
	testhelpers.AssertPermutationsDifferent(t, perm1, perm2)
}

// TestAssertBufferCapacityPreserved demonstrates AssertBufferCapacityPreserved helper
func TestAssertBufferCapacityPreserved(t *testing.T) {
	fy := fisheryates.NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	_ = random.Seed([]byte("test"))

	// Create buffer with specific capacity
	buf := make([]int, 0, 100)
	initialCap := cap(buf)

	// Generate into buffer
	buf, err := fy.GenerateInto(buf, 50, random)
	if err != nil {
		t.Fatalf("GenerateInto failed: %v", err)
	}

	// This should not fail - capacity is preserved
	testhelpers.AssertBufferCapacityPreserved(t, buf, initialCap)
}

// TestAssertNoAllocations demonstrates AssertNoAllocations helper
func TestAssertNoAllocations(t *testing.T) {
	fy := fisheryates.NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	_ = random.Seed([]byte("test"))

	// Pre-allocate buffer
	buf := make([]int, 0, 100)

	// This should have minimal allocations
	testhelpers.AssertNoAllocations(t, 10, func() {
		buf, _ = fy.GenerateInto(buf[:0], 50, random) //nolint:errcheck // benchmark callback
	}, 50) // Allow up to 50 allocs per run for PRNG overhead
}

// TestGenerateTestVector demonstrates GenerateTestVector helper
func TestGenerateTestVector(t *testing.T) {
	fy := fisheryates.NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	seed := []byte("test-vector")
	_ = random.Seed(seed)

	perm, err := fy.Generate(10, random)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Create test vector
	vec := testhelpers.GenerateTestVector(seed, 10, perm)

	// Verify test vector fields
	if len(vec.Seed) != len(seed) {
		t.Errorf("Expected seed length %d, got %d", len(seed), len(vec.Seed))
	}
	if vec.Size != 10 {
		t.Errorf("Expected size 10, got %d", vec.Size)
	}
	if len(vec.Output) != 10 {
		t.Errorf("Expected output length 10, got %d", len(vec.Output))
	}

	t.Logf("Test Vector: seed=%q, size=%d, output=%v", vec.Seed, vec.Size, vec.Output)
}

// TestCountInversions demonstrates CountInversions helper
func TestCountInversions(t *testing.T) {
	// Identity permutation has 0 inversions
	identity := []int{0, 1, 2, 3, 4}
	inversions := testhelpers.CountInversions(identity)
	if inversions != 0 {
		t.Errorf("Identity permutation should have 0 inversions, got %d", inversions)
	}

	// Reversed permutation has maximum inversions
	reversed := []int{4, 3, 2, 1, 0}
	inversions = testhelpers.CountInversions(reversed)
	// Maximum inversions for size n is n×(n-1)/2
	expected := (5 * 4) / 2
	if inversions != expected {
		t.Errorf("Reversed permutation should have %d inversions, got %d", expected, inversions)
	}

	t.Logf("Identity inversions: 0, Reversed inversions: %d", inversions)
}

// TestIsIdentityPermutation demonstrates IsIdentityPermutation helper
func TestIsIdentityPermutation(t *testing.T) {
	// Identity permutation
	identity := []int{0, 1, 2, 3, 4}
	if !testhelpers.IsIdentityPermutation(identity) {
		t.Error("Expected identity permutation to return true")
	}

	// Non-identity permutation
	shuffled := []int{1, 0, 2, 3, 4}
	if testhelpers.IsIdentityPermutation(shuffled) {
		t.Error("Expected shuffled permutation to return false")
	}
}

// TestNewMockRandomSource demonstrates NewMockRandomSource helper
func TestNewMockRandomSource(t *testing.T) {
	// Create mock with specific values
	mock := testhelpers.NewMockRandomSource(2, 1, 0)

	// Test Seed method (no-op for mock, but should execute for coverage)
	_ = mock.Seed([]byte("test-seed"))

	fy := fisheryates.NewFisherYates()
	perm, err := fy.Generate(5, mock)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify permutation is valid
	testhelpers.AssertValidPermutation(t, perm)

	t.Logf("Generated permutation with mock: %v", perm)
}

// TestAssertValidPermutation_EdgeCases tests edge cases
func TestAssertValidPermutation_EdgeCases(t *testing.T) {
	// Test empty permutation
	testhelpers.AssertValidPermutation(t, []int{})

	// Test single element
	testhelpers.AssertValidPermutation(t, []int{0})

	// Test two elements
	testhelpers.AssertValidPermutation(t, []int{1, 0})
	testhelpers.AssertValidPermutation(t, []int{0, 1})
}

// TestAssertPermutationsEqual_EdgeCases tests edge cases
func TestAssertPermutationsEqual_EdgeCases(t *testing.T) {
	// Test empty permutations
	testhelpers.AssertPermutationsEqual(t, []int{}, []int{})

	// Test single element
	testhelpers.AssertPermutationsEqual(t, []int{0}, []int{0})
}

// TestAssertPermutationsDifferent_EdgeCases tests edge cases
func TestAssertPermutationsDifferent_EdgeCases(t *testing.T) {
	fy := fisheryates.NewFisherYates()

	// Generate multiple different permutations
	perms := make([][]int, 3)
	for i := 0; i < 3; i++ {
		hasher := sha1.NewSHA1(sha1.NewBigEndian())
		random := securerandom.NewSecureRandom(hasher)
		_ = random.Seed([]byte{byte(i)})
		perm, err := fy.Generate(10, random)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}
		perms[i] = perm
	}

	// Verify they're all different
	testhelpers.AssertPermutationsDifferent(t, perms[0], perms[1])
	testhelpers.AssertPermutationsDifferent(t, perms[1], perms[2])
	testhelpers.AssertPermutationsDifferent(t, perms[0], perms[2])
}

// TestAssertBufferCapacityPreserved_EdgeCases tests edge cases
func TestAssertBufferCapacityPreserved_EdgeCases(t *testing.T) {
	// Test zero capacity
	buf := make([]int, 0)
	testhelpers.AssertBufferCapacityPreserved(t, buf, 0)

	// Test small capacity
	buf = make([]int, 0, 1)
	testhelpers.AssertBufferCapacityPreserved(t, buf, 1)
}

// TestAssertNoAllocations_EdgeCases tests edge cases
func TestAssertNoAllocations_EdgeCases(t *testing.T) {
	// Test function with preallocated resources (no allocations)
	fy := fisheryates.NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	_ = random.Seed([]byte("test"))
	buf := make([]int, 0, 100)

	testhelpers.AssertNoAllocations(t, 5, func() {
		buf, _ = fy.GenerateInto(buf[:0], 50, random) //nolint:errcheck // benchmark callback
	}, 50) // Allow some allocations for PRNG overhead
}

// TestNewMockRandomSource_EmptyValues tests default behavior
func TestNewMockRandomSource_EmptyValues(t *testing.T) {
	// Test with no values (should default to 0)
	mock := testhelpers.NewMockRandomSource()
	val := mock.NextInt()
	if val != 0 {
		t.Errorf("Expected default value 0, got %d", val)
	}
}

// TestMockRandomSource_AllMethods tests all mock methods
func TestMockRandomSource_AllMethods(t *testing.T) {
	mock := testhelpers.NewMockRandomSource(5, 10, 15)

	// Test NextInt
	if val := mock.NextInt(); val != 5 {
		t.Errorf("Expected 5, got %d", val)
	}
	if val := mock.NextInt(); val != 10 {
		t.Errorf("Expected 10, got %d", val)
	}

	// Test Reset
	mock.Reset()
	if val := mock.NextInt(); val != 5 {
		t.Errorf("After reset, expected 5, got %d", val)
	}

	// Test SetValues with values
	mock.SetValues(20, 25, 30)
	if val := mock.NextInt(); val != 20 {
		t.Errorf("After SetValues, expected 20, got %d", val)
	}

	// Test SetValues with no values (should default to 0)
	mock.SetValues()
	if val := mock.NextInt(); val != 0 {
		t.Errorf("After SetValues(), expected 0, got %d", val)
	}

	// Test Seed (no-op but should not panic)
	_ = mock.Seed([]byte("test"))

	// Test NextBytes
	bytes := mock.NextBytes(10)
	if len(bytes) != 10 {
		t.Errorf("Expected 10 bytes, got %d", len(bytes))
	}
	// All bytes should be zero
	for i, b := range bytes {
		if b != 0 {
			t.Errorf("Expected byte %d to be 0, got %d", i, b)
		}
	}
}

// TestAssertValidPermutation_ErrorPaths tests error detection in AssertValidPermutation
func TestAssertValidPermutation_ErrorPaths(t *testing.T) {
	t.Run("out of range value", func(t *testing.T) {
		mock := &mockReporter{}
		invalidPerm := []int{0, 1, 5} // 5 is out of range for size 3
		testhelpers.AssertValidPermutation(mock, invalidPerm)
		if mock.errorfCalls != 1 {
			t.Errorf("Expected 1 Errorf call, got %d", mock.errorfCalls)
		}
	})

	t.Run("duplicate value", func(t *testing.T) {
		mock := &mockReporter{}
		invalidPerm := []int{0, 1, 1} // duplicate 1
		testhelpers.AssertValidPermutation(mock, invalidPerm)
		if mock.errorfCalls != 1 {
			t.Errorf("Expected 1 Errorf call, got %d", mock.errorfCalls)
		}
	})
}

// TestAssertPermutationsEqual_ErrorPaths tests error detection
func TestAssertPermutationsEqual_ErrorPaths(t *testing.T) {
	t.Run("different lengths", func(t *testing.T) {
		mock := &mockReporter{}
		perm1 := []int{0, 1, 2}
		perm2 := []int{0, 1}

		// Fatalf will panic, so we need to recover from it
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic from Fatalf")
			}
		}()

		testhelpers.AssertPermutationsEqual(mock, perm1, perm2)
		if mock.fatalfCalls != 1 {
			t.Errorf("Expected 1 Fatalf call, got %d", mock.fatalfCalls)
		}
	})

	t.Run("different values", func(t *testing.T) {
		mock := &mockReporter{}
		perm1 := []int{0, 1, 2}
		perm2 := []int{0, 2, 1}
		testhelpers.AssertPermutationsEqual(mock, perm1, perm2)
		// The loop calls Errorf for each differing index
		if mock.errorfCalls < 1 {
			t.Errorf("Expected at least 1 Errorf call, got %d", mock.errorfCalls)
		}
	})
}

// TestAssertPermutationsDifferent_ErrorPath tests identical permutations
func TestAssertPermutationsDifferent_ErrorPath(t *testing.T) {
	t.Run("identical permutations", func(t *testing.T) {
		mock := &mockReporter{}
		perm := []int{0, 1, 2, 3, 4}
		testhelpers.AssertPermutationsDifferent(mock, perm, perm)
		if mock.errorCalls != 1 {
			t.Errorf("Expected 1 Error call, got %d", mock.errorCalls)
		}
	})

	t.Run("different lengths are considered different", func(t *testing.T) {
		mock := &mockReporter{}
		perm1 := []int{0, 1, 2}
		perm2 := []int{0, 1, 2, 3, 4}
		// Should return early without error since different lengths = different
		testhelpers.AssertPermutationsDifferent(mock, perm1, perm2)
		if mock.errorCalls != 0 {
			t.Errorf("Expected 0 Error calls for different lengths, got %d", mock.errorCalls)
		}
	})
}

// TestAssertBufferCapacityPreserved_ErrorPath tests capacity mismatch
func TestAssertBufferCapacityPreserved_ErrorPath(t *testing.T) {
	t.Run("capacity mismatch", func(t *testing.T) {
		mock := &mockReporter{}
		buf := make([]int, 0, 10)
		testhelpers.AssertBufferCapacityPreserved(mock, buf, 20) // Wrong expected capacity
		if mock.errorfCalls != 1 {
			t.Errorf("Expected 1 Errorf call, got %d", mock.errorfCalls)
		}
	})
}

// TestAssertNoAllocations_ErrorPath tests too many allocations
func TestAssertNoAllocations_ErrorPath(t *testing.T) {
	t.Run("too many allocations", func(t *testing.T) {
		mock := &mockReporter{}
		var sink interface{}
		testhelpers.AssertNoAllocations(mock, 10, func() {
			// Interface assignment forces heap allocation
			sink = make([]byte, 1)
		}, 0) // Expect zero but we're allocating
		_ = sink
		if mock.errorfCalls != 1 {
			t.Errorf("Expected 1 Errorf call, got %d", mock.errorfCalls)
		}
	})
}

package testing

import (
	"testing"
)

// TestReporter is a minimal interface for test reporting used by assertion helpers.
// This interface is automatically satisfied by *testing.T and *testing.B.
type TestReporter interface {
	Helper()
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Error(args ...interface{})
}

// AssertValidPermutation verifies that a slice is a valid permutation of [0, n-1].
// A valid permutation must contain all integers from 0 to n-1 exactly once.
//
// This helper checks:
//   - All values are in the range [0, n-1]
//   - No duplicate values exist
//   - All expected values are present
//
// Parameters:
//   - t: The test reporter (typically *testing.T)
//   - perm: The permutation slice to validate
//
// Example:
//
//	perm := fy.Generate(10, random)
//	AssertValidPermutation(t, perm) // Fails test if invalid
func AssertValidPermutation(t TestReporter, perm []int) {
	t.Helper()

	n := len(perm)
	seen := make(map[int]bool, n)

	for i, val := range perm {
		// Check range
		if val < 0 || val >= n {
			t.Errorf("Invalid value at index %d: %d (expected 0-%d)", i, val, n-1)
			return
		}

		// Check for duplicates
		if seen[val] {
			t.Errorf("Duplicate value at index %d: %d", i, val)
			return
		}
		seen[val] = true
	}

	// Note: Completeness check (len(seen) == n) is mathematically guaranteed here.
	// If we completed the loop: processed n elements, all in range [0,n), all unique.
	// Therefore len(seen) MUST equal n. No defensive check needed.
}

// AssertPermutationsEqual verifies that two permutations are identical.
//
// Parameters:
//   - t: The test reporter (typically *testing.T)
//   - perm1: First permutation
//   - perm2: Second permutation
//
// Example:
//
//	perm1 := fy.Generate(10, random)
//	random.Seed(sameSeed) // Reset to same state
//	perm2 := fy.Generate(10, random)
//	AssertPermutationsEqual(t, perm1, perm2) // Verify determinism
func AssertPermutationsEqual(t TestReporter, perm1, perm2 []int) {
	t.Helper()

	if len(perm1) != len(perm2) {
		t.Fatalf("Permutation lengths differ: %d vs %d", len(perm1), len(perm2))
	}

	for i := 0; i < len(perm1); i++ {
		if perm1[i] != perm2[i] {
			t.Errorf("Permutations differ at index %d: %d vs %d", i, perm1[i], perm2[i])
		}
	}
}

// AssertPermutationsDifferent verifies that two permutations are different.
// This is useful for testing that different seeds produce different results.
//
// Parameters:
//   - t: The test reporter (typically *testing.T)
//   - perm1: First permutation
//   - perm2: Second permutation
//
// Example:
//
//	perm1 := fy.Generate(10, random1)
//	perm2 := fy.Generate(10, random2) // Different seed
//	AssertPermutationsDifferent(t, perm1, perm2)
func AssertPermutationsDifferent(t TestReporter, perm1, perm2 []int) {
	t.Helper()

	if len(perm1) != len(perm2) {
		return // Different lengths = definitely different
	}

	identical := true
	for i := 0; i < len(perm1); i++ {
		if perm1[i] != perm2[i] {
			identical = false
			break
		}
	}

	if identical {
		t.Error("Expected different permutations, but they are identical")
	}
}

// AssertBufferCapacityPreserved verifies that a buffer's capacity hasn't changed.
// This is useful for testing the zero-allocation pattern with GenerateInto().
//
// Parameters:
//   - t: The test reporter (typically *testing.T)
//   - buf: The buffer to check
//   - expectedCap: The expected capacity
//
// Example:
//
//	buf := make([]int, 100)
//	initialCap := cap(buf)
//	buf = fy.GenerateInto(buf, 50, random)
//	AssertBufferCapacityPreserved(t, buf, initialCap)
func AssertBufferCapacityPreserved(t TestReporter, buf []int, expectedCap int) {
	t.Helper()

	if cap(buf) != expectedCap {
		t.Errorf("Buffer capacity changed: expected %d, got %d", expectedCap, cap(buf))
	}
}

// AssertNoAllocations verifies that a function performs no heap allocations.
// This uses testing.AllocsPerRun to measure allocations.
//
// Parameters:
//   - t: The test reporter (typically *testing.T)
//   - runs: Number of runs to average over
//   - fn: The function to test
//   - maxAllocs: Maximum allowed allocations per run
//
// Example:
//
//	buf := make([]int, 100)
//	AssertNoAllocations(t, 100, func() {
//	    buf = fy.GenerateInto(buf, 100, random)
//	}, 0) // Expect zero allocations
func AssertNoAllocations(t TestReporter, runs int, fn func(), maxAllocs float64) {
	t.Helper()

	allocs := testing.AllocsPerRun(runs, fn)
	if allocs > maxAllocs {
		t.Errorf("Too many allocations: %.2f per run (expected ≤%.2f)", allocs, maxAllocs)
	}
}

// GenerateTestVector creates a test vector for cross-platform verification.
// A test vector includes the input parameters and expected output for
// verifying implementations across different languages.
//
// Parameters:
//   - seed: The seed bytes
//   - size: The permutation size
//   - perm: The expected permutation output
//
// Returns:
//   - A TestVector struct
//
// Example:
//
//	vec := GenerateTestVector([]byte("test"), 10, expectedPerm)
//	t.Logf("Test Vector: seed=%q, size=%d, output=%v", vec.Seed, vec.Size, vec.Output)
func GenerateTestVector(seed []byte, size int, perm []int) TestVector {
	return TestVector{
		Seed:   seed,
		Size:   size,
		Output: perm,
	}
}

// TestVector represents a cross-platform test vector.
// Test vectors are used to verify that implementations in different
// programming languages (Go, Java, TypeScript, Rust) produce identical results.
type TestVector struct {
	Seed   []byte // The seed bytes used for the PRNG
	Output []int  // The expected permutation output
	Size   int    // The permutation size
}

// CountInversions counts the number of inversions in a permutation.
// An inversion is a pair of indices (i, j) where i < j but perm[i] > perm[j].
// This can be useful for analyzing the "randomness" of a permutation.
//
// Parameters:
//   - perm: The permutation to analyze
//
// Returns:
//   - The number of inversions
//
// Example:
//
//	inversions := CountInversions(perm)
//	t.Logf("Permutation has %d inversions", inversions)
func CountInversions(perm []int) int {
	count := 0
	n := len(perm)

	for i := 0; i < n-1; i++ {
		for j := i + 1; j < n; j++ {
			if perm[i] > perm[j] {
				count++
			}
		}
	}

	return count
}

// IsIdentityPermutation checks if a permutation is the identity permutation [0, 1, 2, ..., n-1].
//
// Parameters:
//   - perm: The permutation to check
//
// Returns:
//   - true if perm is the identity permutation, false otherwise
//
// Example:
//
//	if IsIdentityPermutation(perm) {
//	    t.Error("Expected shuffled permutation, got identity")
//	}
func IsIdentityPermutation(perm []int) bool {
	for i, val := range perm {
		if val != i {
			return false
		}
	}
	return true
}

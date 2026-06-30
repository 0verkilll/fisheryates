package fisheryates

import (
	"testing"

	"github.com/0verkilll/f5prng"
	"github.com/0verkilll/sha1"
)

// ========================================
// f5prng Integration Tests
// ========================================
//
// These tests verify that fisheryates works correctly with f5prng.RandomSource.

// TestF5PRNGIntegration_PermutationGeneration tests that permutation generation
// works correctly when using an f5prng.RandomSource implementation.
func TestF5PRNGIntegration_PermutationGeneration(t *testing.T) {
	// Create f5prng.RandomSource using the f5prng package's SecureRandom
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := f5prng.NewSecureRandom(hasher)
	_ = random.Seed([]byte("test-seed"))

	// Create permutator and generate
	fy := NewFisherYates()
	size := 100
	perm, err := fy.Generate(size, random)

	// Verify no error
	if err != nil {
		t.Fatalf("Generate with f5prng.RandomSource failed: %v", err)
	}

	// Verify permutation properties
	if len(perm) != size {
		t.Errorf("Expected permutation length %d, got %d", size, len(perm))
	}

	// Verify all elements are present (0 to size-1)
	seen := make(map[int]bool)
	for _, val := range perm {
		if val < 0 || val >= size {
			t.Errorf("Value %d out of range [0, %d)", val, size)
		}
		if seen[val] {
			t.Errorf("Duplicate value %d in permutation", val)
		}
		seen[val] = true
	}

	// Verify all values from 0 to size-1 are present
	for i := 0; i < size; i++ {
		if !seen[i] {
			t.Errorf("Missing value %d in permutation", i)
		}
	}
}

// TestF5PRNGIntegration_GenerateInto tests that GenerateInto works correctly
// with f5prng.RandomSource.
func TestF5PRNGIntegration_GenerateInto(t *testing.T) {
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := f5prng.NewSecureRandom(hasher)
	_ = random.Seed([]byte("generate-into-test"))

	fy := NewFisherYates()
	size := 50
	buf := make([]int, 0, 100) // Pre-allocate larger buffer

	result, err := fy.GenerateInto(buf, size, random)
	if err != nil {
		t.Fatalf("GenerateInto with f5prng.RandomSource failed: %v", err)
	}

	if len(result) != size {
		t.Errorf("Expected result length %d, got %d", size, len(result))
	}

	// Verify permutation properties
	seen := make(map[int]bool)
	for _, val := range result {
		if val < 0 || val >= size {
			t.Errorf("Value %d out of range [0, %d)", val, size)
		}
		if seen[val] {
			t.Errorf("Duplicate value %d in permutation", val)
		}
		seen[val] = true
	}
}

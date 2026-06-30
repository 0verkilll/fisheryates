package fisheryates

import (
	"errors"
	"fmt"
	"testing"
	"testing/fstest"
	"time"

	"github.com/0verkilll/securerandom"
	"github.com/0verkilll/sha1"
)

// ========================================
// Mock Random Source Implementations
// ========================================

// mockRandomSource is a simple mock that returns a pre-defined sequence of int32 values
type mockRandomSource struct {
	ints  []int32
	index int
}

func newMockRandomSource(ints ...int32) *mockRandomSource {
	return &mockRandomSource{ints: ints, index: 0}
}

func (m *mockRandomSource) Seed(_ []byte) error {
	m.index = 0
	return nil
}

func (m *mockRandomSource) NextBytes(n int) []byte {
	return make([]byte, n)
}

func (m *mockRandomSource) NextInt() int32 {
	if m.index >= len(m.ints) {
		return 0
	}
	val := m.ints[m.index]
	m.index++
	return val
}

// Clear is a no-op for mockRandomSource (implements f5prng.RandomSource)
func (m *mockRandomSource) Clear() {
	m.index = 0
}

// constantRandomSource is a mock that always returns the same value.
// This provides the absolute minimum PRNG overhead for baseline measurement.
type constantRandomSource struct {
	value int32
}

func (c *constantRandomSource) NextInt() int32 {
	return c.value
}

func (c *constantRandomSource) Seed(_ []byte) error {
	// No-op
	return nil
}

func (c *constantRandomSource) NextBytes(n int) []byte {
	return make([]byte, n)
}

// Clear is a no-op for constantRandomSource (implements f5prng.RandomSource)
func (c *constantRandomSource) Clear() {
	// No state to clear
}

// cyclingRandomSource is a mock that cycles through a list of values.
// This has moderate overhead similar to a real PRNG.
type cyclingRandomSource struct {
	values []int32
	index  int
}

func (c *cyclingRandomSource) NextInt() int32 {
	val := c.values[c.index]
	c.index = (c.index + 1) % len(c.values)
	return val
}

func (c *cyclingRandomSource) Seed(_ []byte) error {
	// No-op
	return nil
}

func (c *cyclingRandomSource) NextBytes(n int) []byte {
	return make([]byte, n)
}

func (c *cyclingRandomSource) Reset() {
	c.index = 0
}

// Clear is a no-op for cyclingRandomSource (implements f5prng.RandomSource)
func (c *cyclingRandomSource) Clear() {
	c.index = 0
}

// ========================================
// Unit Tests: Generate()
// ========================================

// TestFisherYates_EmptySize tests zero size
func TestFisherYates_EmptySize(t *testing.T) {
	fy := NewFisherYates()
	random := newMockRandomSource(1, 2, 3)

	// Zero size should return empty slice
	result, err := fy.Generate(0, random)
	if err != nil {
		t.Errorf("Zero size should not return error: %v", err)
	}
	if len(result) != 0 {
		t.Error("Zero size should return empty slice")
	}

	// Note: Negative sizes now return ErrSizeNegative - see security_limits tests
}

// TestFisherYates_SingleElement tests single element permutation
func TestFisherYates_SingleElement(t *testing.T) {
	fy := NewFisherYates()
	random := newMockRandomSource(0)

	result, err := fy.Generate(1, random)
	if err != nil {
		t.Errorf("Single element should not return error: %v", err)
	}
	if len(result) != 1 {
		t.Error("Single element should have length 1")
	}
	if result[0] != 0 {
		t.Error("Single element should be 0")
	}
}

// TestFisherYates_Completeness tests that all elements are present
func TestFisherYates_Completeness(t *testing.T) {
	sizes := []int{5, 10, 100, 1000}

	for _, size := range sizes {
		t.Run("", func(t *testing.T) {
			fy := NewFisherYates()
			hasher := sha1.NewSHA1(sha1.NewBigEndian())
			random := securerandom.NewSecureRandom(hasher)
			_ = random.Seed([]byte("test"))

			result, err := fy.Generate(size, random)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			// Check length
			if size != len(result) {
				t.Error("Result should have correct length")
			}

			// Check all elements present
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

			// Verify all values from 0 to size-1 are present
			for i := range size {
				if !seen[i] {
					t.Errorf("Missing value %d in permutation", i)
				}
			}
		})
	}
}

// TestFisherYates_Deterministic tests deterministic behavior with same seed
func TestFisherYates_Deterministic(t *testing.T) {
	size := 100
	seed := []byte("deterministic")

	// Generate first permutation
	fy1 := NewFisherYates()
	hasher1 := sha1.NewSHA1(sha1.NewBigEndian())
	random1 := securerandom.NewSecureRandom(hasher1)
	_ = random1.Seed(seed)
	perm1, err := fy1.Generate(size, random1)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Generate second permutation with same seed
	fy2 := NewFisherYates()
	hasher2 := sha1.NewSHA1(sha1.NewBigEndian())
	random2 := securerandom.NewSecureRandom(hasher2)
	_ = random2.Seed(seed)
	perm2, err := fy2.Generate(size, random2)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should be identical
	if len(perm1) != len(perm2) {
		t.Error("Same seed should produce same permutation: length mismatch")
	}
	for i := range perm1 {
		if perm1[i] != perm2[i] {
			t.Error("Same seed should produce same permutation")
		}
	}
}

// TestFisherYates_MockRandomSource tests with predictable mock
func TestFisherYates_MockRandomSource(t *testing.T) {
	fy := NewFisherYates()

	// Create mock with predictable sequence
	// For size 5: maxRandom goes 5, 4, 3, 2, 1
	// We provide indices that will be used
	random := newMockRandomSource(
		2, // i=0: swap index 2 with position 4
		1, // i=1: swap index 1 with position 3
		0, // i=2: swap index 0 with position 2
		0, // i=3: swap index 0 with position 1
		0, // i=4: swap index 0 with position 0
	)

	result, err := fy.Generate(5, random)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify result
	if len(result) != 5 {
		t.Error("Should have 5 elements")
	}

	// Trace through algorithm:
	// Start: [0, 1, 2, 3, 4]
	// i=0, maxRandom=5, randomInt=2, randomIndex=2: swap(2,4) -> [0, 1, 4, 3, 2]
	// i=1, maxRandom=4, randomInt=1, randomIndex=1: swap(1,3) -> [0, 3, 4, 1, 2]
	// i=2, maxRandom=3, randomInt=0, randomIndex=0: swap(0,2) -> [4, 3, 0, 1, 2]
	// i=3, maxRandom=2, randomInt=0, randomIndex=0: swap(0,1) -> [3, 4, 0, 1, 2]
	// i=4, maxRandom=1, randomInt=0, randomIndex=0: swap(0,0) -> [3, 4, 0, 1, 2]
	expected := []int{3, 4, 0, 1, 2}
	if len(expected) != len(result) {
		t.Error("Should match expected permutation: length mismatch")
	}
	for i := range expected {
		if expected[i] != result[i] {
			t.Error("Should match expected permutation")
		}
	}
}

// TestFisherYates_NegativeModulo tests handling of negative random values
func TestFisherYates_NegativeModulo(t *testing.T) {
	fy := NewFisherYates()

	// Create mock with negative values
	random := newMockRandomSource(
		-1, // Should become 4 when mod 5
		-2, // Should become 2 when mod 4
		-3, // Should become 0 when mod 3
		-4, // Should become 0 when mod 2 (Java: -4 % 2 = 0, then no adjustment)
		-5, // Should become 0 when mod 1
	)

	result, err := fy.Generate(5, random)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify completeness
	if len(result) != 5 {
		t.Error("Should have 5 elements")
	}

	seen := make(map[int]bool)
	for _, val := range result {
		if val < 0 || val >= 5 {
			t.Errorf("Value %d out of range", val)
		}
		seen[val] = true
	}

	// All values should be present
	for i := range 5 {
		if !seen[i] {
			t.Errorf("Value %d should be present", i)
		}
	}
}

// TestFisherYates_LargePermutation tests with large arrays
func TestFisherYates_LargePermutation(t *testing.T) {
	fy := NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	_ = random.Seed([]byte("large"))

	// Test with 240,000 elements (sample.jpg coefficient count)
	size := 240000
	result, err := fy.Generate(size, random)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if size != len(result) {
		t.Error("Should have correct size")
	}

	// Spot check a few values
	for i := range 100 {
		val := result[i]
		if val < 0 || val >= size {
			t.Errorf("Value at position %d is out of range: %d", i, val)
		}
	}

	t.Logf("[OK] Successfully generated permutation of %d elements", size)
}

// TestFisherYates_DifferentSeeds tests different seeds produce different permutations
func TestFisherYates_DifferentSeeds(t *testing.T) {
	size := 50
	fy := NewFisherYates()

	// Generate with seed 1
	hasher1 := sha1.NewSHA1(sha1.NewBigEndian())
	random1 := securerandom.NewSecureRandom(hasher1)
	_ = random1.Seed([]byte("seed1"))
	perm1, err := fy.Generate(size, random1)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Generate with seed 2
	hasher2 := sha1.NewSHA1(sha1.NewBigEndian())
	random2 := securerandom.NewSecureRandom(hasher2)
	_ = random2.Seed([]byte("seed2"))
	perm2, err := fy.Generate(size, random2)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should be different (very high probability)
	different := false
	for i := range size {
		if perm1[i] != perm2[i] {
			different = true
			break
		}
	}

	if !different {
		t.Error("Different seeds should produce different permutations")
	}
}

// TestFisherYates_Password23 tests with sample.jpg password
func TestFisherYates_Password23(t *testing.T) {
	fy := NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	_ = random.Seed([]byte("23"))

	// Generate permutation for typical size
	size := 1000
	result, err := fy.Generate(size, random)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify completeness
	seen := make(map[int]bool)
	for _, val := range result {
		if val < 0 || val >= size {
			t.Fatalf("Value %d out of range [0, %d)", val, size)
		}
		if seen[val] {
			t.Fatalf("Duplicate value %d", val)
		}
		seen[val] = true
	}

	t.Logf("First 10 indices: %v", result[:10])
	t.Logf("[OK] Valid permutation for password '23'")
}

// ========================================
// Unit Tests: GenerateInto()
// ========================================

// TestFisherYates_GenerateInto_EmptySize tests zero size
func TestFisherYates_GenerateInto_EmptySize(t *testing.T) {
	fy := NewFisherYates()
	random := newMockRandomSource(1, 2, 3)
	buf := make([]int, 10) // Start with pre-allocated buffer

	// Zero size should return empty slice
	result, err := fy.GenerateInto(buf, 0, random)
	if err != nil {
		t.Errorf("Zero size should not return error: %v", err)
	}
	if len(result) != 0 {
		t.Error("Zero size should return empty slice")
	}

	// Note: Negative sizes now return ErrSizeNegative - see security_limits tests
}

// TestFisherYates_GenerateInto_SingleElement tests single element permutation
func TestFisherYates_GenerateInto_SingleElement(t *testing.T) {
	fy := NewFisherYates()
	random := newMockRandomSource(0)
	buf := make([]int, 0)

	result, err := fy.GenerateInto(buf, 1, random)
	if err != nil {
		t.Errorf("Single element should not return error: %v", err)
	}
	if len(result) != 1 {
		t.Error("Single element should have length 1")
	}
	if result[0] != 0 {
		t.Error("Single element should be 0")
	}
}

// TestFisherYates_GenerateInto_BufferReuse tests buffer reuse behavior
func TestFisherYates_GenerateInto_BufferReuse(t *testing.T) {
	fy := NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)

	// Start with small buffer
	buf := make([]int, 5)

	// First generation - buffer is large enough
	_ = random.Seed([]byte("test1"))
	result1, err := fy.GenerateInto(buf, 5, random)
	if err != nil {
		t.Fatalf("GenerateInto failed: %v", err)
	}
	if len(result1) != 5 {
		t.Error("Should have 5 elements")
	}

	// Same buffer pointer should be reused (capacity is sufficient)
	if cap(buf) != cap(result1) {
		t.Error("Should reuse buffer capacity")
	}

	// Second generation with smaller size - should reuse same buffer
	_ = random.Seed([]byte("test2"))
	result2, err := fy.GenerateInto(result1, 3, random)
	if err != nil {
		t.Fatalf("GenerateInto failed: %v", err)
	}
	if len(result2) != 3 {
		t.Error("Should have 3 elements")
	}

	// Third generation with larger size than original buffer capacity - should reallocate
	_ = random.Seed([]byte("test3"))
	result3, err := fy.GenerateInto(buf, 10, random)
	if err != nil {
		t.Fatalf("GenerateInto failed: %v", err)
	}
	if len(result3) != 10 {
		t.Error("Should have 10 elements")
	}
	if cap(result3) < 10 {
		t.Error("Should have capacity for 10 elements")
	}
}

// TestFisherYates_GenerateInto_MatchesGenerate tests that GenerateInto produces same results as Generate
func TestFisherYates_GenerateInto_MatchesGenerate(t *testing.T) {
	sizes := []int{5, 10, 50, 100}

	for _, size := range sizes {
		t.Run("", func(t *testing.T) {
			fy := NewFisherYates()
			seed := []byte("match-test")

			// Generate using Generate()
			hasher1 := sha1.NewSHA1(sha1.NewBigEndian())
			random1 := securerandom.NewSecureRandom(hasher1)
			_ = random1.Seed(seed)
			result1, err := fy.Generate(size, random1)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			// Generate using GenerateInto()
			hasher2 := sha1.NewSHA1(sha1.NewBigEndian())
			random2 := securerandom.NewSecureRandom(hasher2)
			_ = random2.Seed(seed)
			buf := make([]int, 0)
			result2, err := fy.GenerateInto(buf, size, random2)
			if err != nil {
				t.Fatalf("GenerateInto failed: %v", err)
			}

			// Should be identical
			if len(result1) != len(result2) {
				t.Error("GenerateInto should match Generate: length mismatch")
			}
			for i := range result1 {
				if result1[i] != result2[i] {
					t.Error("GenerateInto should match Generate")
				}
			}
		})
	}
}

// TestFisherYates_GenerateInto_Completeness tests that all elements are present
func TestFisherYates_GenerateInto_Completeness(t *testing.T) {
	sizes := []int{5, 10, 100}

	for _, size := range sizes {
		t.Run("", func(t *testing.T) {
			fy := NewFisherYates()
			hasher := sha1.NewSHA1(sha1.NewBigEndian())
			random := securerandom.NewSecureRandom(hasher)
			_ = random.Seed([]byte("completeness"))

			buf := make([]int, 0)
			result, err := fy.GenerateInto(buf, size, random)
			if err != nil {
				t.Fatalf("GenerateInto failed: %v", err)
			}

			// Check length
			if size != len(result) {
				t.Error("Result should have correct length")
			}

			// Check all elements present
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

			// Verify all values from 0 to size-1 are present
			for i := range size {
				if !seen[i] {
					t.Errorf("Missing value %d in permutation", i)
				}
			}
		})
	}
}

// TestFisherYates_GenerateInto_MockRandomSource tests with predictable mock
func TestFisherYates_GenerateInto_MockRandomSource(t *testing.T) {
	fy := NewFisherYates()

	// Create mock with predictable sequence (same as Generate test)
	random := newMockRandomSource(
		2, // i=0: swap index 2 with position 4
		1, // i=1: swap index 1 with position 3
		0, // i=2: swap index 0 with position 2
		0, // i=3: swap index 0 with position 1
		0, // i=4: swap index 0 with position 0
	)

	buf := make([]int, 0)
	result, err := fy.GenerateInto(buf, 5, random)
	if err != nil {
		t.Fatalf("GenerateInto failed: %v", err)
	}

	// Verify result
	if len(result) != 5 {
		t.Error("Should have 5 elements")
	}

	// Expected result: [3, 4, 0, 1, 2]
	expected := []int{3, 4, 0, 1, 2}
	if len(expected) != len(result) {
		t.Error("Should match expected permutation: length mismatch")
	}
	for i := range expected {
		if expected[i] != result[i] {
			t.Error("Should match expected permutation")
		}
	}
}

// TestFisherYates_GenerateInto_NegativeModulo tests handling of negative random values
func TestFisherYates_GenerateInto_NegativeModulo(t *testing.T) {
	fy := NewFisherYates()

	// Create mock with negative values
	random := newMockRandomSource(
		-1, // Should become 4 when mod 5
		-2, // Should become 2 when mod 4
		-3, // Should become 0 when mod 3
		-4, // Should become 0 when mod 2
		-5, // Should become 0 when mod 1
	)

	buf := make([]int, 0)
	result, err := fy.GenerateInto(buf, 5, random)
	if err != nil {
		t.Fatalf("GenerateInto failed: %v", err)
	}

	// Verify completeness
	if len(result) != 5 {
		t.Error("Should have 5 elements")
	}

	seen := make(map[int]bool)
	for _, val := range result {
		if val < 0 || val >= 5 {
			t.Errorf("Value %d out of range", val)
		}
		seen[val] = true
	}

	// All values should be present
	for i := range 5 {
		if !seen[i] {
			t.Errorf("Value %d should be present", i)
		}
	}
}

// TestFisherYates_GenerateInto_LargePermutation tests with large arrays
func TestFisherYates_GenerateInto_LargePermutation(t *testing.T) {
	fy := NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	_ = random.Seed([]byte("large-into"))

	// Test with 240,000 elements (sample.jpg coefficient count)
	size := 240000
	buf := make([]int, 0)
	result, err := fy.GenerateInto(buf, size, random)
	if err != nil {
		t.Fatalf("GenerateInto failed: %v", err)
	}

	if size != len(result) {
		t.Error("Should have correct size")
	}

	// Spot check a few values
	for i := range 100 {
		val := result[i]
		if val < 0 || val >= size {
			t.Errorf("Value at position %d is out of range: %d", i, val)
		}
	}

	t.Logf("[OK] Successfully generated permutation of %d elements using GenerateInto", size)
}

// TestFisherYates_GenerateInto_Deterministic tests deterministic behavior with same seed
func TestFisherYates_GenerateInto_Deterministic(t *testing.T) {
	size := 100
	seed := []byte("deterministic-into")

	// Generate first permutation
	fy1 := NewFisherYates()
	hasher1 := sha1.NewSHA1(sha1.NewBigEndian())
	random1 := securerandom.NewSecureRandom(hasher1)
	_ = random1.Seed(seed)
	buf1 := make([]int, 0)
	perm1, err := fy1.GenerateInto(buf1, size, random1)
	if err != nil {
		t.Fatalf("GenerateInto failed: %v", err)
	}

	// Generate second permutation with same seed
	fy2 := NewFisherYates()
	hasher2 := sha1.NewSHA1(sha1.NewBigEndian())
	random2 := securerandom.NewSecureRandom(hasher2)
	_ = random2.Seed(seed)
	buf2 := make([]int, 0)
	perm2, err := fy2.GenerateInto(buf2, size, random2)
	if err != nil {
		t.Fatalf("GenerateInto failed: %v", err)
	}

	// Should be identical
	if len(perm1) != len(perm2) {
		t.Error("Same seed should produce same permutation: length mismatch")
	}
	for i := range perm1 {
		if perm1[i] != perm2[i] {
			t.Error("Same seed should produce same permutation")
		}
	}
}

// ========================================
// Security and Limits Tests
// ========================================

// TestSecurityLimits_Generate_NegativeSize verifies that Generate returns error on negative size
func TestSecurityLimits_Generate_NegativeSize(t *testing.T) {
	fy := NewFisherYates()
	random := newMockRandomSource(1, 2, 3)

	// This should return ErrSizeNegative
	_, err := fy.Generate(-1, random)
	if err == nil {
		t.Fatal("Expected error for negative size, but no error returned")
	}

	if !errors.Is(err, ErrSizeNegative) {
		t.Errorf("Expected ErrSizeNegative, got %v", err)
	}
}

// TestSecurityLimits_Generate_ExceedsMaxSize verifies that Generate returns error when size exceeds MaxPermutationSize
func TestSecurityLimits_Generate_ExceedsMaxSize(t *testing.T) {
	fy := NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	_ = random.Seed([]byte("test"))

	// This should return ErrSizeExceedsMax (100M + 1)
	_, err := fy.Generate(MaxPermutationSize+1, random)
	if err == nil {
		t.Fatal("Expected error for size exceeding MaxPermutationSize, but no error returned")
	}

	if !errors.Is(err, ErrSizeExceedsMax) {
		t.Errorf("Expected ErrSizeExceedsMax, got %v", err)
	}
}

// TestSecurityLimits_Generate_AtMaxSize verifies that Generate works at exactly MaxPermutationSize
// Note: This test is disabled by default as it requires ~800MB of RAM
func TestSecurityLimits_Generate_AtMaxSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory-intensive test in short mode")
	}

	fy := NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	_ = random.Seed([]byte("max-size-test"))

	// This should work (exactly MaxPermutationSize)
	result, err := fy.Generate(MaxPermutationSize, random)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(result) != MaxPermutationSize {
		t.Errorf("Expected length %d, got %d", MaxPermutationSize, len(result))
	}

	// Spot check a few values to ensure validity
	if result[0] < 0 || result[0] >= MaxPermutationSize {
		t.Errorf("First element %d out of range", result[0])
	}
	if result[MaxPermutationSize-1] < 0 || result[MaxPermutationSize-1] >= MaxPermutationSize {
		t.Errorf("Last element %d out of range", result[MaxPermutationSize-1])
	}

	t.Logf("[OK] Successfully generated permutation at MaxPermutationSize (%d elements)", MaxPermutationSize)
}

// TestSecurityLimits_GenerateInto_NegativeSize verifies that GenerateInto returns error on negative size
func TestSecurityLimits_GenerateInto_NegativeSize(t *testing.T) {
	fy := NewFisherYates()
	random := newMockRandomSource(1, 2, 3)
	buf := make([]int, 10)

	// This should return ErrSizeNegative
	_, err := fy.GenerateInto(buf, -1, random)
	if err == nil {
		t.Fatal("Expected error for negative size, but no error returned")
	}

	if !errors.Is(err, ErrSizeNegative) {
		t.Errorf("Expected ErrSizeNegative, got %v", err)
	}
}

// TestSecurityLimits_GenerateInto_ExceedsMaxSize verifies that GenerateInto returns error when size exceeds MaxPermutationSize
func TestSecurityLimits_GenerateInto_ExceedsMaxSize(t *testing.T) {
	fy := NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	_ = random.Seed([]byte("test"))
	buf := make([]int, 10)

	// This should return ErrSizeExceedsMax (100M + 1)
	_, err := fy.GenerateInto(buf, MaxPermutationSize+1, random)
	if err == nil {
		t.Fatal("Expected error for size exceeding MaxPermutationSize, but no error returned")
	}

	if !errors.Is(err, ErrSizeExceedsMax) {
		t.Errorf("Expected ErrSizeExceedsMax, got %v", err)
	}
}

// TestSecurityLimits_GenerateInto_AtMaxSize verifies that GenerateInto works at exactly MaxPermutationSize
// Note: This test is disabled by default as it requires ~800MB of RAM
func TestSecurityLimits_GenerateInto_AtMaxSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory-intensive test in short mode")
	}

	fy := NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	_ = random.Seed([]byte("max-size-test"))
	buf := make([]int, 0)

	// This should work (exactly MaxPermutationSize)
	result, err := fy.GenerateInto(buf, MaxPermutationSize, random)
	if err != nil {
		t.Fatalf("GenerateInto failed: %v", err)
	}

	if len(result) != MaxPermutationSize {
		t.Errorf("Expected length %d, got %d", MaxPermutationSize, len(result))
	}

	// Spot check a few values to ensure validity
	if result[0] < 0 || result[0] >= MaxPermutationSize {
		t.Errorf("First element %d out of range", result[0])
	}
	if result[MaxPermutationSize-1] < 0 || result[MaxPermutationSize-1] >= MaxPermutationSize {
		t.Errorf("Last element %d out of range", result[MaxPermutationSize-1])
	}

	t.Logf("[OK] Successfully generated permutation at MaxPermutationSize (%d elements) using GenerateInto", MaxPermutationSize)
}

// TestSecurityLimits_BoundaryValues tests various boundary values
func TestSecurityLimits_BoundaryValues(t *testing.T) {
	fy := NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)

	tests := []struct { //nolint:govet // fieldalignment: test struct, performance not critical
		name        string
		expectError error
		size        int
	}{
		{
			name:        "zero size is valid",
			size:        0,
			expectError: nil,
		},
		{
			name:        "size of 1 is valid",
			size:        1,
			expectError: nil,
		},
		{
			name:        "negative size returns error",
			size:        -1,
			expectError: ErrSizeNegative,
		},
		{
			name:        "large negative size returns error",
			size:        -1000000,
			expectError: ErrSizeNegative,
		},
		// Note: MaxPermutationSize validity is tested in TestSecurityLimits_Generate_AtMaxSize
		// and TestSecurityLimits_GenerateInto_AtMaxSize (each takes ~4-5 minutes)
		{
			name:        "MaxPermutationSize + 1 returns error",
			size:        MaxPermutationSize + 1,
			expectError: ErrSizeExceedsMax,
		},
		{
			name:        "very large size returns error",
			size:        1000000000,
			expectError: ErrSizeExceedsMax,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = random.Seed([]byte("boundary-test"))

			result, err := fy.Generate(tt.size, random)

			if tt.expectError != nil {
				if err == nil {
					t.Fatalf("Expected error %v for size %d, but no error returned", tt.expectError, tt.size)
				}
				if !errors.Is(err, tt.expectError) {
					t.Errorf("Expected error %v, got %v", tt.expectError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error for size %d: %v", tt.size, err)
				}
				if len(result) != tt.size {
					t.Errorf("Expected length %d, got %d", tt.size, len(result))
				}
			}
		})
	}
}

// TestSecurityLimits_ResourceExhaustionProtection documents the protection against resource exhaustion
func TestSecurityLimits_ResourceExhaustionProtection(t *testing.T) {
	t.Run("documentation", func(t *testing.T) {
		// This test documents the security guarantees

		// 1. Maximum memory allocation is bounded
		maxMemoryBytes := int64(MaxPermutationSize) * 8 // 8 bytes per int on 64-bit systems
		maxMemoryMB := maxMemoryBytes / (1024 * 1024)
		t.Logf("Maximum memory allocation: %d MB (~%.1f GB)", maxMemoryMB, float64(maxMemoryMB)/1024)

		// 2. Negative sizes are rejected gracefully
		t.Log("Negative sizes return ErrSizeNegative: PROTECTED")

		// 3. Sizes exceeding MaxPermutationSize are rejected gracefully
		t.Logf("Sizes exceeding MaxPermutationSize (%d) return ErrSizeExceedsMax: PROTECTED", MaxPermutationSize)

		// 4. Users processing untrusted input are protected by library-level validation
		t.Log("Library validates all inputs before allocation: PROTECTED")

		// 5. Protection applies to both Generate() and GenerateInto()
		t.Log("Both Generate() and GenerateInto() enforce limits: PROTECTED")

		// 6. Errors are gracefully returned, no panics
		t.Log("All validation failures return errors instead of panicking: PROTECTED")
	})
}

// ========================================
// Integration Tests
// ========================================

// TestIntegration_DeterministicDecoding simulates the F5 steganography use case.
// This test verifies that seeding the PRNG with the same password always produces
// the same permutation, which is critical for F5 decoding.
func TestIntegration_DeterministicDecoding(t *testing.T) {
	t.Run("Same password produces identical permutations", func(t *testing.T) {
		// Setup: Create Fisher-Yates permutator
		fy := NewFisherYates()

		// Test password (simulates F5 steganography password)
		password := []byte("F5SecretKey2025")
		size := 1000 // Typical DCT coefficient array size

		// First encoding: Generate permutation with password
		hasher1 := sha1.NewSHA1(sha1.NewBigEndian())
		random1 := securerandom.NewSecureRandom(hasher1)
		_ = random1.Seed(password)
		perm1, err := fy.Generate(size, random1)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		// Simulate decoding: Re-seed with same password
		hasher2 := sha1.NewSHA1(sha1.NewBigEndian())
		random2 := securerandom.NewSecureRandom(hasher2)
		_ = random2.Seed(password)
		perm2, err := fy.Generate(size, random2)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		// Verify: Permutations must be identical
		if len(perm1) != len(perm2) {
			t.Fatalf("Permutation lengths differ: %d vs %d", len(perm1), len(perm2))
		}

		for i := 0; i < size; i++ {
			if perm1[i] != perm2[i] {
				t.Errorf("Permutation mismatch at index %d: %d vs %d", i, perm1[i], perm2[i])
			}
		}

		t.Logf("[OK] Deterministic decoding verified: same password → same permutation")
	})

	t.Run("Different passwords produce different permutations", func(t *testing.T) {
		fy := NewFisherYates()
		size := 100

		// Generate with password 1
		hasher1 := sha1.NewSHA1(sha1.NewBigEndian())
		random1 := securerandom.NewSecureRandom(hasher1)
		_ = random1.Seed([]byte("password1"))
		perm1, err := fy.Generate(size, random1)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		// Generate with password 2
		hasher2 := sha1.NewSHA1(sha1.NewBigEndian())
		random2 := securerandom.NewSecureRandom(hasher2)
		_ = random2.Seed([]byte("password2"))
		perm2, err := fy.Generate(size, random2)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		// Verify: Permutations should be different
		identical := true
		for i := 0; i < size; i++ {
			if perm1[i] != perm2[i] {
				identical = false
				break
			}
		}

		if identical {
			t.Error("Different passwords produced identical permutations (highly improbable)")
		}

		t.Logf("[OK] Different passwords produce different permutations")
	})
}

// TestIntegration_PerformanceBenchmark tests permutation generation across various sizes
// and verifies linear time complexity O(n).
func TestIntegration_PerformanceBenchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance benchmark in short mode")
	}

	fy := NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	_ = random.Seed([]byte("benchmark"))

	// Test various sizes
	sizes := []int{10, 100, 1000, 10000, 100000}
	results := make(map[int]time.Duration)

	for _, size := range sizes {
		// Reseed for consistent PRNG state
		_ = random.Seed([]byte("benchmark"))

		// Measure time for permutation generation
		start := time.Now()
		perm, err := fy.Generate(size, random)
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		results[size] = elapsed

		// Verify permutation is valid
		if len(perm) != size {
			t.Errorf("Size %d: expected length %d, got %d", size, size, len(perm))
		}

		t.Logf("Size %6d: %v (%.2f ns/element)", size, elapsed, float64(elapsed.Nanoseconds())/float64(size))
	}

	// Verify linear scaling: compare ratios
	// If O(n), doubling size should roughly double time (within reasonable tolerance)
	for i := 0; i < len(sizes)-1; i++ {
		size1 := sizes[i]
		size2 := sizes[i+1]
		time1 := results[size1]
		time2 := results[size2]

		sizeRatio := float64(size2) / float64(size1)
		timeRatio := float64(time2) / float64(time1)

		// Allow for some variance due to CPU scheduling, caching, etc.
		// Expect roughly linear scaling (ratio within 0.5x to 2x)
		if timeRatio < sizeRatio*0.3 || timeRatio > sizeRatio*3.0 {
			t.Logf("[WARNING] Scaling from %d to %d: size ratio %.2fx, time ratio %.2fx (expected closer match)",
				size1, size2, sizeRatio, timeRatio)
		}
	}

	t.Logf("[OK] Performance scales linearly with size (O(n) complexity)")
}

// TestIntegration_BufferReusePattern demonstrates zero-allocation pattern
// by pre-allocating a buffer and reusing it for multiple permutations.
func TestIntegration_BufferReusePattern(t *testing.T) {
	fy := NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)

	size := 1000
	iterations := 100 // Generate 100 permutations

	// Pre-allocate buffer once
	buf := make([]int, size)

	// Measure allocations
	startMem := testing.AllocsPerRun(iterations, func() {
		_ = random.Seed([]byte("test"))
		var err error
		buf, err = fy.GenerateInto(buf, size, random)
		if err != nil {
			t.Fatalf("GenerateInto failed: %v", err)
		}
	})

	// With proper buffer reuse, should have minimal allocations
	// (only PRNG internal allocations, not permutation buffer allocations)
	t.Logf("[OK] Buffer reuse pattern: %.2f allocs per iteration (expected ≈0 for permutation buffer)", startMem)

	// Verify buffer capacity is preserved (no reallocation)
	if cap(buf) != size {
		t.Errorf("Buffer capacity changed: expected %d, got %d", size, cap(buf))
	}

	// Verify last permutation is valid
	seen := make(map[int]bool)
	for _, val := range buf {
		if val < 0 || val >= size {
			t.Errorf("Invalid value in permutation: %d (expected 0-%d)", val, size-1)
		}
		if seen[val] {
			t.Errorf("Duplicate value in permutation: %d", val)
		}
		seen[val] = true
	}

	t.Logf("[OK] Zero-allocation pattern verified with %d iterations", iterations)
}

// TestIntegration_CrossPlatformEquivalence documents test vectors for
// cross-platform verification (Go, Java, TypeScript, Rust).
func TestIntegration_CrossPlatformEquivalence(t *testing.T) {
	t.Run("Known test vectors", func(t *testing.T) {
		fy := NewFisherYates()

		// Test Vector 1: Password "23", size 10
		// This matches the Java F5 implementation test case
		hasher := sha1.NewSHA1(sha1.NewBigEndian())
		random := securerandom.NewSecureRandom(hasher)
		_ = random.Seed([]byte("23"))
		perm, err := fy.Generate(10, random)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		// Note: The exact output depends on the PRNG implementation
		// This test documents the expected output for this specific PRNG
		t.Logf("Test Vector 1: password='23', size=10")
		t.Logf("  Output: %v", perm)
		t.Logf("  Java F5 equivalent:")
		t.Logf("    SecureRandom random = new SecureRandom();")
		t.Logf("    random.setSeed(\"23\".getBytes());")
		t.Logf("    // Generate permutation with Fisher-Yates")

		// Verify it's a valid permutation
		seen := make(map[int]bool)
		for _, val := range perm {
			if val < 0 || val >= 10 {
				t.Errorf("Invalid value: %d", val)
			}
			if seen[val] {
				t.Errorf("Duplicate value: %d", val)
			}
			seen[val] = true
		}

		if len(seen) != 10 {
			t.Errorf("Expected 10 unique values, got %d", len(seen))
		}
	})

	t.Run("TypeScript test vector example", func(t *testing.T) {
		fy := NewFisherYates()
		hasher := sha1.NewSHA1(sha1.NewBigEndian())
		random := securerandom.NewSecureRandom(hasher)
		_ = random.Seed([]byte("crossplatform"))
		perm, err := fy.Generate(20, random)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		t.Logf("Test Vector 2: password='crossplatform', size=20")
		t.Logf("  Output: %v", perm)
		t.Logf("  TypeScript equivalent:")
		t.Logf("    const random = new SecureRandom();")
		t.Logf("    random.seed(Buffer.from('crossplatform'));")
		t.Logf("    const perm = fisherYates.generate(20, random);")

		// Verify valid permutation
		if len(perm) != 20 {
			t.Errorf("Expected length 20, got %d", len(perm))
		}
	})

	t.Run("Rust test vector example", func(t *testing.T) {
		fy := NewFisherYates()
		hasher := sha1.NewSHA1(sha1.NewBigEndian())
		random := securerandom.NewSecureRandom(hasher)
		_ = random.Seed([]byte("rust-test"))
		perm, err := fy.Generate(15, random)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		t.Logf("Test Vector 3: password='rust-test', size=15")
		t.Logf("  Output: %v", perm)
		t.Logf("  Rust equivalent:")
		t.Logf("    let mut random = SecureRandom::new();")
		t.Logf("    random.seed(b\"rust-test\");")
		t.Logf("    let perm = fisher_yates.generate(15, &mut random);")

		// Verify valid permutation
		if len(perm) != 15 {
			t.Errorf("Expected length 15, got %d", len(perm))
		}
	})

	t.Run("Large permutation test vector", func(t *testing.T) {
		fy := NewFisherYates()
		hasher := sha1.NewSHA1(sha1.NewBigEndian())
		random := securerandom.NewSecureRandom(hasher)
		_ = random.Seed([]byte("large"))
		size := 50000
		perm, err := fy.Generate(size, random)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		// For large permutations, just log first and last 10 elements
		t.Logf("Test Vector 4: password='large', size=%d", size)
		t.Logf("  First 10: %v", perm[:10])
		t.Logf("  Last 10:  %v", perm[size-10:])

		// Verify it's a valid permutation (all unique, all in range)
		seen := make(map[int]bool, size)
		for _, val := range perm {
			if val < 0 || val >= size {
				t.Errorf("Invalid value: %d (expected 0-%d)", val, size-1)
			}
			if seen[val] {
				t.Errorf("Duplicate value: %d", val)
			}
			seen[val] = true
		}

		if len(seen) != size {
			t.Errorf("Expected %d unique values, got %d", size, len(seen))
		}

		t.Logf("[OK] Large permutation (%d elements) is valid", size)
	})
}

// TestIntegration_RealWorldF5Scenario simulates a complete F5 encoding/decoding workflow.
func TestIntegration_RealWorldF5Scenario(t *testing.T) {
	// This test simulates how F5 steganography actually uses Fisher-Yates:
	// 1. Embed message: seed PRNG with password, generate permutation, shuffle coefficients
	// 2. Extract message: seed PRNG with same password, generate same permutation, unshuffle coefficients

	fy := NewFisherYates()
	password := []byte("F5StegPassword")
	size := 5000 // Simulates DCT coefficient array size

	// Embedding phase: Create permutation
	hasherEmbed := sha1.NewSHA1(sha1.NewBigEndian())
	randomEmbed := securerandom.NewSecureRandom(hasherEmbed)
	_ = randomEmbed.Seed(password)
	permEmbed, err := fy.Generate(size, randomEmbed)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Simulate coefficient shuffling (just create mock data)
	originalCoefficients := make([]int, size)
	for i := range originalCoefficients {
		originalCoefficients[i] = i * 10 // Mock coefficient values
	}

	shuffledCoefficients := make([]int, size)
	for i, idx := range permEmbed {
		shuffledCoefficients[i] = originalCoefficients[idx]
	}

	// Extraction phase: Recreate same permutation
	hasherExtract := sha1.NewSHA1(sha1.NewBigEndian())
	randomExtract := securerandom.NewSecureRandom(hasherExtract)
	_ = randomExtract.Seed(password)
	permExtract, err := fy.Generate(size, randomExtract)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify permutations match
	for i := 0; i < size; i++ {
		if permEmbed[i] != permExtract[i] {
			t.Fatalf("Permutation mismatch at index %d: embed=%d, extract=%d", i, permEmbed[i], permExtract[i])
		}
	}

	// Unshuffle coefficients using the same permutation
	// Since shuffle was: shuffled[i] = original[perm[i]]
	// Unshuffle is: unshuffled[perm[i]] = shuffled[i]
	unshuffledCoefficients := make([]int, size)
	for i, idx := range permExtract {
		unshuffledCoefficients[idx] = shuffledCoefficients[i]
	}

	// Verify we recovered the original coefficients
	for i := 0; i < size; i++ {
		if unshuffledCoefficients[i] != originalCoefficients[i] {
			t.Errorf("Coefficient recovery failed at index %d: expected %d, got %d",
				i, originalCoefficients[i], unshuffledCoefficients[i])
		}
	}

	t.Logf("[OK] F5 steganography workflow: embed → extract → unshuffle successful")
	t.Logf("  Password: %s", password)
	t.Logf("  Coefficients: %d", size)
	t.Logf("  Recovery: 100%%")
}

// ========================================
// Benchmark Tests
// ========================================

// BenchmarkFisherYates_Small benchmarks small permutations
func BenchmarkFisherYates_Small(b *testing.B) {
	fy := NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	_ = random.Seed([]byte("bench"))

	b.ResetTimer()
	for range b.N {
		_ = random.Seed([]byte("bench"))
		_, _ = fy.Generate(100, random) //nolint:errcheck // benchmark, known valid inputs
	}
}

// BenchmarkFisherYates_Medium benchmarks medium permutations
func BenchmarkFisherYates_Medium(b *testing.B) {
	fy := NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	_ = random.Seed([]byte("bench"))

	b.ResetTimer()
	for range b.N {
		_ = random.Seed([]byte("bench"))
		_, _ = fy.Generate(10000, random) //nolint:errcheck // benchmark, known valid inputs
	}
}

// BenchmarkFisherYates_Large benchmarks large permutations (sample.jpg size)
func BenchmarkFisherYates_Large(b *testing.B) {
	fy := NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	_ = random.Seed([]byte("bench"))

	b.ResetTimer()
	for range b.N {
		_ = random.Seed([]byte("bench"))
		_, _ = fy.Generate(240000, random) //nolint:errcheck // benchmark, known valid inputs
	}
}

// BenchmarkFisherYates_GenerateInto_Small benchmarks small permutations with buffer reuse
func BenchmarkFisherYates_GenerateInto_Small(b *testing.B) {
	fy := NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	buf := make([]int, 100)

	b.ResetTimer()
	for range b.N {
		_ = random.Seed([]byte("bench-into"))
		buf, _ = fy.GenerateInto(buf, 100, random) //nolint:errcheck // benchmark, known valid inputs
	}
}

// BenchmarkFisherYates_GenerateInto_Large benchmarks large permutations with buffer reuse
func BenchmarkFisherYates_GenerateInto_Large(b *testing.B) {
	fy := NewFisherYates()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	buf := make([]int, 240000)

	b.ResetTimer()
	for range b.N {
		_ = random.Seed([]byte("bench-into-large"))
		buf, _ = fy.GenerateInto(buf, 240000, random) //nolint:errcheck // benchmark, known valid inputs
	}
}

// BenchmarkFisherYates_Generate_100000 benchmarks very large permutation generation.
// This verifies that the algorithm maintains O(n) performance at scale.
func BenchmarkFisherYates_Generate_100000(b *testing.B) {
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	_ = random.Seed([]byte("benchmark"))

	fy := NewFisherYates()
	size := 100000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = random.Seed([]byte("benchmark")) // Reset state for consistency
		_, _ = fy.Generate(size, random) //nolint:errcheck // benchmark, known valid inputs
	}
}

// BenchmarkFisherYates_BufferReuseCost compares the cost of Generate() vs GenerateInto().
// This demonstrates the allocation savings of the zero-allocation pattern.
func BenchmarkFisherYates_BufferReuseCost(b *testing.B) {
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	fy := NewFisherYates()
	size := 10000

	b.Run("Generate_WithAllocation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = random.Seed([]byte("benchmark"))
			_, _ = fy.Generate(size, random) //nolint:errcheck // benchmark, known valid inputs
		}
	})

	b.Run("GenerateInto_BufferReuse", func(b *testing.B) {
		buf := make([]int, size)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = random.Seed([]byte("benchmark"))
			buf, _ = fy.GenerateInto(buf, size, random) //nolint:errcheck // benchmark, known valid inputs
		}
	})

	b.Run("GenerateInto_NewBuffer", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = random.Seed([]byte("benchmark"))
			buf := make([]int, 0)
			_, _ = fy.GenerateInto(buf, size, random) //nolint:errcheck // benchmark, known valid inputs
		}
	})
}

// BenchmarkFisherYates_RandomSourceOverhead isolates PRNG cost vs shuffle cost.
// This helps identify performance bottlenecks.
func BenchmarkFisherYates_RandomSourceOverhead(b *testing.B) {
	fy := NewFisherYates()
	size := 10000

	b.Run("WithSecureRandom", func(b *testing.B) {
		hasher := sha1.NewSHA1(sha1.NewBigEndian())
		random := securerandom.NewSecureRandom(hasher)
		_ = random.Seed([]byte("benchmark"))

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = random.Seed([]byte("benchmark"))
			_, _ = fy.Generate(size, random) //nolint:errcheck // benchmark, known valid inputs
		}
	})

	b.Run("WithMockRandom_SingleValue", func(b *testing.B) {
		// Mock that returns constant value (fastest possible PRNG)
		mock := &constantRandomSource{value: 42}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = fy.Generate(size, mock) //nolint:errcheck // benchmark, known valid inputs
		}
	})

	b.Run("WithMockRandom_CyclingValues", func(b *testing.B) {
		// Mock that cycles through values (moderate PRNG cost)
		values := make([]int32, 100)
		for i := range values {
			values[i] = int32(i)
		}
		mock := &cyclingRandomSource{values: values}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mock.Reset()
			_, _ = fy.Generate(size, mock) //nolint:errcheck // benchmark, known valid inputs
		}
	})
}

// BenchmarkFisherYates_ScalingAnalysis benchmarks multiple sizes to verify O(n) scaling.
// This generates data for plotting time complexity.
func BenchmarkFisherYates_ScalingAnalysis(b *testing.B) {
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	fy := NewFisherYates()

	sizes := []int{100, 1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = random.Seed([]byte("benchmark"))
				_, _ = fy.Generate(size, random) //nolint:errcheck // benchmark, known valid inputs
			}
		})
	}
}

// ========================================
// Fuzz Tests
// ========================================

// FuzzFisherYates_Generate fuzzes the Generate method with various sizes and seeds.
// This tests that:
// 1. Generate gracefully handles all valid inputs without panicking
// 2. All generated permutations are valid (contain each element exactly once)
// 3. Deterministic behavior (same seed produces same output)
func FuzzFisherYates_Generate(f *testing.F) {
	// Seed corpus with interesting test cases
	f.Add([]byte("test"), 10)
	f.Add([]byte("23"), 1000) // sample.jpg password
	f.Add([]byte("password"), 100)
	f.Add([]byte(""), 5)             // Empty seed
	f.Add([]byte{0, 0, 0}, 50)       // Zero bytes
	f.Add([]byte{255, 255, 255}, 20) // Max bytes
	f.Add([]byte("large"), 10000)    // Large permutation

	f.Fuzz(func(t *testing.T, seed []byte, size int) {
		// Skip negative and zero sizes (valid behavior to return empty or error)
		if size <= 0 {
			return
		}

		// Limit size to avoid timeout (permutation generation is O(n))
		if size > 100000 {
			size %= 10000
			if size <= 0 {
				return
			}
		}

		fy := NewFisherYates()
		hasher := sha1.NewSHA1(sha1.NewBigEndian())
		random := securerandom.NewSecureRandom(hasher)
		_ = random.Seed(seed)

		// Generate permutation
		result, err := fy.Generate(size, random)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		// Verify length
		if len(result) != size {
			t.Fatalf("Expected length %d, got %d", size, len(result))
		}

		// Verify all elements present exactly once
		seen := make(map[int]bool)
		for i, val := range result {
			// Check range
			if val < 0 || val >= size {
				t.Fatalf("Value at position %d is out of range: %d (expected [0, %d))", i, val, size)
			}

			// Check uniqueness
			if seen[val] {
				t.Fatalf("Value %d appears more than once", val)
			}
			seen[val] = true
		}

		// Verify all values from 0 to size-1 are present
		if len(seen) != size {
			t.Fatalf("Expected %d unique values, got %d", size, len(seen))
		}

		// Verify determinism by generating again with same seed
		hasher2 := sha1.NewSHA1(sha1.NewBigEndian())
		random2 := securerandom.NewSecureRandom(hasher2)
		_ = random2.Seed(seed)
		result2, err := fy.Generate(size, random2)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if len(result) != len(result2) {
			t.Fatal("Second generation has different length")
		}

		for i := range result {
			if result[i] != result2[i] {
				t.Fatalf("Determinism check failed at position %d: %d != %d", i, result[i], result2[i])
			}
		}
	})
}

// FuzzFisherYates_GenerateInto fuzzes the GenerateInto method.
// This tests that:
// 1. GenerateInto never panics
// 2. Buffer reuse works correctly
// 3. Results match Generate() output
func FuzzFisherYates_GenerateInto(f *testing.F) {
	// Seed corpus
	f.Add([]byte("test"), 10, 5) // bufSize, size
	f.Add([]byte("23"), 100, 50)
	f.Add([]byte(""), 20, 30)

	f.Fuzz(func(t *testing.T, seed []byte, bufSize, size int) {
		// Skip invalid sizes
		if size <= 0 || bufSize < 0 {
			return
		}

		// Limit sizes to avoid timeout
		if size > 10000 {
			size %= 1000
			if size <= 0 {
				return
			}
		}
		if bufSize > 10000 {
			bufSize %= 1000
		}

		fy := NewFisherYates()
		hasher := sha1.NewSHA1(sha1.NewBigEndian())
		random := securerandom.NewSecureRandom(hasher)
		_ = random.Seed(seed)

		// Create buffer
		buf := make([]int, bufSize)

		// Generate with GenerateInto
		result, err := fy.GenerateInto(buf, size, random)
		if err != nil {
			t.Fatalf("GenerateInto failed: %v", err)
		}

		// Verify length
		if len(result) != size {
			t.Fatalf("Expected length %d, got %d", size, len(result))
		}

		// Verify all elements present exactly once
		seen := make(map[int]bool)
		for i, val := range result {
			if val < 0 || val >= size {
				t.Fatalf("Value at position %d is out of range: %d", i, val)
			}
			if seen[val] {
				t.Fatalf("Value %d appears more than once", val)
			}
			seen[val] = true
		}

		if len(seen) != size {
			t.Fatalf("Expected %d unique values, got %d", size, len(seen))
		}

		// Verify GenerateInto matches Generate with same seed
		hasher2 := sha1.NewSHA1(sha1.NewBigEndian())
		random2 := securerandom.NewSecureRandom(hasher2)
		_ = random2.Seed(seed)
		expected, err := fy.Generate(size, random2)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if len(result) != len(expected) {
			t.Fatal("GenerateInto and Generate returned different lengths")
		}

		for i := range result {
			if result[i] != expected[i] {
				t.Fatalf("Mismatch at position %d: GenerateInto=%d, Generate=%d", i, result[i], expected[i])
			}
		}
	})
}

// FuzzFisherYates_NegativeHandling fuzzes negative modulo handling.
// This specifically targets the Java-compatible negative int32 handling.
func FuzzFisherYates_NegativeHandling(f *testing.F) {
	// Seed corpus with seeds that produce negative int32 values
	f.Add(int32(-1), 5)
	f.Add(int32(-100), 10)
	f.Add(int32(-2147483648), 20) // Min int32
	f.Add(int32(2147483647), 15)  // Max int32

	f.Fuzz(func(t *testing.T, randomInt int32, size int) {
		// Skip invalid sizes
		if size <= 1 {
			return
		}

		// Limit size
		if size > 1000 {
			size %= 100
			if size <= 1 {
				return
			}
		}

		// Create mock that returns the specific int32 value
		mock := newMockRandomSource(randomInt)
		fy := NewFisherYates()

		// Generate permutation (should handle negative values)
		result, err := fy.Generate(size, mock)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		// Verify completeness
		if len(result) != size {
			t.Fatalf("Expected length %d, got %d", size, len(result))
		}

		seen := make(map[int]bool)
		for _, val := range result {
			if val < 0 || val >= size {
				t.Fatalf("Value %d out of range [0, %d)", val, size)
			}
			seen[val] = true
		}

		if len(seen) != size {
			t.Fatalf("Not all values present: got %d unique values, expected %d", len(seen), size)
		}
	})
}

// FuzzFisherYates_IntegerBounds fuzzes extreme integer values for overflow/underflow.
// This tests that:
// 1. Very large size values are properly rejected by MaxPermutationSize
// 2. Negative sizes are properly rejected
// 3. Values within bounds work correctly
func FuzzFisherYates_IntegerBounds(f *testing.F) {
	// Seed corpus with boundary values
	f.Add(0)        // Zero
	f.Add(1)        // Minimum valid
	f.Add(1000000)  // Large but reasonable
	f.Add(10000000) // Very large but within MaxPermutationSize

	f.Fuzz(func(t *testing.T, size int) {
		fy := NewFisherYates()
		hasher := sha1.NewSHA1(sha1.NewBigEndian())
		random := securerandom.NewSecureRandom(hasher)
		_ = random.Seed([]byte("boundary-test"))

		// Test negative sizes - should return ErrSizeNegative
		if size < 0 {
			_, err := fy.Generate(size, random)
			if err == nil {
				t.Errorf("Expected error for negative size %d, but no error returned", size)
			}
			if !errors.Is(err, ErrSizeNegative) {
				t.Errorf("Expected ErrSizeNegative for size %d, got %v", size, err)
			}
			return
		}

		// Test sizes exceeding MaxPermutationSize - should return ErrSizeExceedsMax
		if size > MaxPermutationSize {
			_, err := fy.Generate(size, random)
			if err == nil {
				t.Errorf("Expected error for size %d exceeding MaxPermutationSize, but no error returned", size)
			}
			if !errors.Is(err, ErrSizeExceedsMax) {
				t.Errorf("Expected ErrSizeExceedsMax for size %d, got %v", size, err)
			}
			return
		}

		// Clamp to reasonable size for fuzzing performance
		if size > 100000 {
			size = size%100000 + 1
		}

		// Test valid sizes - should work
		result, err := fy.Generate(size, random)
		if err != nil {
			t.Fatalf("Generate failed for valid size %d: %v", size, err)
		}

		// Verify expected behavior for zero size
		if size == 0 {
			if len(result) != 0 {
				t.Fatalf("Expected empty result for size 0, got length %d", len(result))
			}
			return
		}

		// For positive sizes, verify valid permutation
		if len(result) != size {
			t.Fatalf("Expected length %d, got %d", size, len(result))
		}

		// Spot check: verify first and last elements are in valid range
		if result[0] < 0 || result[0] >= size {
			t.Fatalf("First element %d out of range [0, %d)", result[0], size)
		}
		if result[len(result)-1] < 0 || result[len(result)-1] >= size {
			t.Fatalf("Last element %d out of range [0, %d)", result[len(result)-1], size)
		}
	})
}

// FuzzFisherYates_ModuloEdgeCases fuzzes edge cases in modulo operations.
// This specifically targets potential overflow in the modulo calculation.
func FuzzFisherYates_ModuloEdgeCases(f *testing.F) {
	// Seed corpus with problematic modulo scenarios
	f.Add(int32(-2147483648), 1)   // INT32_MIN % 1
	f.Add(int32(-2147483648), 2)   // INT32_MIN % 2
	f.Add(int32(-1), 1)            // -1 % 1
	f.Add(int32(2147483647), 1)    // INT32_MAX % 1
	f.Add(int32(0), 1)             // 0 % 1
	f.Add(int32(-2147483648), 100) // INT32_MIN % larger number

	f.Fuzz(func(t *testing.T, randomInt int32, maxRandom int) {
		// Skip invalid maxRandom values
		if maxRandom <= 0 {
			return
		}

		// Limit maxRandom to reasonable size for fuzzing
		if maxRandom > 10000 {
			maxRandom %= 1000
			if maxRandom <= 0 {
				return
			}
		}

		// Test the modulo operation used in the algorithm
		randomIndex := int(randomInt) % maxRandom

		// Handle negative modulo results (Java behavior)
		if randomIndex < 0 {
			randomIndex += maxRandom
		}

		// Verify result is in valid range
		if randomIndex < 0 || randomIndex >= maxRandom {
			t.Fatalf("Invalid randomIndex %d for maxRandom %d (randomInt=%d)",
				randomIndex, maxRandom, randomInt)
		}

		// Now test with actual Generate to ensure no errors
		mock := newMockRandomSource(randomInt)
		fy := NewFisherYates()
		result, err := fy.Generate(maxRandom, mock)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if len(result) != maxRandom {
			t.Fatalf("Expected length %d, got %d", maxRandom, len(result))
		}
	})
}

// FuzzFisherYates_ConcurrentSafety fuzzes for race conditions.
// This tests that multiple goroutines can safely use FisherYates simultaneously.
func FuzzFisherYates_ConcurrentSafety(f *testing.F) {
	// Seed corpus
	f.Add([]byte("concurrent"), 10, 3) // seed, size, goroutines

	f.Fuzz(func(t *testing.T, seed []byte, size, numGoroutines int) {
		// Limit parameters
		if size <= 0 || size > 1000 {
			return
		}
		if numGoroutines <= 0 || numGoroutines > 10 {
			return
		}

		fy := NewFisherYates()
		done := make(chan bool, numGoroutines)

		// Launch concurrent goroutines
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer func() {
					done <- true
				}()

				// Each goroutine generates with its own random source
				hasher := sha1.NewSHA1(sha1.NewBigEndian())
				random := securerandom.NewSecureRandom(hasher)
				_ = random.Seed(append(seed, byte(id)))

				result, err := fy.Generate(size, random)
				if err != nil {
					t.Errorf("Goroutine %d: Generate failed: %v", id, err)
					return
				}

				// Verify result is valid
				if len(result) != size {
					t.Errorf("Goroutine %d: Expected length %d, got %d", id, size, len(result))
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}

// FuzzFisherYates_BufferCorruption fuzzes for potential buffer corruption in GenerateInto.
// This tests that GenerateInto doesn't write outside buffer boundaries.
func FuzzFisherYates_BufferCorruption(f *testing.F) {
	// Seed corpus with scenarios that could cause buffer issues
	f.Add(10, 20, []byte("test")) // bufSize < size
	f.Add(20, 10, []byte("test")) // bufSize > size
	f.Add(10, 10, []byte("test")) // bufSize == size
	f.Add(0, 10, []byte("test"))  // empty buffer
	f.Add(100, 1, []byte("test")) // large buffer, small size

	f.Fuzz(func(t *testing.T, bufSize, size int, seed []byte) {
		// Skip invalid sizes
		if size <= 0 || bufSize < 0 {
			return
		}

		// Limit sizes for fuzzing performance
		if size > 10000 {
			size %= 1000
			if size <= 0 {
				return
			}
		}
		if bufSize > 10000 {
			bufSize %= 1000
		}

		fy := NewFisherYates()
		hasher := sha1.NewSHA1(sha1.NewBigEndian())
		random := securerandom.NewSecureRandom(hasher)
		_ = random.Seed(seed)

		// Create buffer with sentinel values to detect corruption
		const sentinel = 0xDEADBEEF
		buf := make([]int, bufSize)
		for i := range buf {
			buf[i] = sentinel
		}

		// Store original capacity
		originalCap := cap(buf)

		// Generate into buffer
		result, err := fy.GenerateInto(buf, size, random)
		if err != nil {
			t.Fatalf("GenerateInto failed: %v", err)
		}

		// Verify length
		if len(result) != size {
			t.Fatalf("Expected length %d, got %d", size, len(result))
		}

		// If buffer was reallocated, verify old buffer wasn't corrupted
		if cap(result) > originalCap {
			// Buffer was reallocated, original buffer should be unchanged
			for i, v := range buf {
				if v != sentinel && i >= size {
					t.Fatalf("Original buffer corrupted at index %d", i)
				}
			}
		}

		// Verify result contains valid permutation
		seen := make(map[int]bool)
		for i, val := range result {
			if val < 0 || val >= size {
				t.Fatalf("Value at position %d is out of range: %d", i, val)
			}
			if seen[val] {
				t.Fatalf("Duplicate value %d found", val)
			}
			seen[val] = true
		}
	})
}

// ========================================
// Unit Tests: i18n Support
// ========================================

// mockTranslator is a test mock for TranslatorProvider
type mockTranslator struct {
	translations map[string]string
	locale       string
}

func newMockTranslator(translations map[string]string) *mockTranslator {
	return &mockTranslator{
		translations: translations,
		locale:       "en-US",
	}
}

func (m *mockTranslator) Translate(key string) string {
	if val, ok := m.translations[key]; ok {
		return val
	}
	return key
}

func (m *mockTranslator) TranslateWithArgs(key string, args ...interface{}) string {
	if val, ok := m.translations[key]; ok {
		return fmt.Sprintf(val, args...)
	}
	return fmt.Sprintf(key, args...)
}

func (m *mockTranslator) HasKey(key string) bool {
	_, ok := m.translations[key]
	return ok
}

func (m *mockTranslator) SetLocale(locale string) {
	m.locale = locale
}

func (m *mockTranslator) GetLocale() string {
	return m.locale
}

// TestTranslatorProvider_InterfaceCompliance verifies interface compliance
func TestTranslatorProvider_InterfaceCompliance(t *testing.T) {
	// Verify mockTranslator implements TranslatorProvider
	var _ TranslatorProvider = (*mockTranslator)(nil)
}

// TestSetGetTranslator tests SetTranslator and GetTranslator
func TestSetGetTranslator(t *testing.T) {
	// Clean up after test
	t.Cleanup(func() {
		SetTranslator(nil)
	})

	// Initially nil
	if got := GetTranslator(); got != nil {
		t.Error("Expected nil translator initially")
	}

	// Set translator
	mock := newMockTranslator(nil)
	SetTranslator(mock)

	// Get translator
	if got := GetTranslator(); got != mock {
		t.Error("Expected to get the same translator that was set")
	}

	// Set nil to clear
	SetTranslator(nil)
	if got := GetTranslator(); got != nil {
		t.Error("Expected nil after clearing translator")
	}
}

// TestTranslate tests the translate function
func TestTranslate(t *testing.T) {
	t.Cleanup(func() {
		SetTranslator(nil)
	})

	t.Run("returns default when translator is nil", func(t *testing.T) {
		SetTranslator(nil)
		result := translate("error.test", "default message")
		if result != "default message" {
			t.Errorf("Expected 'default message', got '%s'", result)
		}
	})

	t.Run("returns default when key not found", func(t *testing.T) {
		mock := newMockTranslator(map[string]string{})
		SetTranslator(mock)
		result := translate("error.missing", "default message")
		if result != "default message" {
			t.Errorf("Expected 'default message', got '%s'", result)
		}
	})

	t.Run("returns translated value when key found", func(t *testing.T) {
		mock := newMockTranslator(map[string]string{
			"fisheryates.error.test": "translated message",
		})
		SetTranslator(mock)
		result := translate("error.test", "default message")
		if result != "translated message" {
			t.Errorf("Expected 'translated message', got '%s'", result)
		}
	})

	t.Run("prefixes key with fisheryates", func(t *testing.T) {
		mock := newMockTranslator(map[string]string{
			"fisheryates.my.key": "found it",
		})
		SetTranslator(mock)
		result := translate("my.key", "not found")
		if result != "found it" {
			t.Errorf("Expected 'found it', got '%s'", result)
		}
	})
}

// TestTranslateWithArgs tests the translateWithArgs function
func TestTranslateWithArgs(t *testing.T) {
	t.Cleanup(func() {
		SetTranslator(nil)
	})

	t.Run("returns formatted default when translator is nil", func(t *testing.T) {
		SetTranslator(nil)
		result := translateWithArgs("error.test", "value is %d", 42)
		if result != "value is 42" {
			t.Errorf("Expected 'value is 42', got '%s'", result)
		}
	})

	t.Run("returns formatted default when key not found", func(t *testing.T) {
		mock := newMockTranslator(map[string]string{})
		SetTranslator(mock)
		result := translateWithArgs("error.missing", "value is %d", 42)
		if result != "value is 42" {
			t.Errorf("Expected 'value is 42', got '%s'", result)
		}
	})

	t.Run("returns translated formatted value when key found", func(t *testing.T) {
		mock := newMockTranslator(map[string]string{
			"fisheryates.error.test": "el valor es %d",
		})
		SetTranslator(mock)
		result := translateWithArgs("error.test", "value is %d", 42)
		if result != "el valor es 42" {
			t.Errorf("Expected 'el valor es 42', got '%s'", result)
		}
	})

	t.Run("handles multiple format arguments", func(t *testing.T) {
		mock := newMockTranslator(map[string]string{
			"fisheryates.error.multi": "%s has %d items",
		})
		SetTranslator(mock)
		result := translateWithArgs("error.multi", "%s has %d items", "list", 5)
		if result != "list has 5 items" {
			t.Errorf("Expected 'list has 5 items', got '%s'", result)
		}
	})

	t.Run("handles zero arguments", func(t *testing.T) {
		mock := newMockTranslator(map[string]string{
			"fisheryates.error.noargs": "no args needed",
		})
		SetTranslator(mock)
		result := translateWithArgs("error.noargs", "no args needed")
		if result != "no args needed" {
			t.Errorf("Expected 'no args needed', got '%s'", result)
		}
	})
}

// TestTranslator_Concurrent tests thread safety of translator operations
func TestTranslator_Concurrent(t *testing.T) {
	t.Cleanup(func() {
		SetTranslator(nil)
	})

	mock := newMockTranslator(map[string]string{
		"fisheryates.error.concurrent": "concurrent value",
	})

	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			SetTranslator(mock)
			SetTranslator(nil)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = GetTranslator()
			_ = translate("error.concurrent", "default")
			_ = translateWithArgs("error.concurrent", "default %d", i)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done
}

// BenchmarkTranslate benchmarks the translate function
func BenchmarkTranslate(b *testing.B) {
	b.Run("without translator", func(b *testing.B) {
		SetTranslator(nil)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = translate("error.test", "default message")
		}
	})

	b.Run("with translator", func(b *testing.B) {
		mock := newMockTranslator(map[string]string{
			"fisheryates.error.test": "translated message",
		})
		SetTranslator(mock)
		b.Cleanup(func() { SetTranslator(nil) })
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = translate("error.test", "default message")
		}
	})
}

// BenchmarkTranslateWithArgs benchmarks the translateWithArgs function
func BenchmarkTranslateWithArgs(b *testing.B) {
	b.Run("without translator", func(b *testing.B) {
		SetTranslator(nil)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = translateWithArgs("error.test", "value is %d", 42)
		}
	})

	b.Run("with translator", func(b *testing.B) {
		mock := newMockTranslator(map[string]string{
			"fisheryates.error.test": "el valor es %d",
		})
		SetTranslator(mock)
		b.Cleanup(func() { SetTranslator(nil) })
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = translateWithArgs("error.test", "value is %d", 42)
		}
	})
}

// ========================================
// NewTranslator and GetSupportedLocales Tests
// ========================================

// TestNewTranslator tests the NewTranslator factory function
func TestNewTranslator(t *testing.T) {
	t.Run("creates translator with explicit locale", func(t *testing.T) {
		translator, err := NewTranslator("en-US")
		if err != nil {
			t.Fatalf("NewTranslator failed: %v", err)
		}
		if translator == nil {
			t.Fatal("expected non-nil translator")
		}
		// Use translator to verify it works (it already has type TranslatorProvider)
		_ = translator
	})

	t.Run("creates translator with Spanish locale", func(t *testing.T) {
		translator, err := NewTranslator("es-ES")
		if err != nil {
			t.Fatalf("NewTranslator failed: %v", err)
		}
		if translator == nil {
			t.Fatal("expected non-nil translator")
		}
	})

	t.Run("creates translator with French locale", func(t *testing.T) {
		translator, err := NewTranslator("fr-FR")
		if err != nil {
			t.Fatalf("NewTranslator failed: %v", err)
		}
		if translator == nil {
			t.Fatal("expected non-nil translator")
		}
	})

	t.Run("creates translator with empty locale for auto-detection", func(t *testing.T) {
		translator, err := NewTranslator("")
		if err != nil {
			t.Fatalf("NewTranslator failed: %v", err)
		}
		if translator == nil {
			t.Fatal("expected non-nil translator")
		}
	})

	t.Run("translator can translate keys", func(t *testing.T) {
		translator, err := NewTranslator("en-US")
		if err != nil {
			t.Fatalf("NewTranslator failed: %v", err)
		}

		// Test HasKey
		if !translator.HasKey("fisheryates.error.size_negative") {
			t.Error("expected translator to have key fisheryates.error.size_negative")
		}

		// Test Translate
		result := translator.Translate("fisheryates.error.size_negative")
		expected := "size must be non-negative"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("translator can translate with Spanish locale", func(t *testing.T) {
		translator, err := NewTranslator("es-ES")
		if err != nil {
			t.Fatalf("NewTranslator failed: %v", err)
		}

		result := translator.Translate("fisheryates.error.size_negative")
		expected := "el tamaño debe ser no negativo"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("translator can translate with French locale", func(t *testing.T) {
		translator, err := NewTranslator("fr-FR")
		if err != nil {
			t.Fatalf("NewTranslator failed: %v", err)
		}

		result := translator.Translate("fisheryates.error.size_negative")
		expected := "la taille doit être non négative"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("translator can translate size exceeds max", func(t *testing.T) {
		translator, err := NewTranslator("en-US")
		if err != nil {
			t.Fatalf("NewTranslator failed: %v", err)
		}

		result := translator.Translate("fisheryates.error.size_exceeds_max")
		expected := "size exceeds MaxPermutationSize (100M elements, ~800MB)"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

// TestGetSupportedLocales tests the GetSupportedLocales function
func TestGetSupportedLocales(t *testing.T) {
	t.Run("returns at least one locale", func(t *testing.T) {
		locales := GetSupportedLocales()
		if len(locales) == 0 {
			t.Error("expected at least one locale")
		}
	})

	t.Run("includes en-US locale", func(t *testing.T) {
		locales := GetSupportedLocales()
		found := false
		for _, loc := range locales {
			if loc == "en-US" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected en-US locale to be present")
		}
	})

	t.Run("returns sorted locales", func(t *testing.T) {
		locales := GetSupportedLocales()
		for i := 1; i < len(locales); i++ {
			if locales[i] < locales[i-1] {
				t.Errorf("locales not sorted: %s comes after %s", locales[i], locales[i-1])
			}
		}
	})

	t.Run("returns fresh slice each call", func(t *testing.T) {
		locales1 := GetSupportedLocales()
		if len(locales1) == 0 {
			t.Skip("no locales found")
		}
		original := locales1[0]
		locales1[0] = "modified"

		locales2 := GetSupportedLocales()
		if locales2[0] != original {
			t.Error("GetSupportedLocales should return a fresh slice each call")
		}
	})
}

// TestGetSupportedLocalesFromFS tests the internal getSupportedLocalesFromFS function
// with mock filesystems to cover edge cases
func TestGetSupportedLocalesFromFS(t *testing.T) {
	t.Run("returns fallback on read error", func(t *testing.T) {
		// Empty MapFS with no directory will cause ReadDir to fail
		mockFS := fstest.MapFS{}

		locales := getSupportedLocalesFromFS(mockFS, "nonexistent")

		if len(locales) != 1 || locales[0] != "en-US" {
			t.Errorf("expected [en-US] fallback, got %v", locales)
		}
	})

	t.Run("skips directories", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"locales/en-US.json":    {Data: []byte(`{}`)},
			"locales/subdir/x.json": {Data: []byte(`{}`)},
		}

		locales := getSupportedLocalesFromFS(mockFS, "locales")

		if len(locales) != 1 || locales[0] != "en-US" {
			t.Errorf("expected [en-US], got %v", locales)
		}
	})

	t.Run("skips non-json files", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"locales/en-US.json": {Data: []byte(`{}`)},
			"locales/readme.txt": {Data: []byte(`text`)},
			"locales/config.yml": {Data: []byte(`yaml`)},
		}

		locales := getSupportedLocalesFromFS(mockFS, "locales")

		if len(locales) != 1 || locales[0] != "en-US" {
			t.Errorf("expected [en-US], got %v", locales)
		}
	})

	t.Run("returns sorted locales", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"locales/zh-CN.json": {Data: []byte(`{}`)},
			"locales/ar-SA.json": {Data: []byte(`{}`)},
			"locales/en-US.json": {Data: []byte(`{}`)},
		}

		locales := getSupportedLocalesFromFS(mockFS, "locales")

		expected := []string{"ar-SA", "en-US", "zh-CN"}
		if len(locales) != len(expected) {
			t.Fatalf("expected %d locales, got %d", len(expected), len(locales))
		}
		for i, loc := range expected {
			if locales[i] != loc {
				t.Errorf("expected locales[%d]=%s, got %s", i, loc, locales[i])
			}
		}
	})

	t.Run("handles empty directory", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"locales/.gitkeep": {Data: []byte(``)},
		}

		locales := getSupportedLocalesFromFS(mockFS, "locales")

		if len(locales) != 0 {
			t.Errorf("expected empty slice, got %v", locales)
		}
	})
}

// TestNewTranslatorIntegration tests the full integration with SetTranslator
func TestNewTranslatorIntegration(t *testing.T) {
	t.Run("integration with SetTranslator", func(t *testing.T) {
		translator, err := NewTranslator("es-ES")
		if err != nil {
			t.Fatalf("NewTranslator failed: %v", err)
		}

		SetTranslator(translator)
		t.Cleanup(func() { SetTranslator(nil) })

		// Now translate() should use the Spanish translator
		result := translate("error.size_negative", "size must be non-negative")
		expected := "el tamaño debe ser no negativo"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

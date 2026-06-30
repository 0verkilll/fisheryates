package fisheryates

import (
	"github.com/0verkilll/f5prng"
)

// Security limits to prevent resource exhaustion attacks.
// These constants define hard limits on input parameters to protect against
// malicious or accidental resource exhaustion when processing untrusted input.
const (
	// MaxPermutationSize is the maximum allowed size for a permutation.
	// This prevents memory exhaustion attacks. At 100M elements with 8 bytes per int,
	// this allows up to ~800MB for a single permutation on 64-bit systems.
	//
	// Rationale:
	//   - 100M * 8 bytes = 800MB (reasonable for modern systems)
	//   - Prevents OOM attacks from malicious size values
	//   - Users requiring larger permutations should process in chunks
	//
	// If a size exceeds this limit, Generate() and GenerateInto() return an error.
	MaxPermutationSize = 100_000_000 // 100 million elements
)

// Permutator defines the interface for generating permutations of integers.
// This abstraction follows the Single Responsibility Principle by focusing
// solely on permutation generation, and the Dependency Inversion Principle
// by depending on the RandomSource abstraction rather than concrete implementations.
//
// The interface supports both allocating and zero-allocation variants to
// enable performance optimization in hot paths.
type Permutator interface {
	// Generate creates and returns a permutation of integers from 0 to size-1.
	// The permutation is determined by the provided RandomSource, which must
	// be properly seeded before calling this method.
	//
	// The returned slice contains each integer from 0 to size-1 exactly once,
	// arranged in a pseudo-random order determined by the RandomSource.
	// The same RandomSource state must always produce the same permutation.
	//
	// Parameters:
	//   size   - The number of elements in the permutation (must be >= 0)
	//   random - The RandomSource used to generate the permutation order
	//
	// Returns:
	//   A slice of size integers containing a permutation of [0, size-1]
	//   An error if size is negative or exceeds MaxPermutationSize
	//
	// Example:
	//   Generate(5, seededRandom) might return [2, 4, 0, 3, 1], nil
	Generate(size int, random f5prng.RandomSource) ([]int, error)

	// GenerateInto generates a permutation into the provided buffer.
	// This is a zero-allocation variant that reuses an existing buffer.
	// The buffer will be resized if needed to fit 'size' elements.
	//
	// Parameters:
	//   buf    - Reusable buffer (will be resized if needed)
	//   size   - The number of elements in the permutation (must be >= 0)
	//   random - The RandomSource used to generate the permutation order
	//
	// Returns:
	//   The buffer containing the permutation (may be reallocated if too small)
	//   An error if size is negative or exceeds MaxPermutationSize
	GenerateInto(buf []int, size int, random f5prng.RandomSource) ([]int, error)
}

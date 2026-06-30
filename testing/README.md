# Fisher-Yates Test Utilities

This package provides reusable test utilities, mocks, and helpers for testing Fisher-Yates implementations.

## Overview

The testing package includes:
- **MockRandomSource**: Deterministic random source for reproducible tests
- **Test Helpers**: Functions for validating permutations and test assertions
- **Test Vectors**: Cross-platform verification support

## Components

### MockRandomSource

A deterministic implementation of the `RandomSource` interface that cycles through predefined int32 values.

#### Features

- Predictable output for reproducible tests
- Configurable value sequences
- Reset and update functionality
- Full `RandomSource` interface compliance

#### Usage

```go
import testing "github.com/0verkilll/fisheryates/testing"

// Create mock with specific values
mock := testing.NewMockRandomSource(5, -10, 3, -1, 0)

// Use with Fisher-Yates
fy := fisheryates.NewFisherYates()
perm := fy.Generate(5, mock)

// Reset for reuse
mock.Reset()

// Change values
mock.SetValues(10, 20, 30)
```

#### Example: Testing Negative Int32 Handling

```go
func TestNegativeHandling(t *testing.T) {
    mock := testing.NewMockRandomSource(
        -2147483648, // math.MinInt32
        -1,
        0,
        1,
        2147483647, // math.MaxInt32
    )

    fy := fisheryates.NewFisherYates()
    perm := fy.Generate(10, mock)

    testing.AssertValidPermutation(t, perm)
}
```

### Test Helpers

#### AssertValidPermutation

Verifies that a slice is a valid permutation of [0, n-1].

```go
perm := fy.Generate(10, random)
testing.AssertValidPermutation(t, perm)
```

Checks:
- All values in range [0, n-1]
- No duplicates
- All expected values present

#### AssertPermutationsEqual

Verifies determinism by comparing two permutations.

```go
random.Seed([]byte("test"))
perm1 := fy.Generate(10, random)

random.Seed([]byte("test")) // Same seed
perm2 := fy.Generate(10, random)

testing.AssertPermutationsEqual(t, perm1, perm2)
```

#### AssertPermutationsDifferent

Verifies that different seeds produce different results.

```go
random1.Seed([]byte("seed1"))
perm1 := fy.Generate(10, random1)

random2.Seed([]byte("seed2"))
perm2 := fy.Generate(10, random2)

testing.AssertPermutationsDifferent(t, perm1, perm2)
```

#### AssertBufferCapacityPreserved

Verifies zero-allocation pattern with `GenerateInto()`.

```go
buf := make([]int, 100)
initialCap := cap(buf)

buf = fy.GenerateInto(buf, 50, random)

testing.AssertBufferCapacityPreserved(t, buf, initialCap)
```

#### AssertNoAllocations

Verifies that a function performs no heap allocations.

```go
buf := make([]int, 100)
testing.AssertNoAllocations(t, 100, func() {
    buf = fy.GenerateInto(buf, 100, random)
}, 0) // Expect 0 allocations
```

### Test Vectors

Test vectors enable cross-platform verification between Go, Java, TypeScript, and Rust implementations.

#### GenerateTestVector

Creates a test vector for cross-platform comparison.

```go
vec := testing.GenerateTestVector(
    []byte("test"),
    10,
    []int{5, 3, 1, 4, 2, 8, 6, 9, 0, 7},
)

t.Logf("Test Vector:")
t.Logf("  Seed: %q", vec.Seed)
t.Logf("  Size: %d", vec.Size)
t.Logf("  Output: %v", vec.Output)
```

#### Cross-Platform Example

**Go:**
```go
hasher := sha1.NewSHA1(sha1.NewBigEndian())
random := securerandom.NewSecureRandom(hasher)
random.Seed([]byte("23"))

fy := fisheryates.NewFisherYates()
perm := fy.Generate(10, random)
// Expected: [5 3 1 4 2 8 6 9 0 7]
```

**Java Equivalent:**
```java
SecureRandom random = new SecureRandom();
random.setSeed("23".getBytes());
int[] perm = FisherYates.generate(10, random);
// Expected: same output as Go
```

### Utility Functions

#### CountInversions

Counts inversions in a permutation (useful for randomness analysis).

```go
inversions := testing.CountInversions(perm)
t.Logf("Permutation has %d inversions", inversions)

// Theoretical average for random permutation of size n:
// expected ≈ n*(n-1)/4
```

#### IsIdentityPermutation

Checks if a permutation is the identity [0, 1, 2, ..., n-1].

```go
if testing.IsIdentityPermutation(perm) {
    t.Error("Expected shuffled permutation, got identity")
}
```

## Complete Example

```go
package fisheryates

import (
    "testing"

    "github.com/0verkilll/fisheryates"
    testing "github.com/0verkilll/fisheryates/testing"
)

func TestWithUtilities(t *testing.T) {
    // Create mock random source
    mock := testing.NewMockRandomSource(3, 1, 4, 1, 5, 9, 2, 6)

    // Generate permutation
    fy := fisheryates.NewFisherYates()
    perm := fy.Generate(8, mock)

    // Validate using helpers
    testing.AssertValidPermutation(t, perm)

    // Test determinism
    mock.Reset() // Reset to beginning
    perm2 := fy.Generate(8, mock)
    testing.AssertPermutationsEqual(t, perm, perm2)

    // Test buffer reuse
    buf := make([]int, 10)
    initialCap := cap(buf)
    buf = fy.GenerateInto(buf, 8, mock)
    testing.AssertBufferCapacityPreserved(t, buf, initialCap)
}
```

## Benefits

### 1. **Deterministic Testing**
MockRandomSource eliminates PRNG non-determinism, making tests reproducible and debuggable.

### 2. **Reduced Boilerplate**
Helper functions eliminate repetitive validation code in tests.

### 3. **Better Error Messages**
Helpers use `t.Helper()` to report errors at the correct call site.

### 4. **Cross-Platform Verification**
Test vectors enable validation across Go, Java, TypeScript, and Rust.

### 5. **Performance Validation**
Allocation helpers verify zero-allocation patterns work correctly.

## Package Design

The testing package follows SOLID principles:

- **Single Responsibility**: Each helper has one clear purpose
- **Open/Closed**: Extensible through composition
- **Liskov Substitution**: MockRandomSource satisfies RandomSource interface
- **Interface Segregation**: Minimal, focused interfaces
- **Dependency Inversion**: Depends on fisheryates interfaces, not implementations

## See Also

- [Main Fisher-Yates Package](../README.md)
- [Examples](../examples/)

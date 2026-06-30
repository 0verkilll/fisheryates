# Multiple Permutations Example

Demonstrates the deterministic nature of Fisher-Yates by generating multiple permutations with different seeds.

## What This Demonstrates

- Different seeds produce different permutations
- Same seed always produces the same permutation
- Creating fresh PRNG instances per permutation

## Running

```bash
go run main.go
```

## Expected Output

```
Demonstrating Fisher-Yates Determinism
======================================

1. Password: "password1"
   Permutation: [8 4 7 2 0 5 3 9 1 6]

2. Password: "mySecret"
   Permutation: [3 9 1 6 4 0 8 2 5 7]

3. Password: "test123"
   Permutation: [1 5 3 9 2 6 0 4 8 7]

4. Password: "password1"
   Permutation: [8 4 7 2 0 5 3 9 1 6]
   ^ Same password = same permutation (deterministic)

Key Insight:
The same seed always produces the same permutation.
This enables F5 steganography decoding.
```

## Key Concepts

**Determinism Proof:** Entry #1 and #4 use the same password and produce identical permutations.

**Fresh PRNG:** Each permutation requires a fresh PRNG instance seeded independently. Reusing a seeded PRNG would continue from its current state, not reset.

```go
for _, password := range passwords {
    // Create fresh PRNG for each permutation
    hasher := sha1.NewSHA1(sha1.NewBigEndian())
    random := securerandom.NewSecureRandom(hasher)
    random.Seed([]byte(password))

    perm, err := fy.Generate(10, random)
}
```

## Why This Matters

F5 steganography relies on this property:
- **Encoder** creates a permutation from password/key
- **Decoder** recreates the exact same permutation
- Without identical permutations, hidden data cannot be recovered

# Basic Example

Simple permutation generation using a password seed.

## What This Demonstrates

- Creating the Fisher-Yates permutator
- Setting up the SHA1-based PRNG chain
- Generating a deterministic permutation

## Running

```bash
go run main.go
```

## Expected Output

```
Password: secret123
Permutation of [0..9]: [5 3 1 4 2 8 6 9 0 7]

This permutation is deterministic:
Same password + same size = same permutation
```

## Code Walkthrough

```go
// 1. Create dependencies: SHA-1 hasher for PRNG
hasher := sha1.NewSHA1(sha1.NewBigEndian())
random := securerandom.NewSecureRandom(hasher)
fy := fisheryates.NewFisherYates()

// 2. Seed with a password (like F5 steganography)
password := "secret123"
random.Seed([]byte(password))

// 3. Generate a permutation of 10 elements
perm, err := fy.Generate(10, random)
```

## Key Concepts

**Determinism:** The same password always produces the same permutation. This is essential for F5 steganography where both encoder and decoder must generate identical permutations.

**Component Chain:** FisherYates -> SecureRandom -> SHA1 -> BigEndian. Each component has a single responsibility.

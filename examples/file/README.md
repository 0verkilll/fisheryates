# File-Seeded Example

Generate permutations using file content as the seed, demonstrating the F5 steganography pattern.

## What This Demonstrates

- Using arbitrary binary data as seed
- F5 steganography use case
- Command-line argument handling

## Running

```bash
# Using a text file
go run main.go /path/to/file.txt

# Using an image (F5 pattern)
go run main.go /path/to/image.jpg
```

## Expected Output

```
File: /path/to/file.txt
File size: 1234 bytes
Permutation size: 50 elements

First 20 elements: [23 7 41 12 ...]

This demonstrates F5 steganography permutation:
- File content seeds the PRNG
- Same file = same permutation (deterministic)
- Used to shuffle DCT coefficients for message embedding
```

## F5 Steganography Context

In F5 steganography:

1. **Embedding:** The cover image seeds the PRNG. A permutation shuffles DCT coefficients so message bits are spread unpredictably across the image.

2. **Extraction:** Using the same image content as seed regenerates the identical permutation, allowing recovery of the hidden message.

3. **Security:** Without knowing the original image (or password), an attacker cannot determine where bits are hidden.

## Code Walkthrough

```go
// Read file content as seed
data, err := os.ReadFile(filename)

// Create PRNG chain
hasher := sha1.NewSHA1(sha1.NewBigEndian())
random := securerandom.NewSecureRandom(hasher)

// Seed with file content
random.Seed(data)

// Generate permutation
// Size would typically be the DCT coefficient array length
perm, err := fy.Generate(size, random)
```

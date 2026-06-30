package main

import (
	"fmt"
	"os"

	"github.com/0verkilll/fisheryates"
	"github.com/0verkilll/securerandom"
	"github.com/0verkilll/sha1"
)

func main() {
	if len(os.Args) < 2 {
		_, _ = fmt.Fprintf(os.Stderr, "Usage: %s <file>\n", os.Args[0])
		_, _ = fmt.Fprintf(os.Stderr, "\nGenerates a Fisher-Yates permutation using file content as seed.\n")
		_, _ = fmt.Fprintf(os.Stderr, "This demonstrates F5 steganography use case.\n")
		os.Exit(1)
	}

	filename := os.Args[1]

	// Read the file content as seed
	data, err := os.ReadFile(filename)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Create dependencies
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	fy := fisheryates.NewFisherYates()

	// Seed PRNG with file content
	random.Seed(data)

	// Generate permutation
	// In F5 steganography, this would be the size of DCT coefficient array
	size := 50
	perm, err := fy.Generate(size, random)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error generating permutation: %v\n", err)
		os.Exit(1)
	}

	// Display results
	fmt.Printf("File: %s\n", filename)
	fmt.Printf("File size: %d bytes\n", len(data))
	fmt.Printf("Permutation size: %d elements\n", size)
	fmt.Println()
	fmt.Printf("First 20 elements: %v\n", perm[:20])
	fmt.Println()
	fmt.Println("This demonstrates F5 steganography permutation:")
	fmt.Println("- File content seeds the PRNG")
	fmt.Println("- Same file = same permutation (deterministic)")
	fmt.Println("- Used to shuffle DCT coefficients for message embedding")
}

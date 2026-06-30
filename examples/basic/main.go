package main

import (
	"fmt"

	"github.com/0verkilll/fisheryates"
	"github.com/0verkilll/securerandom"
	"github.com/0verkilll/sha1"
)

func main() {
	// Create dependencies: SHA-1 hasher for PRNG
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	fy := fisheryates.NewFisherYates()

	// Seed with a password (like F5 steganography)
	password := "secret123"
	random.Seed([]byte(password))

	// Generate a permutation of 10 elements
	size := 10
	perm, err := fy.Generate(size, random)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Display the result
	fmt.Printf("Password: %s\n", password)
	fmt.Printf("Permutation of [0..%d]: %v\n", size-1, perm)
	fmt.Println()
	fmt.Println("This permutation is deterministic:")
	fmt.Println("Same password + same size = same permutation")
}

package main

import (
	"fmt"

	"github.com/0verkilll/fisheryates"
	"github.com/0verkilll/securerandom"
	"github.com/0verkilll/sha1"
)

func main() {
	// Create Fisher-Yates permutator
	fy := fisheryates.NewFisherYates()

	// Test passwords
	passwords := []string{
		"password1",
		"mySecret",
		"test123",
		"password1", // Duplicate to show determinism
	}

	fmt.Println("Demonstrating Fisher-Yates Determinism")
	fmt.Println("======================================")
	fmt.Println()

	for i, password := range passwords {
		// Create fresh PRNG for each permutation
		hasher := sha1.NewSHA1(sha1.NewBigEndian())
		random := securerandom.NewSecureRandom(hasher)
		random.Seed([]byte(password))

		// Generate permutation
		perm, err := fy.Generate(10, random)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Printf("%d. Password: %q\n", i+1, password)
		fmt.Printf("   Permutation: %v\n", perm)

		// Highlight duplicate
		if i == 3 {
			fmt.Println("   ↑ Same password = same permutation (deterministic)")
		}
		fmt.Println()
	}

	fmt.Println("Key Insight:")
	fmt.Println("The same seed always produces the same permutation.")
	fmt.Println("This enables F5 steganography decoding.")
}

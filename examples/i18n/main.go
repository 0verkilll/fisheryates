package main

import (
	"fmt"
	"os"

	"github.com/0verkilll/fisheryates"
	"github.com/0verkilll/securerandom"
	"github.com/0verkilll/sha1"
)

func main() {
	// Show some example locales (130+ supported)
	fmt.Println("Example locales (130+ supported):")
	fmt.Println("en-US, es-ES, fr-FR, de-DE, ja-JP, zh-CN, ar-SA, hi-IN, pt-BR, ru-RU, ...")
	fmt.Println()

	// Test with a few locales
	locales := []string{"en-US", "es-ES", "fr-FR"}

	for _, locale := range locales {
		fmt.Printf("=== Locale: %s ===\n", locale)

		// Create and set translator for this locale
		translator, err := fisheryates.NewTranslator(locale)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error creating translator: %v\n", err)
			continue
		}
		fisheryates.SetTranslator(translator)

		// Trigger an error to see localized message
		hasher := sha1.NewSHA1(sha1.NewBigEndian())
		random := securerandom.NewSecureRandom(hasher)
		random.Seed([]byte("test"))

		fy := fisheryates.NewFisherYates()

		// Try negative size to trigger error
		_, err = fy.Generate(-1, random)
		if err != nil {
			fmt.Printf("Negative size: %v\n", err)
		}

		// Try size exceeds max
		_, err = fy.Generate(fisheryates.MaxPermutationSize+1, random)
		if err != nil {
			fmt.Printf("Exceeds max:   %v\n", err)
		}

		fmt.Println()
	}

	// Auto-detect locale (uses system locale)
	fmt.Println("=== Auto-detect locale ===")
	translator, err := fisheryates.NewTranslator("")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	fisheryates.SetTranslator(translator)
	fmt.Println("Translator configured with system locale")
}

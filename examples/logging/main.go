// Example: enabling debug logging for Fisher-Yates
//
// This example demonstrates how to inject a logger to see
// debug output from the fisheryates package.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/0verkilll/fisheryates"
	"github.com/0verkilll/logger"
	"github.com/0verkilll/securerandom"
	"github.com/0verkilll/sha1"
)

func main() {
	fmt.Println("=== Fisher-Yates Logging Example ===")
	fmt.Println()

	// Create dependencies
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	random := securerandom.NewSecureRandom(hasher)
	fy := fisheryates.NewFisherYates()

	// Example 1: Default behavior (silent)
	fmt.Println("1. Default behavior (silent - no logger set):")
	random.Seed([]byte("test"))
	perm, _ := fy.Generate(5, random)
	fmt.Printf("   Generated permutation: %v\n", perm)
	fmt.Println("   (No debug output - this is the default)")
	fmt.Println()

	// Example 2: Enable debug logging
	fmt.Println("2. With debug logging enabled:")
	fisheryates.SetLogger(&SimpleLogger{level: logger.LevelDebug})

	random.Seed([]byte("test"))
	perm, _ = fy.Generate(5, random)
	fmt.Printf("   Result: %v\n", perm)
	fmt.Println()

	// Example 3: Use a logger with fields
	fmt.Println("3. With structured fields:")
	fieldLogger := &SimpleLogger{
		level:  logger.LevelDebug,
		fields: map[string]any{"component": "steganography", "version": "1.0"},
	}
	fisheryates.SetLogger(fieldLogger)

	random.Seed([]byte("secret"))
	perm, _ = fy.Generate(10, random)
	fmt.Printf("   Result: %v\n", perm)
	fmt.Println()

	// Example 4: Filter to only show warnings and above
	fmt.Println("4. With level filtering (Warn and above only):")
	fisheryates.SetLogger(&SimpleLogger{level: logger.LevelWarn})

	random.Seed([]byte("test"))
	perm, _ = fy.Generate(5, random) // Debug logs won't show
	fmt.Printf("   Result: %v\n", perm)
	fmt.Println("   (Debug messages filtered out)")
	fmt.Println()

	// Example 5: Disable logging again
	fmt.Println("5. Disable logging:")
	fisheryates.SetLogger(nil)

	random.Seed([]byte("test"))
	perm, _ = fy.Generate(5, random)
	fmt.Printf("   Result: %v\n", perm)
	fmt.Println("   (Back to silent mode)")
}

// SimpleLogger is a basic logger implementation for demonstration.
// In production, you would use zerolog, zap, or another logging library.
type SimpleLogger struct {
	level  logger.Level
	fields map[string]any
	ctx    context.Context
}

func (s *SimpleLogger) Debug(msg string, args ...any) { s.log(logger.LevelDebug, msg, args) }
func (s *SimpleLogger) Info(msg string, args ...any)  { s.log(logger.LevelInfo, msg, args) }
func (s *SimpleLogger) Warn(msg string, args ...any)  { s.log(logger.LevelWarn, msg, args) }
func (s *SimpleLogger) Error(msg string, args ...any) { s.log(logger.LevelError, msg, args) }
func (s *SimpleLogger) Fatal(msg string, args ...any) {
	s.log(logger.LevelFatal, msg, args)
	os.Exit(1)
}

func (s *SimpleLogger) log(level logger.Level, msg string, args []any) {
	if level < s.level {
		return
	}

	timestamp := time.Now().Format("15:04:05.000")
	fmt.Printf("   [%s] %s: %s", timestamp, level, msg)

	// Print inline args
	for i := 0; i < len(args)-1; i += 2 {
		fmt.Printf(" %v=%v", args[i], args[i+1])
	}

	// Print stored fields
	for k, v := range s.fields {
		fmt.Printf(" %s=%v", k, v)
	}

	fmt.Println()
}

func (s *SimpleLogger) WithFields(fields ...any) logger.Logger {
	newFields := make(map[string]any)
	for k, v := range s.fields {
		newFields[k] = v
	}
	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(string); ok {
			newFields[key] = fields[i+1]
		}
	}
	return &SimpleLogger{level: s.level, fields: newFields, ctx: s.ctx}
}

func (s *SimpleLogger) WithContext(ctx context.Context) logger.Logger {
	return &SimpleLogger{level: s.level, fields: s.fields, ctx: ctx}
}

func (s *SimpleLogger) WithLevel(level logger.Level) logger.Logger {
	return &SimpleLogger{level: level, fields: s.fields, ctx: s.ctx}
}

func (s *SimpleLogger) Enabled(level logger.Level) bool {
	return level >= s.level
}

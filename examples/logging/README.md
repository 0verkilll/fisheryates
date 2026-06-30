# Logging Example

Demonstrates how to enable debug logging for the Fisher-Yates package.

## Overview

The fisheryates package supports optional debug logging via a pluggable logger interface. By default, logging is completely silent (no output). You can inject your own logger implementation to see debug messages.

## Run

```bash
go run main.go
```

## Expected Output

```
=== Fisher-Yates Logging Example ===

1. Default behavior (silent - no logger set):
   Generated permutation: [3 4 0 1 2]
   (No debug output - this is the default)

2. With debug logging enabled:
   [15:04:05.123] DEBUG: Generate called size=5
   [15:04:05.123] DEBUG: Generate completed size=5
   Result: [3 4 0 1 2]

3. With structured fields:
   [15:04:05.124] DEBUG: Generate called size=10 component=steganography version=1.0
   [15:04:05.124] DEBUG: Generate completed size=10 component=steganography version=1.0
   Result: [5 3 1 4 2 8 6 9 0 7]

4. With level filtering (Warn and above only):
   Result: [3 4 0 1 2]
   (Debug messages filtered out)

5. Disable logging:
   Result: [3 4 0 1 2]
   (Back to silent mode)
```

## Key Concepts

### Default Silent Behavior

```go
// By default, no output
perm, _ := fy.Generate(10, random)
// No debug messages printed
```

### Enable Debug Logging

```go
import "github.com/0verkilll/fisheryates"

// Set your logger implementation
fisheryates.SetLogger(myLogger)

// Now Generate() will emit debug messages
perm, _ := fy.Generate(10, random)
// Debug: Generate called size=10
// Debug: Generate completed size=10
```

### Disable Logging

```go
// Set nil to disable
fisheryates.SetLogger(nil)
```

## Using with Popular Loggers

### Zerolog

```go
import (
    "github.com/rs/zerolog"
    "github.com/0verkilll/fisheryates"
)

zlog := zerolog.New(os.Stdout).With().Timestamp().Logger()
adapter := NewZerologAdapter(zlog)  // See logger package examples
fisheryates.SetLogger(adapter)
```

### Standard Library

```go
import (
    "log"
    "github.com/0verkilll/fisheryates"
)

stdlog := log.New(os.Stdout, "", log.LstdFlags)
adapter := NewStdLogAdapter(stdlog)  // See logger package examples
fisheryates.SetLogger(adapter)
```

## Logger Interface

To create a custom logger, implement the `logger.Logger` interface:

```go
type Logger interface {
    Debug(msg string, args ...any)
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
    Fatal(msg string, args ...any)
    WithFields(fields ...any) Logger
    WithContext(ctx context.Context) Logger
    WithLevel(level Level) Logger
    Enabled(level Level) bool
}
```

See [github.com/0verkilll/logger](https://github.com/0verkilll/logger) for adapter examples and documentation.

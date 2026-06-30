# Examples

Runnable examples demonstrating Fisher-Yates permutation generation.

## Quick Run

```bash
# Basic permutation
cd examples/basic && go run main.go

# Multiple permutations (determinism demo)
cd examples/multiple && go run main.go

# File-seeded permutation
cd examples/file && go run main.go <filename>

# Internationalization
cd examples/i18n && go run main.go

# Concurrent usage patterns
cd examples/concurrent && go run main.go

# Debug logging
cd examples/logging && go run main.go
```

## Available Examples

| Example | Description | README |
|---------|-------------|--------|
| [basic/](basic/) | Simple permutation with password seed | [README](basic/README.md) |
| [multiple/](multiple/) | Demonstrates deterministic behavior | [README](multiple/README.md) |
| [file/](file/) | File content as seed (F5 pattern) | [README](file/README.md) |
| [i18n/](i18n/) | Localized error messages | [README](i18n/README.md) |
| [concurrent/](concurrent/) | Thread-safe usage patterns | [README](concurrent/README.md) |
| [logging/](logging/) | Debug logging with custom logger | [README](logging/README.md) |

## IDE Support

Run configurations are included for:
- **IntelliJ IDEA / GoLand:** `.run/` directory
- **VS Code:** `.vscode/launch.json`

Click the play button in your IDE to run any example.

## Common Pattern

All examples follow the same component chain:

```go
// 1. Create the dependency chain
hasher := sha1.NewSHA1(sha1.NewBigEndian())
random := securerandom.NewSecureRandom(hasher)
fy := fisheryates.NewFisherYates()

// 2. Seed the PRNG
random.Seed([]byte("your-seed"))

// 3. Generate permutation
perm, err := fy.Generate(size, random)
```

Each example's README contains detailed explanations and expected output.

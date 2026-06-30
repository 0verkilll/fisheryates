# Internationalization (i18n) Example

Demonstrates localized error messages in multiple languages.

## Supported Locales

The i18n system supports **130+ languages** via the [i18n package](https://github.com/0verkilll/i18n).

Common locales include: `en-US`, `es-ES`, `fr-FR`, `de-DE`, `ja-JP`, `zh-CN`, `ar-SA`, `hi-IN`, `pt-BR`, `ru-RU`, and many more.

## Running

```bash
go run main.go
```

## Expected Output

```
Example locales (130+ supported):
en-US, es-ES, fr-FR, de-DE, ja-JP, zh-CN, ar-SA, hi-IN, pt-BR, ru-RU, ...

=== Locale: en-US ===
Negative size: size must be non-negative: fisheryates: size must be non-negative
Exceeds max:   size exceeds MaxPermutationSize (100M elements, ~800MB): fisheryates: ...

=== Locale: es-ES ===
Negative size: el tamaño debe ser no negativo: fisheryates: size must be non-negative
Exceeds max:   el tamaño excede MaxPermutationSize (100M elementos, ~800MB): fisheryates: ...

=== Locale: fr-FR ===
Negative size: la taille doit être non négative: fisheryates: size must be non-negative
Exceeds max:   la taille dépasse MaxPermutationSize (100M éléments, ~800MB): fisheryates: ...

=== Auto-detect locale ===
Translator configured with system locale
```

## Usage Patterns

### Explicit Locale

```go
translator, _ := fisheryates.NewTranslator("es-ES")
fisheryates.SetTranslator(translator)
```

### Auto-Detect System Locale

```go
translator, _ := fisheryates.NewTranslator("")  // Empty string = auto-detect
fisheryates.SetTranslator(translator)
```

### Query Supported Locales

```go
locales := fisheryates.GetSupportedLocales()
// Returns: ["en-US", "es-ES", "fr-FR"]
```

## Shared Translator (Multiple Packages)

When using multiple 0verkilll packages, share a single translator:

```go
import (
    "github.com/0verkilll/fisheryates"
    "github.com/0verkilll/sha1"
    "github.com/0verkilll/securerandom"
    "github.com/0verkilll/i18n"
)

func main() {
    // Create shared translator
    translator, _ := i18n.New(
        i18n.WithFileSystemLoader("locales"),
        i18n.WithDefaultLocale("es-ES"),
    )

    // Configure all packages
    fisheryates.SetTranslator(translator)
    sha1.SetTranslator(translator)
    securerandom.SetTranslator(translator)
}
```

## Zero Dependency

The i18n feature adds no external dependencies. Translation files are embedded directly in the binary.

package fisheryates

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/0verkilll/i18n"
)

//go:embed locales/*.json
var localeFS embed.FS

// TranslatorProvider defines the interface for translation providers.
// This allows loose coupling - the package works with or without i18n.
// The interface matches github.com/0verkilll/i18n.Translator.
type TranslatorProvider interface {
	// Translate returns the translated string for the given key.
	Translate(key string) string

	// TranslateWithArgs returns the translated string with format arguments.
	TranslateWithArgs(key string, args ...interface{}) string

	// HasKey returns true if the translation key exists.
	HasKey(key string) bool

	// SetLocale changes the current locale.
	SetLocale(locale string)

	// GetLocale returns the current locale.
	GetLocale() string
}

// translatorBox wraps TranslatorProvider so atomic.Value sees a consistent
// concrete type regardless of which implementation is stored. Standard Go
// idiom for atomic swap of an interface value.
type translatorBox struct{ t TranslatorProvider }

// globalTranslator holds the optional translator instance inside a
// translatorBox. It is swapped atomically by SetTranslator and read
// lock-free by GetTranslator.
var globalTranslator atomic.Value // holds translatorBox

// loadTranslator returns the currently stored TranslatorProvider, or nil if none.
func loadTranslator() TranslatorProvider {
	if v := globalTranslator.Load(); v != nil {
		if b, ok := v.(translatorBox); ok {
			return b.t
		}
	}
	return nil
}

// NewTranslator creates a new translator configured for the fisheryates package.
// If locale is empty, it will auto-detect from environment variables.
// The translator uses embedded locale files and the i18n package for translation.
func NewTranslator(locale string) (TranslatorProvider, error) {
	opts := []i18n.Option{
		i18n.WithLoader(i18n.NewEmbedFSLoader(localeFS, "locales")),
	}
	if locale != "" {
		opts = append(opts, i18n.WithDefaultLocale(locale))
	}
	return i18n.New(opts...)
}

// SetTranslator sets the global translator for this package.
// Pass nil to disable translations and use default English messages.
// The translator is shared across all goroutines and is thread-safe.
func SetTranslator(translator TranslatorProvider) {
	globalTranslator.Store(translatorBox{t: translator})
}

// GetTranslator returns the global translator, or nil if not set.
func GetTranslator() TranslatorProvider {
	return loadTranslator()
}

// GetSupportedLocales returns the list of locales supported by this package.
//
// It reads the list of embedded locale files and returns their locale codes.
// The returned array is sorted alphabetically for consistency.
//
// If the embedded filesystem cannot be read (which should never happen in
// normal operation), this function returns a fallback array containing only
// "en-US" to ensure graceful degradation.
func GetSupportedLocales() []string {
	return getSupportedLocalesFromFS(localeFS, "locales")
}

// getSupportedLocalesFromFS extracts locale codes from a filesystem directory.
// This internal function enables testing of edge cases with mock filesystems.
func getSupportedLocalesFromFS(fsys fs.FS, dir string) []string {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		// Fallback to English if FS cannot be read
		return []string{"en-US"}
	}

	locales := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}

		// Extract locale code by removing .json extension
		locale := strings.TrimSuffix(name, ".json")
		locales = append(locales, locale)
	}

	// Sort for consistent ordering
	sort.Strings(locales)

	return locales
}

// translate returns the translated message or the default if no translator is set.
// The key is automatically prefixed with "fisheryates." for namespacing.
func translate(key, defaultValue string) string {
	t := loadTranslator()
	if t == nil {
		return defaultValue
	}

	fullKey := "fisheryates." + key
	if t.HasKey(fullKey) {
		return t.Translate(fullKey)
	}
	return defaultValue
}

// translateWithArgs returns the translated message with format arguments.
// The key is automatically prefixed with "fisheryates." for namespacing.
func translateWithArgs(key, defaultValue string, args ...interface{}) string {
	t := loadTranslator()
	if t == nil {
		return fmt.Sprintf(defaultValue, args...)
	}

	fullKey := "fisheryates." + key
	if t.HasKey(fullKey) {
		return t.TranslateWithArgs(fullKey, args...)
	}
	return fmt.Sprintf(defaultValue, args...)
}

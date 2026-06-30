package fisheryates

import (
	"sync/atomic"

	"github.com/0verkilll/logger"
)

// loggerBox wraps logger.Logger so atomic.Value sees a consistent concrete
// type regardless of which implementation is stored. Standard Go idiom for
// atomic swap of an interface value.
type loggerBox struct{ l logger.Logger }

// pkgLogger holds the package-level Logger inside a loggerBox. It is swapped
// atomically by SetLogger and read lock-free by GetLogger.
var pkgLogger atomic.Value // holds loggerBox

func init() {
	pkgLogger.Store(loggerBox{l: logger.NopLogger{}})
}

// SetLogger sets the logger for the fisheryates package.
// Pass nil to disable logging and reset to the default NopLogger.
// The logger is shared across all goroutines and is thread-safe.
func SetLogger(l logger.Logger) {
	if l == nil {
		pkgLogger.Store(loggerBox{l: logger.NopLogger{}})
		return
	}
	pkgLogger.Store(loggerBox{l: l})
}

// GetLogger returns the package logger, or NopLogger if not set.
func GetLogger() logger.Logger {
	if v := pkgLogger.Load(); v != nil {
		if b, ok := v.(loggerBox); ok && b.l != nil {
			return b.l
		}
	}
	return logger.NopLogger{}
}

// log returns the package logger for internal use.
func log() logger.Logger {
	return GetLogger()
}

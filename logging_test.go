package fisheryates

import (
	"testing"

	"github.com/0verkilll/logger"
	logtesting "github.com/0verkilll/logger/testing"
)

func TestSetLogger(t *testing.T) {
	// Save original and restore after test
	original := GetLogger()
	defer SetLogger(original)

	mock := logtesting.NewMockLogger()
	SetLogger(mock)

	got := GetLogger()
	if got != mock {
		t.Error("GetLogger did not return the logger set by SetLogger")
	}
}

func TestSetLoggerNil(t *testing.T) {
	// Save original and restore after test
	original := GetLogger()
	defer SetLogger(original)

	// First set a non-nil logger
	mock := logtesting.NewMockLogger()
	SetLogger(mock)

	// Then set nil
	SetLogger(nil)

	got := GetLogger()
	if _, ok := got.(logger.NopLogger); !ok {
		t.Errorf("SetLogger(nil) should reset to NopLogger, got %T", got)
	}
}

func TestDefaultLoggerIsNop(t *testing.T) {
	// Create fresh package state by setting nil
	SetLogger(nil)

	got := GetLogger()
	if _, ok := got.(logger.NopLogger); !ok {
		t.Errorf("default logger should be NopLogger, got %T", got)
	}
}

func TestGenerateLogsDebugMessages(t *testing.T) {
	// Save original and restore after test
	original := GetLogger()
	defer SetLogger(original)

	mock := logtesting.NewMockLogger()
	SetLogger(mock)

	fy := NewFisherYates()
	mockRandom := newMockRandomSource(1, 2, 3, 4, 5)

	_, err := fy.Generate(5, mockRandom)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should have logged debug messages
	entries := mock.Entries()
	if len(entries) < 2 {
		t.Errorf("expected at least 2 log entries (start and end), got %d", len(entries))
	}

	// First entry should mention Generate called
	logtesting.AssertLogContains(t, mock, logger.LevelDebug, "Generate called")
	logtesting.AssertLogContains(t, mock, logger.LevelDebug, "Generate completed")
}

func TestGenerateIntoLogsDebugMessages(t *testing.T) {
	// Save original and restore after test
	original := GetLogger()
	defer SetLogger(original)

	mock := logtesting.NewMockLogger()
	SetLogger(mock)

	fy := NewFisherYates()
	mockRandom := newMockRandomSource(1, 2, 3, 4, 5)
	buf := make([]int, 0)

	_, err := fy.GenerateInto(buf, 5, mockRandom)
	if err != nil {
		t.Fatalf("GenerateInto failed: %v", err)
	}

	// Should have logged debug messages
	logtesting.AssertLogContains(t, mock, logger.LevelDebug, "GenerateInto called")
	logtesting.AssertLogContains(t, mock, logger.LevelDebug, "GenerateInto completed")
}

func TestNopLoggerDoesNotPanic(t *testing.T) {
	// Ensure default NopLogger doesn't cause issues
	SetLogger(nil)

	fy := NewFisherYates()
	mockRandom := newMockRandomSource(1, 2, 3, 4, 5)

	// Should not panic with NopLogger
	_, err := fy.Generate(5, mockRandom)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
}

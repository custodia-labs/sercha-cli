package logger

import (
	"bytes"
	"os"
	"testing"
)

func TestSetVerbose(t *testing.T) {
	// Reset state after test
	defer func() {
		SetVerbose(false)
		SetOutput(os.Stderr)
	}()

	// Initially not verbose
	SetVerbose(false)
	if IsVerbose() {
		t.Error("expected verbose to be false initially")
	}

	// Enable verbose
	SetVerbose(true)
	if !IsVerbose() {
		t.Error("expected verbose to be true after SetVerbose(true)")
	}

	// Disable verbose
	SetVerbose(false)
	if IsVerbose() {
		t.Error("expected verbose to be false after SetVerbose(false)")
	}
}

func TestDebug_WhenVerbose(t *testing.T) {
	defer func() {
		SetVerbose(false)
		SetOutput(os.Stderr)
	}()

	var buf bytes.Buffer
	SetOutput(&buf)
	SetVerbose(true)

	Debug("test message %s", "arg")

	output := buf.String()
	if output == "" {
		t.Error("expected output when verbose is enabled")
	}
	if output != "[DEBUG] test message arg\n" {
		t.Errorf("unexpected output: %q", output)
	}
}

func TestDebug_WhenNotVerbose(t *testing.T) {
	defer func() {
		SetVerbose(false)
		SetOutput(os.Stderr)
	}()

	var buf bytes.Buffer
	SetOutput(&buf)
	SetVerbose(false)

	Debug("test message")

	if buf.Len() > 0 {
		t.Error("expected no output when verbose is disabled")
	}
}

func TestSection(t *testing.T) {
	defer func() {
		SetVerbose(false)
		SetOutput(os.Stderr)
	}()

	var buf bytes.Buffer
	SetOutput(&buf)
	SetVerbose(true)

	Section("Test Section")

	output := buf.String()
	if output != "\n=== Test Section ===\n" {
		t.Errorf("unexpected section output: %q", output)
	}
}

func TestInfo(t *testing.T) {
	defer func() {
		SetVerbose(false)
		SetOutput(os.Stderr)
	}()

	var buf bytes.Buffer
	SetOutput(&buf)
	SetVerbose(true)

	Info("info message %d", 42)

	output := buf.String()
	if output != "[INFO] info message 42\n" {
		t.Errorf("unexpected info output: %q", output)
	}
}

func TestWarn(t *testing.T) {
	defer func() {
		SetVerbose(false)
		SetOutput(os.Stderr)
	}()

	var buf bytes.Buffer
	SetOutput(&buf)
	SetVerbose(true)

	Warn("warning message")

	output := buf.String()
	if output != "[WARN] warning message\n" {
		t.Errorf("unexpected warn output: %q", output)
	}
}

func TestConcurrentAccess(t *testing.T) {
	defer func() {
		SetVerbose(false)
		SetOutput(os.Stderr)
	}()

	var buf bytes.Buffer
	SetOutput(&buf)

	// Run concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			SetVerbose(true)
			Debug("concurrent %d", i)
			IsVerbose()
			SetVerbose(false)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	// Test passes if no race conditions
}

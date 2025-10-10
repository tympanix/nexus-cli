package util

import (
	"bytes"
	"testing"
)

// TestLogger tests that Logger writes to the provided writer
func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf)

	logger.Println("test message")
	expected := "test message\n"
	if buf.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, buf.String())
	}

	buf.Reset()
	logger.Printf("formatted %s %d\n", "message", 42)
	expected = "formatted message 42\n"
	if buf.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, buf.String())
	}
}

// TestVerboseLogger tests that verbose logger writes verbose messages
func TestVerboseLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewVerboseLogger(&buf)

	// Normal messages should always be written
	logger.Println("normal message")
	expected := "normal message\n"
	if buf.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, buf.String())
	}

	// Verbose messages should be written in verbose mode
	buf.Reset()
	logger.VerbosePrintf("verbose %s\n", "message")
	expected = "verbose message\n"
	if buf.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, buf.String())
	}

	buf.Reset()
	logger.VerbosePrintln("verbose println")
	expected = "verbose println\n"
	if buf.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, buf.String())
	}
}

// TestNonVerboseLogger tests that non-verbose logger suppresses verbose messages
func TestNonVerboseLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf)

	// Normal messages should always be written
	logger.Println("normal message")
	expected := "normal message\n"
	if buf.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, buf.String())
	}

	// Verbose messages should NOT be written in non-verbose mode
	buf.Reset()
	logger.VerbosePrintf("verbose %s\n", "message")
	if buf.String() != "" {
		t.Errorf("Expected no output, got '%s'", buf.String())
	}

	logger.VerbosePrintln("verbose println")
	if buf.String() != "" {
		t.Errorf("Expected no output, got '%s'", buf.String())
	}
}

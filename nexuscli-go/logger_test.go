package main

import (
	"bytes"
	"testing"
)

// TestStdLogger tests that StdLogger writes to the provided writer
func TestStdLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := &StdLogger{writer: &buf}
	
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

// TestNoopLogger tests that NoopLogger discards all output
func TestNoopLogger(t *testing.T) {
	logger := NewNoopLogger()
	
	// These calls should not panic and should do nothing
	logger.Println("test message")
	logger.Printf("formatted %s %d\n", "message", 42)
}

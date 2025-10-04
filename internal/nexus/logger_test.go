package nexus

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

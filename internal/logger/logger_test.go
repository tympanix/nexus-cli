package logger

import (
	"bytes"
	"testing"
)

// TestLogger tests that Logger writes to the provided writer
func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf)
	
	log.Println("test message")
	expected := "test message\n"
	if buf.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, buf.String())
	}
	
	buf.Reset()
	log.Printf("formatted %s %d\n", "message", 42)
	expected = "formatted message 42\n"
	if buf.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, buf.String())
	}
}

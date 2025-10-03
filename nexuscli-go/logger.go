package main

import (
	"fmt"
	"io"
	"os"
)

// Logger interface for output operations
type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

// StdLogger writes to the given writer (typically os.Stdout)
type StdLogger struct {
	writer io.Writer
}

// NewStdLogger creates a new logger that writes to stdout
func NewStdLogger() Logger {
	return &StdLogger{writer: os.Stdout}
}

func (l *StdLogger) Printf(format string, v ...interface{}) {
	fmt.Fprintf(l.writer, format, v...)
}

func (l *StdLogger) Println(v ...interface{}) {
	fmt.Fprintln(l.writer, v...)
}

// NoopLogger discards all output
type NoopLogger struct{}

// NewNoopLogger creates a new logger that discards all output
func NewNoopLogger() Logger {
	return &NoopLogger{}
}

func (l *NoopLogger) Printf(format string, v ...interface{}) {
	// Do nothing
}

func (l *NoopLogger) Println(v ...interface{}) {
	// Do nothing
}

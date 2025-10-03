package main

import (
	"fmt"
	"io"
)

// Logger interface for output operations
type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

// SimpleLogger writes to the given writer
type SimpleLogger struct {
	writer io.Writer
}

// NewLogger creates a new logger that writes to the given writer
func NewLogger(writer io.Writer) Logger {
	return &SimpleLogger{writer: writer}
}

func (l *SimpleLogger) Printf(format string, v ...interface{}) {
	fmt.Fprintf(l.writer, format, v...)
}

func (l *SimpleLogger) Println(v ...interface{}) {
	fmt.Fprintln(l.writer, v...)
}

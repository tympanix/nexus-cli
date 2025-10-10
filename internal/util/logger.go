package util

import (
	"fmt"
	"io"
)

// Logger interface for output operations
type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
	VerbosePrintf(format string, v ...interface{})
	VerbosePrintln(v ...interface{})
}

// SimpleLogger writes to the given writer
type SimpleLogger struct {
	writer  io.Writer
	verbose bool
}

// NewLogger creates a new logger that writes to the given writer
func NewLogger(writer io.Writer) Logger {
	return &SimpleLogger{writer: writer, verbose: false}
}

// NewVerboseLogger creates a new logger with verbose mode enabled
func NewVerboseLogger(writer io.Writer) Logger {
	return &SimpleLogger{writer: writer, verbose: true}
}

func (l *SimpleLogger) Printf(format string, v ...interface{}) {
	fmt.Fprintf(l.writer, format, v...)
}

func (l *SimpleLogger) Println(v ...interface{}) {
	fmt.Fprintln(l.writer, v...)
}

func (l *SimpleLogger) VerbosePrintf(format string, v ...interface{}) {
	if l.verbose {
		fmt.Fprintf(l.writer, format, v...)
	}
}

func (l *SimpleLogger) VerbosePrintln(v ...interface{}) {
	if l.verbose {
		fmt.Fprintln(l.writer, v...)
	}
}

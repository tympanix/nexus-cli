package progress

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

// ProgressBar wraps a progress bar to track whether progress should be shown
type ProgressBar struct {
	bar          *progressbar.ProgressBar
	showProgress bool
}

// Write implements io.Writer
func (p *ProgressBar) Write(b []byte) (int, error) {
	return p.bar.Write(b)
}

// Add64 adds to the progress bar
func (p *ProgressBar) Add64(n int64) error {
	return p.bar.Add64(n)
}

// Describe sets the description of the progress bar
func (p *ProgressBar) Describe(description string) {
	p.bar.Describe(description)
}

// Finish completes the progress bar and prints a newline if progress is shown
func (p *ProgressBar) Finish() error {
	err := p.bar.Finish()
	if p.showProgress {
		fmt.Println()
	}
	return err
}

// NewProgressBar creates a new progress bar with standard configuration
// The description parameter should describe the operation (e.g., "Uploading", "Downloading")
// The currentFile and totalFiles parameters track which file is being processed
// The showProgress parameter controls whether progress should be shown (typically util.IsATTY() && !quietMode)
func NewProgressBar(totalBytes int64, description string, currentFile, totalFiles int, showProgress bool) *ProgressBar {
	var writer io.Writer = ansi.NewAnsiStdout()
	if !showProgress {
		writer = io.Discard
	}

	descWithCount := fmt.Sprintf("[cyan][%d/%d][reset] %s", currentFile, totalFiles, description)

	bar := progressbar.NewOptions64(totalBytes,
		progressbar.OptionSetWriter(writer),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetDescription(descWithCount),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	return &ProgressBar{
		bar:          bar,
		showProgress: showProgress,
	}
}

// ProgressBarWithCount wraps a progress bar to track file count atomically
// Used for parallel download operations where multiple goroutines update progress
type ProgressBarWithCount struct {
	bar          *ProgressBar
	current      *int32
	total        int
	description  string
	mu           sync.Mutex // Protects bar.Describe() calls
	showProgress bool       // Whether progress is being shown (not quiet mode and is TTY)
}

func (p *ProgressBarWithCount) Write(b []byte) (int, error) {
	return p.bar.Write(b)
}

func (p *ProgressBarWithCount) Add64(n int64) error {
	return p.bar.Add64(n)
}

func (p *ProgressBarWithCount) IncrementFile() {
	newCount := atomic.AddInt32(p.current, 1)
	p.mu.Lock()
	p.bar.Describe(fmt.Sprintf("[cyan][%d/%d][reset] %s", newCount, p.total, p.description))
	p.mu.Unlock()
}

func (p *ProgressBarWithCount) Finish() error {
	return p.bar.Finish()
}

// NewProgressBarWithCount creates a new progress bar with file count tracking
// The showProgress parameter controls whether progress should be shown (typically util.IsATTY() && !quietMode)
func NewProgressBarWithCount(totalBytes int64, description string, total int, showProgress bool) *ProgressBarWithCount {
	var current int32
	baseBar := NewProgressBar(totalBytes, description, 0, total, showProgress)
	return &ProgressBarWithCount{
		bar:          baseBar,
		current:      &current,
		total:        total,
		description:  description,
		showProgress: showProgress,
	}
}

// CappingWriter wraps an io.Writer and caps the total bytes written to a maximum value
// This is useful for progress bars where the actual size might exceed the estimated max
// (e.g., compressed data with headers/footers)
type CappingWriter struct {
	writer       io.Writer
	maxBytes     int64
	bytesWritten int64
}

// NewCappingWriter creates a new capping writer
func NewCappingWriter(writer io.Writer, maxBytes int64) *CappingWriter {
	return &CappingWriter{
		writer:   writer,
		maxBytes: maxBytes,
	}
}

func (cw *CappingWriter) Write(p []byte) (int, error) {
	n := len(p)

	// Calculate how many bytes we should actually report to stay under max
	remaining := cw.maxBytes - cw.bytesWritten
	if remaining <= 0 {
		// Already at or over max, don't report any more progress
		return n, nil
	}

	// Only report up to the remaining bytes
	toReport := int64(n)
	if toReport > remaining {
		toReport = remaining
	}

	// Update the underlying writer with the capped amount
	if toReport > 0 {
		reportBytes := make([]byte, toReport)
		copy(reportBytes, p[:toReport])
		if _, err := cw.writer.Write(reportBytes); err != nil {
			return n, err
		}
	}

	cw.bytesWritten += toReport
	return n, nil
}

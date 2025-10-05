package nexus

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

// Config holds the configuration for connecting to Nexus
type Config struct {
	NexusURL string
	Username string
	Password string
}

// NewConfig creates a new Config with values from environment variables or defaults
func NewConfig() *Config {
	return &Config{
		NexusURL: getenv("NEXUS_URL", "http://localhost:8081"),
		Username: getenv("NEXUS_USER", "admin"),
		Password: getenv("NEXUS_PASS", "admin"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func isatty() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// newProgressBar creates a new progress bar with standard configuration
// The description parameter should describe the operation (e.g., "Uploading", "Downloading")
// The currentFile and totalFiles parameters track which file is being processed
// The quietMode parameter controls whether progress should be shown
func newProgressBar(totalBytes int64, description string, currentFile, totalFiles int, quietMode bool) *progressbar.ProgressBar {
	showProgress := isatty() && !quietMode
	var writer io.Writer = ansi.NewAnsiStdout()
	if !showProgress {
		writer = io.Discard
	}

	descWithCount := fmt.Sprintf("[cyan][%d/%d][reset] %s", currentFile, totalFiles, description)

	return progressbar.NewOptions64(totalBytes,
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
}

// progressBarWithCount wraps a progress bar to track file count atomically
// Used for parallel download operations where multiple goroutines update progress
type progressBarWithCount struct {
	bar         *progressbar.ProgressBar
	current     *int32
	total       int
	description string
	mu          sync.Mutex // Protects bar.Describe() calls
}

func (p *progressBarWithCount) Write(b []byte) (int, error) {
	return p.bar.Write(b)
}

func (p *progressBarWithCount) Add64(n int64) error {
	return p.bar.Add64(n)
}

func (p *progressBarWithCount) incrementFile() {
	newCount := atomic.AddInt32(p.current, 1)
	p.mu.Lock()
	p.bar.Describe(fmt.Sprintf("[cyan][%d/%d][reset] %s", newCount, p.total, p.description))
	p.mu.Unlock()
}

func (p *progressBarWithCount) Finish() error {
	return p.bar.Finish()
}

// cappingWriter wraps an io.Writer and caps the total bytes written to a maximum value
// This is useful for progress bars where the actual size might exceed the estimated max
// (e.g., compressed data with headers/footers)
type cappingWriter struct {
	writer       io.Writer
	maxBytes     int64
	bytesWritten int64
}

func newCappingWriter(writer io.Writer, maxBytes int64) *cappingWriter {
	return &cappingWriter{
		writer:   writer,
		maxBytes: maxBytes,
	}
}

func (cw *cappingWriter) Write(p []byte) (int, error) {
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

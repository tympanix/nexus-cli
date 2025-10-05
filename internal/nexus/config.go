package nexus

import (
	"fmt"
	"io"
	"os"

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

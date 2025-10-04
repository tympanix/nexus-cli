package nexus

import (
	"io"
	"os"

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
// The description parameter should describe the operation (e.g., "Uploading bytes", "Downloading bytes")
// The quietMode parameter controls whether progress should be shown
func newProgressBar(totalBytes int64, description string, quietMode bool) *progressbar.ProgressBar {
	showProgress := isatty() && !quietMode
	progressWriter := io.Writer(os.Stdout)
	if !showProgress {
		progressWriter = io.Discard
	}
	return progressbar.NewOptions64(totalBytes,
		progressbar.OptionSetWriter(progressWriter),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetDescription(description),
		progressbar.OptionFullWidth(),
	)
}

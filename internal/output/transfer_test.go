package output

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/tympanix/nexus-cli/internal/util"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero", 0, "0 B"},
		{"bytes", 512, "512 B"},
		{"1 KiB", 1024, "1.0 KiB"},
		{"1.5 KiB", 1536, "1.5 KiB"},
		{"1 MiB", 1048576, "1.0 MiB"},
		{"2.5 MiB", 2621440, "2.5 MiB"},
		{"1 GiB", 1073741824, "1.0 GiB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %s, want %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		contains string
	}{
		{"milliseconds", 500 * time.Millisecond, "ms"},
		{"seconds", 5 * time.Second, "s"},
		{"minutes", 2 * time.Minute, "m"},
		{"hours", 2 * time.Hour, "h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("formatDuration(%v) = %s, should contain %s", tt.duration, result, tt.contains)
			}
		})
	}
}

func TestTransferTracker(t *testing.T) {
	var buf bytes.Buffer
	logger := util.NewLogger(&buf)

	tracker := NewTransferTracker(TransferTypeUpload, "test-repo", logger, false, false, false)

	tracker.PrintHeader(3, 1024*1024)

	tracker.RecordFile(FileTransfer{
		Path:      "file1.txt",
		Size:      1024,
		Status:    TransferStatusSuccess,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(100 * time.Millisecond),
	})

	tracker.RecordFile(FileTransfer{
		Path:   "file2.txt",
		Size:   2048,
		Status: TransferStatusSkipped,
	})

	tracker.RecordFile(FileTransfer{
		Path:   "file3.txt",
		Size:   512,
		Status: TransferStatusFailed,
		Error:  errors.New("network error"),
	})

	tracker.PrintSummary()

	output := buf.String()

	if !strings.Contains(output, "Uploading to test-repo") {
		t.Errorf("Expected header message in output, got: %s", output)
	}

	if !strings.Contains(output, "Files uploaded: 1") {
		t.Errorf("Expected 'Files uploaded: 1' in summary, got: %s", output)
	}

	if !strings.Contains(output, "skipped: 1") {
		t.Errorf("Expected 'skipped: 1' in summary, got: %s", output)
	}

	if !strings.Contains(output, "failed: 1") {
		t.Errorf("Expected 'failed: 1' in summary, got: %s", output)
	}
}

func TestTransferTrackerQuietMode(t *testing.T) {
	var buf bytes.Buffer
	logger := util.NewLogger(&buf)

	tracker := NewTransferTracker(TransferTypeDownload, "test-repo", logger, true, false, false)

	tracker.PrintHeader(1, 1024)

	tracker.RecordFile(FileTransfer{
		Path:   "file1.txt",
		Size:   1024,
		Status: TransferStatusSuccess,
	})

	tracker.PrintSummary()

	output := buf.String()
	// In quiet mode, header is suppressed but summary is still shown
	if strings.Contains(output, "Downloading from") {
		t.Errorf("Expected no header in quiet mode, got: %s", output)
	}
	if !strings.Contains(output, "Files downloaded: 1") {
		t.Errorf("Expected summary in quiet mode, got: %s", output)
	}
}

func TestTransferTrackerVerboseMode(t *testing.T) {
	var buf bytes.Buffer
	logger := util.NewVerboseLogger(&buf)

	tracker := NewTransferTracker(TransferTypeUpload, "test-repo", logger, false, true, false)

	tracker.PrintHeader(1, 1024)

	output := buf.String()
	if !strings.Contains(output, "Total files: 1") {
		t.Errorf("Expected verbose header info in output, got: %s", output)
	}
}

func TestProgressWriter(t *testing.T) {
	var buf bytes.Buffer
	pw := NewProgressWriter(&buf)

	data := []byte("hello world")
	n, err := pw.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	if pw.BytesWritten() != int64(len(data)) {
		t.Errorf("Expected BytesWritten() = %d, got %d", len(data), pw.BytesWritten())
	}

	if buf.String() != string(data) {
		t.Errorf("Expected buffer to contain %q, got %q", data, buf.String())
	}
}

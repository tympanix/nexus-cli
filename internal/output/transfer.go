package output

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/tympanix/nexus-cli/internal/util"
)

type TransferType string

const (
	TransferTypeUpload   TransferType = "upload"
	TransferTypeDownload TransferType = "download"
)

type TransferStatus string

const (
	TransferStatusSuccess TransferStatus = "success"
	TransferStatusSkipped TransferStatus = "skipped"
	TransferStatusFailed  TransferStatus = "failed"
)

type FileTransfer struct {
	Path       string
	Size       int64
	Status     TransferStatus
	Error      error
	StartTime  time.Time
	EndTime    time.Time
	BytesCount int64
}

type TransferTracker struct {
	transferType TransferType
	target       string
	startTime    time.Time
	endTime      time.Time
	files        []FileTransfer
	mu           sync.Mutex
	logger       util.Logger
	quietMode    bool
	verboseMode  bool
	showProgress bool
}

func NewTransferTracker(transferType TransferType, target string, logger util.Logger, quietMode, verboseMode, showProgress bool) *TransferTracker {
	return &TransferTracker{
		transferType: transferType,
		target:       target,
		startTime:    time.Now(),
		files:        make([]FileTransfer, 0),
		logger:       logger,
		quietMode:    quietMode,
		verboseMode:  verboseMode,
		showProgress: showProgress,
	}
}

func (t *TransferTracker) PrintHeader(totalFiles int, totalSize int64) {
	if t.quietMode {
		return
	}
	action := "Uploading to"
	if t.transferType == TransferTypeDownload {
		action = "Downloading from"
	}
	t.logger.Printf("%s %s\n", action, t.target)
	if t.verboseMode {
		t.logger.Printf("Total files: %d, Total size: %s\n", totalFiles, formatBytes(totalSize))
	}
}

func (t *TransferTracker) RecordFile(file FileTransfer) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.files = append(t.files, file)

	if t.quietMode {
		return
	}

	if !t.showProgress {
		status := ""
		switch file.Status {
		case TransferStatusSuccess:
			elapsed := file.EndTime.Sub(file.StartTime)
			if elapsed > 0 {
				speed := float64(file.Size) / elapsed.Seconds()
				status = fmt.Sprintf("✓ %s (%s, %s/s)", file.Path, formatBytes(file.Size), formatBytes(int64(speed)))
			} else {
				status = fmt.Sprintf("✓ %s (%s)", file.Path, formatBytes(file.Size))
			}
		case TransferStatusSkipped:
			status = fmt.Sprintf("- %s (skipped)", file.Path)
		case TransferStatusFailed:
			status = fmt.Sprintf("✗ %s (failed: %v)", file.Path, file.Error)
		}
		t.logger.Println(status)
	} else if file.Status == TransferStatusFailed && t.verboseMode {
		t.logger.Printf("Error: %s: %v\n", file.Path, file.Error)
	}
}

func (t *TransferTracker) PrintSummary() {
	t.endTime = time.Now()

	t.mu.Lock()
	defer t.mu.Unlock()

	var successful, skipped, failed int
	var totalBytes int64

	for _, file := range t.files {
		switch file.Status {
		case TransferStatusSuccess:
			successful++
			totalBytes += file.Size
		case TransferStatusSkipped:
			skipped++
		case TransferStatusFailed:
			failed++
		}
	}

	elapsed := t.endTime.Sub(t.startTime)
	avgSpeed := float64(0)
	if elapsed.Seconds() > 0 {
		avgSpeed = float64(totalBytes) / elapsed.Seconds()
	}

	action := "uploaded"
	if t.transferType == TransferTypeDownload {
		action = "downloaded"
	}

	summary := fmt.Sprintf("Files %s: %d", action, successful)
	if skipped > 0 {
		summary += fmt.Sprintf(", skipped: %d", skipped)
	}
	if failed > 0 {
		summary += fmt.Sprintf(", failed: %d", failed)
	}
	summary += fmt.Sprintf(", size: %s", formatBytes(totalBytes))
	summary += fmt.Sprintf(", time: %s", formatDuration(elapsed))
	if avgSpeed > 0 {
		summary += fmt.Sprintf(", speed: %s/s", formatBytes(int64(avgSpeed)))
	}

	t.logger.Println(summary)
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", d.Seconds()*1000)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

type ProgressWriter struct {
	writer       io.Writer
	bytesWritten int64
	mu           sync.Mutex
}

func NewProgressWriter(writer io.Writer) *ProgressWriter {
	return &ProgressWriter{writer: writer}
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	pw.mu.Lock()
	pw.bytesWritten += int64(n)
	pw.mu.Unlock()
	return n, err
}

func (pw *ProgressWriter) BytesWritten() int64 {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	return pw.bytesWritten
}

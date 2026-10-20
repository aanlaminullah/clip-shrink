package backend

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type PresetType string

const (
	PresetYouTubeStandard PresetType = "youtube"
	PresetMaxSaving       PresetType = "hevc"
	PresetCustom          PresetType = "custom"
)

// CompressionConfig holds options passed from UI
type CompressionConfig struct {
	InputPath      string     `json:"inputPath"`
	OutputPath     string     `json:"outputPath,omitempty"`
	Preset         PresetType `json:"preset"` // "youtube", "hevc", "custom"
	CustomCodec    string     `json:"customCodec,omitempty"`
	CRF            int        `json:"crf,omitempty"`
	Resolution     string     `json:"resolution,omitempty"` // "original", "1080p", "720p", "480p"
	UseHardwareAcc bool       `json:"useHardwareAcc"`
	TotalDuration  float64    `json:"totalDuration"` // in seconds
}

// ProgressPayload is sent via Wails event bus to the frontend
type ProgressPayload struct {
	InputPath     string  `json:"inputPath"`
	Progress      float64 `json:"progress"` // 0.0 - 100.0
	CurrentTime   float64 `json:"currentTime"`
	TotalDuration float64 `json:"totalDuration"`
	FPS           float64 `json:"fps"`
	Speed         string  `json:"speed"`
	Bitrate       string  `json:"bitrate"`
	Size          int64   `json:"size"`
	Status        string  `json:"status"` // "running", "completed", "cancelled", "error"
	Message       string  `json:"message,omitempty"`
}

// CompressionResult is returned when compression finishes
type CompressionResult struct {
	InputPath       string  `json:"inputPath"`
	OutputPath      string  `json:"outputPath"`
	OriginalSize    int64   `json:"originalSize"`
	CompressedSize  int64   `json:"compressedSize"`
	SavedBytes      int64   `json:"savedBytes"`
	SavingsPercent  float64 `json:"savingsPercent"`
	DurationSeconds float64 `json:"durationSeconds"`
}

// Compressor manages active compression jobs
type Compressor struct {
	mu           sync.Mutex
	cancelFunc   context.CancelFunc
	activeCmd    *exec.Cmd
	activeOutput string
	isCancelled  bool
}

// NewCompressor initializes a new Compressor instance
func NewCompressor() *Compressor {
	return &Compressor{}
}

// Cancel terminates any currently running compression and removes incomplete output
func (c *Compressor) Cancel() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancelFunc != nil {
		c.isCancelled = true
		c.cancelFunc()
		if c.activeCmd != nil && c.activeCmd.Process != nil {
			_ = c.activeCmd.Process.Kill()
		}
	}
}

// Compress executes FFmpeg with progress tracking and event callbacks
func (c *Compressor) Compress(
	ffmpegBin string,
	cfg CompressionConfig,
	onProgress func(ProgressPayload),
) (*CompressionResult, error) {
	if _, err := os.Stat(cfg.InputPath); err != nil {
		return nil, fmt.Errorf("input file does not exist: %w", err)
	}

	// Determine output path if not specified
	outputPath := cfg.OutputPath
	if outputPath == "" {
		outputPath = GenerateDefaultOutputPath(cfg.InputPath)
	}

	// If output path is identical to input path (e.g. user chose to overwrite/replace),
	// we write to a temporary file in the same directory first to avoid corrupting the input file.
	// On Windows, file paths must be compared case-insensitively and resolved to physical files.
	isReplacingInput := isSameFilePath(outputPath, cfg.InputPath)
	actualOutputPath := outputPath
	if isReplacingInput {
		ext := filepath.Ext(outputPath)
		if ext == "" {
			ext = ".mp4"
		}
		tempName := fmt.Sprintf("clipshrink_tmp_%d_%d%s", os.Getpid(), time.Now().UnixNano(), ext)
		actualOutputPath = filepath.Join(filepath.Dir(outputPath), tempName)
	}

	// Prepare cancellation context
	ctx, cancel := context.WithCancel(context.Background())

	c.mu.Lock()
	c.cancelFunc = cancel
	c.activeOutput = actualOutputPath
	c.isCancelled = false
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.cancelFunc = nil
		c.activeCmd = nil
		c.mu.Unlock()
	}()

	args := buildFFmpegArgs(cfg, actualOutputPath)
	cmd := exec.CommandContext(ctx, ffmpegBin, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open stderr pipe: %w", err)
	}

	c.mu.Lock()
	c.activeCmd = cmd
	c.mu.Unlock()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Collect stderr in background in case of error
	var stderrOutput strings.Builder
	go func() {
		reader := bufio.NewReader(stderr)
		for {
			line, err := reader.ReadString('\n')
			if len(line) > 0 {
				stderrOutput.WriteString(line)
			}
			if err != nil {
				break
			}
		}
	}()

	// Parse progress on stdout
	c.parseProgress(stdout, cfg, onProgress)

	err = cmd.Wait()

	c.mu.Lock()
	wasCancelled := c.isCancelled
	c.mu.Unlock()

	if wasCancelled {
		// Clean up partial output file
		_ = os.Remove(actualOutputPath)
		if onProgress != nil {
			onProgress(ProgressPayload{
				InputPath: cfg.InputPath,
				Status:    "cancelled",
				Message:   "Compression cancelled by user",
			})
		}
		return nil, fmt.Errorf("compression was cancelled")
	}

	if err != nil {
		_ = os.Remove(actualOutputPath)
		stderrStr := stderrOutput.String()
		lowerErr := strings.ToLower(stderrStr)

		// If hardware acceleration failed due to missing or unsupported GPU/driver, gracefully fallback to CPU encoding
		if cfg.UseHardwareAcc && (strings.Contains(lowerErr, "nvenc") || strings.Contains(lowerErr, "cuda") || strings.Contains(lowerErr, "device") || strings.Contains(lowerErr, "cannot load") || strings.Contains(lowerErr, "driver")) {
			if onProgress != nil {
				onProgress(ProgressPayload{
					InputPath: cfg.InputPath,
					Status:    "running",
					Message:   "GPU NVENC tidak tersedia/gagal, beralih otomatis ke CPU encoding...",
				})
			}
			fallbackCfg := cfg
			fallbackCfg.UseHardwareAcc = false
			return c.Compress(ffmpegBin, fallbackCfg, onProgress)
		}

		errMsg := fmt.Sprintf("FFmpeg failed: %v | %s", err, truncateString(stderrStr, 400))
		if onProgress != nil {
			onProgress(ProgressPayload{
				InputPath: cfg.InputPath,
				Status:    "error",
				Message:   errMsg,
			})
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	// Calculate stats
	inputStat, _ := os.Stat(cfg.InputPath)
	outputStat, err := os.Stat(actualOutputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat compressed output file: %w", err)
	}

	origSize := int64(0)
	if inputStat != nil {
		origSize = inputStat.Size()
	}
	compSize := outputStat.Size()
	savedBytes := origSize - compSize
	var savingsPercent float64
	if origSize > 0 {
		savingsPercent = (float64(savedBytes) / float64(origSize)) * 100.0
	}

	// If replacing the original input file, swap temp file with the original safely
	if isReplacingInput {
		if err := replaceOriginalFile(actualOutputPath, cfg.InputPath); err != nil {
			_ = os.Remove(actualOutputPath)
			return nil, fmt.Errorf("failed to replace original file: %w", err)
		}
	}

	result := &CompressionResult{
		InputPath:       cfg.InputPath,
		OutputPath:      outputPath,
		OriginalSize:    origSize,
		CompressedSize:  compSize,
		SavedBytes:      savedBytes,
		SavingsPercent:  savingsPercent,
		DurationSeconds: cfg.TotalDuration,
	}

	if onProgress != nil {
		onProgress(ProgressPayload{
			InputPath:     cfg.InputPath,
			Progress:      100.0,
			CurrentTime:   cfg.TotalDuration,
			TotalDuration: cfg.TotalDuration,
			Size:          compSize,
			Status:        "completed",
		})
	}

	return result, nil
}

func (c *Compressor) parseProgress(
	stdout io.Reader,
	cfg CompressionConfig,
	onProgress func(ProgressPayload),
) {
	scanner := bufio.NewScanner(stdout)
	currentProgress := ProgressPayload{
		InputPath:     cfg.InputPath,
		TotalDuration: cfg.TotalDuration,
		Status:        "running",
	}

	for scanner.Scan() {
		c.mu.Lock()
		cancelled := c.isCancelled
		c.mu.Unlock()
		if cancelled {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "frame":
			// can record frame if needed
		case "fps":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				currentProgress.FPS = f
			}
		case "speed":
			currentProgress.Speed = val
		case "bitrate":
			currentProgress.Bitrate = val
		case "total_size":
			if s, err := strconv.ParseInt(val, 10, 64); err == nil {
				currentProgress.Size = s
			}
		case "out_time_us":
			if us, err := strconv.ParseInt(val, 10, 64); err == nil {
				currentSec := float64(us) / 1000000.0
				currentProgress.CurrentTime = currentSec
				if cfg.TotalDuration > 0 {
					pct := (currentSec / cfg.TotalDuration) * 100.0
					if pct > 99.9 {
						pct = 99.9
					}
					if pct < 0 {
						pct = 0
					}
					currentProgress.Progress = pct
				}
			}
		case "progress":
			if val == "continue" {
				if onProgress != nil {
					onProgress(currentProgress)
				}
			}
		}
	}
}

func buildFFmpegArgs(cfg CompressionConfig, outputPath string) []string {
	var args []string

	// Overwrite without prompt
	args = append(args, "-y", "-hide_banner", "-loglevel", "error")

	// Input file
	args = append(args, "-i", cfg.InputPath)

	// Scale filter if requested
	scaleFilter := ""
	switch strings.ToLower(cfg.Resolution) {
	case "1080p":
		scaleFilter = "scale=-2:'min(1080,ih)'"
	case "720p":
		scaleFilter = "scale=-2:'min(720,ih)'"
	case "480p":
		scaleFilter = "scale=-2:'min(480,ih)'"
	}
	if scaleFilter != "" {
		args = append(args, "-vf", scaleFilter)
	}

	// Presets configuration
	switch cfg.Preset {
	case PresetMaxSaving:
		// H.265 / HEVC - CRF 28
		if cfg.UseHardwareAcc {
			args = append(args, "-c:v", "hevc_nvenc", "-cq", "28", "-preset", "p4")
		} else {
			args = append(args, "-c:v", "libx265", "-crf", "28", "-preset", "medium")
		}
		args = append(args, "-c:a", "aac", "-b:a", "128k")

	case PresetCustom:
		crf := cfg.CRF
		if crf < 16 || crf > 40 {
			crf = 23
		}
		codec := strings.ToLower(cfg.CustomCodec)
		if strings.Contains(codec, "hevc") || strings.Contains(codec, "265") {
			if cfg.UseHardwareAcc {
				args = append(args, "-c:v", "hevc_nvenc", "-cq", strconv.Itoa(crf), "-preset", "p4")
			} else {
				args = append(args, "-c:v", "libx265", "-crf", strconv.Itoa(crf), "-preset", "medium")
			}
		} else {
			if cfg.UseHardwareAcc {
				args = append(args, "-c:v", "h264_nvenc", "-cq", strconv.Itoa(crf), "-preset", "p4")
			} else {
				args = append(args, "-c:v", "libx264", "-crf", strconv.Itoa(crf), "-preset", "medium")
			}
		}
		args = append(args, "-c:a", "aac", "-b:a", "128k")

	case PresetYouTubeStandard:
		fallthrough
	default:
		// H.264 - CRF 23
		if cfg.UseHardwareAcc {
			args = append(args, "-c:v", "h264_nvenc", "-cq", "23", "-preset", "p4")
		} else {
			args = append(args, "-c:v", "libx264", "-crf", "23", "-preset", "medium")
		}
		args = append(args, "-c:a", "aac", "-b:a", "128k", "-movflags", "+faststart")
	}

	// Progress reporting to pipe:1 (stdout)
	args = append(args, "-progress", "pipe:1")

	// Destination
	args = append(args, outputPath)

	return args
}

// GenerateDefaultOutputPath creates a non-colliding output path with `_compressed.mp4`
func GenerateDefaultOutputPath(inputPath string) string {
	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)

	candidate := filepath.Join(dir, fmt.Sprintf("%s_compressed.mp4", base))
	if !fileExists(candidate) {
		return candidate
	}

	for i := 1; i < 1000; i++ {
		candidate = filepath.Join(dir, fmt.Sprintf("%s_compressed_%d.mp4", base, i))
		if !fileExists(candidate) {
			return candidate
		}
	}
	return candidate
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[len(s)-maxLen:]
}

// isSameFilePath checks if two file paths point to the same file on disk.
// On Windows, paths are case-insensitive and can differ by drive letter casing, slashes, or 8.3 aliases.
func isSameFilePath(path1, path2 string) bool {
	if path1 == "" || path2 == "" {
		return false
	}
	p1, err1 := filepath.Abs(filepath.Clean(path1))
	p2, err2 := filepath.Abs(filepath.Clean(path2))
	if err1 == nil && err2 == nil {
		path1 = p1
		path2 = p2
	}
	// Case-insensitive comparison (Windows standard NTFS/FAT/ReFS)
	if strings.EqualFold(path1, path2) {
		return true
	}
	// If both files exist, check if their OS file identity is identical
	stat1, err1 := os.Stat(path1)
	stat2, err2 := os.Stat(path2)
	if err1 == nil && err2 == nil && os.SameFile(stat1, stat2) {
		return true
	}
	return false
}

// replaceOriginalFile safely replaces targetPath with tempPath on Windows
func replaceOriginalFile(tempPath, targetPath string) error {
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*50) * time.Millisecond)
		}

		backupPath := targetPath + fmt.Sprintf(".clipshrink_bak_%d", time.Now().UnixNano())
		_ = os.Remove(backupPath)

		// Step 1: Rename original target to backup
		if err := os.Rename(targetPath, backupPath); err != nil {
			if errRem := os.Remove(targetPath); errRem != nil {
				lastErr = fmt.Errorf("failed to remove existing target file: %w (backup err: %v)", errRem, err)
				continue
			}
		}

		// Step 2: Rename temp to original target
		if err := os.Rename(tempPath, targetPath); err != nil {
			if fileExists(backupPath) {
				_ = os.Rename(backupPath, targetPath)
			}
			lastErr = fmt.Errorf("failed to move temp file into place: %w", err)
			continue
		}

		// Step 3: Clean up backup
		_ = os.Remove(backupPath)
		return nil
	}

	return fmt.Errorf("failed to replace original file after retries: %w", lastErr)
}


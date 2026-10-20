package backend

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func ensureTestSample(t *testing.T, ffmpegBin string) string {
	samplePath, err := filepath.Abs("test_sample.mp4")
	if err != nil {
		t.Fatalf("Failed to get abs path: %v", err)
	}

	if !fileExists(samplePath) {
		cmd := exec.Command(ffmpegBin,
			"-y",
			"-f", "lavfi", "-i", "testsrc=duration=2:size=640x360:rate=30",
			"-f", "lavfi", "-i", "sine=frequency=1000:duration=2",
			"-c:v", "libx264", "-c:a", "aac",
			samplePath,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to generate test sample video: %v\n%s", err, string(out))
		}
	}
	return samplePath
}

func TestResolveBinaryPaths(t *testing.T) {
	paths, err := ResolveBinaryPaths()
	if err != nil {
		t.Fatalf("ResolveBinaryPaths failed: %v", err)
	}

	if paths.FFmpeg == "" {
		t.Errorf("Expected ffmpeg path, got empty string")
	}
	if paths.FFprobe == "" {
		t.Errorf("Expected ffprobe path, got empty string")
	}
	t.Logf("Found FFmpeg at: %s", paths.FFmpeg)
	t.Logf("Found FFprobe at: %s", paths.FFprobe)
}

func TestProbeVideo(t *testing.T) {
	paths, err := ResolveBinaryPaths()
	if err != nil {
		t.Fatalf("ResolveBinaryPaths failed: %v", err)
	}

	samplePath := ensureTestSample(t, paths.FFmpeg)

	meta, err := ProbeVideo(paths.FFprobe, samplePath)
	if err != nil {
		t.Fatalf("ProbeVideo failed: %v", err)
	}

	if meta.Width != 640 || meta.Height != 360 {
		t.Errorf("Expected resolution 640x360, got %dx%d", meta.Width, meta.Height)
	}

	if meta.Codec != "h264" {
		t.Errorf("Expected codec h264, got %s", meta.Codec)
	}

	if meta.Duration <= 0 {
		t.Errorf("Expected positive duration, got %f", meta.Duration)
	}

	if meta.FPS != 30 {
		t.Errorf("Expected 30 FPS, got %f", meta.FPS)
	}

	t.Logf("Probe result: %+v", meta)
}

func TestCompressYouTubeStandard(t *testing.T) {
	paths, err := ResolveBinaryPaths()
	if err != nil {
		t.Fatalf("ResolveBinaryPaths failed: %v", err)
	}

	samplePath := ensureTestSample(t, paths.FFmpeg)
	outputPath, _ := filepath.Abs("test_sample_compressed.mp4")
	_ = os.Remove(outputPath)
	defer os.Remove(outputPath)

	compressor := NewCompressor()
	progressCount := 0

	cfg := CompressionConfig{
		InputPath:     samplePath,
		OutputPath:    outputPath,
		Preset:        PresetYouTubeStandard,
		TotalDuration: 2.0,
	}

	res, err := compressor.Compress(paths.FFmpeg, cfg, func(p ProgressPayload) {
		progressCount++
	})

	if err != nil {
		t.Fatalf("Compression failed: %v", err)
	}

	if res.CompressedSize <= 0 {
		t.Errorf("Expected compressed file size > 0, got %d", res.CompressedSize)
	}

	if !fileExists(outputPath) {
		t.Errorf("Output file %s was not created", outputPath)
	}

	t.Logf("Compression completed successfully. Size: %d -> %d bytes. Savings: %.2f%%. Progress events: %d",
		res.OriginalSize, res.CompressedSize, res.SavingsPercent, progressCount)
}

func TestCompressCancellation(t *testing.T) {
	paths, err := ResolveBinaryPaths()
	if err != nil {
		t.Fatalf("ResolveBinaryPaths failed: %v", err)
	}

	samplePath := ensureTestSample(t, paths.FFmpeg)
	outputPath, _ := filepath.Abs("test_cancel_output.mp4")
	_ = os.Remove(outputPath)

	compressor := NewCompressor()

	// Trigger cancellation quickly
	go func() {
		time.Sleep(30 * time.Millisecond)
		compressor.Cancel()
	}()

	cfg := CompressionConfig{
		InputPath:     samplePath,
		OutputPath:    outputPath,
		Preset:        PresetMaxSaving,
		TotalDuration: 2.0,
	}

	_, err = compressor.Compress(paths.FFmpeg, cfg, nil)
	if err == nil {
		t.Errorf("Expected error on cancellation, got nil")
	}

	// Output file should have been cleaned up
	if fileExists(outputPath) {
		t.Errorf("Cancelled output file was not cleaned up!")
		_ = os.Remove(outputPath)
	}
}

func TestCompressInPlaceReplacement(t *testing.T) {
	paths, err := ResolveBinaryPaths()
	if err != nil {
		t.Fatalf("ResolveBinaryPaths failed: %v", err)
	}

	samplePath := ensureTestSample(t, paths.FFmpeg)

	// Create a copy to test in-place overwrite
	testCopyPath, _ := filepath.Abs("test_replace_target.mp4")
	_ = os.Remove(testCopyPath)
	defer os.Remove(testCopyPath)

	data, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatalf("Failed to read sample: %v", err)
	}
	if err := os.WriteFile(testCopyPath, data, 0644); err != nil {
		t.Fatalf("Failed to write copy: %v", err)
	}

	compressor := NewCompressor()
	cfg := CompressionConfig{
		InputPath:     testCopyPath,
		OutputPath:    testCopyPath, // Identical to input path: in-place replacement
		Preset:        PresetYouTubeStandard,
		TotalDuration: 2.0,
	}

	res, err := compressor.Compress(paths.FFmpeg, cfg, nil)
	if err != nil {
		t.Fatalf("In-place compression failed: %v", err)
	}

	if res.OutputPath != testCopyPath {
		t.Errorf("Expected output path %s, got %s", testCopyPath, res.OutputPath)
	}

	if !fileExists(testCopyPath) {
		t.Errorf("Target file %s does not exist after in-place replacement", testCopyPath)
	}

	meta, err := ProbeVideo(paths.FFprobe, testCopyPath)
	if err != nil {
		t.Fatalf("Failed to probe replaced file: %v", err)
	}

	if meta.Duration <= 0 {
		t.Errorf("Replaced file has invalid duration: %f", meta.Duration)
	}
	t.Logf("In-place replacement succeeded. Original: %d, New: %d", res.OriginalSize, res.CompressedSize)
}

func TestCompressCaseMismatchOverwrite(t *testing.T) {
	paths, err := ResolveBinaryPaths()
	if err != nil {
		t.Fatalf("ResolveBinaryPaths failed: %v", err)
	}

	// Create a 10-second video named test_case_target.MP4
	testCopyPath, _ := filepath.Abs("test_case_target.MP4")
	_ = os.Remove(testCopyPath)
	defer os.Remove(testCopyPath)

	genCmd := exec.Command(paths.FFmpeg,
		"-y",
		"-f", "lavfi", "-i", "testsrc=duration=10:size=1280x720:rate=30",
		"-f", "lavfi", "-i", "sine=frequency=1000:duration=10",
		"-c:v", "libx264", "-c:a", "aac",
		testCopyPath,
	)
	if out, err := genCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to generate 10s video: %v\n%s", err, string(out))
	}

	// Output path is lowercase .mp4
	outputPath := strings.TrimSuffix(testCopyPath, ".MP4") + ".mp4"

	compressor := NewCompressor()
	cfg := CompressionConfig{
		InputPath:     testCopyPath,
		OutputPath:    outputPath,
		Preset:        PresetYouTubeStandard,
		TotalDuration: 10.0,
	}

	res, err := compressor.Compress(paths.FFmpeg, cfg, nil)
	if err != nil {
		t.Fatalf("Case mismatch compression failed: %v", err)
	}

	meta, err := ProbeVideo(paths.FFprobe, res.OutputPath)
	if err != nil {
		t.Fatalf("Failed to probe output file: %v", err)
	}

	t.Logf("Probe duration: %f, original was 10.0s", meta.Duration)
	if meta.Duration < 9.5 {
		t.Errorf("Corrupt output! Expected ~10s duration, but got %f seconds!", meta.Duration)
	}
}

func TestCompressDriveLetterAndSlashDifferences(t *testing.T) {
	paths, err := ResolveBinaryPaths()
	if err != nil {
		t.Fatalf("ResolveBinaryPaths failed: %v", err)
	}

	samplePath := ensureTestSample(t, paths.FFmpeg)

	testCopyPath, _ := filepath.Abs("test_drive_slash_target.mp4")
	_ = os.Remove(testCopyPath)
	defer os.Remove(testCopyPath)

	data, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatalf("Failed to read sample: %v", err)
	}
	if err := os.WriteFile(testCopyPath, data, 0644); err != nil {
		t.Fatalf("Failed to write copy: %v", err)
	}

	// Create input path with lowercase drive and forward slashes: e.g. "d:/project/..."
	inputWithSlash := filepath.ToSlash(testCopyPath)
	if len(inputWithSlash) > 1 && inputWithSlash[1] == ':' {
		inputWithSlash = strings.ToLower(string(inputWithSlash[0])) + inputWithSlash[1:]
	}

	// Output path with uppercase drive and backslashes
	outputWithBackslash := filepath.FromSlash(testCopyPath)
	if len(outputWithBackslash) > 1 && outputWithBackslash[1] == ':' {
		outputWithBackslash = strings.ToUpper(string(outputWithBackslash[0])) + outputWithBackslash[1:]
	}

	compressor := NewCompressor()
	cfg := CompressionConfig{
		InputPath:     inputWithSlash,
		OutputPath:    outputWithBackslash,
		Preset:        PresetYouTubeStandard,
		TotalDuration: 2.0,
	}

	res, err := compressor.Compress(paths.FFmpeg, cfg, nil)
	if err != nil {
		t.Fatalf("Drive letter / slash mismatch compression failed: %v", err)
	}

	meta, err := ProbeVideo(paths.FFprobe, res.OutputPath)
	if err != nil {
		t.Fatalf("Failed to probe output file: %v", err)
	}

	if meta.Duration <= 0 {
		t.Errorf("Invalid duration: %f", meta.Duration)
	}
}


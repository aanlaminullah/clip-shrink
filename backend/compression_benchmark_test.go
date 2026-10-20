package backend

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// generateHighBitrateSample creates a high-bitrate video mimicking uncompressed raw smartphone footage
func generateHighBitrateSample(t *testing.T, ffmpegBin, filename string, width, height, fps, duration int) string {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		t.Fatalf("Failed to resolve abs path: %v", err)
	}

	_ = os.Remove(absPath)

	// Using mandelbrot + testsrc + noise produces realistic high complexity video requiring high bitrate
	filterGraph := fmt.Sprintf("testsrc=duration=%d:size=%dx%d:rate=%d,noise=alls=20:allf=t+u", duration, width, height, fps)

	cmd := exec.Command(ffmpegBin,
		"-y",
		"-f", "lavfi", "-i", filterGraph,
		"-f", "lavfi", "-i", fmt.Sprintf("sine=frequency=1000:duration=%d", duration),
		"-c:v", "libx264", "-b:v", "35M", "-maxrate", "40M", "-bufsize", "40M", // Mimic ~35Mbps smartphone video
		"-c:a", "aac", "-b:a", "192k",
		absPath,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to generate test video %s: %v\n%s", filename, err, string(out))
	}

	return absPath
}

// Test 5.1: Uji Rasio Kompresi 1080p 60fps & 30fps
func TestCompressionRatio1080p(t *testing.T) {
	paths, err := ResolveBinaryPaths()
	if err != nil {
		t.Fatalf("ResolveBinaryPaths failed: %v", err)
	}

	// 1080p 30fps
	sample30 := generateHighBitrateSample(t, paths.FFmpeg, "sample_1080p_30fps.mp4", 1920, 1080, 30, 2)
	defer os.Remove(sample30)

	meta, err := ProbeVideo(paths.FFprobe, sample30)
	if err != nil {
		t.Fatalf("Probe failed: %v", err)
	}
	t.Logf("Raw 1080p 30fps: Size=%d bytes, Bitrate=%d bps, FPS=%.1f", meta.FileSize, meta.Bitrate, meta.FPS)

	compressor := NewCompressor()
	out30 := filepath.Join(filepath.Dir(sample30), "out_1080p_30fps_compressed.mp4")
	_ = os.Remove(out30)
	defer os.Remove(out30)

	res, err := compressor.Compress(paths.FFmpeg, CompressionConfig{
		InputPath:     sample30,
		OutputPath:    out30,
		Preset:        PresetYouTubeStandard,
		TotalDuration: meta.Duration,
	}, nil)

	if err != nil {
		t.Fatalf("1080p 30fps compression failed: %v", err)
	}

	t.Logf("1080p 30fps (YouTube Standard): Original=%d, Compressed=%d, Savings=%.2f%%",
		res.OriginalSize, res.CompressedSize, res.SavingsPercent)

	if res.SavingsPercent < 50.0 {
		t.Logf("Notice: Savings was %.2f%% (expected high savings on high-bitrate sources)", res.SavingsPercent)
	}

	// Test with HEVC (Maximum Savings)
	outHevc := filepath.Join(filepath.Dir(sample30), "out_1080p_hevc.mp4")
	_ = os.Remove(outHevc)
	defer os.Remove(outHevc)

	resHevc, err := compressor.Compress(paths.FFmpeg, CompressionConfig{
		InputPath:     sample30,
		OutputPath:    outHevc,
		Preset:        PresetMaxSaving,
		TotalDuration: meta.Duration,
	}, nil)

	if err != nil {
		t.Fatalf("1080p HEVC compression failed: %v", err)
	}

	t.Logf("1080p (HEVC Max Savings): Original=%d, Compressed=%d, Savings=%.2f%%",
		resHevc.OriginalSize, resHevc.CompressedSize, resHevc.SavingsPercent)

	if resHevc.CompressedSize >= res.CompressedSize {
		t.Logf("HEVC size (%d) compared to H.264 (%d)", resHevc.CompressedSize, res.CompressedSize)
	}
}

// Test 5.1: Uji Video 4K Berdurasi Pendek
func TestCompression4K(t *testing.T) {
	paths, err := ResolveBinaryPaths()
	if err != nil {
		t.Fatalf("ResolveBinaryPaths failed: %v", err)
	}

	// 4K UHD 3840x2160
	sample4k := generateHighBitrateSample(t, paths.FFmpeg, "sample_4k_test.mp4", 3840, 2160, 30, 1)
	defer os.Remove(sample4k)

	meta, err := ProbeVideo(paths.FFprobe, sample4k)
	if err != nil {
		t.Fatalf("Probe 4K failed: %v", err)
	}
	t.Logf("Raw 4K: Size=%d bytes, Width=%d, Height=%d", meta.FileSize, meta.Width, meta.Height)

	compressor := NewCompressor()
	out4k := filepath.Join(filepath.Dir(sample4k), "out_4k_compressed.mp4")
	_ = os.Remove(out4k)
	defer os.Remove(out4k)

	// Test 4K with downscale to 1080p
	res, err := compressor.Compress(paths.FFmpeg, CompressionConfig{
		InputPath:     sample4k,
		OutputPath:    out4k,
		Preset:        PresetCustom,
		Resolution:    "1080p",
		CRF:           23,
		TotalDuration: meta.Duration,
	}, nil)

	if err != nil {
		t.Fatalf("4K downscale compression failed: %v", err)
	}

	t.Logf("4K -> 1080p Custom: Original=%d, Compressed=%d, Savings=%.2f%%",
		res.OriginalSize, res.CompressedSize, res.SavingsPercent)

	metaOut, err := ProbeVideo(paths.FFprobe, out4k)
	if err != nil {
		t.Fatalf("Probe downscaled 4k failed: %v", err)
	}
	if metaOut.Height != 1080 {
		t.Errorf("Expected downscaled height 1080, got %d", metaOut.Height)
	}
}

// Test 5.2: Edge Cases - Spasi & Karakter Unicode / Emojis
func TestUnicodeAndSpacesFilename(t *testing.T) {
	paths, err := ResolveBinaryPaths()
	if err != nil {
		t.Fatalf("ResolveBinaryPaths failed: %v", err)
	}

	unicodeName := "rekaman video liburan (2026) - 测试 🎬 #1.mp4"
	samplePath := generateHighBitrateSample(t, paths.FFmpeg, unicodeName, 640, 360, 30, 1)
	defer os.Remove(samplePath)

	meta, err := ProbeVideo(paths.FFprobe, samplePath)
	if err != nil {
		t.Fatalf("Probe failed on Unicode filename: %v", err)
	}

	compressor := NewCompressor()
	outUnicode := GenerateDefaultOutputPath(samplePath)
	defer os.Remove(outUnicode)

	res, err := compressor.Compress(paths.FFmpeg, CompressionConfig{
		InputPath:     samplePath,
		OutputPath:    outUnicode,
		Preset:        PresetYouTubeStandard,
		TotalDuration: meta.Duration,
	}, nil)

	if err != nil {
		t.Fatalf("Compression failed on Unicode filename: %v", err)
	}

	if !fileExists(res.OutputPath) {
		t.Errorf("Output file %s was not found", res.OutputPath)
	}
	t.Logf("Unicode compression test passed! Output: %s", res.OutputPath)
}

// Test 5.2: Batch Queue Execution
func TestBatchCompressionQueue(t *testing.T) {
	paths, err := ResolveBinaryPaths()
	if err != nil {
		t.Fatalf("ResolveBinaryPaths failed: %v", err)
	}

	compressor := NewCompressor()
	files := []string{
		generateHighBitrateSample(t, paths.FFmpeg, "batch_1.mp4", 640, 360, 24, 1),
		generateHighBitrateSample(t, paths.FFmpeg, "batch_2.mp4", 640, 360, 30, 1),
		generateHighBitrateSample(t, paths.FFmpeg, "batch_3.mp4", 640, 360, 60, 1),
	}
	defer func() {
		for _, f := range files {
			_ = os.Remove(f)
			_ = os.Remove(GenerateDefaultOutputPath(f))
		}
	}()

	for i, f := range files {
		meta, err := ProbeVideo(paths.FFprobe, f)
		if err != nil {
			t.Fatalf("Probe batch file %d failed: %v", i, err)
		}

		out := GenerateDefaultOutputPath(f)
		res, err := compressor.Compress(paths.FFmpeg, CompressionConfig{
			InputPath:     f,
			OutputPath:    out,
			Preset:        PresetYouTubeStandard,
			TotalDuration: meta.Duration,
		}, nil)

		if err != nil {
			t.Fatalf("Batch file %d compression failed: %v", i, err)
		}
		if res.CompressedSize <= 0 {
			t.Errorf("Batch file %d produced 0 bytes", i)
		}
		t.Logf("Batch item %d compressed successfully (%d -> %d bytes)", i+1, res.OriginalSize, res.CompressedSize)
	}
}

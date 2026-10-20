package backend

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// FFmpegPaths holds resolved binary paths for ffmpeg and ffprobe.
type FFmpegPaths struct {
	FFmpeg  string `json:"ffmpeg"`
	FFprobe string `json:"ffprobe"`
}

// ResolveBinaryPaths finds ffmpeg and ffprobe binaries.
// Search order:
// 1. Next to the executable in ./bin/ or current directory
// 2. Relative to working directory in ./bin/ or ./
// 3. System PATH via exec.LookPath
func ResolveBinaryPaths() (*FFmpegPaths, error) {
	ffmpegPath, err := resolveExecutable("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not found: %w", err)
	}

	ffprobePath, err := resolveExecutable("ffprobe")
	if err != nil {
		return nil, fmt.Errorf("ffprobe not found: %w", err)
	}

	return &FFmpegPaths{
		FFmpeg:  ffmpegPath,
		FFprobe: ffprobePath,
	}, nil
}

func resolveExecutable(name string) (string, error) {
	filename := name
	if os.PathSeparator == '\\' {
		filename += ".exe"
	}

	// 1. Check relative to current running executable
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates := []string{
			filepath.Join(exeDir, "bin", filename),
			filepath.Join(exeDir, filename),
		}
		for _, candidate := range candidates {
			if fileExists(candidate) {
				return filepath.Clean(candidate), nil
			}
		}
	}

	// 2. Check relative to current working directory (including parent for tests)
	if cwd, err := os.Getwd(); err == nil {
		candidates := []string{
			filepath.Join(cwd, "bin", filename),
			filepath.Join(cwd, filename),
			filepath.Join(cwd, "..", "bin", filename),
			filepath.Join(cwd, "..", filename),
		}
		for _, candidate := range candidates {
			if fileExists(candidate) {
				return filepath.Clean(candidate), nil
			}
		}
	}

	// 3. Check common Windows install locations (WinGet Links, etc.)
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		wingetLink := filepath.Join(localAppData, "Microsoft", "WinGet", "Links", filename)
		if fileExists(wingetLink) {
			return filepath.Clean(wingetLink), nil
		}
	}

	// 4. Check system PATH
	if path, err := exec.LookPath(name); err == nil {
		return filepath.Clean(path), nil
	}

	return "", fmt.Errorf("could not find %s in exe dir, ./bin/, or system PATH", filename)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return true
}

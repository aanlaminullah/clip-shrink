package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"clipshrink/backend"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	compressor *backend.Compressor
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		compressor: backend.NewCompressor(),
	}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Register OS native drag-and-drop listener to receive full absolute paths
	runtime.OnFileDrop(ctx, func(x, y int, paths []string) {
		runtime.EventsEmit(ctx, "file:dropped", paths)
	})
}

// CheckBinaries returns resolved paths for ffmpeg and ffprobe
func (a *App) CheckBinaries() (*backend.FFmpegPaths, error) {
	return backend.ResolveBinaryPaths()
}

// SelectFile opens native single file picker
func (a *App) SelectFile() (string, error) {
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Pilih Berkas Video",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Video Files (*.mp4, *.mov, *.mkv, *.avi, *.webm)",
				Pattern:     "*.mp4;*.mov;*.mkv;*.avi;*.webm;*.flv;*.wmv",
			},
			{
				DisplayName: "Semua Berkas (*.*)",
				Pattern:     "*.*",
			},
		},
	})
	if err != nil {
		return "", err
	}
	return selection, nil
}

// SelectMultipleFiles opens native multiple file picker
func (a *App) SelectMultipleFiles() ([]string, error) {
	selections, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Pilih Berkas Video (Batch)",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Video Files (*.mp4, *.mov, *.mkv, *.avi, *.webm)",
				Pattern:     "*.mp4;*.mov;*.mkv;*.avi;*.webm;*.flv;*.wmv",
			},
			{
				DisplayName: "Semua Berkas (*.*)",
				Pattern:     "*.*",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return selections, nil
}

// SelectOutputFolder opens native directory selection dialog
func (a *App) SelectOutputFolder() (string, error) {
	folder, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Pilih Folder Tujuan",
	})
	if err != nil {
		return "", err
	}
	return folder, nil
}

// GetVideoMetadata extracts metadata using ffprobe
func (a *App) GetVideoMetadata(filePath string) (*backend.VideoMetadata, error) {
	paths, err := backend.ResolveBinaryPaths()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg/ffprobe tidak ditemukan: %w", err)
	}

	return backend.ProbeVideo(paths.FFprobe, filePath)
}

// StartCompression starts compression process and emits progress events
func (a *App) StartCompression(config backend.CompressionConfig) (*backend.CompressionResult, error) {
	paths, err := backend.ResolveBinaryPaths()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg/ffprobe tidak ditemukan: %w", err)
	}

	result, err := a.compressor.Compress(paths.FFmpeg, config, func(p backend.ProgressPayload) {
		runtime.EventsEmit(a.ctx, "compression:progress", p)
	})

	return result, err
}

// CancelCompression stops the active compression
func (a *App) CancelCompression() {
	a.compressor.Cancel()
}

// OpenFolder opens the containing folder in Windows Explorer and selects the file if it exists
func (a *App) OpenFolder(targetPath string) error {
	info, err := os.Stat(targetPath)
	if err != nil {
		dir := filepath.Dir(targetPath)
		if _, errDir := os.Stat(dir); errDir == nil {
			return exec.Command("explorer", dir).Start()
		}
		return err
	}

	if info.IsDir() {
		return exec.Command("explorer", targetPath).Start()
	}

	return exec.Command("explorer", "/select,", targetPath).Start()
}

// OpenFile opens the file with the default associated application
func (a *App) OpenFile(filePath string) error {
	return exec.Command("cmd", "/c", "start", "", filePath).Start()
}

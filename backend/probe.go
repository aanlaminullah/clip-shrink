package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// FFprobeOutput maps ffprobe JSON root
type FFprobeOutput struct {
	Streams []FFprobeStream `json:"streams"`
	Format  FFprobeFormat   `json:"format"`
}

// FFprobeStream maps ffprobe individual stream
type FFprobeStream struct {
	Index        int    `json:"index"`
	CodecType    string `json:"codec_type"`
	CodecName    string `json:"codec_name"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	RFrameRate   string `json:"r_frame_rate"`
	AvgFrameRate string `json:"avg_frame_rate"`
	BitRate      string `json:"bit_rate"`
	Duration     string `json:"duration"`
}

// FFprobeFormat maps ffprobe container format
type FFprobeFormat struct {
	Filename   string `json:"filename"`
	FormatName string `json:"format_name"`
	Duration   string `json:"duration"`
	Size       string `json:"size"`
	BitRate    string `json:"bit_rate"`
}

// VideoMetadata is the normalized metadata returned to the UI
type VideoMetadata struct {
	FilePath     string  `json:"filePath"`
	FileName     string  `json:"fileName"`
	FileSize     int64   `json:"fileSize"`
	Duration     float64 `json:"duration"` // in seconds
	Width        int     `json:"width"`
	Height       int     `json:"height"`
	Bitrate      int64   `json:"bitrate"` // in bps
	Codec        string  `json:"codec"`
	FPS          float64 `json:"fps"`
	AudioCodec   string  `json:"audioCodec,omitempty"`
	AudioBitrate int64   `json:"audioBitrate,omitempty"`
}

// ProbeVideo runs ffprobe on the target file and returns structured VideoMetadata
func ProbeVideo(ffprobeBin, filePath string) (*VideoMetadata, error) {
	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, ffprobeBin,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffprobe failed (%v): %s", err, stderr.String())
	}

	var data FFprobeOutput
	if err := json.Unmarshal(stdout.Bytes(), &data); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe json: %w", err)
	}

	fileInfo, _ := os.Stat(filePath)
	var fileSize int64
	if fileInfo != nil {
		fileSize = fileInfo.Size()
	} else if data.Format.Size != "" {
		fileSize, _ = strconv.ParseInt(data.Format.Size, 10, 64)
	}

	meta := &VideoMetadata{
		FilePath: filePath,
		FileName: filepath.Base(filePath),
		FileSize: fileSize,
	}

	// Parse duration from container format
	if data.Format.Duration != "" {
		if dur, err := strconv.ParseFloat(data.Format.Duration, 64); err == nil {
			meta.Duration = dur
		}
	}

	// Parse container bitrate
	if data.Format.BitRate != "" {
		if br, err := strconv.ParseInt(data.Format.BitRate, 10, 64); err == nil {
			meta.Bitrate = br
		}
	}

	// Parse video and audio streams
	for _, stream := range data.Streams {
		if stream.CodecType == "video" && meta.Codec == "" {
			meta.Codec = stream.CodecName
			meta.Width = stream.Width
			meta.Height = stream.Height
			meta.FPS = parseFPS(stream.RFrameRate, stream.AvgFrameRate)

			if meta.Duration == 0 && stream.Duration != "" {
				if dur, err := strconv.ParseFloat(stream.Duration, 64); err == nil {
					meta.Duration = dur
				}
			}

			if meta.Bitrate == 0 && stream.BitRate != "" {
				if br, err := strconv.ParseInt(stream.BitRate, 10, 64); err == nil {
					meta.Bitrate = br
				}
			}
		} else if stream.CodecType == "audio" && meta.AudioCodec == "" {
			meta.AudioCodec = stream.CodecName
			if stream.BitRate != "" {
				if br, err := strconv.ParseInt(stream.BitRate, 10, 64); err == nil {
					meta.AudioBitrate = br
				}
			}
		}
	}

	return meta, nil
}

func parseFPS(rFrameRate, avgFrameRate string) float64 {
	target := rFrameRate
	if target == "" || target == "0/0" {
		target = avgFrameRate
	}

	if target == "" || target == "0/0" {
		return 0
	}

	parts := strings.Split(target, "/")
	if len(parts) == 1 {
		fps, _ := strconv.ParseFloat(parts[0], 64)
		return fps
	} else if len(parts) == 2 {
		num, err1 := strconv.ParseFloat(parts[0], 64)
		den, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 == nil && err2 == nil && den > 0 {
			return num / den
		}
	}
	return 0
}

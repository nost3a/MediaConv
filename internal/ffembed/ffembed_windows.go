//go:build windows

package ffembed

import (
	_ "embed"
	"os"
	"path/filepath"
)

// ffmpegExe 内嵌的 Windows 静态 ffmpeg
//
//go:embed ffmpeg.exe
var ffmpegExe []byte

// EnsureFFmpeg 把内嵌 ffmpeg.exe 释放到临时目录，返回路径
// Extract embedded ffmpeg.exe to temp dir, return path
func EnsureFFmpeg() string {
	dst := filepath.Join(os.TempDir(), "mediaconv_ffmpeg.exe")
	if _, err := os.Stat(dst); err == nil {
		return dst
	}
	if err := os.WriteFile(dst, ffmpegExe, 0755); err != nil {
		return ""
	}
	return dst
}

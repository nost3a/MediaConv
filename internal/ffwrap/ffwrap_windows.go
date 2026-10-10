//go:build windows

package ffwrap

import "github.com/nost3a/MediaConv/internal/ffembed"

// findEmbeddedFFmpeg Windows 下从内嵌资源释放 ffmpeg
// On Windows, extract embedded ffmpeg
func findEmbeddedFFmpeg() string {
	return ffembed.EnsureFFmpeg()
}

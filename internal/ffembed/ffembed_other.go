//go:build !windows

package ffembed

// EnsureFFmpeg 非 Windows 平台返回空，走系统 PATH
// Non-Windows: no-op, use system ffmpeg
func EnsureFFmpeg() string {
	return ""
}

//go:build !windows

package ffwrap

// findEmbeddedFFmpeg 非 Windows 不嵌入
// No embedding on non-Windows
func findEmbeddedFFmpeg() string {
	return ""
}

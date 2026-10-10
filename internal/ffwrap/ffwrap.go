package ffwrap

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"time"
)

// 启动时找 ffmpeg：先 PATH，再 exe 同目录，最后内嵌
func FindFFmpeg() (string, error) {
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p, nil
	}
	if p, err := exec.LookPath("ffmpeg.exe"); err == nil {
		return p, nil
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for _, name := range []string{"ffmpeg", "ffmpeg.exe"} {
			p := filepath.Join(dir, name)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p, nil
			}
		}
	}
	if p := findEmbeddedFFmpeg(); p != "" {
		return p, nil
	}
	return "", fmt.Errorf("ffmpeg not found")
}

func FindFFprobe() (string, error) {
	if p, err := exec.LookPath("ffprobe"); err == nil {
		return p, nil
	}
	if p, err := exec.LookPath("ffprobe.exe"); err == nil {
		return p, nil
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for _, name := range []string{"ffprobe", "ffprobe.exe"} {
			p := filepath.Join(dir, name)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p, nil
			}
		}
	}
	return "", fmt.Errorf("ffprobe not found")
}

// ProbeDuration 拿输入媒体总时长（秒），拿不到返回 -1
func ProbeDuration(probeBin, inPath string) float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, probeBin,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "json",
		inPath)
	out, err := cmd.Output()
	if err != nil {
		return -1
	}
	var v struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return -1
	}
	f, err := strconv.ParseFloat(v.Format.Duration, 64)
	if err != nil {
		return -1
	}
	return f
}

var timeRe = regexp.MustCompile(`time=(\d+):(\d+):(\d+(?:\.\d+)?)`)

// Run 跑 ffmpeg，把进度回调给 onProgress(秒)
func Run(ctx context.Context, bin string, args []string, onProgress func(sec float64)) error {
	all := append([]string{"-y", "-hide_banner"}, args...)
	cmd := exec.CommandContext(ctx, bin, all...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		sc := bufio.NewScanner(stderr)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			line := sc.Text()
			m := timeRe.FindStringSubmatch(line)
			if len(m) == 4 {
				h, _ := strconv.Atoi(m[1])
				mn, _ := strconv.Atoi(m[2])
				s, _ := strconv.ParseFloat(m[3], 64)
				onProgress(float64(h*3600+mn*60) + s)
			}
		}
	}()
	return cmd.Wait()
}

// InstallHint 友好的 ffmpeg 未安装提示
func InstallHint() string {
	switch runtime.GOOS {
	case "windows":
		return "未检测到 ffmpeg。请把 ffmpeg.exe 放在本程序同目录，或加入 PATH 环境变量。"
	case "linux":
		if _, err := os.Stat("/data/data/com.termux"); err == nil {
			return "未检测到 ffmpeg。请运行: pkg install ffmpeg"
		}
		return "未检测到 ffmpeg。Debian/Ubuntu 运行: sudo apt install ffmpeg"
	default:
		return "未检测到 ffmpeg，请自行安装。"
	}
}

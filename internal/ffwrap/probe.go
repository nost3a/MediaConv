package ffwrap

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// StreamInfo 单条流的探测结果
type StreamInfo struct {
	CodecType  string `json:"codec_type"`
	CodecName  string `json:"codec_name"`
	SampleRate string `json:"sample_rate"`
	Channels   int    `json:"channels"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}

// ProbeInfo ffprobe 的汇总结果
type ProbeInfo struct {
	Format   string       `json:"format"`
	Duration float64      `json:"duration"`
	BitRate  int64        `json:"bit_rate"`
	Streams  []StreamInfo `json:"streams"`
}

// Probe 用 ffprobe 取媒体信息（容器/时长/编码/分辨率）
func Probe(probeBin, inPath string) (ProbeInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, probeBin,
		"-v", "error",
		"-show_format",
		"-show_streams",
		"-of", "json",
		inPath)
	out, err := cmd.Output()
	if err != nil {
		return ProbeInfo{}, fmt.Errorf("ffprobe 执行失败: %w", err)
	}
	var v struct {
		Streams []StreamInfo `json:"streams"`
		Format  struct {
			FormatName string `json:"format_name"`
			Duration   string `json:"duration"`
			BitRate    string `json:"bit_rate"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return ProbeInfo{}, fmt.Errorf("解析 ffprobe 输出失败: %w", err)
	}
	info := ProbeInfo{
		Format:  v.Format.FormatName,
		Streams: v.Streams,
	}
	if d, err := parseFloat(v.Format.Duration); err == nil {
		info.Duration = d
	}
	if b, err := parseInt64(v.Format.BitRate); err == nil {
		info.BitRate = b
	}
	return info, nil
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func parseInt64(s string) (int64, error) {
	var i int64
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

// Package convert 单一转码核心。
// 所有入口（CLI、GUI、Web 任务、交互式 REPL）都走这里，避免重复实现导致行为分叉。
package convert

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nost3a/MediaConv/internal/ffwrap"
	"github.com/nost3a/MediaConv/internal/musicdecrypt"
	"github.com/nost3a/MediaConv/internal/preset"
)

// ProgressFn 进度回调：已处理秒数、总秒数（-1 表示未知）
type ProgressFn func(processedSec, totalSec float64)

// Options 单次转码参数
type Options struct {
	InPath     string // 输入文件
	OutFmt     string // 目标格式后缀（不含点），为空则按输入自动推断
	OutPath    string // 输出文件完整路径，为空则放到当前目录、用输入基名
	Quality    string // 音质档，如 192k / original / 720p，为空按类别取默认
	FFmpegBin  string
	FFprobeBin string
	OnProgress ProgressFn
}

// videoExts 视频后缀集合（包级，避免每次调用重建）
var videoExts = map[string]bool{
	"mp4": true, "mkv": true, "avi": true, "mov": true, "flv": true,
	"webm": true, "wmv": true, "mpg": true, "mpeg": true, "3gp": true,
	"ts": true, "mts": true, "vob": true, "m4v": true, "f4v": true, "ogv": true,
}

// Convert 执行一次完整转码，返回最终输出文件路径。
// 内部流程：还原私有格式 -> 推断输出格式/路径 -> 查预设 -> 音质校验 ->
// 直接复制优化 -> 跑 ffmpeg -> 清理中间文件与失败产物。
func Convert(ctx context.Context, opts Options) (string, error) {
	if opts.InPath == "" {
		return "", fmt.Errorf("必须指定输入文件")
	}
	fi, err := os.Stat(opts.InPath)
	if err != nil {
		return "", fmt.Errorf("输入文件不存在: %w", err)
	}
	if fi.IsDir() {
		return "", fmt.Errorf("输入不能是目录: %s", opts.InPath)
	}

	// 1) 私有格式先还原到临时文件（用后清理）
	realIn := opts.InPath
	decrypted := false
	if musicdecrypt.IsPrivateExt(opts.InPath) {
		decPath, _, err := musicdecrypt.Decrypt(opts.InPath)
		if err != nil {
			return "", fmt.Errorf("还原私有格式失败: %w", err)
		}
		realIn = decPath
		decrypted = true
		defer os.Remove(realIn)
	}

	// 2) 推断输出格式
	outFmt := strings.ToLower(strings.TrimSpace(opts.OutFmt))
	if outFmt == "" && opts.OutPath != "" {
		outFmt = strings.TrimPrefix(strings.ToLower(filepath.Ext(opts.OutPath)), ".")
	}
	if outFmt == "" {
		inL := strings.ToLower(strings.TrimPrefix(filepath.Ext(realIn), "."))
		if videoExts[inL] {
			outFmt = "mp4"
		} else {
			outFmt = "mp3"
		}
	}

	// 3) 推断输出路径
	outPath := opts.OutPath
	if outPath == "" {
		base := strings.TrimSuffix(filepath.Base(opts.InPath), filepath.Ext(opts.InPath))
		outPath = base + "." + outFmt
	}

	// 4) 查预设
	inExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(realIn), "."))
	p, ok := preset.Find(inExt, outFmt)
	if !ok {
		return "", fmt.Errorf("不支持的组合: %s -> %s", inExt, outFmt)
	}

	// 5) 音质默认值与校验（不再静默降级）
	qual := opts.Quality
	if qual == "" {
		if p.Category == "video2video" {
			qual = "original"
		} else {
			qual = "192k"
		}
	}
	if len(p.QualityArgs) > 0 {
		if _, hit := p.QualityArgs[qual]; !hit {
			return "", fmt.Errorf("音质 %q 不支持，可选: %s", qual, strings.Join(p.ValidQualities(), ", "))
		}
	}

	// 6) 直接复制优化：解密后格式已匹配目标，不再重编码
	if decrypted {
		decExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(realIn), "."))
		if decExt == outFmt {
			data, err := os.ReadFile(realIn)
			if err != nil {
				return "", fmt.Errorf("读取还原文件失败: %w", err)
			}
			if err := os.WriteFile(outPath, data, 0644); err != nil {
				return "", fmt.Errorf("写入失败: %w", err)
			}
			return outPath, nil
		}
	}

	// 7) 跑 ffmpeg
	args := p.BuildArgs(qual, outPath)
	args = append([]string{"-y", "-i", realIn}, args...)

	dur := float64(-1)
	if opts.FFprobeBin != "" {
		dur = ffwrap.ProbeDuration(opts.FFprobeBin, realIn)
	}
	onProg := opts.OnProgress
	if onProg == nil {
		onProg = func(float64, float64) {}
	}
	if err := ffwrap.Run(ctx, opts.FFmpegBin, args, func(sec float64) {
		onProg(sec, dur)
	}); err != nil {
		os.Remove(outPath) // 清理失败产物
		return "", fmt.Errorf("转码失败: %w", err)
	}
	return outPath, nil
}

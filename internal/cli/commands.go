package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nost3a/MediaConv/internal/convert"
	"github.com/nost3a/MediaConv/internal/ffwrap"
	"github.com/nost3a/MediaConv/internal/musicdecrypt"
	"github.com/nost3a/MediaConv/internal/preset"
	"github.com/nost3a/MediaConv/internal/tui"
	"github.com/nost3a/MediaConv/internal/webapi"
)

// registerAll 注册全部命令
func registerAll() {
	Register(Command{
		Name: "help", Aliases: []string{"h", "?"},
		Description: "显示帮助",
		Usage:       "help [命令]",
		Run:         cmdHelp,
	})
	Register(Command{
		Name: "start", Aliases: []string{"web", "server"},
		Description: "启动本地 Web 服务（无桌面环境首选）",
		Usage:       "start [--port 8090] [--host 127.0.0.1]",
		Run:         cmdStart,
	})
	Register(Command{
		Name: "gui", Aliases: []string{"app"},
		Description: "启动原生图形界面（需 -tags gui 构建）",
		Run:         cmdGUI, Hidden: false,
	})
	Register(Command{
		Name: "convert", Aliases: []string{"cvt", "c"},
		Description: "转码单个文件",
		Usage:       "convert -i 输入 [-f 格式] [-q 音质] [-o 输出]",
		Run:         cmdConvert,
	})
	Register(Command{
		Name: "up", Aliases: []string{"upload"},
		Description: "TUI 选择文件→转码→TUI 选择输出目录（无桌面专用）",
		Usage:       "up [-f 格式] [-q 音质]",
		Run:         cmdUp,
	})
	Register(Command{
		Name: "batch", Aliases: []string{"b"},
		Description: "批量转码目录下的文件",
		Usage:       "batch -d 目录 -f 格式 [-q 音质] [--recursive]",
		Run:         cmdBatch,
	})
	Register(Command{
		Name: "list", Aliases: []string{"formats", "ls"},
		Description: "列出支持的格式与音质",
		Run:         cmdList,
	})
	Register(Command{
		Name: "info", Aliases: []string{"i"},
		Description: "查看媒体文件信息（时长/编码/分辨率）",
		Usage:       "info <文件>",
		Run:         cmdInfo,
	})
	Register(Command{
		Name: "decrypt", Aliases: []string{"d"},
		Description: "还原私有音乐格式（不重编码）",
		Usage:       "decrypt <文件.ncm/kgm/qmc/kwm>",
		Run:         cmdDecrypt,
	})
	Register(Command{
		Name: "ffmpeg", Aliases: []string{"ff", "deps"},
		Description: "查看 ffmpeg/ffprobe 状态与安装提示",
		Run:         cmdFFmpeg,
	})
	Register(Command{
		Name: "clean", Aliases: []string{"cl"},
		Description: "清理临时工作目录",
		Run:         cmdClean,
	})
	Register(Command{
		Name: "version", Aliases: []string{"ver", "v"},
		Description: "显示版本号",
		Run:         cmdVersion,
	})
	Register(Command{
		Name: "repl", Aliases: []string{"shell", "sh"},
		Description: "进入交互式命令行",
		Run:         func(env *Env, args []string) error { return REPL(env) },
	})
	Register(Command{
		Name: "exit", Aliases: []string{"quit"},
		Description: "退出交互模式",
		Hidden:      true,
		Run:         func(env *Env, args []string) error { return ErrExit },
	})
}

// cmdHelp 帮助
func cmdHelp(env *Env, args []string) error {
	if len(args) > 0 {
		c, ok := Lookup(args[0])
		if !ok {
			return fmt.Errorf("未知命令: %s", args[0])
		}
		fmt.Fprintf(env.stdout(), "%s — %s\n", c.Name, c.Description)
		if c.Usage != "" {
			fmt.Fprintf(env.stdout(), "用法: %s\n", c.Usage)
		}
		if len(c.Aliases) > 0 {
			fmt.Fprintf(env.stdout(), "别名: %s\n", strings.Join(c.Aliases, ", "))
		}
		return nil
	}
	PrintHelp(env)
	return nil
}

// cmdVersion 版本
func cmdVersion(env *Env, args []string) error {
	fmt.Fprintf(env.stdout(), "MediaConv %s\n", env.Version)
	return nil
}

// cmdStart 启动 Web 服务
func cmdStart(env *Env, args []string) error {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	port := fs.String("port", "", "端口")
	host := fs.String("host", "127.0.0.1", "绑定地址（仅允许回环）")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *port == "" {
		*port = env.Port
		if *port == "" {
			*port = "8090"
		}
	}
	if err := env.ResolveFFmpeg(); err != nil {
		return err
	}
	// 安全：强制回环
	if !IsLoopback(*host) {
		return fmt.Errorf("安全策略：仅允许绑定回环地址（127.0.0.1），收到 %q", *host)
	}
	fmt.Fprintf(env.stdout(), "MediaConv Web 服务: http://%s:%s\n", *host, *port)
	return webapi.Serve(env.WebFS, *host, *port, env.FFmpegBin, env.FFprobeBin)
}

// cmdGUI 原生 GUI
func cmdGUI(env *Env, args []string) error {
	if env.GUI == nil {
		return fmt.Errorf("此构建未包含原生 GUI，请用 `go build -tags gui` 重新编译，或直接用 `start` 启动 Web 版")
	}
	return env.GUI()
}

// cmdConvert 一次性转码
func cmdConvert(env *Env, args []string) error {
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)
	in := fs.String("i", "", "输入文件")
	inL := fs.String("input", "", "输入文件（别名）")
	out := fs.String("o", "", "输出文件")
	outL := fs.String("output", "", "输出文件（别名）")
	fmtStr := fs.String("f", "", "输出格式")
	fmtL := fs.String("format", "", "输出格式（别名）")
	qual := fs.String("q", "", "音质")
	qualL := fs.String("quality", "", "音质（别名）")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *inL != "" {
		*in = *inL
	}
	if *outL != "" {
		*out = *outL
	}
	if *fmtL != "" {
		*fmtStr = *fmtL
	}
	if *qualL != "" {
		*qual = *qualL
	}

	// 没有 -i 时尝试用 TUI 选择
	if *in == "" {
		p, err := tui.PickFile("", nil)
		if err != nil {
			return fmt.Errorf("未指定 -i 且无法启动 TUI 选择器: %w", err)
		}
		*in = p
	}
	if _, err := os.Stat(*in); err != nil {
		return fmt.Errorf("输入文件不存在: %w", err)
	}
	if err := env.ResolveFFmpeg(); err != nil {
		return err
	}

	opts := convert.Options{
		InPath:     *in,
		OutFmt:     strings.ToLower(strings.TrimSpace(*fmtStr)),
		OutPath:    *out,
		Quality:    *qual,
		FFmpegBin:  env.FFmpegBin,
		FFprobeBin: env.FFprobeBin,
		OnProgress: makeProgress(env),
	}
	outPath, err := convert.Convert(context.Background(), opts)
	if err != nil {
		return err
	}
	fmt.Fprintf(env.stdout(), "完成: %s\n", outPath)
	return nil
}

// cmdUp TUI 上传式转码：选源→转码→选输出目录
func cmdUp(env *Env, args []string) error {
	fs := flag.NewFlagSet("up", flag.ContinueOnError)
	fmtStr := fs.String("f", "", "输出格式")
	qual := fs.String("q", "192k", "音质")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if err := env.ResolveFFmpeg(); err != nil {
		return err
	}

	// 1) TUI 选择源文件
	src, err := tui.PickFile("", nil)
	if err != nil {
		if errorsIsNoTTY(err) {
			return fmt.Errorf("当前无交互终端，无法使用 TUI 选择器；请改用: convert -i <文件> -f <格式>")
		}
		return err
	}
	fmt.Fprintf(env.stdout(), "已选择源文件: %s\n", src)

	// 2) 询问目标格式
	defFmt := defaultOutFmt(src)
	if *fmtStr == "" {
		*fmtStr = promptLine(env, "输出格式 (mp3/mp4/mkv/mov/webm/flac/wav/ogg/opus...)", defFmt)
	}
	if *fmtStr == "" {
		*fmtStr = defFmt
	}

	// 3) 询问音质
	if *qual == "" {
		*qual = promptLine(env, "音质 (192k/320k/original/720p/1080p...)", "192k")
	}

	// 4) 转码到临时目录
	tmp, err := os.MkdirTemp("", "mediaconv-up-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	outName := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src)) + "." + strings.ToLower(*fmtStr)
	tmpOut := filepath.Join(tmp, outName)

	opts := convert.Options{
		InPath:     src,
		OutFmt:     strings.ToLower(*fmtStr),
		OutPath:    tmpOut,
		Quality:    *qual,
		FFmpegBin:  env.FFmpegBin,
		FFprobeBin: env.FFprobeBin,
		OnProgress: makeProgress(env),
	}
	finalOut, err := convert.Convert(context.Background(), opts)
	if err != nil {
		return err
	}

	// 5) TUI 选择输出目录，并把成品放进去
	dir, err := tui.PickDir("")
	if err != nil {
		if errorsIsNoTTY(err) {
			// 退化：放到当前目录
			dest := filepath.Join(".", filepath.Base(finalOut))
			if err := copyFile(finalOut, dest); err != nil {
				return err
			}
			fmt.Fprintf(env.stdout(), "已保存到当前目录: %s\n", dest)
			return nil
		}
		return err
	}
	dest := filepath.Join(dir, filepath.Base(finalOut))
	if err := copyFile(finalOut, dest); err != nil {
		return err
	}
	fmt.Fprintf(env.stdout(), "转码完成，已保存到: %s\n", dest)
	return nil
}

// cmdBatch 批量转码
func cmdBatch(env *Env, args []string) error {
	fs := flag.NewFlagSet("batch", flag.ContinueOnError)
	dir := fs.String("d", "", "源目录")
	fmtStr := fs.String("f", "", "输出格式")
	qual := fs.String("q", "192k", "音质")
	recursive := fs.Bool("recursive", false, "递归子目录")
	outDir := fs.String("o", "", "输出目录（默认与源同级）")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" {
		return fmt.Errorf("必须指定 -d 源目录")
	}
	if *fmtStr == "" {
		return fmt.Errorf("必须指定 -f 输出格式")
	}
	if err := env.ResolveFFmpeg(); err != nil {
		return err
	}
	info, err := os.Stat(*dir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("源目录不存在: %s", *dir)
	}
	if *outDir == "" {
		*outDir = *dir
	}
	os.MkdirAll(*outDir, 0755)

	count := 0
	walkErr := filepath.Walk(*dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			if !*recursive && path != *dir {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
		if !preset.SupportsInput(ext) {
			return nil
		}
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		dest := filepath.Join(*outDir, base+"."+strings.ToLower(*fmtStr))
		opts := convert.Options{
			InPath:     path,
			OutFmt:     strings.ToLower(*fmtStr),
			OutPath:    dest,
			Quality:    *qual,
			FFmpegBin:  env.FFmpegBin,
			FFprobeBin: env.FFprobeBin,
		}
		fmt.Fprintf(env.stdout(), "[%d] %s -> %s\n", count+1, filepath.Base(path), filepath.Base(dest))
		if _, e := convert.Convert(context.Background(), opts); e != nil {
			fmt.Fprintf(env.stderr(), "  失败: %v\n", e)
			return nil
		}
		count++
		return nil
	})
	if walkErr != nil {
		return walkErr
	}
	fmt.Fprintf(env.stdout(), "批量转码完成，成功 %d 个文件。\n", count)
	return nil
}

// cmdList 列出格式
func cmdList(env *Env, args []string) error {
	list := preset.ListFormats()
	cur := ""
	for _, p := range list {
		if p.Category != cur {
			cur = p.Category
			fmt.Fprintf(env.stdout(), "\n== %s ==\n", p.Category)
		}
		qs := strings.Join(p.Qualities, "/")
		if qs == "" {
			qs = "-"
		}
		if p.InputOnly {
			fmt.Fprintf(env.stdout(), "  %-10s (仅输入) 输入: %v\n", p.OutFmt, p.InExts)
		} else {
			fmt.Fprintf(env.stdout(), "  %-10s 音质: %s  输入: %v\n", p.OutFmt, qs, p.InExts)
		}
	}
	return nil
}

// cmdInfo 媒体信息
func cmdInfo(env *Env, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("用法: info <文件>")
	}
	if err := env.ResolveFFmpeg(); err != nil {
		return err
	}
	info, err := ffwrap.Probe(env.FFprobeBin, args[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(env.stdout(), "文件: %s\n", args[0])
	fmt.Fprintf(env.stdout(), "容器: %s\n", info.Format)
	fmt.Fprintf(env.stdout(), "时长: %.2f 秒\n", info.Duration)
	if info.BitRate > 0 {
		fmt.Fprintf(env.stdout(), "码率: %d bps\n", info.BitRate)
	}
	for i, s := range info.Streams {
		fmt.Fprintf(env.stdout(), "流%d: %s 编码=%s", i, s.CodecType, s.CodecName)
		if s.CodecType == "audio" {
			fmt.Fprintf(env.stdout(), " 采样率=%s 声道=%d", s.SampleRate, s.Channels)
		} else if s.CodecType == "video" {
			fmt.Fprintf(env.stdout(), " 分辨率=%dx%d", s.Width, s.Height)
		}
		fmt.Fprintln(env.stdout())
	}
	return nil
}

// cmdDecrypt 还原私有格式
func cmdDecrypt(env *Env, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("用法: decrypt <文件>")
	}
	src := args[0]
	if !musicdecrypt.IsPrivateExt(src) {
		return fmt.Errorf("不是支持的私有格式: %s", src)
	}
	// 优先让用户用 TUI 选输出目录；无 TTY 则放当前目录
	outDir := "."
	if dir, err := tui.PickDir(""); err == nil {
		outDir = dir
	}
	decPath, outExt, err := musicdecrypt.Decrypt(src)
	if err != nil {
		return err
	}
	defer os.Remove(decPath)
	base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	dest := filepath.Join(outDir, base+outExt)
	if err := copyFile(decPath, dest); err != nil {
		return err
	}
	fmt.Fprintf(env.stdout(), "已还原: %s\n", dest)
	return nil
}

// cmdFFmpeg 依赖状态
func cmdFFmpeg(env *Env, args []string) error {
	f, errF := ffwrap.FindFFmpeg()
	p, errP := ffwrap.FindFFprobe()
	fmt.Fprintf(env.stdout(), "ffmpeg : ")
	if errF != nil {
		fmt.Fprintf(env.stdout(), "未找到\n")
	} else {
		fmt.Fprintf(env.stdout(), "%s\n", f)
	}
	fmt.Fprintf(env.stdout(), "ffprobe: ")
	if errP != nil {
		fmt.Fprintf(env.stdout(), "未找到\n")
	} else {
		fmt.Fprintf(env.stdout(), "%s\n", p)
	}
	if errF != nil {
		fmt.Fprintln(env.stdout(), ffwrap.InstallHint())
	}
	return nil
}

// cmdClean 清理临时目录
func cmdClean(env *Env, args []string) error {
	if env.WorkDir == "" {
		t, err := os.MkdirTemp("", "mediaconv-*")
		if err == nil {
			env.WorkDir = t
		}
	}
	if env.WorkDir == "" {
		return nil
	}
	entries, err := os.ReadDir(env.WorkDir)
	if err != nil {
		return err
	}
	n := 0
	for _, e := range entries {
		if os.RemoveAll(filepath.Join(env.WorkDir, e.Name())) == nil {
			n++
		}
	}
	fmt.Fprintf(env.stdout(), "已清理 %d 个临时文件。\n", n)
	return nil
}

// ───────────── 工具函数 ─────────────

func makeProgress(env *Env) convert.ProgressFn {
	var lastPct float64 = -1
	return func(sec, total float64) {
		if total <= 0 {
			return
		}
		pct := sec / total * 100
		if pct > 99.5 {
			pct = 99.5
		}
		if pct-lastPct >= 1 {
			fmt.Fprintf(env.stderr(), "进度: %.0f%%\n", pct)
			lastPct = pct
		}
	}
}

func defaultOutFmt(src string) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(src), "."))
	video := map[string]bool{
		"mp4": true, "mkv": true, "avi": true, "mov": true, "flv": true,
		"webm": true, "wmv": true, "mpg": true, "mpeg": true, "3gp": true,
		"ts": true, "mts": true, "vob": true, "m4v": true, "f4v": true, "ogv": true,
	}
	if video[ext] {
		return "mp4"
	}
	return "mp3"
}

func errorsIsNoTTY(err error) bool {
	return strings.Contains(err.Error(), "没有交互式终端") || strings.Contains(err.Error(), "ErrNoTTY")
}

func IsLoopback(host string) bool {
	h := strings.TrimSpace(host)
	return h == "127.0.0.1" || h == "::1" || h == "localhost"
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// Package main MediaConv 本地多媒体转码工具入口
// MediaConv local media transcoder entrypoint
package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nost3a/MediaConv/internal/ffwrap"
	"github.com/nost3a/MediaConv/internal/job"
	"github.com/nost3a/MediaConv/internal/musicdecrypt"
	"github.com/nost3a/MediaConv/internal/preset"
)

//go:embed web
var webFS embed.FS

var (
	mgr     *job.Manager
	workDir string
)

const version = "1.1.0"

func main() {
	// mconv 二进制默认进 REPL
	// mconv binary defaults to REPL
	if defaultCmd == "repl" && len(os.Args) < 2 {
		runREPL()
		return
	}
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}
	switch os.Args[1] {
	case "repl":
		runREPL()
	case "convert":
		runConvert(os.Args[2:])
	case "start":
		runStart()
	case "list-formats":
		runListFormats()
	case "version", "-v", "--version":
		fmt.Printf("MediaConv %s\n", version)
	case "-h", "--help", "help":
		printHelp()
	default:
		arg := os.Args[1]
		if strings.HasPrefix(arg, "-") {
			fmt.Fprintf(os.Stderr, "未知选项: %s\n", arg)
			printHelp()
			os.Exit(2)
		}
		args := append([]string{"-i", arg}, os.Args[2:]...)
		runConvert(args)
	}
}

// printHelp 打印帮助
// Print usage help
func printHelp() {
	fmt.Print(`MediaConv - 本地音视频转码 / Local media transcoder

用法 / Usage:
  mediaconv input.mp4                  直接转码，自动选输出格式 / Convert with auto output
  mediaconv input.mp4 -o out.mp3      指定输出 / Specify output
  mediaconv start                      启动 GUI 或本地 Web 服务 / Start GUI or web UI
  mediaconv list-formats               列出支持格式 / List formats
  mediaconv -h                         显示本帮助 / Show this help

参数 / Flags:
  -o, --output    输出文件路径（后缀决定格式） / Output path (ext decides format)
  -f, --format    输出格式（可选，默认按输入自动判断） / Output format
  -q, --quality   音质档位 (默认 192k) / Quality (default 192k)

默认输出 / Default output:
  视频输入 -> .mp4    / video input -> .mp4
  音频输入 -> .mp3    / audio input -> .mp3
  私有格式 -> .mp3    / private containers -> .mp3

示例 / Examples:
  mediaconv input.mp4
  mediaconv input.wav
  mediaconv song.ncm
  mediaconv input.mkv -o out.mp4
  mediaconv input.mp4 -q 320k -o out.mp3
`)
}

// runListFormats 列出全部预设
// List all format presets
func runListFormats() {
	list := preset.ListFormats()
	for _, p := range list {
		fmt.Printf("%s -> %s  (输入: %v)\n", p.Desc, p.OutFmt, p.InExts)
	}
}

// runStart 启动 GUI 或 headless 服务
// Start GUI or headless server
func runStart() {
	port := os.Getenv("MEDIACONV_PORT")
	if port == "" {
		port = "8090"
	}
	headless := os.Getenv("MEDIACONV_HEADLESS") == "1"
	for _, a := range os.Args[2:] {
		if a == "--no-webview" || a == "--headless" {
			headless = true
		}
	}
	if headless {
		runHeadless(port)
		return
	}
	if _, err := ffwrap.FindFFmpeg(); err != nil {
		log.Printf("警告: %s", ffwrap.InstallHint())
	}
	startGUI()
}

// runConvert 命令行直接转码（不起 HTTP）
// CLI direct transcode, no HTTP server
func runConvert(argv []string) {
	fs := flag.NewFlagSet("convert", flag.ExitOnError)
	in := fs.String("i", "", "输入文件 / input")
	out := fs.String("o", "", "输出文件 / output")
	fmtStr := fs.String("f", "", "输出格式 / format")
	qual := fs.String("q", "192k", "音质 / quality")
	fs.StringVar(in, "input", "", "")
	fs.StringVar(out, "output", "", "")
	fs.StringVar(fmtStr, "format", "", "")
	fs.StringVar(qual, "quality", "", "")
	_ = fs.Parse(argv)

	if *in == "" {
		fmt.Fprintln(os.Stderr, "错误: 必须指定 -i 输入文件")
		os.Exit(2)
	}
	if _, err := os.Stat(*in); err != nil {
		fmt.Fprintf(os.Stderr, "错误: 输入文件不存在: %v\n", err)
		os.Exit(2)
	}

	ffBin, err := ffwrap.FindFFmpeg()
	if err != nil {
		fmt.Fprintln(os.Stderr, ffwrap.InstallHint())
		os.Exit(1)
	}
	probeBin, err := ffwrap.FindFFprobe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "未检测到 ffprobe，请一并安装")
		os.Exit(1)
	}

	// 私有格式先还原
	realIn := *in
	if musicdecrypt.IsPrivateExt(*in) {
		fmt.Fprintf(os.Stderr, "还原私有格式 %s ...\n", filepath.Base(*in))
		dec, _, err := musicdecrypt.Decrypt(*in)
		if err != nil {
			fmt.Fprintf(os.Stderr, "还原失败: %v\n", err)
			os.Exit(1)
		}
		realIn = dec
	}

	// 推断输出格式
	outFmt := *fmtStr
	if outFmt == "" {
		if *out != "" {
			outFmt = strings.TrimPrefix(filepath.Ext(*out), ".")
		}
	}
	if outFmt == "" {
		// 按输入类型默认选：视频->mp4, 音频/私有->mp3
		// Default: video->mp4, audio/private->mp3
		inL := strings.ToLower(strings.TrimPrefix(filepath.Ext(realIn), "."))
		videoExts := map[string]bool{
			"mp4": true, "mkv": true, "avi": true, "mov": true, "flv": true,
			"webm": true, "wmv": true, "mpg": true, "mpeg": true, "3gp": true,
			"ts": true, "mts": true, "vob": true, "m4v": true, "f4v": true, "ogv": true,
		}
		if videoExts[inL] {
			outFmt = "mp4"
		} else {
			outFmt = "mp3"
		}
	}

	inExt := strings.TrimPrefix(filepath.Ext(realIn), ".")
	p, ok := preset.Find(inExt, outFmt, *qual)
	if !ok {
		fmt.Fprintf(os.Stderr, "错误: 不支持 %s -> %s\n", inExt, outFmt)
		os.Exit(1)
	}

	// 推断输出路径：默认放当前工作目录，用原始输入的基名（不是解密后的基名）
	// Default output to CWD, use original input base name
	outPath := *out
	if outPath == "" {
		base := strings.TrimSuffix(filepath.Base(*in), filepath.Ext(*in))
		outPath = base + "." + p.OutFmt
	}

	// 私有格式还原后如果格式已经匹配目标，直接复制，不再走 ffmpeg 重编码
	// If decrypt already produced the target format, copy bytes directly
	if musicdecrypt.IsPrivateExt(*in) {
		decExt := strings.TrimPrefix(filepath.Ext(realIn), ".")
		if decExt == p.OutFmt {
			data, err := os.ReadFile(realIn)
			if err != nil {
				fmt.Fprintf(os.Stderr, "读取还原文件失败: %v\n", err)
				os.Exit(1)
			}
			if err := os.WriteFile(outPath, data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "写入失败: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "完成（直接还原，未重编码）: %s\n", outPath)
			return
		}
	}

	args := p.BuildArgs(*qual, outPath)
	args = append([]string{"-y", "-i", realIn}, args...)

	dur := ffwrap.ProbeDuration(probeBin, realIn)
	fmt.Fprintf(os.Stderr, "转码中: %s -> %s\n", filepath.Base(realIn), filepath.Base(outPath))

	ctx := context.Background()
	lastPct := -1.0
	err = ffwrap.Run(ctx, ffBin, args, func(sec float64) {
		if dur > 0 {
			pct := sec / dur * 100
			if pct > 99.5 {
				pct = 99.5
			}
			if pct-lastPct >= 1 {
				fmt.Fprintf(os.Stderr, "进度: %.0f%%\n", pct)
				lastPct = pct
			}
		}
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "完成: %s\n", outPath)
}

// runHeadless 起 HTTP 服务（Termux/服务器）
// Start HTTP headless server
func runHeadless(port string) {
	ffBin, err := ffwrap.FindFFmpeg()
	if err != nil {
		fmt.Fprintln(os.Stderr, ffwrap.InstallHint())
		os.Exit(1)
	}
	probeBin, err := ffwrap.FindFFprobe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "未检测到 ffprobe，请一并安装。")
		os.Exit(1)
	}
	log.Printf("使用 ffmpeg: %s", ffBin)
	log.Printf("使用 ffprobe: %s", probeBin)

	workDir, err = os.MkdirTemp("", "mediaconv-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(workDir)
	log.Printf("工作目录: %s", workDir)
	mgr = job.NewManager(workDir, ffBin, probeBin)

	go func() {
		t := time.NewTicker(1 * time.Minute)
		for range t.C {
			mgr.CleanupDone()
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/upload", handleUpload)
	mux.HandleFunc("/api/convert", handleConvert)
	mux.HandleFunc("/api/progress/", handleProgress)
	mux.HandleFunc("/api/download/", handleDownload)
	mux.HandleFunc("/api/formats", handleFormats)

	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))

	addr := mustLoopback("127.0.0.1") + ":" + port
	log.Printf("MediaConv headless: http://%s", addr)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("监听 %s 失败: %v", addr, err)
	}
	log.Fatal(http.Serve(ln, mux))
}

// mustLoopback 强制只允许绑回环地址
// Force loopback-only binding
func mustLoopback(host string) string {
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		log.Fatalf("安全策略：本工具仅允许本机使用，禁止绑定非回环地址。(收到 host=%q)", host)
	}
	return host
}

// handleUpload POST /api/upload
func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	if err := r.ParseMultipartForm(500 << 20); err != nil {
		http.Error(w, "解析表单失败: "+err.Error(), 400)
		return
	}
	f, hdr, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "缺少 file 字段", 400)
		return
	}
	defer f.Close()

	id := fmt.Sprintf("up_%d", time.Now().UnixNano())
	safeName := filepath.Base(hdr.Filename)
	inPath := filepath.Join(workDir, id+"_"+safeName)
	out, err := os.Create(inPath)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, f); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, map[string]string{
		"file_id": inPath,
		"name":    safeName,
		"size":    fmt.Sprintf("%d", hdr.Size),
	})
}

type convertReq struct {
	FileID       string `json:"file_id"`
	TargetFormat string `json:"target_format"`
	Quality      string `json:"quality"`
}

// handleConvert POST /api/convert
func handleConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	var req convertReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON 解析失败: "+err.Error(), 400)
		return
	}
	if req.FileID == "" || req.TargetFormat == "" {
		http.Error(w, "file_id / target_format 必填", 400)
		return
	}
	if _, err := os.Stat(req.FileID); err != nil {
		http.Error(w, "输入文件不存在", 400)
		return
	}
	jobID, err := mgr.Submit(req.FileID, req.TargetFormat, req.Quality)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	writeJSON(w, map[string]string{"job_id": jobID})
}

// handleProgress GET /api/progress/:id
func handleProgress(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/progress/")
	j, ok := mgr.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, j)
}

// handleDownload GET /api/download/:id
func handleDownload(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/download/")
	j, ok := mgr.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if j.Status != job.StatusDone {
		http.Error(w, "任务未完成", 409)
		return
	}
	data, err := os.ReadFile(j.OutPath)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	ext := filepath.Ext(j.OutName)
	ct := mime.TypeByExtension(ext)
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="convert%s"`, ext))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Write(data)
}

// handleFormats GET /api/formats
func handleFormats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, preset.ListFormats())
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

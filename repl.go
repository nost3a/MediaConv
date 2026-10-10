package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nost3a/MediaConv/internal/ffwrap"
	"github.com/nost3a/MediaConv/internal/musicdecrypt"
	"github.com/nost3a/MediaConv/internal/preset"
)

// 默认启动命令（由 ldflags 注入，mconv 二进制默认 repl）
// Default command injected via ldflags; mconv defaults to repl
var defaultCmd = ""

var replInPath string

// runREPL 交互式命令行
// Interactive REPL
func runREPL() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("MediaConv  v1.0.0")
	fmt.Println("本地音视频转换工具 / Local media transcoder")
	fmt.Println()
	fmt.Println("输入 help 查看命令 / Type help for commands.")
	fmt.Println()

	for {
		fmt.Print("mconv> ")
		if !scanner.Scan() {
			return
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		switch parts[0] {
		case "help":
			printReplHelp()
		case "exit", "quit":
			fmt.Println("再见 / Bye.")
			return
		case "up":
			if len(parts) < 2 {
				fmt.Println("用法: up <文件路径> / Usage: up <path>")
				continue
			}
			handleUp(strings.Join(parts[1:], " "), scanner)
		case "dl":
			fmt.Println("临时文件已清理 / Temp files cleaned.")
		default:
			fmt.Printf("未知命令: %s / Unknown command: %s\n", parts[0], parts[0])
		}
	}
}

func printReplHelp() {
	fmt.Println("MediaConv v1.0.0 - 本地音视频转换工具")
	fmt.Println("MediaConv v1.0.0 - Local media transcoder")
	fmt.Println()
	fmt.Println("命令 / Commands:")
	fmt.Println("  up <文件>      加载文件并列出格式 / Load file and list formats")
	fmt.Println("  dl             清理临时文件 / Clean temp files")
	fmt.Println("  help           显示本帮助 / Show this help")
	fmt.Println("  exit           退出 / Quit")
	fmt.Println()
	fmt.Println("示例 / Examples:")
	fmt.Println("  mconv> up song.mp4")
	fmt.Println("  mconv> up /sdcard/Music/song.ncm")
	fmt.Println("  mconv> up clip.mkv")
}

// handleUp 加载文件并列出格式菜单
// Load file and show format menu
func handleUp(path string, scanner *bufio.Scanner) {
	st, err := os.Stat(path)
	if err != nil {
		fmt.Printf("文件不存在 / File not found: %s\n", path)
		return
	}
	if st.IsDir() {
		fmt.Printf("这是目录不是文件 / Is a directory: %s\n", path)
		return
	}
	replInPath = path
	fmt.Printf("已加载 %s\n", path)
	fmt.Printf("Loaded %s\n", path)

	// 列出可选输出格式
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	if musicdecrypt.IsPrivateExt(path) {
		ext = "mp3" // 私有格式先当音频处理
	}
	isVideo := false
	for _, v := range []string{"mp4", "mkv", "avi", "mov", "flv", "webm", "wmv", "mpg", "mpeg", "3gp", "ts", "mts", "vob", "m4v", "f4v", "ogv"} {
		if v == ext {
			isVideo = true
			break
		}
	}

	// 收集可用输出格式
	var outs []string
	if isVideo {
		outs = []string{"mp4", "mkv", "avi", "mov", "flv", "webm", "wmv", "mpeg", "mpg", "3gp", "ts", "m4v", "ogv", "gif"}
	} else {
		outs = []string{"mp3", "flac", "wav", "aac", "ogg", "m4a", "opus", "wma", "aiff", "ac3"}
	}

	fmt.Println()
	fmt.Println("可转换格式 / Available formats:")
	for i, f := range outs {
		fmt.Printf("  \"%d\"  %s\n", i+1, f)
	}
	fmt.Println()
	fmt.Print("请输入编号 / Enter number: ")
	if !scanner.Scan() {
		return
	}
	var n int
	_, err = fmt.Sscanf(strings.TrimSpace(scanner.Text()), "%d", &n)
	if err != nil || n < 1 || n > len(outs) {
		fmt.Println("编号无效 / Invalid number.")
		return
	}
	target := outs[n-1]
	fmt.Printf("转换为 %s / Converting to %s\n", target, target)

	// 保存路径
	outPath := askSavePath(target, scanner)
	if outPath == "" {
		fmt.Println("已取消 / Cancelled.")
		return
	}
	// 如果用户在保存路径里指定了别的后缀，以路径后缀为准
	// If save path has different ext, prefer it
	if extFromPath := strings.TrimPrefix(filepath.Ext(outPath), "."); extFromPath != "" {
		target = extFromPath
	}

	err = doConvertInteractive(path, target, outPath)
	if err != nil {
		fmt.Printf("失败 / Failed: %v\n", err)
		return
	}
	fmt.Println("转换成功 / Conversion success!")
	fmt.Printf("已保存到 %s\nSaved to %s\n", outPath, outPath)
}

// askSavePath 弹保存对话框或手动输入
// Show save dialog or ask manually
func askSavePath(ext string, scanner *bufio.Scanner) string {
	// 先试系统对话框
	if p := trySaveDialog(ext); p != "" {
		return p
	}
	fmt.Println("请输入保存路径 / Enter save path: ")
	fmt.Print("> ")
	if !scanner.Scan() {
		return ""
	}
	return strings.TrimSpace(scanner.Text())
}

// trySaveDialog 调系统保存对话框，失败返回空串
// Try native save dialog, return "" on failure
func trySaveDialog(ext string) string {
	candidates := [][]string{
		{"zenity", "--file-selection", "--save", "--confirm-overwrite", "--filename=output." + ext},
		{"kdialog", "--getsavefilename", "output." + ext},
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		cmd := exec.Command(c[0], c[1:]...)
		out, err := cmd.Output()
		if err == nil {
			p := strings.TrimSpace(string(out))
			if p != "" {
				return p
			}
		}
	}
	return ""
}

// doConvertInteractive 跑一次转码，带进度打印
// Run one transcode with progress
func doConvertInteractive(inPath, outFmt, outPath string) error {
	realIn := inPath
	if musicdecrypt.IsPrivateExt(inPath) {
		dec, _, err := musicdecrypt.Decrypt(inPath)
		if err != nil {
			return err
		}
		realIn = dec
	}
	inExt := strings.TrimPrefix(filepath.Ext(realIn), ".")
	p, ok := preset.Find(inExt, outFmt, "192k")
	if !ok {
		return fmt.Errorf("不支持 %s -> %s", inExt, outFmt)
	}
	args := p.BuildArgs("192k", outPath)
	args = append([]string{"-y", "-i", realIn}, args...)

	ffBin, err := ffwrap.FindFFmpeg()
	if err != nil {
		return err
	}
	probeBin, err := ffwrap.FindFFprobe()
	if err != nil {
		return err
	}
	dur := ffwrap.ProbeDuration(probeBin, realIn)
	last := -1.0
	return ffwrap.Run(context.Background(), ffBin, args, func(sec float64) {
		if dur > 0 {
			pct := sec / dur * 100
			if pct > 99 {
				pct = 99
			}
			if pct-last >= 5 {
				fmt.Printf("转码中 %.0f%% / Converting %.0f%%\n", pct, pct)
				last = pct
			}
		}
	})
}

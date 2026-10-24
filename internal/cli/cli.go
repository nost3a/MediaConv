// Package cli 共享命令行指令库。
//
// Linux / Windows 两个 CLI 构建产物共用同一套命令注册表，保证行为一致。
// 每条命令实现一次，由 main 在启动时注入运行环境（Env）后统一分发。
package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
	"strings"

	"github.com/nost3a/MediaConv/internal/ffwrap"
)

// ErrExit 由 exit/quit 命令返回，REPL 据此退出交互循环
var ErrExit = errors.New("exit")

// Env 运行时环境，由 main 在启动时填充并注入
type Env struct {
	Version    string
	FFmpegBin  string
	FFprobeBin string
	WorkDir    string       // 临时工作目录（上传/任务产物）
	OutDir     string       // 默认输出目录（空则当前目录）
	Port       string       // web 服务端口
	WebFS      fs.FS        // 前端静态资源（main 通过 //go:embed 注入）
	GUI        func() error // 原生 GUI 启动函数（main 注入；可能为空）
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
}

func (e *Env) stdin() io.Reader {
	if e.Stdin != nil {
		return e.Stdin
	}
	return os.Stdin
}
func (e *Env) stdout() io.Writer {
	if e.Stdout != nil {
		return e.Stdout
	}
	return os.Stdout
}
func (e *Env) stderr() io.Writer {
	if e.Stderr != nil {
		return e.Stderr
	}
	return os.Stderr
}

// Command 一条命令定义
type Command struct {
	Name        string
	Aliases     []string
	Usage       string
	Description string
	Run         func(env *Env, args []string) error
	Hidden      bool
}

var registry = map[string]*Command{}

// Register 注册命令（名字 + 别名都映射到同一实例）
func Register(c Command) {
	registry[c.Name] = &c
	for _, a := range c.Aliases {
		registry[a] = &c
	}
}

// Lookup 按名字或别名查命令
func Lookup(name string) (*Command, bool) {
	c, ok := registry[name]
	return c, ok
}

// Sorted 返回按名字排序的所有非隐藏命令（用于帮助）
func Sorted() []*Command {
	seen := map[string]bool{}
	out := make([]*Command, 0, len(registry))
	for _, c := range registry {
		if c.Hidden || seen[c.Name] {
			continue
		}
		seen[c.Name] = true
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ResolveFFmpeg 惰性解析 ffmpeg/ffprobe 并缓存到 Env；解析失败返回安装提示
func (e *Env) ResolveFFmpeg() error {
	if e.FFmpegBin != "" && e.FFprobeBin != "" {
		return nil
	}
	f, err := ffwrap.FindFFmpeg()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, ffwrap.InstallHint())
	}
	p, err := ffwrap.FindFFprobe()
	if err != nil {
		return fmt.Errorf("未检测到 ffprobe: %w", err)
	}
	e.FFmpegBin, e.FFprobeBin = f, p
	return nil
}

// Run 解析首个 token 为命令并分发；无参数打印帮助
func Run(env *Env, args []string) error {
	if len(args) == 0 {
		PrintHelp(env)
		return nil
	}
	name := args[0]
	if name == "-h" || name == "--help" {
		PrintHelp(env)
		return nil
	}
	if name == "-v" || name == "--version" {
		fmt.Fprintf(env.stdout(), "MediaConv %s\n", env.Version)
		return nil
	}
	cmd, ok := Lookup(name)
	if !ok {
		fmt.Fprintf(env.stderr(), "未知命令: %s\n输入 \"help\" 查看可用命令\n", name)
		return fmt.Errorf("unknown command: %s", name)
	}
	return cmd.Run(env, args[1:])
}

// REPL 进入交互式命令行循环（mconv 二进制或 `repl` 命令使用）
func REPL(env *Env) error {
	fmt.Fprintf(env.stdout(), "MediaConv %s 交互模式（输入 help 查看命令，exit 退出）\n", env.Version)
	reader := bufio.NewReader(env.stdin())
	for {
		fmt.Fprintf(env.stdout(), "mediaconv> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Fprintln(env.stdout())
				return nil
			}
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		toks := strings.Fields(line)
		if toks[0] == "exit" || toks[0] == "quit" {
			return nil
		}
		if err := Run(env, toks); err != nil {
			if errors.Is(err, ErrExit) {
				return nil
			}
			fmt.Fprintf(env.stderr(), "错误: %v\n", err)
		}
	}
}

// PrintHelp 打印总帮助
func PrintHelp(env *Env) {
	fmt.Fprint(env.stdout(), "MediaConv - 本地音视频转码 / Local media transcoder\n\n")
	fmt.Fprint(env.stdout(), "可用命令 / Commands:\n")
	for _, c := range Sorted() {
		line := fmt.Sprintf("  %-10s %s\n", c.Name, c.Description)
		fmt.Fprint(env.stdout(), line)
		if c.Usage != "" {
			fmt.Fprintf(env.stdout(), "      用法: %s\n", c.Usage)
		}
		for _, a := range c.Aliases {
			fmt.Fprintf(env.stdout(), "      别名: %s\n", a)
		}
	}
	fmt.Fprintf(env.stdout(), "\n输入 \"help <命令>\" 查看某条命令的详细说明。\n")
}

// promptLine 在 stderr 打印提示并从 stdin 读一行；空输入返回默认值
func promptLine(env *Env, prompt, def string) string {
	if def != "" {
		fmt.Fprintf(env.stderr(), "%s[%s]: ", prompt, def)
	} else {
		fmt.Fprintf(env.stderr(), "%s: ", prompt)
	}
	reader := bufio.NewReader(env.stdin())
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// init 注册全部命令（保证 import 即生效）
func init() {
	registerAll()
}

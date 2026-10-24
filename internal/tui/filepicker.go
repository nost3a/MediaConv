// Package tui 提供纯终端的文件/目录选择器（基于 bubbletea）。
//
// 它不依赖任何桌面环境或外部程序，专为无图形界面的服务器/SSH/Windows
// 控制台场景设计：用户用方向键浏览目录、回车选中，体验等同于一个轻量
// 的 TUI 文件管理器（如 lf/ranger），但全部打包进二进制、零依赖。
package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// 错误：当前没有可用终端（非 TTY，例如管道/重定向场景）
var ErrNoTTY = errors.New("当前没有交互式终端，无法启动 TUI 文件管理器；请直接提供文件路径")

// Mode 选择模式
type Mode int

const (
	// FileMode 选择单个文件
	FileMode Mode = iota
	// DirMode 选择单个目录
	DirMode
)

var (
	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	stylePath  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleDir   = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	styleSel   = lipgloss.NewStyle().Background(lipgloss.Color("63")).Foreground(lipgloss.Color("0"))
	styleDim   = lipgloss.NewStyle().Faint(true)
	styleHelp  = lipgloss.NewStyle().Faint(true)
)

type item struct {
	name  string
	isDir bool
}

// pickerModel 是 bubbletea 的状态机
type pickerModel struct {
	mode      Mode
	allowed   map[string]bool // 允许的扩展名（小写，含点）；空表示不限制
	dir       string
	entries   []item
	cursor    int
	quitting  bool
	cancelled bool
	err       error
}

type errMsg struct{ err error }

func (m pickerModel) Init() tea.Cmd { return nil }

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.cancelled = true
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		case "home":
			home, _ := os.UserHomeDir()
			m.chdir(home)
		case "backspace", "esc":
			parent := filepath.Dir(m.dir)
			if parent != m.dir {
				m.chdir(parent)
			} else {
				// 已在根，回车取消
				m.cancelled = true
				m.quitting = true
				return m, tea.Quit
			}
		case "enter":
			if len(m.entries) == 0 {
				return m, nil
			}
			sel := m.entries[m.cursor]
			full := filepath.Join(m.dir, sel.name)
			if sel.isDir {
				if m.mode == DirMode {
					m.quitting = true
					return m, tea.Quit // 选中目录
				}
				m.chdir(full)
				return m, nil
			}
			// 文件
			if m.mode == DirMode {
				return m, nil // 目录模式不接收文件
			}
			if len(m.allowed) > 0 && !m.allowed[strings.ToLower(filepath.Ext(sel.name))] {
				return m, nil // 非目标类型，忽略
			}
			m.dir = full // 用选中文件作为结果
			m.quitting = true
			return m, tea.Quit
		}
	case errMsg:
		m.err = msg.err
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *pickerModel) chdir(dir string) {
	m.dir = dir
	m.cursor = 0
	m.entries = m.load()
}

func (m *pickerModel) load() []item {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		// 目录读不了就回退父目录
		parent := filepath.Dir(m.dir)
		if parent != m.dir {
			m.dir = parent
			entries, _ = os.ReadDir(m.dir)
		}
	}
	items := make([]item, 0, len(entries)+1)
	// 始终提供“.. 返回上级”
	items = append(items, item{name: "..", isDir: true})
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			items = append(items, item{name: name, isDir: true})
			continue
		}
		// 文件：有扩展名过滤时跳过不匹配的
		if len(m.allowed) > 0 && !m.allowed[strings.ToLower(filepath.Ext(name))] {
			continue
		}
		items = append(items, item{name: name, isDir: false})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].isDir != items[j].isDir {
			return items[i].isDir // 目录优先
		}
		return strings.ToLower(items[i].name) < strings.ToLower(items[j].name)
	})
	return items
}

func (m pickerModel) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("MediaConv 文件选择器"))
	b.WriteString("  ")
	b.WriteString(stylePath.Render(m.dir))
	b.WriteString("\n\n")

	for i, it := range m.entries {
		line := it.name
		if it.isDir {
			line = styleDir.Render(it.name + "/")
		}
		if i == m.cursor {
			b.WriteString(styleSel.Render("> " + line))
		} else {
			b.WriteString("  " + line)
		}
		b.WriteString("\n")
	}

	hint := "[↑↓]移动 [Enter]"
	if m.mode == DirMode {
		hint += "选中目录"
	} else {
		hint += "选中文件"
	}
	hint += " [Backspace/Esc]上级 [Home]主目录 [q]取消"
	b.WriteString("\n" + styleHelp.Render(hint))
	return b.String()
}

// filteredExt 把扩展名切片规整为 map（小写、含点）
func filteredExt(exts []string) map[string]bool {
	if len(exts) == 0 {
		return nil
	}
	m := make(map[string]bool, len(exts))
	for _, e := range exts {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		m[e] = true
	}
	return m
}

// runPicker 公共运行入口
func runPicker(mode Mode, startDir string, allowed []string) (string, error) {
	if !isTTY() {
		return "", ErrNoTTY
	}
	if startDir == "" {
		d, _ := os.Getwd()
		startDir = d
	}
	startDir = filepath.Clean(startDir)
	m := pickerModel{
		mode:    mode,
		allowed: filteredExt(allowed),
		dir:     startDir,
	}
	m.entries = m.load()

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	res, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("TUI 文件选择器运行失败: %w", err)
	}
	final := res.(pickerModel)
	if final.err != nil {
		return "", final.err
	}
	if final.cancelled {
		return "", fmt.Errorf("已取消选择")
	}
	// 选中结果是 m.dir 的最终值（文件为完整路径，目录为目录路径）
	if final.dir == startDir && len(final.entries) > 0 {
		// 没移动过也可能直接选中当前项，这里安全返回
	}
	out := final.dir
	if mode == FileMode && !filepath.IsAbs(out) {
		out, _ = filepath.Abs(out)
	}
	return out, nil
}

// PickFile 启动 TUI 选择单个文件；allowed 为空表示接受任意文件，
// 否则只列出/接受这些扩展名（如 []string{"mp4","mp3"}）。
func PickFile(startDir string, allowed []string) (string, error) {
	return runPicker(FileMode, startDir, allowed)
}

// PickDir 启动 TUI 选择单个目录。
func PickDir(startDir string) (string, error) {
	return runPicker(DirMode, startDir, nil)
}

// isTTY 判断 stdin 是否连到真实终端（bubbletea 从 stdin 读键盘）
func isTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

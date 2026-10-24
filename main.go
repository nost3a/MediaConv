// Package main MediaConv 本地多媒体转码工具入口
// MediaConv local media transcoder entrypoint
//
// 这里的 main 只是个瘦入口：它把运行环境（版本、静态资源、GUI 启动函数）
// 注入到共享命令库 internal/cli，然后交给 cli.Run 统一分发。
// Linux / Windows 两个 CLI 产物共用同一套命令，保证行为一致。
package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/nost3a/MediaConv/internal/cli"
)

//go:embed web
var webFS embed.FS

// version 由构建时 -ldflags -X main.version=... 注入；默认与代码一致
var version = "1.2.0"

func main() {
	env := &cli.Env{
		Version: version,
		WebFS:   webFS,
		Port:    envPort(),
		// startGUI 由 gui_stub / gui_fyne / gui_windows 之一提供（按构建标签）
		GUI: func() error {
			startGUI()
			return nil
		},
	}
	if err := cli.Run(env, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

func envPort() string {
	if p := os.Getenv("MEDIACONV_PORT"); p != "" {
		return p
	}
	return "8090"
}

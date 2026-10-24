//go:build !windows && (!linux || !cgo)

package main

import (
	"log"
)

// startGUI 无原生窗口支持的构建（Termux / 服务器 / 非 cgo）下，
// 直接提示用户改用 Web 服务，而不是阻塞进程。
func startGUI() {
	log.Printf("当前构建不含原生 GUI（无 cgo/非桌面平台）。请使用 `mediaconv start` 启动本地 Web 服务。")
}

//go:build !windows && (!linux || !cgo)

package main

import (
	"log"
)

func startGUI() {
	log.Printf("headless 构建，无原生窗口")
	select {}
}

package main

import (
	"testing"

	"github.com/nost3a/MediaConv/internal/cli"
)

// TestIsLoopbackAccepts 验证合法的回环地址被接受
func TestIsLoopbackAccepts(t *testing.T) {
	for _, h := range []string{"127.0.0.1", "::1", "localhost", " 127.0.0.1 "} {
		if !cli.IsLoopback(h) {
			t.Fatalf("应判定为回环地址: %q", h)
		}
	}
}

// TestIsLoopbackRejects 验证非回环地址被拒绝（安全策略：Web 仅绑回环）
func TestIsLoopbackRejects(t *testing.T) {
	for _, h := range []string{"0.0.0.0", "192.168.1.1", "10.0.0.1", "example.com"} {
		if cli.IsLoopback(h) {
			t.Fatalf("不应判定为回环地址: %q", h)
		}
	}
}

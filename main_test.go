package main

import (
	"os"
	"os/exec"
	"testing"
)

// 验证 mustLoopback 对非回环地址会 os.Exit(1)
// 用子进程方式跑：自己再调一次 binary，传 -test.main-exit 标志
func TestMustLoopbackRejects(t *testing.T) {
	if os.Getenv("MC_SELF_TEST") == "1" {
		// 子进程里直接调 mustLoopback，应该 fatal
		mustLoopback("0.0.0.0")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestMustLoopbackRejects")
	cmd.Env = append(os.Environ(), "MC_SELF_TEST=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("预期 mustLoopback(0.0.0.0) 会退出，但子进程正常返回了")
	}
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() == 0 {
		t.Fatalf("预期非零退出码，实际 err=%v", err)
	}
	t.Log("mustLoopback(0.0.0.0) 正确触发 fatal 退出")
}

func TestMustLoopbackAccepts(t *testing.T) {
	if got := mustLoopback("127.0.0.1"); got != "127.0.0.1" {
		t.Fatalf("loopback 应被接受，got=%s", got)
	}
	if got := mustLoopback("::1"); got != "::1" {
		t.Fatalf("::1 应被接受，got=%s", got)
	}
}

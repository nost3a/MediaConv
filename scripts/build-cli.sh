#!/usr/bin/env bash
# build-cli.sh: 编译 Linux 纯 CLI 版（无桌面依赖，CGO 关闭）
#
# 这是默认分发形态：单文件、无图形依赖，内置 TUI 文件选择器
# （bubbletea），`start` 命令可启动本地 Web 服务。
# Linux / Windows 两个 CLI 产物共用 internal/cli 同一套命令注册表。
#
# 依赖: 系统 PATH 里的 ffmpeg / ffprobe（或 scripts/fetch-... 自备）。
set -e
cd "$(dirname "$0")/.."

VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
LDFLAGS="-s -w -X main.version=${VERSION}"

echo "==> 编译 linux/amd64 CLI 版 (version=${VERSION})"
mkdir -p bin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath \
    -ldflags "${LDFLAGS}" \
    -o bin/mediaconv-linux-amd64 .

echo "==> 完成: bin/mediaconv-linux-amd64"
./bin/mediaconv-linux-amd64 version

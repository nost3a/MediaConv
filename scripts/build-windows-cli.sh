#!/usr/bin/env bash
# build-windows-cli.sh: 交叉编译 Windows 纯 CLI 版 exe（无桌面依赖，CGO 关闭）
#
# 与 Linux CLI 共用 internal/cli 同一套命令；Windows 版会把 ffmpeg.exe
# 内嵌进二进制（internal/ffembed），开箱即用、无需另行安装 ffmpeg。
#
# 注意: Windows 构建需要 internal/ffembed/ffmpeg.exe（162M，不进仓库）。
#       若缺失，本脚本会自动调用 scripts/fetch-ffmpeg.sh 下载一次。
set -e
cd "$(dirname "$0")/.."

VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
LDFLAGS="-s -w -X main.version=${VERSION}"

# 确保内嵌 ffmpeg.exe 就位（Windows 构建的 //go:embed 必需）
if [ ! -s internal/ffembed/ffmpeg.exe ]; then
    echo "[*] 未找到内嵌 ffmpeg.exe，尝试下载..."
    bash scripts/fetch-ffmpeg.sh
fi

echo "==> 交叉编译 windows/amd64 CLI 版 (version=${VERSION})"
mkdir -p bin
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
    go build -trimpath \
    -ldflags "${LDFLAGS}" \
    -o bin/mediaconv-windows-amd64.exe .

echo "==> 完成: bin/mediaconv-windows-amd64.exe"
file bin/mediaconv-windows-amd64.exe

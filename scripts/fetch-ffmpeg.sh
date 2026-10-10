#!/usr/bin/env bash
# 下载 Windows 版编译所需的 ffmpeg.exe（被 //go:embed 嵌进 Windows 二进制）
# 仓库里不存这个 162M 的文件；克隆仓库后跑一次本脚本即可。
set -e
cd "$(dirname "$0")/.."

DST="internal/ffembed/ffmpeg.exe"
URL="https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip"

if [ -s "$DST" ]; then
  echo "已存在 $DST（$(du -h "$DST" | cut -f1)），跳过下载。"
  exit 0
fi

mkdir -p internal/ffembed
echo "下载 ffmpeg.exe ..."
TMP_ZIP="$(mktemp -d)/ffmpeg.zip"
curl -L -o "$TMP_ZIP" "$URL"
unzip -p "$TMP_ZIP" "ffmpeg-master-latest-win64-gpl/bin/ffmpeg.exe" > "$DST"
chmod +x "$DST"
echo "完成：$DST（$(du -h "$DST" | cut -f1)）"

#!/bin/bash
# MediaConv 卸载脚本
# Uninstall script
set -e
PREFIX="$HOME/.local/bin"
for f in mediaconv mconv mediaconv-cli; do
    rm -f "$PREFIX/$f" && echo "已删除 $PREFIX/$f"
done
echo "完成 / Done."

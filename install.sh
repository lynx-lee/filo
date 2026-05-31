#!/bin/bash

set -e

echo ""
echo "  ███████╗██╗██╗      ██████╗ "
echo "  ██╔════╝██║██║     ██╔═══██╗"
echo "  █████╗  ██║██║     ██║   ██║"
echo "  ██╔══╝  ██║██║     ██║   ██║"
echo "  ██║     ██║███████╗╚██████╔╝"
echo "  ╚═╝     ╚═╝╚══════╝ ╚═════╝ "
echo ""
echo "  文件智理 · 越用越懂你"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 检测系统
OS=$(uname -s)
ARCH=$(uname -m)

echo ""
echo "📍 系统: $OS $ARCH"

# 确定下载文件名
case "$OS" in
    Darwin)
        case "$ARCH" in
            arm64) BINARY="filo-mac-arm64" ;;
            x86_64) BINARY="filo-mac-amd64" ;;
            *) echo "❌ 不支持的架构: $ARCH"; exit 1 ;;
        esac
        ;;
    Linux)
        case "$ARCH" in
            aarch64) BINARY="filo-linux-arm64" ;;
            x86_64) BINARY="filo-linux-amd64" ;;
            *) echo "❌ 不支持的架构: $ARCH"; exit 1 ;;
        esac
        ;;
    *)
        echo "❌ 不支持的系统: $OS"
        exit 1
        ;;
esac

# 下载地址（替换为实际地址）
DOWNLOAD_URL="https://github.com/yourname/filo/releases/latest/download/$BINARY"

echo ""
echo "📥 下载 filo..."

# 下载
INSTALL_DIR="/usr/local/bin"
TMP_FILE="/tmp/filo"

if command -v curl &> /dev/null; then
    curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE"
elif command -v wget &> /dev/null; then
    wget -q "$DOWNLOAD_URL" -O "$TMP_FILE"
else
    echo "❌ 需要 curl 或 wget"
    exit 1
fi

chmod +x "$TMP_FILE"

# 安装
echo "📦 安装到 $INSTALL_DIR..."
sudo mv "$TMP_FILE" "$INSTALL_DIR/filo"

echo ""
echo "✅ 安装完成！"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "下一步："
echo ""
echo "  1. 运行安装向导: filo setup"
echo "  2. 预览整理效果: filo ~/Downloads -n"
echo "  3. 执行整理:     filo ~/Downloads"
echo ""

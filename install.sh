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
echo ""
echo "⚠️  注意：当前版本尚未发布正式 Release，建议使用源码编译安装"
echo "   或使用: go install github.com/lynx-lee/filo@latest"
echo ""

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

# 下载地址
DOWNLOAD_URL="https://github.com/lynx-lee/filo/releases/latest/download/$BINARY"
BACKUP_URL="https://ghproxy.net/https://github.com/lynx-lee/filo/releases/latest/download/$BINARY"

echo ""
echo "📥 下载 filo..."

# 下载（带重试机制）
INSTALL_DIR="/usr/local/bin"
TMP_FILE="/tmp/filo"
MAX_RETRIES=3
DOWNLOAD_SUCCESS=false

echo "  下载地址: $DOWNLOAD_URL"
echo ""

for i in $(seq 1 $MAX_RETRIES); do
    echo "  尝试下载 (第 $i/$MAX_RETRIES 次)..."
    
    if command -v curl &> /dev/null; then
        if curl -fsSL --connect-timeout 10 --max-time 60 "$DOWNLOAD_URL" -o "$TMP_FILE" 2>/dev/null; then
            DOWNLOAD_SUCCESS=true
            break
        fi
    elif command -v wget &> /dev/null; then
        if wget -q --timeout=10 --tries=1 "$DOWNLOAD_URL" -O "$TMP_FILE" 2>/dev/null; then
            DOWNLOAD_SUCCESS=true
            break
        fi
    else
        echo "❌ 需要 curl 或 wget"
        exit 1
    fi
    
    echo "  ⚠️  第 $i 次尝试失败"
    if [ $i -lt $MAX_RETRIES ]; then
        echo "  等待 2 秒后重试..."
        sleep 2
    fi
done

# 如果主地址失败，尝试备用地址
if [ "$DOWNLOAD_SUCCESS" = false ]; then
    echo ""
    echo "  尝试备用下载地址..."
    echo "  备用地址: $BACKUP_URL"
    echo ""
    
    for i in $(seq 1 $MAX_RETRIES); do
        echo "  尝试下载 (第 $i/$MAX_RETRIES 次)..."
        
        if command -v curl &> /dev/null; then
            if curl -fsSL --connect-timeout 10 --max-time 60 "$BACKUP_URL" -o "$TMP_FILE" 2>/dev/null; then
                DOWNLOAD_SUCCESS=true
                break
            fi
        elif command -v wget &> /dev/null; then
            if wget -q --timeout=10 --tries=1 "$BACKUP_URL" -O "$TMP_FILE" 2>/dev/null; then
                DOWNLOAD_SUCCESS=true
                break
            fi
        fi
        
        echo "  ⚠️  第 $i 次尝试失败"
        if [ $i -lt $MAX_RETRIES ]; then
            sleep 2
        fi
    done
fi

# 检查下载是否成功
if [ "$DOWNLOAD_SUCCESS" = false ]; then
    echo ""
    echo "❌ 下载失败！所有下载地址均不可用。"
    echo ""
    echo "这可能是因为："
    echo "  1. GitHub Releases 尚未发布正式版本"
    echo "  2. 网络连接问题"
    echo ""
    echo "请使用以下替代安装方式："
    echo ""
    echo "  方式一：源码编译（推荐）"
    echo "    cd filo"
    echo "    make build"
    echo "    sudo make install"
    echo ""
    echo "  方式二：Go 安装"
    echo "    go install github.com/lynx-lee/filo@latest"
    echo ""
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

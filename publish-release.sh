#!/bin/bash
# publish-release.sh - 使用 GitHub CLI 发布新版本
# 需要先安装 gh: brew install gh
# 需要登录: gh auth login

set -e

VERSION="v2.1.0-alpha"
TAG="v2.1.0-alpha"
RELEASE_NOTES="RELEASE_NOTES_v2.1.0-alpha.md"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🚀 Filo Release Publisher"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 检查 gh 是否安装
if ! command -v gh &> /dev/null; then
    echo "❌ GitHub CLI (gh) 未安装"
    echo ""
    echo "请先安装:"
    echo "  macOS: brew install gh"
    echo "  Linux: sudo apt install gh"
    echo ""
    exit 1
fi

# 检查是否已登录
if ! gh auth status &> /dev/null; then
    echo "❌ 未登录 GitHub"
    echo ""
    echo "请先运行: gh auth login"
    echo ""
    exit 1
fi

# 检查 release notes 是否存在
if [ ! -f "$RELEASE_NOTES" ]; then
    echo "❌ Release notes 文件不存在: $RELEASE_NOTES"
    exit 1
fi

echo "📋 版本信息:"
echo "   Tag: $TAG"
echo "   Version: $VERSION"
echo "   Notes: $RELEASE_NOTES"
echo ""

# 确认发布
read -p "确认发布 $TAG? (y/N): " confirm
if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
    echo "❌ 取消发布"
    exit 0
fi

echo ""
echo "🔨 构建多平台二进制文件..."
./release.sh "$VERSION"

echo ""
echo "📤 上传到 GitHub Releases..."

# 创建 Release
gh release create "$TAG" \
    --title "Filo $VERSION - 智能体系统正式发布" \
    --notes-file "$RELEASE_NOTES" \
    --draft \
    dist/*

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Release 创建成功！"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📝 这是一个 Draft Release，请访问 GitHub 页面进行最终发布："
echo "   https://github.com/lynx-lee/filo/releases"
echo ""
echo "💡 提示："
echo "   1. 在 GitHub 页面上检查 Release 内容"
echo "   2. 点击 'Publish release' 正式发布"
echo "   3. 或者运行: gh release edit $TAG --draft=false"
echo ""

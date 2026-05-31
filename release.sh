#!/bin/bash
# release.sh - 自动化构建和打包 Filo 多平台版本
# 用法: ./release.sh <version>
# 示例: ./release.sh v2.1.0-alpha

set -e

VERSION=${1:-"v2.1.0-alpha"}
BUILD_DIR="build"
DIST_DIR="dist"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📦 Filo Release Builder"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "版本: $VERSION"
echo ""

# 清理旧的构建文件
echo "🧹 清理旧文件..."
rm -rf "$BUILD_DIR" "$DIST_DIR"
mkdir -p "$BUILD_DIR" "$DIST_DIR"

# 定义目标平台
PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "windows/amd64"
)

# 编译各个平台
for platform in "${PLATFORMS[@]}"; do
    OS=$(echo $platform | cut -d'/' -f1)
    ARCH=$(echo $platform | cut -d'/' -f2)
    
    echo ""
    echo "🔨 编译 $OS/$ARCH..."
    
    # 设置环境变量
    export GOOS=$OS
    export GOARCH=$ARCH
    
    # 确定输出文件名
    if [ "$OS" = "windows" ]; then
        OUTPUT="$BUILD_DIR/filo-${OS}-${ARCH}.exe"
    else
        OUTPUT="$BUILD_DIR/filo-${OS}-${ARCH}"
    fi
    
    # 编译
    CGO_ENABLED=0 go build \
        -ldflags="-s -w -X github.com/lynx-lee/filo/internal/config.Version=${VERSION#v} -X github.com/lynx-lee/filo/internal/config.BuildDate=$(date +%Y%m%d)" \
        -o "$OUTPUT" \
        .
    
    echo "✅ $OS/$ARCH 编译完成: $OUTPUT"
done

# 取消环境变量
unset GOOS
unset GOARCH

echo ""
echo "📦 打包分发文件..."

# 为每个平台创建压缩包
for platform in "${PLATFORMS[@]}"; do
    OS=$(echo $platform | cut -d'/' -f1)
    ARCH=$(echo $platform | cut -d'/' -f2)
    
    if [ "$OS" = "windows" ]; then
        BINARY="filo-${OS}-${ARCH}.exe"
        ARCHIVE="filo-${VERSION#v}-${OS}-${ARCH}.zip"
        
        cd "$BUILD_DIR"
        zip "../$DIST_DIR/$ARCHIVE" "$BINARY" ../LICENSE ../README.md > /dev/null
        cd ..
    else
        BINARY="filo-${OS}-${ARCH}"
        ARCHIVE="filo-${VERSION#v}-${OS}-${ARCH}.tar.gz"
        
        cd "$BUILD_DIR"
        tar czf "../$DIST_DIR/$ARCHIVE" "$BINARY" ../LICENSE ../README.md
        cd ..
    fi
    
    echo "✅ 打包完成: $DIST_DIR/$ARCHIVE"
done

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✨ 构建完成！"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📁 输出目录: $DIST_DIR/"
echo ""
echo "📋 生成的文件:"
ls -lh "$DIST_DIR/" | tail -n +2
echo ""
echo "💡 下一步："
echo "   1. 测试各个平台的二进制文件"
echo "   2. 上传到 GitHub Releases"
echo "   3. 更新 Homebrew formula（如需要）"
echo ""

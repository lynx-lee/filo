# Filo - 文件智理，越用越懂你
# Copyright (c) 2024-2026 lynx-lee
# https://github.com/lynx-lee/filo

APP_NAME := filo
VERSION := 2.1.0-alpha
AUTHOR := lynx-lee
BUILD_TIME := $(shell date +%Y%m%d)
LDFLAGS := -ldflags="-s -w -X github.com/lynx-lee/filo/internal/config.Version=$(VERSION) -X github.com/lynx-lee/filo/internal/config.BuildDate=$(BUILD_TIME)"

.PHONY: all build clean install test run

# 默认构建
all: build

# 当前平台构建
build:
	@echo "🔨 Building $(APP_NAME) v$(VERSION) by $(AUTHOR)..."
	@go build $(LDFLAGS) -o bin/$(APP_NAME) .
	@echo "✅ Build complete: bin/$(APP_NAME)"

# 所有平台构建
# 注意：Windows 交叉编译可能需要在本机执行或使用 CI/CD
build-all: build-darwin build-linux build-windows
	@echo "✅ All platforms built"

build-darwin:
	@echo "🍎 Building for macOS..."
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-mac-amd64 .
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(APP_NAME)-mac-arm64 .

build-linux:
	@echo "🐧 Building for Linux..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-linux-amd64 .
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(APP_NAME)-linux-arm64 .

build-windows:
	@echo "🪟 Building for Windows..."
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-windows.exe .
	@GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o bin/$(APP_NAME)-windows-arm64.exe .
	@GOOS=windows GOARCH=386 go build $(LDFLAGS) -o bin/$(APP_NAME)-windows-x86.exe .

# 安装到系统
install: build
	@echo "📦 Installing to /usr/local/bin..."
	@sudo cp bin/$(APP_NAME) /usr/local/bin/
	@echo "✅ Installed! Run 'filo' to start"

# 清理
clean:
	@rm -rf bin/
	@echo "🧹 Cleaned"

# 测试
test:
	@go test ./... -v

# 开发运行
run:
	@go run . ~/Downloads -n

# 初始化依赖
deps:
	@go mod tidy
	@echo "✅ Dependencies ready"

# 帮助
help:
	@echo "filo - 文件智理 by $(AUTHOR)"
	@echo ""
	@echo "构建命令:"
	@echo "  make build      构建当前平台"
	@echo "  make build-all  构建所有平台"
	@echo "  make install    安装到系统"
	@echo "  make clean      清理构建文件"
	@echo ""
	@echo "开发命令:"
	@echo "  make run        开发测试运行"
	@echo "  make test       运行测试"
	@echo "  make deps       更新依赖"

// Package cmd 命令行入口模块
// setup.go - 安装向导命令，用于配置 Ollama 环境和下载模型
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"filo/internal/config"
	"filo/internal/llm"
	"filo/internal/ui"
)

// setupCmd 安装向导命令定义
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "安装向导",
	Long:  "安装和配置 Ollama 及推荐模型",
	Run:   runSetup,
}

// init 注册 setup 子命令
func init() {
	rootCmd.AddCommand(setupCmd)
}

// runSetup 执行安装向导
// 流程：检查 Ollama -> 启动服务 -> 检查模型 -> 下载推荐模型
func runSetup(cmd *cobra.Command, args []string) {
	ui.Banner()
	ui.Title("🚀", "安装向导")
	ui.Divider()

	// ========== 步骤1: 检查 Ollama 是否安装 ==========
	fmt.Println()
	ui.Info("检查 Ollama...")
	ollamaPath, err := exec.LookPath("ollama")
	if err != nil {
		ui.Error("Ollama 未安装")
		printInstallInstructions() // 显示安装指引
		return
	}
	ui.Success("Ollama 已安装: %s", ollamaPath)

	// ========== 步骤2: 启动 Ollama 服务 ==========
	ui.Info("启动 Ollama 服务...")
	startOllama()
	time.Sleep(2 * time.Second) // 等待服务启动

	// 检查服务是否成功启动（最多等待5秒）
	client := llm.NewClient()
	for i := 0; i < 5; i++ {
		if client.IsAvailable() {
			break
		}
		time.Sleep(time.Second)
	}

	if !client.IsAvailable() {
		ui.Error("无法连接 Ollama 服务")
		ui.Info("请手动运行: ollama serve")
		return
	}
	ui.Success("Ollama 服务已启动")

	// ========== 步骤3: 检查已安装的模型 ==========
	fmt.Println()
	ui.Info("已安装的模型:")
	models, _ := client.ListModels()
	if len(models) == 0 {
		ui.Dim("  (无)")
	} else {
		for _, m := range models {
			ui.Info("  - %s", m)
		}
	}

	// ========== 步骤4: 检查并下载推荐模型 ==========
	cfg := config.Get()
	recommended := cfg.LLMModel // 获取推荐模型名称

	// 检查推荐模型是否已安装
	hasModel := false
	for _, m := range models {
		if strings.HasPrefix(m, strings.Split(recommended, ":")[0]) {
			hasModel = true
			break
		}
	}

	// 如果未安装，提示下载
	if !hasModel {
		fmt.Println()
		ui.Warning("推荐模型 %s 未安装", recommended)
		if ui.Confirm("是否下载?", true) {
			downloadModel(recommended)
		}
	} else {
		ui.Success("推荐模型已安装")
	}

	// ========== 步骤5: 显示完成信息 ==========
	fmt.Println()
	ui.Divider()
	ui.Success("设置完成！")
	fmt.Println()
	ui.Info("现在可以使用:")
	fmt.Println()
	fmt.Println("  " + ui.Cyan("filo ~/Downloads -n") + "    # 预览整理效果")
	fmt.Println("  " + ui.Cyan("filo ~/Downloads") + "       # 执行整理")
	fmt.Println()
}

// printInstallInstructions 打印 Ollama 安装指引
// 根据不同操作系统显示对应的安装命令
func printInstallInstructions() {
	fmt.Println()
	ui.Info("安装方法:")
	switch runtime.GOOS {
	case "darwin":
		// macOS 安装方式
		fmt.Println("  brew install ollama")
		fmt.Println("  或访问: https://ollama.com/download/mac")
	case "linux":
		// Linux 安装方式
		fmt.Println("  curl -fsSL https://ollama.com/install.sh | sh")
	case "windows":
		// Windows 安装方式
		fmt.Println("  访问: https://ollama.com/download/windows")
	default:
		// 其他系统
		fmt.Println("  访问: https://ollama.com/download")
	}
}

// startOllama 后台启动 Ollama 服务
// 以非阻塞方式启动，不等待命令完成
func startOllama() {
	cmd := exec.Command("ollama", "serve")
	cmd.Stdout = nil // 不捕获输出
	cmd.Stderr = nil
	cmd.Start() // 异步启动
}

// downloadModel 下载指定的模型
// 调用 ollama pull 命令下载模型，实时显示下载进度
func downloadModel(model string) {
	ui.Info("下载 %s ...", model)
	cmd := exec.Command("ollama", "pull", model)
	cmd.Stdout = os.Stdout // 直接输出到终端
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		ui.Error("下载失败: %v", err)
	} else {
		ui.Success("下载完成")
	}
}

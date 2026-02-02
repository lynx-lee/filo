// Package cmd 命令行入口模块
// config.go - 配置管理命令，用于查看和修改 Filo 配置
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"filo/internal/config"
	"filo/internal/ui"
)

// 配置选项标志
var (
	setModel       string  // 设置默认模型
	setThreshold   float64 // 设置置信度阈值
	setBatchSize   int     // 设置批处理大小
	toggleLearning bool    // 切换学习功能开关
)

// configCmd 配置管理命令定义
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "配置管理",
	Long:  "查看或修改 Filo 配置",
	Run:   runConfig,
}

// init 注册 config 子命令及其标志
func init() {
	configCmd.Flags().StringVar(&setModel, "model", "", "设置默认模型")
	configCmd.Flags().Float64Var(&setThreshold, "threshold", 0, "设置置信度阈值 (0.5-1.0)")
	configCmd.Flags().IntVar(&setBatchSize, "batch", 0, "设置批处理大小 (5-50)")
	configCmd.Flags().BoolVar(&toggleLearning, "toggle-learning", false, "切换学习功能开关")
	rootCmd.AddCommand(configCmd)
}

// runConfig 执行配置命令
// 如果没有设置选项，显示当前配置；否则修改配置
func runConfig(cmd *cobra.Command, args []string) {
	ui.Banner()
	cfg := config.Get()

	// 检查是否有设置选项
	hasChanges := false

	// 设置模型
	if setModel != "" {
		cfg.LLMModel = setModel
		ui.Success("默认模型已设置为: %s", setModel)
		hasChanges = true
	}

	// 设置置信度阈值
	if setThreshold > 0 {
		if setThreshold < 0.5 || setThreshold > 1.0 {
			ui.Error("置信度阈值必须在 0.5 到 1.0 之间")
			return
		}
		cfg.ConfidenceThreshold = setThreshold
		ui.Success("置信度阈值已设置为: %.2f", setThreshold)
		hasChanges = true
	}

	// 设置批处理大小
	if setBatchSize > 0 {
		if setBatchSize < 5 || setBatchSize > 50 {
			ui.Error("批处理大小必须在 5 到 50 之间")
			return
		}
		cfg.BatchSize = setBatchSize
		ui.Success("批处理大小已设置为: %d", setBatchSize)
		hasChanges = true
	}

	// 切换学习功能
	if toggleLearning {
		cfg.EnableLearning = !cfg.EnableLearning
		status := "开启"
		if !cfg.EnableLearning {
			status = "关闭"
		}
		ui.Success("学习功能已%s", status)
		hasChanges = true
	}

	// 如果有更改，保存配置
	if hasChanges {
		if err := cfg.Save(); err != nil {
			ui.Error("保存配置失败: %v", err)
		} else {
			ui.Success("配置已保存")
		}
		return
	}

	// 没有设置选项，显示当前配置
	showConfig(cfg)
}

// showConfig 显示当前配置
func showConfig(cfg *config.Config) {
	ui.Title("⚙️", "当前配置")
	ui.Divider()

	fmt.Println()
	ui.Info("模型配置:")
	ui.Info("  LLM 模型:      %s", cfg.LLMModel)
	ui.Info("  嵌入模型:      %s", cfg.EmbeddingModel)
	ui.Info("  Ollama 地址:   %s", cfg.OllamaURL)
	ui.Info("  温度参数:      %.2f", cfg.Temperature)

	fmt.Println()
	ui.Info("学习配置:")
	learning := "开启"
	if !cfg.EnableLearning {
		learning = "关闭"
	}
	ui.Info("  学习功能:      %s", learning)
	ui.Info("  相似度阈值:    %.2f", cfg.SimilarityThreshold)
	ui.Info("  置信度阈值:    %.2f", cfg.ConfidenceThreshold)
	ui.Info("  最小样本数:    %d", cfg.MinSamplesForRule)

	fmt.Println()
	ui.Info("处理配置:")
	ui.Info("  批处理大小:    %d", cfg.BatchSize)

	fmt.Println()
	ui.Info("数据路径:")
	ui.Info("  数据目录:      %s", cfg.DataDir)
	ui.Info("  数据库文件:    %s", cfg.DBPath)

	fmt.Println()
	ui.Dim("修改配置示例:")
	ui.Dim("  filo config --model qwen3:8b")
	ui.Dim("  filo config --threshold 0.8")
	ui.Dim("  filo config --batch 20")
	ui.Dim("  filo config --toggle-learning")
}

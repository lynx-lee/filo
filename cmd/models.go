// Package cmd 命令行入口模块
// models.go - 模型管理命令，列出可用的本地模型和性能统计
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"filo/internal/config"
	"filo/internal/llm"
	"filo/internal/storage"
	"filo/internal/ui"
)

// modelsCmd 模型管理命令定义
var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "模型管理",
	Long: `列出可用的本地模型和性能统计。

示例:
  filo models              # 列出可用模型
  filo models --stats      # 显示模型性能对比
  filo models --recommend  # 显示推荐模型`,
	Run: runModels,
}

// models 命令行参数
var (
	showStats     bool // 显示性能统计
	showRecommend bool // 显示推荐
)

// init 注册 models 子命令
func init() {
	rootCmd.AddCommand(modelsCmd)
	modelsCmd.Flags().BoolVarP(&showStats, "stats", "s", false, "显示模型性能对比")
	modelsCmd.Flags().BoolVarP(&showRecommend, "recommend", "r", false, "显示推荐模型")
}

// runModels 执行模型管理命令
// 列出 Ollama 中已安装的所有模型，并标记当前使用的模型
func runModels(cmd *cobra.Command, args []string) {
	ui.Banner()

	// 显示性能统计
	if showStats {
		showModelStats()
		return
	}

	// 显示推荐
	if showRecommend {
		showModelRecommendation()
		return
	}

	// 默认：列出可用模型
	listAvailableModels()
}

// listAvailableModels 列出可用模型
func listAvailableModels() {
	ui.Title("🤖", "可用模型")
	ui.Divider()

	// 创建 LLM 客户端
	client := llm.NewClient()

	// 检查 Ollama 服务状态
	if !client.IsAvailable() {
		ui.Error("Ollama 服务未运行")
		ui.Info("请运行: ollama serve")
		return
	}

	// 获取已安装的模型列表
	models, err := client.ListModels()
	if err != nil {
		ui.Error("获取模型列表失败: %v", err)
		return
	}

	cfg := config.Get()

	// 检查是否有已安装的模型
	if len(models) == 0 {
		ui.Warning("未找到已安装的模型")
		ui.Info("运行 'filo setup' 安装模型")
		return
	}

	// 获取模型性能数据
	db, _ := storage.NewDatabase()
	var summaries []storage.ModelSummary
	if db != nil {
		summaries, _ = db.GetModelSummaries()
		db.Close()
	}

	// 构建性能数据映射
	statsMap := make(map[string]storage.ModelSummary)
	for _, s := range summaries {
		statsMap[s.ModelName] = s
	}

	// 显示模型列表
	fmt.Println()
	for _, m := range models {
		var suffix string
		if m == cfg.LLMModel {
			suffix = ui.Green(" (当前)")
		}

		// 显示性能信息（如果有）
		if stats, ok := statsMap[m]; ok {
			fmt.Printf("  %s %s%s\n", ui.Green("✓"), m, suffix)
			ui.Dim("      📊 %d 文件 | ⏱️ %.0fms/文件 | 🎯 %.0f%%准确",
				stats.TotalFiles, stats.AvgTimePerFileMs, stats.AccuracyRate*100)
		} else {
			if m == cfg.LLMModel {
				fmt.Printf("  %s %s%s\n", ui.Green("✓"), m, suffix)
			} else {
				fmt.Printf("    %s\n", m)
			}
		}
	}

	// 显示切换模型的提示
	fmt.Println()
	ui.Info("切换模型: filo -m <模型名> <目录>")
	ui.Info("性能对比: filo models --stats")
}

// showModelStats 显示模型性能对比
func showModelStats() {
	ui.Title("📊", "模型性能对比")
	ui.Divider()

	db, err := storage.NewDatabase()
	if err != nil {
		ui.Error("无法连接数据库: %v", err)
		return
	}
	defer db.Close()

	summaries, err := db.GetModelSummaries()
	if err != nil || len(summaries) == 0 {
		ui.Warning("暂无性能数据")
		ui.Info("使用不同模型整理文件后，这里会显示性能对比")
		return
	}

	cfg := config.Get()
	fmt.Println()

	// 表头
	fmt.Printf("  %-20s %8s %10s %10s %8s %8s\n",
		"模型", "文件数", "速度", "置信度", "准确率", "评分")
	ui.Divider()

	// 显示每个模型的统计
	for i, s := range summaries {
		// 标记当前模型和推荐模型
		var marker string
		if s.ModelName == cfg.LLMModel {
			marker = ui.Green(" ◀ 当前")
		}
		if i == 0 && s.TotalFiles >= 10 {
			marker = ui.Green(" ★ 推荐")
			if s.ModelName == cfg.LLMModel {
				marker = ui.Green(" ★ 当前")
			}
		}

		// 格式化速度
		speedStr := fmt.Sprintf("%.0fms", s.AvgTimePerFileMs)

		// 显示一行统计
		fmt.Printf("  %-20s %8d %10s %9.0f%% %7.0f%% %7.0f%%%s\n",
			truncateModelName(s.ModelName, 20),
			s.TotalFiles,
			speedStr,
			s.AvgConfidence*100,
			s.AccuracyRate*100,
			s.Score*100,
			marker,
		)
	}

	fmt.Println()
	ui.Dim("评分 = 准确率×50%% + 置信度×30%% + 速度×20%%")
	ui.Dim("准确率基于用户确认/纠正计算，需积累足够数据")
}

// showModelRecommendation 显示推荐模型
func showModelRecommendation() {
	ui.Title("⭐", "模型推荐")
	ui.Divider()

	db, err := storage.NewDatabase()
	if err != nil {
		ui.Error("无法连接数据库: %v", err)
		return
	}
	defer db.Close()

	cfg := config.Get()

	// 获取最佳模型
	bestModel := db.GetBestModel()
	if bestModel == "" {
		ui.Warning("暂无足够数据推荐模型")
		ui.Info("使用不同模型整理更多文件后，系统会自动推荐最佳模型")
		return
	}

	fmt.Println()
	if bestModel == cfg.LLMModel {
		ui.Success("当前使用的 %s 就是推荐模型！", ui.Bold(bestModel))
	} else {
		ui.Info("推荐切换到: %s", ui.Bold(bestModel))
		ui.Info("当前使用: %s", cfg.LLMModel)
		fmt.Println()
		ui.Dim("切换命令: filo -m %s <目录>", bestModel)
		ui.Dim("或修改配置: filo config --model %s", bestModel)
	}

	// 显示推荐理由
	summaries, _ := db.GetModelSummaries()
	for _, s := range summaries {
		if s.ModelName == bestModel {
			fmt.Println()
			ui.Info("推荐理由:")
			fmt.Printf("  • 处理 %d 个文件的经验\n", s.TotalFiles)
			fmt.Printf("  • 平均速度: %.0f ms/文件\n", s.AvgTimePerFileMs)
			fmt.Printf("  • 平均置信度: %.0f%%\n", s.AvgConfidence*100)
			if s.TotalConfirmed+s.TotalCorrected > 0 {
				fmt.Printf("  • 用户反馈准确率: %.0f%%\n", s.AccuracyRate*100)
			}
			break
		}
	}
}

// truncateModelName 截断模型名称
func truncateModelName(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}
	return name[:maxLen-3] + "..."
}

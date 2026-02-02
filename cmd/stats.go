// Package cmd 命令行入口模块
// stats.go - 学习统计命令，显示系统学习状态和分类分布
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"filo/internal/classifier"
	"filo/internal/config"
	"filo/internal/ui"
)

// statsCmd 统计命令定义
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "学习统计",
	Long:  "显示学习记录和统计信息",
	Run:   runStats,
}

// init 注册 stats 子命令
func init() {
	rootCmd.AddCommand(statsCmd)
}

// runStats 执行统计命令
// 显示系统状态、学习记录数量、分类分布等信息
func runStats(cmd *cobra.Command, args []string) {
	ui.Banner()
	ui.Title("📊", "学习统计")
	ui.Divider()

	// 初始化分类器以获取统计数据
	clf, err := classifier.NewClassifier()
	if err != nil {
		ui.Error("初始化失败: %v", err)
		return
	}
	defer clf.Close()

	// 获取统计信息
	stats, err := clf.GetStatistics()
	if err != nil {
		ui.Error("获取统计失败: %v", err)
		return
	}

	cfg := config.Get()

	// 显示系统状态
	fmt.Println()
	ui.Info("系统状态:")
	ui.Info("  历史分类:  %v 条", stats["total_records"])     // 总分类记录数
	ui.Info("  用户确认:  %v 条", stats["confirmed_records"]) // 用户确认的记录数
	ui.Info("  学习规则:  %v 条", stats["learned_rules"])     // 已学习的规则数
	ui.Info("  向量记录:  %v 条", stats["vector_count"])      // 向量嵌入记录数
	ui.Info("  用户反馈:  %v 条", stats["feedback_count"])    // 用户纠正反馈数

	// 显示学习功能状态
	learning := "开启"
	if !cfg.EnableLearning {
		learning = "关闭"
	}
	ui.Info("  学习功能:  %s", learning)
	ui.Info("  当前模型:  %s", cfg.LLMModel)

	// 显示分类分布（如果有数据）
	if dist, ok := stats["category_distribution"].(map[string]int); ok && len(dist) > 0 {
		fmt.Println()
		ui.Info("分类分布:")
		for cat, cnt := range dist {
			ui.Info("  %-12s %d", cat, cnt)
		}
	}
}

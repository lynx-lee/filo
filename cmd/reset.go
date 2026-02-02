// Package cmd 命令行入口模块
// reset.go - 重置命令，用于清除学习数据和记忆
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package cmd

import (
	"github.com/spf13/cobra"

	"filo/internal/storage"
	"filo/internal/ui"
)

// 重置选项标志
var (
	resetRules   bool // 重置学习规则
	resetHistory bool // 重置历史记录
	resetAll     bool // 重置所有数据
)

// resetCmd 重置命令定义
var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "重置数据",
	Long:  "重置学习数据和记忆",
	Run:   runReset,
}

// init 注册 reset 子命令及其标志
func init() {
	resetCmd.Flags().BoolVar(&resetRules, "rules", false, "重置学习规则")
	resetCmd.Flags().BoolVar(&resetHistory, "history", false, "重置历史记录")
	resetCmd.Flags().BoolVar(&resetAll, "all", false, "重置所有数据")
	rootCmd.AddCommand(resetCmd)
}

// runReset 执行重置命令
// 根据不同的标志重置相应的数据
func runReset(cmd *cobra.Command, args []string) {
	ui.Banner()

	// 连接数据库
	db, err := storage.NewDatabase()
	if err != nil {
		ui.Error("数据库连接失败: %v", err)
		return
	}
	defer db.Close()

	// 重置所有数据
	if resetAll {
		if !ui.ConfirmDanger("确认重置所有数据?") {
			return
		}
		if err := db.ResetAll(); err != nil {
			ui.Error("重置失败: %v", err)
			return
		}
		ui.Success("已重置所有数据")
		return
	}

	// 重置学习规则
	if resetRules {
		if !ui.ConfirmDanger("确认重置学习规则?") {
			return
		}
		if err := db.ResetRules(); err != nil {
			ui.Error("重置失败: %v", err)
			return
		}
		ui.Success("已重置规则")
	}

	// 重置历史记录
	if resetHistory {
		if !ui.ConfirmDanger("确认重置历史记录?") {
			return
		}
		db.ResetHistory() // 重置分类历史
		db.ResetVectors() // 重置向量记录
		ui.Success("已重置历史")
	}

	// 如果没有指定任何标志，显示帮助信息
	if !resetRules && !resetHistory && !resetAll {
		cmd.Help()
	}
}

// Package cmd 命令行入口模块
// scan.go - 扫描命令，用于扫描目录并显示文件统计信息
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package cmd

import (
	"github.com/spf13/cobra"

	"filo/internal/scanner"
	"filo/internal/ui"
)

// scanCmd 扫描命令定义
var scanCmd = &cobra.Command{
	Use:   "scan [目录]",
	Short: "扫描统计",
	Long:  "扫描目录并显示文件统计信息",
	Args:  cobra.ExactArgs(1), // 必须提供一个目录参数
	Run:   runScan,
}

// init 注册 scan 子命令
func init() {
	rootCmd.AddCommand(scanCmd)
}

// runScan 执行扫描命令
// 扫描指定目录，统计文件类型、数量和大小
func runScan(cmd *cobra.Command, args []string) {
	ui.Banner()

	dir := args[0]

	// 扫描目录
	files, err := scanner.ScanDirectory(dir, false)
	if err != nil {
		ui.Error("扫描失败: %v", err)
		return
	}

	// 打印统计信息
	scanner.PrintStatistics(files)
}

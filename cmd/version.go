// Package cmd 命令行入口模块
// version.go - 版本命令，显示程序版本和作者信息
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

// versionCmd 版本命令定义
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		ui.Banner()
		fmt.Println()
		fmt.Printf("  版本:   %s\n", config.Version)   // 版本号
		fmt.Printf("  作者:   %s\n", config.Author)    // 作者
		fmt.Printf("  主页:   %s\n", config.Homepage)  // 项目主页
		fmt.Printf("  许可:   %s\n", config.License)   // 开源许可
		fmt.Printf("  构建:   %s\n", config.BuildDate) // 构建日期
		fmt.Println()
	},
}

// init 注册 version 子命令
func init() {
	rootCmd.AddCommand(versionCmd)
}

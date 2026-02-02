// Package ui 终端界面模块
// 提供终端输出美化功能，包括颜色、图标、格式化等
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

// ==================== 颜色定义 ====================
// 使用 fatih/color 库定义各种颜色函数
var (
	Cyan     = color.New(color.FgCyan).SprintFunc()              // 青色
	Green    = color.New(color.FgGreen).SprintFunc()             // 绿色（成功）
	Yellow   = color.New(color.FgYellow).SprintFunc()            // 黄色（警告）
	Red      = color.New(color.FgRed).SprintFunc()               // 红色（错误）
	White    = color.New(color.FgWhite).SprintFunc()             // 白色
	Gray     = color.New(color.FgHiBlack).SprintFunc()           // 灰色（次要信息）
	Bold     = color.New(color.Bold).SprintFunc()                // 粗体
	BoldCyan = color.New(color.FgCyan, color.Bold).SprintFunc()  // 青色粗体
)

// ==================== 输出函数 ====================

// Banner 打印启动横幅
// 显示 ASCII 艺术字 Logo 和版本信息
func Banner() {
	banner := `
` + Cyan(`  ███████╗██╗██╗      ██████╗ `) + `
` + Cyan(`  ██╔════╝██║██║     ██╔═══██╗`) + `
` + Cyan(`  █████╗  ██║██║     ██║   ██║`) + `
` + Cyan(`  ██╔══╝  ██║██║     ██║   ██║`) + `
` + Cyan(`  ██║     ██║███████╗╚██████╔╝`) + `
` + Cyan(`  ╚═╝     ╚═╝╚══════╝ ╚═════╝ `) + `
` + Gray(`  文件智理 · 越用越懂你`) + ` ` + Gray(`v2.0`) + `
` + Gray(`  by lynx-lee`) + `
`
	fmt.Println(banner)
}

// Title 打印标题
// 格式: 图标 + 青色粗体文字
func Title(icon, text string) {
	fmt.Printf("\n%s %s\n", icon, BoldCyan(text))
}

// Success 打印成功消息
// 格式: ✓ + 消息内容（绿色勾号）
func Success(format string, args ...interface{}) {
	fmt.Printf("  %s %s\n", Green("✓"), fmt.Sprintf(format, args...))
}

// Error 打印错误消息
// 格式: ✗ + 消息内容（红色叉号）
func Error(format string, args ...interface{}) {
	fmt.Printf("  %s %s\n", Red("✗"), fmt.Sprintf(format, args...))
}

// Warning 打印警告消息
// 格式: ⚠ + 消息内容（黄色警告号）
func Warning(format string, args ...interface{}) {
	fmt.Printf("  %s %s\n", Yellow("⚠"), fmt.Sprintf(format, args...))
}

// Info 打印信息消息
// 格式: 缩进 + 消息内容
func Info(format string, args ...interface{}) {
	fmt.Printf("  %s\n", fmt.Sprintf(format, args...))
}

// Dim 打印暗色消息
// 用于显示次要信息（灰色文字）
func Dim(format string, args ...interface{}) {
	fmt.Printf("  %s\n", Gray(fmt.Sprintf(format, args...)))
}

// Divider 打印分隔线
// 55个横线字符组成的灰色分隔线
func Divider() {
	fmt.Println(Gray(strings.Repeat("─", 55)))
}

// ==================== 方框绘制 ====================

// Box 绘制带标题的方框
// 用于显示整理计划等结构化信息
func Box(title string, lines []string) {
	width := 55

	// 绘制顶部边框
	fmt.Println(Cyan("╭" + strings.Repeat("─", width-2) + "╮"))

	// 绘制标题行（居中）
	titlePadding := (width - 4 - len(title)) / 2
	fmt.Printf("%s%s%s%s%s\n",
		Cyan("│"),
		strings.Repeat(" ", titlePadding),
		Bold(title),
		strings.Repeat(" ", width-4-titlePadding-len(title)),
		Cyan("│"))

	// 绘制标题下方分隔线
	fmt.Println(Cyan("├" + strings.Repeat("─", width-2) + "┤"))

	// 绘制内容行
	for _, line := range lines {
		padding := width - 4 - displayWidth(line)
		if padding < 0 {
			padding = 0
		}
		fmt.Printf("%s %s%s%s\n", Cyan("│"), line, strings.Repeat(" ", padding), Cyan("│"))
	}

	// 绘制底部边框
	fmt.Println(Cyan("╰" + strings.Repeat("─", width-2) + "╯"))
}

// displayWidth 计算字符串的显示宽度
// 中文字符占2个宽度，ASCII字符占1个宽度
func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		if r > 127 {
			width += 2 // 非 ASCII 字符（如中文）占2个宽度
		} else {
			width += 1 // ASCII 字符占1个宽度
		}
	}
	return width
}

// ==================== 图标函数 ====================

// SourceIcon 获取分类来源图标
// 根据来源类型返回对应的 emoji 图标
func SourceIcon(source string) string {
	switch source {
	case "memory":
		return "🧠" // 记忆来源
	case "llm":
		return "🤖" // LLM 推理
	case "rule":
		return "📋" // 规则匹配
	default:
		return "❓" // 未知来源
	}
}

// ConfidenceIcon 获取置信度图标
// 根据置信度高低返回不同的图标和颜色
func ConfidenceIcon(confidence float64) string {
	if confidence >= 0.8 {
		return Green("✓") // 高置信度：绿色勾
	} else if confidence >= 0.5 {
		return Yellow("◐") // 中置信度：黄色半圆
	}
	return Red("○") // 低置信度：红色空心圆
}

// ==================== 格式化函数 ====================

// FormatSize 格式化文件大小
// 将字节数转换为人类可读的格式（B/KB/MB/GB）
func FormatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.1f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.1f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.1f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// ==================== 交互函数 ====================

// Confirm 显示确认提示并获取用户输入
// defaultYes=true: 默认确认（直接回车确认）[Y/n]
// defaultYes=false: 默认不确认（需要明确输入y）[y/N]
func Confirm(prompt string, defaultYes bool) bool {
	var hint string
	if defaultYes {
		hint = "[Y/n]"
	} else {
		hint = "[y/N]"
	}
	fmt.Printf("%s %s: ", prompt, hint)

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if defaultYes {
		// 默认确认：空输入或 y/yes 都确认
		return input == "" || input == "y" || input == "yes"
	}
	// 默认不确认：只有 y/yes 才确认
	return input == "y" || input == "yes"
}

// ConfirmDanger 显示危险操作确认提示
// 带警告图标，默认不确认
func ConfirmDanger(prompt string) bool {
	fmt.Printf("%s %s [y/N]: ", Yellow("⚠"), prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y"
}

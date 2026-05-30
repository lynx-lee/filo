// Package organizer 文件整理模块
// 负责生成整理计划、交互审查和执行文件移动
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package organizer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lynx-lee/filo/internal/classifier"
	"github.com/lynx-lee/filo/internal/storage"
	"github.com/lynx-lee/filo/internal/ui"
)

// ==================== 常量定义 ====================

const (
	MaxDisplayFiles       = 5   // 计划显示中每个分类最多显示的文件数
	LowConfidenceThreshold = 0.7 // 低置信度阈值，低于此值需要审查
)

// ==================== 类型定义 ====================

// Plan 整理计划
// 存储分类结果和目标目录信息
type Plan struct {
	TargetDir string                           // 目标目录（整理后文件存放位置）
	Actions   map[string][]classifier.Result   // 分类动作：文件夹名 -> 文件列表
}

// TotalFiles 计算计划中的总文件数
func (p *Plan) TotalFiles() int {
	total := 0
	for _, files := range p.Actions {
		total += len(files)
	}
	return total
}

// TotalFolders 计算计划中的总分类数
func (p *Plan) TotalFolders() int {
	return len(p.Actions)
}

// ExecuteResult 执行结果统计
type ExecuteResult struct {
	Success int // 成功移动的文件数
	Errors  int // 失败的文件数
}

// ==================== 计划生成函数 ====================

// GeneratePlan 根据分类结果生成整理计划
// 将文件按分类组织到对应的目标文件夹
func GeneratePlan(results []classifier.Result, targetDir string) *Plan {
	plan := &Plan{
		TargetDir: targetDir,
		Actions:   make(map[string][]classifier.Result),
	}

	for _, r := range results {
		// 确定目标文件夹名称
		var folder string
		if r.Subcategory != "" && r.Subcategory != "其他" && r.Subcategory != "未知" {
			// 有有效子分类时，使用两级目录: 主分类/子分类
			folder = filepath.Join(r.Category, r.Subcategory)
		} else {
			// 否则只使用主分类
			folder = r.Category
		}

		// 将文件添加到对应分类
		plan.Actions[folder] = append(plan.Actions[folder], r)
	}

	return plan
}

// ==================== 计划显示函数 ====================

// PrintPlan 美观地打印整理计划
// 显示目标目录、文件数量和分类详情
func PrintPlan(plan *Plan) {
	// 显示计划概览
	lines := []string{
		fmt.Sprintf("📂 目标: %s", plan.TargetDir),
		fmt.Sprintf("📄 文件: %d 个", plan.TotalFiles()),
		fmt.Sprintf("📁 分类: %d 种", plan.TotalFolders()),
	}
	ui.Box("📋 整理计划", lines)

	// 按文件夹名排序显示
	folders := make([]string, 0, len(plan.Actions))
	for f := range plan.Actions {
		folders = append(folders, f)
	}
	sort.Strings(folders)

	// 显示每个分类下的文件
	for _, folder := range folders {
		files := plan.Actions[folder]
		fmt.Printf("\n  %s %s/ %s\n", ui.Green("📁"), ui.Bold(folder), ui.Gray(fmt.Sprintf("(%d个)", len(files))))

		// 最多显示 MaxDisplayFiles 个文件
		for i, r := range files {
			if i >= MaxDisplayFiles {
				ui.Dim("      ... 还有 %d 个文件", len(files)-MaxDisplayFiles)
				break
			}

			// 显示文件信息：置信度图标 + 来源图标 + 文件名
			icon := ui.ConfidenceIcon(r.Confidence)
			source := ui.SourceIcon(r.Source)
			fmt.Printf("      %s %s %s\n", icon, source, r.FileInfo.Name)

			// 显示分类理由（如果有）
			if r.Reasoning != "" {
				reason := r.Reasoning
				if len(reason) > 45 {
					reason = reason[:45] + "..." // 截断过长的理由
				}
				ui.Dim("         └─ %s", reason)
			}
		}
	}
	fmt.Println()
}

// ==================== 交互审查函数 ====================

// InteractiveReview 交互式审查整理计划
// 对低置信度的分类让用户确认或修改
// 返回可能被修改后的计划
func InteractiveReview(plan *Plan, clf *classifier.Classifier) *Plan {
	ui.Warning("交互审查 (y:确认 n:跳过 c:修改 q:结束)")

	reader := bufio.NewReader(os.Stdin)
	modified := false // 标记计划是否被修改

	// 遍历所有分类
	for folder, files := range plan.Actions {
		for i, r := range files {
			// 只审查低置信度的分类
			if r.Confidence < LowConfidenceThreshold {
				fmt.Println()
				ui.Warning("低置信度: %s", r.FileInfo.Name)
				ui.Info("   分类: %s/%s", r.Category, r.Subcategory)
				ui.Info("   置信度: %.0f%%", r.Confidence*100)
				ui.Dim("   理由: %s", r.Reasoning)

				// 获取用户输入
				fmt.Print("  操作 [y/n/c/q]: ")
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(strings.ToLower(input))

				switch input {
				case "q":
					goto done // 结束审查
				case "y":
					clf.Confirm(r) // 确认分类，学习规则
				case "c":
					// 修改分类
					fmt.Print("  新主分类: ")
					newCat, _ := reader.ReadString('\n')
					newCat = strings.TrimSpace(newCat)
					if newCat == "" {
						newCat = r.Category
					}

					fmt.Print("  新子分类: ")
					newSub, _ := reader.ReadString('\n')
					newSub = strings.TrimSpace(newSub)
					if newSub == "" {
						newSub = r.Subcategory
					}

					// 学习纠正结果
					clf.Correct(r, newCat, newSub)
					// 更新计划中的分类
					plan.Actions[folder][i].Category = newCat
					plan.Actions[folder][i].Subcategory = newSub
					modified = true
				}
			}
		}
	}

done:
	// 如果计划被修改，重新生成计划（重新组织文件夹结构）
	if modified {
		var all []classifier.Result
		for _, files := range plan.Actions {
			all = append(all, files...)
		}
		return GeneratePlan(all, plan.TargetDir)
	}
	return plan
}

// ==================== 执行函数 ====================

// Execute 执行整理计划
// 创建目标目录并移动文件，返回执行结果统计
// 同时记录操作日志，支持撤销功能
func Execute(plan *Plan, clf *classifier.Classifier, verbose bool) ExecuteResult {
	ui.Title("🚀", "执行整理")

	result := ExecuteResult{}

	// 生成批次 ID（用于撤销功能）
	batchID := time.Now().Format("20060102_150405")

	// 初始化数据库连接（用于记录操作日志）
	db, err := storage.NewDatabase()
	if err != nil {
		ui.Error("无法记录操作日志: %v", err)
		// 继续执行，但无法撤销
	}
	defer func() {
		if db != nil {
			db.Close()
		}
	}()

	// 遍历每个分类
	for folder, files := range plan.Actions {
		// 创建目标文件夹
		targetFolder := filepath.Join(plan.TargetDir, folder)
		os.MkdirAll(targetFolder, 0755) // Unix权限，Windows自动忽略

		// 移动文件
		for _, r := range files {
			src := r.FileInfo.Path
			dst := filepath.Join(targetFolder, r.FileInfo.Name)

			// 处理重名文件
			dst = handleDuplicate(dst)

			if verbose {
				ui.Info("移动: %s", r.FileInfo.Name)
				ui.Dim("  → %s", dst)
			}

			// 执行移动
			if err := os.Rename(src, dst); err != nil {
				result.Errors++
				if verbose {
					ui.Error("失败: %v", err)
				}
				// 记录失败的操作
				if db != nil {
					db.AddOperationLog(batchID, src, dst, r.FileInfo.Name, r.Category, r.Subcategory, "failed")
				}
			} else {
				result.Success++
				clf.Confirm(r) // 成功移动后确认分类，学习规则
				// 记录成功的操作（用于撤销）
				if db != nil {
					db.AddOperationLog(batchID, src, dst, r.FileInfo.Name, r.Category, r.Subcategory, "success")
				}
			}
		}
	}

	// 显示执行结果
	fmt.Println()
	ui.Success("成功: %d 个文件", result.Success)
	if result.Errors > 0 {
		ui.Error("失败: %d 个文件", result.Errors)
	}
	ui.Dim("批次: %s (可用 'filo undo' 撤销)", batchID)

	return result
}

// handleDuplicate 处理重名文件
// 如果目标路径已存在文件，自动添加数字后缀
// 例如: file.txt -> file_1.txt -> file_2.txt
func handleDuplicate(path string) string {
	// 文件不存在，直接返回原路径
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}

	// 分解路径
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	name := strings.TrimSuffix(filepath.Base(path), ext)

	// 尝试添加数字后缀
	for i := 1; ; i++ {
		newPath := filepath.Join(dir, fmt.Sprintf("%s_%d%s", name, i, ext))
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
	}
}

// ==================== 确认函数 ====================

// Confirm 显示确认提示
// 默认不确认（需要明确输入 y 或 yes）
func Confirm(prompt string) bool {
	return ui.Confirm(prompt, false)
}

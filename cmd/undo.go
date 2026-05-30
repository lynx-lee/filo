// Package cmd 命令行入口模块
// undo 命令：撤销文件整理操作
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/lynx-lee/filo/internal/storage"
	"github.com/lynx-lee/filo/internal/ui"
)

// undoCmd 撤销命令定义
var undoCmd = &cobra.Command{
	Use:   "undo [批次ID]",
	Short: "撤销文件整理操作",
	Long: `撤销之前的文件整理操作，将文件移回原位置。

不指定批次ID时，默认撤销最近一次操作。

示例:
  filo undo                    # 撤销最近一次整理
  filo undo 20240115_143022    # 撤销指定批次
  filo undo --list             # 查看可撤销的操作列表`,
	Run: runUndo,
}

// undo 命令行参数
var (
	listBatches bool // 是否列出可撤销的批次
)

func init() {
	// 注册 undo 子命令
	rootCmd.AddCommand(undoCmd)

	// 注册命令行标志
	undoCmd.Flags().BoolVarP(&listBatches, "list", "l", false, "列出可撤销的操作")
}

// runUndo 执行撤销操作
func runUndo(cmd *cobra.Command, args []string) {
	ui.Banner()

	// 初始化数据库
	db, err := storage.NewDatabase()
	if err != nil {
		ui.Error("无法连接数据库: %v", err)
		return
	}
	defer db.Close()

	// 列出可撤销的操作
	if listBatches {
		listUndoBatches(db)
		return
	}

	// 确定要撤销的批次
	var batchID string
	if len(args) > 0 {
		batchID = args[0]
	} else {
		// 获取最近一次操作的批次
		batchID = db.GetLatestBatch()
		if batchID == "" {
			ui.Warning("没有可撤销的操作")
			return
		}
	}

	// 执行撤销
	undoBatch(db, batchID)
}

// listUndoBatches 列出可撤销的操作批次
func listUndoBatches(db *storage.Database) {
	ui.Title("📋", "可撤销的操作")

	batches, err := db.GetRecentBatches(10)
	if err != nil || len(batches) == 0 {
		ui.Warning("没有可撤销的操作")
		return
	}

	fmt.Println()
	for i, batch := range batches {
		batchID := batch["batch_id"].(string)
		fileCount := batch["file_count"].(int)
		createdAt := batch["created_at"].(string)
		categories := batch["categories"].(string)

		// 格式化显示
		fmt.Printf("  %s %s\n", ui.Green(fmt.Sprintf("[%d]", i+1)), ui.Bold(batchID))
		fmt.Printf("      📄 %d 个文件  📅 %s\n", fileCount, createdAt)
		fmt.Printf("      📁 %s\n", ui.Gray(truncateString(categories, 50)))
		fmt.Println()
	}

	ui.Dim("使用 'filo undo <批次ID>' 撤销指定操作")
}

// undoBatch 撤销指定批次的操作
func undoBatch(db *storage.Database, batchID string) {
	ui.Title("⏪", fmt.Sprintf("撤销操作: %s", batchID))

	// 获取该批次的所有操作日志
	logs, err := db.GetBatchLogs(batchID)
	if err != nil || len(logs) == 0 {
		ui.Error("找不到批次 %s 的操作记录", batchID)
		return
	}

	// 显示将要撤销的操作
	fmt.Println()
	ui.Info("将撤销 %d 个文件的移动操作:", len(logs))
	fmt.Println()

	// 最多显示 5 个文件
	for i, log := range logs {
		if i >= 5 {
			ui.Dim("  ... 还有 %d 个文件", len(logs)-5)
			break
		}
		fmt.Printf("  %s %s\n", ui.Green("←"), log.Filename)
		ui.Dim("    从: %s", log.DestPath)
		ui.Dim("    到: %s", log.SourcePath)
	}
	fmt.Println()

	// 确认撤销
	if !ui.ConfirmDanger("确认撤销这些操作?") {
		ui.Warning("已取消")
		return
	}

	// 执行撤销
	ui.Title("🔄", "执行撤销")

	success := 0
	errors := 0
	var errorMsgs []string

	for _, log := range logs {
		// 检查目标文件是否存在
		if _, err := os.Stat(log.DestPath); os.IsNotExist(err) {
			errors++
			errorMsgs = append(errorMsgs, fmt.Sprintf("%s: 文件不存在", log.Filename))
			continue
		}

		// 确保源目录存在
		sourceDir := filepath.Dir(log.SourcePath)
		if err := os.MkdirAll(sourceDir, 0755); err != nil {
			errors++
			errorMsgs = append(errorMsgs, fmt.Sprintf("%s: 无法创建目录", log.Filename))
			continue
		}

		// 处理源路径可能已有同名文件的情况
		destPath := log.SourcePath
		if _, err := os.Stat(destPath); err == nil {
			// 源位置已有文件，添加后缀
			ext := filepath.Ext(destPath)
			base := destPath[:len(destPath)-len(ext)]
			for i := 1; ; i++ {
				newPath := fmt.Sprintf("%s_restored_%d%s", base, i, ext)
				if _, err := os.Stat(newPath); os.IsNotExist(err) {
					destPath = newPath
					break
				}
			}
		}

		// 移动文件回原位置
		if err := os.Rename(log.DestPath, destPath); err != nil {
			errors++
			errorMsgs = append(errorMsgs, fmt.Sprintf("%s: %v", log.Filename, err))
		} else {
			success++
		}
	}

	// 标记批次为已撤销
	if success > 0 {
		db.MarkBatchUndone(batchID)
	}

	// 清理空目录
	cleanEmptyDirs(logs)

	// 显示结果
	fmt.Println()
	ui.Success("成功撤销: %d 个文件", success)
	if errors > 0 {
		ui.Error("失败: %d 个文件", errors)
		if len(errorMsgs) <= 3 {
			for _, msg := range errorMsgs {
				ui.Dim("  - %s", msg)
			}
		}
	}
}

// cleanEmptyDirs 清理空目录
func cleanEmptyDirs(logs []storage.OperationLog) {
	// 收集所有涉及的目录
	dirs := make(map[string]bool)
	for _, log := range logs {
		dir := filepath.Dir(log.DestPath)
		dirs[dir] = true
	}

	// 尝试删除空目录
	for dir := range dirs {
		// 检查目录是否为空
		entries, err := os.ReadDir(dir)
		if err == nil && len(entries) == 0 {
			os.Remove(dir)
			// 尝试删除上级目录（如果也为空）
			parentDir := filepath.Dir(dir)
			parentEntries, _ := os.ReadDir(parentDir)
			if len(parentEntries) == 0 {
				os.Remove(parentDir)
			}
		}
	}
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

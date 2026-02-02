// Package scanner 文件扫描模块
// 提供目录扫描和文件统计功能
// 自动过滤隐藏文件、系统文件和已整理的文件
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package scanner

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"filo/internal/ui"
)

// ==================== 类型定义 ====================

// FileInfo 文件信息结构体
// 存储扫描到的文件的基本信息
type FileInfo struct {
	Path         string    // 文件完整路径
	Name         string    // 文件名
	Extension    string    // 扩展名（小写，带点号）
	Size         int64     // 文件大小（字节）
	ModifiedTime time.Time // 最后修改时间
	IsDir        bool      // 是否为目录
}

// skipNames 需要跳过的文件和目录名
// 包括系统文件、版本控制目录、IDE 配置等
var skipNames = map[string]bool{
	".DS_Store":    true, // macOS 系统文件
	"Thumbs.db":    true, // Windows 缩略图缓存
	"desktop.ini":  true, // Windows 文件夹配置
	"$RECYCLE.BIN": true, // Windows 回收站
	".git":         true, // Git 版本控制
	".svn":         true, // SVN 版本控制
	"__pycache__":  true, // Python 缓存
	"node_modules": true, // Node.js 依赖
	".idea":        true, // JetBrains IDE 配置
	".vscode":      true, // VS Code 配置
	".Trash":       true, // macOS 废纸篓
	".filo":        true, // Filo 数据目录
}

// ==================== 核心扫描函数 ====================

// ScanDirectory 扫描目录
// 参数:
//   - dir: 要扫描的目录路径
//   - recursive: 是否递归扫描子目录
// 返回:
//   - []FileInfo: 扫描到的文件列表
//   - error: 错误信息
func ScanDirectory(dir string, recursive bool) ([]FileInfo, error) {
	var files []FileInfo

	// 获取绝对路径
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	// 遍历目录的回调函数
	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略访问错误，继续扫描
		}

		name := info.Name()

		// 跳过隐藏文件（以点开头）
		if strings.HasPrefix(name, ".") {
			if info.IsDir() {
				return filepath.SkipDir // 跳过整个隐藏目录
			}
			return nil
		}

		// 跳过特定文件/目录
		if skipNames[name] {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 跳过已整理目录（避免重复整理）
		if strings.Contains(path, "已整理") || strings.Contains(path, "Organized") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 跳过根目录本身
		if path == absDir {
			return nil
		}

		// 非递归模式：只扫描第一层
		if !recursive {
			rel, _ := filepath.Rel(absDir, path)
			if strings.Contains(rel, string(os.PathSeparator)) {
				if info.IsDir() {
					return filepath.SkipDir // 跳过子目录
				}
				return nil // 跳过子目录中的文件
			}
		}

		// 添加文件信息到列表
		files = append(files, FileInfo{
			Path:         path,
			Name:         name,
			Extension:    strings.ToLower(filepath.Ext(name)), // 扩展名转小写
			Size:         info.Size(),
			ModifiedTime: info.ModTime(),
			IsDir:        info.IsDir(),
		})

		return nil
	}

	// 执行目录遍历
	err = filepath.Walk(absDir, walkFn)
	return files, err
}

// ==================== 统计相关类型 ====================

// Statistics 文件统计信息
type Statistics struct {
	TotalFiles int               // 文件总数
	TotalDirs  int               // 目录总数
	TotalSize  int64             // 总大小（字节）
	ExtStats   map[string]ExtStat // 按扩展名统计
}

// ExtStat 单个扩展名的统计
type ExtStat struct {
	Count int   // 文件数量
	Size  int64 // 总大小
}

// ==================== 统计函数 ====================

// GetStatistics 获取文件列表的统计信息
// 统计文件数、目录数、总大小和按扩展名分布
func GetStatistics(files []FileInfo) Statistics {
	stats := Statistics{
		ExtStats: make(map[string]ExtStat),
	}

	for _, f := range files {
		if f.IsDir {
			stats.TotalDirs++
			continue
		}

		stats.TotalFiles++
		stats.TotalSize += f.Size

		// 按扩展名统计
		ext := f.Extension
		if ext == "" {
			ext = "(无扩展名)"
		}

		es := stats.ExtStats[ext]
		es.Count++
		es.Size += f.Size
		stats.ExtStats[ext] = es
	}

	return stats
}

// PrintStatistics 打印统计信息
// 以美观的格式显示文件统计
func PrintStatistics(files []FileInfo) {
	stats := GetStatistics(files)

	ui.Title("📊", "文件统计")
	ui.Divider()

	// 基本统计
	ui.Info("📁 文件夹: %d 个", stats.TotalDirs)
	ui.Info("📄 文件:   %d 个", stats.TotalFiles)
	ui.Info("💾 总大小: %s", ui.FormatSize(stats.TotalSize))

	// 按扩展名统计（如果有数据）
	if len(stats.ExtStats) > 0 {
		ui.Info("")
		ui.Info("按类型统计:")

		// 按数量排序
		type kv struct {
			Ext  string
			Stat ExtStat
		}
		var sorted []kv
		for k, v := range stats.ExtStats {
			sorted = append(sorted, kv{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Stat.Count > sorted[j].Stat.Count // 按数量降序
		})

		// 显示前12种类型
		for i, kv := range sorted {
			if i >= 12 {
				ui.Dim("  ... 还有 %d 种类型", len(sorted)-12)
				break
			}
			ui.Info("  %-12s %4d 个  %10s", kv.Ext, kv.Stat.Count, ui.FormatSize(kv.Stat.Size))
		}
	}
}

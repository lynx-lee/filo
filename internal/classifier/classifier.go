// Package classifier 智能分类模块
// 实现文件的智能分类功能，结合记忆系统和 LLM 推理
// 采用"记忆优先，AI兜底"的混合策略
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package classifier

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/schollz/progressbar/v3"

	"filo/internal/config"
	"filo/internal/llm"
	"filo/internal/memory"
	"filo/internal/scanner"
	"filo/internal/storage"
	"filo/internal/ui"
)

// ==================== 类型定义 ====================

// Result 分类结果
// 存储单个文件的分类信息
type Result struct {
	FileInfo    scanner.FileInfo // 文件信息
	Category    string           // 主分类
	Subcategory string           // 子分类
	Confidence  float64          // 置信度（0-1）
	Reasoning   string           // 分类理由
	Source      string           // 来源: memory（记忆）, llm（AI推理）
	Keywords    []string         // 提取的关键词
}

// Classifier 分类器
// 整合记忆系统和 LLM 进行智能分类
type Classifier struct {
	memory     *memory.Memory   // 记忆系统
	llm        *llm.Client      // LLM 客户端
	cfg        *config.Config   // 配置
	db         *storage.Database // 数据库（用于记录模型性能）
	batchID    string           // 当前批次 ID
	modelStats struct {         // 模型性能统计
		StartTime     time.Time
		TotalTimeMs   int64
		FileCount     int
		TotalConfidence float64
	}
}

// ==================== 构造函数 ====================

// NewClassifier 创建分类器
// 初始化记忆系统和 LLM 客户端
func NewClassifier() (*Classifier, error) {
	mem, err := memory.NewMemory()
	if err != nil {
		return nil, err
	}

	db, err := storage.NewDatabase()
	if err != nil {
		mem.Close()
		return nil, err
	}

	// 生成批次 ID
	batchID := time.Now().Format("20060102_150405")

	return &Classifier{
		memory:  mem,
		llm:     llm.NewClient(),
		cfg:     config.Get(),
		db:      db,
		batchID: batchID,
	}, nil
}

// Close 关闭分类器
// 释放记忆系统和数据库资源
func (c *Classifier) Close() error {
	// 保存模型性能统计
	if c.modelStats.FileCount > 0 {
		avgConfidence := c.modelStats.TotalConfidence / float64(c.modelStats.FileCount)
		c.db.AddModelStats(c.cfg.LLMModel, c.batchID, c.modelStats.FileCount, c.modelStats.TotalTimeMs, avgConfidence)
	}

	c.db.Close()
	return c.memory.Close()
}

// GetBatchID 获取当前批次 ID
func (c *Classifier) GetBatchID() string {
	return c.batchID
}

// ==================== 核心分类方法 ====================

// Classify 分类文件列表
// 流程：
// 1. 先从记忆系统查询（快速）
// 2. 未命中的文件使用 LLM 分类（准确）
// 3. 学习 LLM 分类结果
func (c *Classifier) Classify(files []scanner.FileInfo, verbose bool) ([]Result, error) {
	var memoryResults []Result  // 记忆命中的结果
	var llmNeeded []scanner.FileInfo // 需要 LLM 分类的文件

	ui.Title("🧠", "检查学习记忆")

	// ========== 阶段1: 记忆查询 ==========
	for _, f := range files {
		if f.IsDir {
			continue // 跳过目录
		}

		// 查询记忆系统
		match := c.memory.Query(f.Name)
		if match != nil && match.Confidence >= c.cfg.SimilarityThreshold {
			// 记忆命中，添加到结果
			memoryResults = append(memoryResults, Result{
				FileInfo:    f,
				Category:    match.Category,
				Subcategory: match.Subcategory,
				Confidence:  match.Confidence,
				Reasoning:   match.Reasoning,
				Source:      "memory",
			})

			if verbose {
				ui.Success("%s → %s (%s)", f.Name, match.Category, match.Source)
			}
		} else {
			// 记忆未命中，加入待分类队列
			llmNeeded = append(llmNeeded, f)
		}
	}

	if len(memoryResults) > 0 {
		ui.Success("从记忆获取 %d 个分类", len(memoryResults))
	}

	// ========== 阶段2: LLM 分类 ==========
	var llmResults []Result
	if len(llmNeeded) > 0 {
		// 显示当前使用的模型
		ui.Title("🤖", fmt.Sprintf("AI分类 %d 个文件", len(llmNeeded)))
		ui.Info("模型: %s", ui.Bold(c.cfg.LLMModel))

		// 获取已学习的规则供 LLM 参考
		rules := c.memory.GetLearnedRules(30)

		// 记录开始时间
		c.modelStats.StartTime = time.Now()

		// 调用 LLM 进行分类
		var err error
		llmResults, err = c.classifyWithLLM(llmNeeded, rules, verbose)
		if err != nil {
			ui.Warning("部分文件分类失败: %v", err)
		}

		// 记录结束时间和统计
		elapsed := time.Since(c.modelStats.StartTime)
		c.modelStats.TotalTimeMs = elapsed.Milliseconds()
		c.modelStats.FileCount = len(llmResults)

		// 计算总置信度
		for _, r := range llmResults {
			c.modelStats.TotalConfidence += r.Confidence
		}

		// 显示性能信息
		if len(llmResults) > 0 {
			avgTime := float64(c.modelStats.TotalTimeMs) / float64(len(llmResults))
			avgConf := c.modelStats.TotalConfidence / float64(len(llmResults))
			ui.Dim("耗时: %.1fs (%.0fms/文件) | 平均置信度: %.0f%%", 
				elapsed.Seconds(), avgTime, avgConf*100)
		}

		// 学习 LLM 分类结果
		if c.cfg.EnableLearning {
			for _, r := range llmResults {
				c.memory.Learn(r.FileInfo.Name, r.Category, r.Subcategory, "llm", r.Confidence, false)
			}
		}
	}

	// ========== 阶段3: 合并结果 ==========
	results := append(memoryResults, llmResults...)

	// 按原始文件顺序排序（使用标准库排序，O(n log n)）
	order := make(map[string]int)
	for i, f := range files {
		order[f.Path] = i
	}
	sort.SliceStable(results, func(i, j int) bool {
		return order[results[i].FileInfo.Path] < order[results[j].FileInfo.Path]
	})

	return results, nil
}

// classifyWithLLM 使用 LLM 批量分类文件
// 将文件分批发送给 LLM，显示进度条
func (c *Classifier) classifyWithLLM(files []scanner.FileInfo, rules []map[string]string, verbose bool) ([]Result, error) {
	var results []Result
	batchSize := c.cfg.BatchSize // 每批处理的文件数

	// 创建进度条
	bar := progressbar.NewOptions(len(files),
		progressbar.OptionSetDescription("  分类中"),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerHead:    "█",
			SaucerPadding: "░",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionShowCount(),
	)

	// 分批处理
	for i := 0; i < len(files); i += batchSize {
		end := i + batchSize
		if end > len(files) {
			end = len(files)
		}

		batch := files[i:end]

		// 准备批次数据
		batchData := make([]map[string]interface{}, len(batch))
		for j, f := range batch {
			batchData[j] = map[string]interface{}{
				"name":      f.Name,
				"extension": f.Extension,
				"size":      f.Size,
			}
		}

		// 调用 LLM API（带超时）
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		resp, err := c.llm.ClassifyFiles(ctx, batchData, rules)
		cancel()

		if err != nil {
			// LLM 调用失败，使用默认分类
			for _, f := range batch {
				results = append(results, Result{
					FileInfo:    f,
					Category:    "未分类",
					Subcategory: "其他",
					Confidence:  0,
					Reasoning:   fmt.Sprintf("分类失败: %v", err),
					Source:      "error",
				})
			}
		} else {
			// 解析 LLM 返回的分类结果
			classifications, _ := resp["classifications"].([]interface{})
			for j, cls := range classifications {
				if j >= len(batch) {
					break
				}
				clsMap, _ := cls.(map[string]interface{})
				if clsMap == nil {
					continue
				}

				results = append(results, Result{
					FileInfo:    batch[j],
					Category:    getString(clsMap, "category", "未分类"),
					Subcategory: getString(clsMap, "subcategory", "其他"),
					Confidence:  getFloat(clsMap, "confidence", 0.5),
					Reasoning:   getString(clsMap, "reasoning", ""),
					Source:      "llm",
					Keywords:    getStringSlice(clsMap, "keywords"),
				})
			}
		}

		bar.Add(len(batch)) // 更新进度条
	}

	fmt.Println() // 进度条结束后换行
	return results, nil
}

// ==================== 学习方法 ====================

// Confirm 确认分类
// 用户确认后调用，将分类结果标记为已确认并学习规则
func (c *Classifier) Confirm(r Result) {
	c.memory.Learn(r.FileInfo.Name, r.Category, r.Subcategory, r.Source, r.Confidence, true)
	// 更新模型准确度（仅 LLM 分类需要统计）
	if r.Source == "llm" {
		c.db.UpdateModelAccuracy(c.batchID, 1, 0)
	}
}

// Correct 纠正分类
// 用户修改分类后调用，学习纠正后的结果
func (c *Classifier) Correct(r Result, newCat, newSub string) {
	c.memory.LearnFromCorrection(r.FileInfo.Name, r.Category, newCat, r.Subcategory, newSub)
	// 更新模型准确度（仅 LLM 分类需要统计）
	if r.Source == "llm" {
		c.db.UpdateModelAccuracy(c.batchID, 0, 1)
	}
}

// ==================== 统计方法 ====================

// GetStatistics 获取统计信息
// 返回分类系统的各项统计数据
func (c *Classifier) GetStatistics() (map[string]interface{}, error) {
	return c.memory.GetStatistics()
}

// ==================== 辅助函数 ====================

// getString 从 map 中安全获取字符串值
func getString(m map[string]interface{}, key, def string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return def
}

// getFloat 从 map 中安全获取浮点数值
func getFloat(m map[string]interface{}, key string, def float64) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return def
}

// getStringSlice 从 map 中安全获取字符串数组
func getStringSlice(m map[string]interface{}, key string) []string {
	if v, ok := m[key].([]interface{}); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

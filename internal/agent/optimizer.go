// Package agent 智能体模块 - 优化器
// 负责对分类结果进行后处理优化，提升整体质量
// 包括：一致性检查、冲突解决、边界优化
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lynx-lee/filo/internal/classifier"
	"github.com/lynx-lee/filo/internal/config"
)

// OptimizerAgent 优化器智能体
// 对分类结果进行后处理，提升一致性和准确性
type OptimizerAgent struct {
	*BaseAgent
	cfg *config.Config
}

// OptimizationRule 优化规则
type OptimizationRule struct {
	Name        string                   // 规则名称
	Description string                   // 规则描述
	Priority    int                      // 优先级 (1-10)
	Apply       func([]classifier.Result) []classifier.Result // 应用规则
}

// ConsistencyIssue 一致性问题
type ConsistencyIssue struct {
	Type        string   // 问题类型: conflict, inconsistency, outlier
	Filenames   []string // 相关文件
	Categories  []string // 涉及的分类
	Description string   // 问题描述
	Suggestion  string   // 建议解决方案
}

// NewOptimizerAgent 创建优化器智能体
func NewOptimizerAgent() *OptimizerAgent {
	return &OptimizerAgent{
		BaseAgent: NewBaseAgent("optimizer"),
		cfg:       config.Get(),
	}
}

// Initialize 初始化优化器
func (o *OptimizerAgent) Initialize(ctx context.Context, config map[string]interface{}) error {
	o.setStatus(StatusWorking)
	defer o.setStatus(StatusIdle)
	return nil
}

// Execute 执行优化任务
func (o *OptimizerAgent) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	startTime := time.Now()
	o.IncrementTaskCount()
	o.setStatus(StatusWorking)
	defer func() {
		o.setStatus(StatusComplete)
	}()

	// 解析任务数据
	results, ok := task.Data["classification_results"].([]classifier.Result)
	if !ok {
		o.IncrementErrorCount()
		return CreateTaskResult(task.ID, false, nil,
			fmt.Errorf("invalid task data: missing classification results"), time.Since(startTime)), nil
	}

	// 应用优化规则
	optimizedResults := o.applyOptimizationRules(results)

	// 检测一致性问题
	issues := o.detectConsistencyIssues(optimizedResults)

	resultData := map[string]interface{}{
		"optimized_results": optimizedResults,
		"issues_found":      len(issues),
		"issues":            issues,
		"improvement_rate":  o.calculateImprovementRate(results, optimizedResults),
	}

	duration := time.Since(startTime)
	return CreateTaskResult(task.ID, true, resultData, nil, duration), nil
}

// applyOptimizationRules 应用所有优化规则
func (o *OptimizerAgent) applyOptimizationRules(results []classifier.Result) []classifier.Result {
	rules := o.getOptimizationRules()

	optimized := results
	for _, rule := range rules {
		optimized = rule.Apply(optimized)
	}

	return optimized
}

// getOptimizationRules 获取优化规则列表
func (o *OptimizerAgent) getOptimizationRules() []OptimizationRule {
	return []OptimizationRule{
		{
			Name:        "相似文件归类一致性",
			Description: "确保相似文件名归入相同类别",
			Priority:    9,
			Apply:       o.ruleSimilarFilesConsistency,
		},
		{
			Name:        "低置信度重新评估",
			Description: "对低置信度结果标记为需要人工审核",
			Priority:    8,
			Apply:       o.ruleLowConfidenceReview,
		},
		{
			Name:        "异常值检测",
			Description: "检测并标记与同类型文件分类不一致的异常值",
			Priority:    7,
			Apply:       o.ruleOutlierDetection,
		},
		{
			Name:        "分类层级规范化",
			Description: "统一分类命名规范",
			Priority:    6,
			Apply:       o.ruleCategoryNormalization,
		},
	}
}

// ruleSimilarFilesConsistency 相似文件归类一致性规则
func (o *OptimizerAgent) ruleSimilarFilesConsistency(results []classifier.Result) []classifier.Result {
	// 构建文件名到结果的映射
	type fileGroup struct {
		prefix    string
		results   []*classifier.Result
	}

	groups := make(map[string]*fileGroup)

	// 按文件名前缀分组（前3个字符或第一个下划线前）
	for i := range results {
		name := results[i].FileInfo.Name
		prefix := o.extractPrefix(name)

		if _, exists := groups[prefix]; !exists {
			groups[prefix] = &fileGroup{prefix: prefix}
		}
		groups[prefix].results = append(groups[prefix].results, &results[i])
	}

	// 对每组应用多数投票
	for _, group := range groups {
		if len(group.results) < 2 {
			continue
		}

		// 统计分类分布
		categoryCount := make(map[string]int)
		for _, r := range group.results {
			categoryCount[r.Category]++
		}

		// 找到最多的分类
		maxCategory := ""
		maxCount := 0
		for cat, count := range categoryCount {
			if count > maxCount {
				maxCount = count
				maxCategory = cat
			}
		}

		// 如果有多数分类，统一其他文件
		if maxCount > len(group.results)/2 {
			for _, r := range group.results {
				if r.Category != maxCategory {
					r.Metadata["optimized"] = "true"
					r.Metadata["original_category"] = r.Category
					r.Category = maxCategory
					r.Reasoning += fmt.Sprintf(" [优化: 与 %d 个相似文件保持一致]", maxCount-1)
				}
			}
		}
	}

	return results
}

// ruleLowConfidenceReview 低置信度重新评估规则
func (o *OptimizerAgent) ruleLowConfidenceReview(results []classifier.Result) []classifier.Result {
	for i := range results {
		if results[i].Confidence < o.cfg.ConfidenceThreshold {
			results[i].Metadata["needs_review"] = "true"
			results[i].Metadata["review_reason"] = fmt.Sprintf("置信度 %.2f 低于阈值 %.2f",
				results[i].Confidence, o.cfg.ConfidenceThreshold)
		}
	}
	return results
}

// ruleOutlierDetection 异常值检测规则
func (o *OptimizerAgent) ruleOutlierDetection(results []classifier.Result) []classifier.Result {
	// 按扩展名分组
	extGroups := make(map[string][]*classifier.Result)
	for i := range results {
		ext := o.getFileExtension(results[i].FileInfo.Name)
		extGroups[ext] = append(extGroups[ext], &results[i])
	}

	// 检测每个组内的异常值
	for _, group := range extGroups {
		if len(group) < 5 {
			continue // 样本太少，不检测
		}

		// 统计分类分布
		categoryCount := make(map[string]int)
		for _, r := range group {
			categoryCount[r.Category]++
		}

		// 找到主导分类
		dominantCategory := ""
		dominantCount := 0
		for cat, count := range categoryCount {
			if count > dominantCount {
				dominantCount = count
				dominantCategory = cat
			}
		}

		// 标记异常值（与主导分类不同且占比 < 10%）
		for _, r := range group {
			if r.Category != dominantCategory {
				count := categoryCount[r.Category]
				if float64(count)/float64(len(group)) < 0.1 {
					r.Metadata["is_outlier"] = "true"
					r.Metadata["outlier_reason"] = fmt.Sprintf(
						"%s 文件中 %.0f%% 归类为 %s，但该文件归类为 %s",
						o.getFileExtension(r.FileInfo.Name),
						float64(dominantCount)/float64(len(group))*100,
						dominantCategory,
						r.Category,
					)
				}
			}
		}
	}

	return results
}

// ruleCategoryNormalization 分类层级规范化规则
func (o *OptimizerAgent) ruleCategoryNormalization(results []classifier.Result) []classifier.Result {
	// 标准化常见分类名称
	categoryMap := map[string]string{
		"doc":         "文档",
		"document":    "文档",
		"documents":   "文档",
		"image":       "图片",
		"images":      "图片",
		"photo":       "图片",
		"photos":      "图片",
		"video":       "视频",
		"videos":      "视频",
		"code":        "代码",
		"source":      "代码",
		"archive":     "压缩包",
		"archives":    "压缩包",
		"compressed":  "压缩包",
	}

	for i := range results {
		cat := strings.ToLower(results[i].Category)
		if normalized, exists := categoryMap[cat]; exists {
			results[i].Metadata["original_category"] = results[i].Category
			results[i].Category = normalized
			results[i].Metadata["normalized"] = "true"
		}
	}

	return results
}

// detectConsistencyIssues 检测一致性问题
func (o *OptimizerAgent) detectConsistencyIssues(results []classifier.Result) []ConsistencyIssue {
	issues := make([]ConsistencyIssue, 0)

	// 检测冲突：相同文件名但不同分类
	nameCategories := make(map[string]map[string]bool)
	for _, r := range results {
		if _, exists := nameCategories[r.FileInfo.Name]; !exists {
			nameCategories[r.FileInfo.Name] = make(map[string]bool)
		}
		nameCategories[r.FileInfo.Name][r.Category] = true
	}

	for name, categories := range nameCategories {
		if len(categories) > 1 {
			cats := make([]string, 0, len(categories))
			for cat := range categories {
				cats = append(cats, cat)
			}
			issues = append(issues, ConsistencyIssue{
				Type:        "conflict",
				Filenames:   []string{name},
				Categories:  cats,
				Description: fmt.Sprintf("文件 %s 被分到多个类别", name),
				Suggestion:  "检查分类规则或使用 LLM 重新分类",
			})
		}
	}

	return issues
}

// calculateImprovementRate 计算优化改进率
func (o *OptimizerAgent) calculateImprovementRate(original, optimized []classifier.Result) float64 {
	if len(original) == 0 {
		return 0
	}

	optimizedCount := 0
	for i := range optimized {
		if optimized[i].Metadata["optimized"] == "true" ||
			optimized[i].Metadata["normalized"] == "true" {
			optimizedCount++
		}
	}

	return float64(optimizedCount) / float64(len(original))
}

// extractPrefix 提取文件名前缀
func (o *OptimizerAgent) extractPrefix(filename string) string {
	// 尝试按下划线分割
	if idx := strings.Index(filename, "_"); idx > 0 && idx <= 10 {
		return filename[:idx]
	}

	// 尝试按连字符分割
	if idx := strings.Index(filename, "-"); idx > 0 && idx <= 10 {
		return filename[:idx]
	}

	// 取前3个字符
	if len(filename) >= 3 {
		return filename[:3]
	}

	return filename
}

// getFileExtension 获取文件扩展名
func (o *OptimizerAgent) getFileExtension(filename string) string {
	idx := strings.LastIndex(filename, ".")
	if idx == -1 {
		return ""
	}
	return strings.ToLower(filename[idx:])
}

// GetMetrics 获取优化器指标
func (o *OptimizerAgent) GetMetrics() map[string]interface{} {
	metrics := o.BaseAgent.GetMetrics()
	metrics["optimization_rules_count"] = len(o.getOptimizationRules())
	return metrics
}

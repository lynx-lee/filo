// Package agent 智能体模块 - 学习器
// 负责从用户反馈和历史数据中学习，持续优化分类策略
// 实现增量学习和自适应调整
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lynx-lee/filo/internal/classifier"
	"github.com/lynx-lee/filo/internal/config"
	"github.com/lynx-lee/filo/internal/storage"
)

// LearnerAgent 学习器智能体
// 从历史分类结果和用户反馈中学习，优化分类策略
type LearnerAgent struct {
	*BaseAgent
	cfg       *config.Config
	db        *storage.Database
	mu        sync.RWMutex
	learningData *LearningData
}

// LearningData 学习数据结构
type LearningData struct {
	UserFeedbacks    []UserFeedback     `json:"user_feedbacks"`    // 用户反馈
	CategoryPatterns map[string]Pattern `json:"category_patterns"` // 分类模式
	LearningMetrics  LearningMetrics    `json:"learning_metrics"`  // 学习指标
	LastUpdated      time.Time          `json:"last_updated"`      // 最后更新时间
}

// UserFeedback 用户反馈
type UserFeedback struct {
	Filename         string    `json:"filename"`          // 文件名
	OriginalCategory string    `json:"original_category"` // 原始分类
	CorrectCategory  string    `json:"correct_category"`  // 正确分类
	Confidence       float64   `json:"confidence"`        // 原始置信度
	Timestamp        time.Time `json:"timestamp"`         // 反馈时间
	FeedbackType     string    `json:"feedback_type"`     // 反馈类型: correction, confirmation, rejection
}

// Pattern 分类模式
type Pattern struct {
	Category     string            `json:"category"`      // 分类名称
	Keywords     []string          `json:"keywords"`      // 关键词
	FileExtensions []string        `json:"file_extensions"` // 文件扩展名
	SampleCount  int               `json:"sample_count"`  // 样本数量
	AvgConfidence float64          `json:"avg_confidence"` // 平均置信度
	LastSeen     time.Time         `json:"last_seen"`     // 最后出现时间
}

// LearningMetrics 学习指标
type LearningMetrics struct {
	TotalFeedbacks    int     `json:"total_feedbacks"`     // 总反馈数
	Corrections       int     `json:"corrections"`         // 修正次数
	AccuracyRate      float64 `json:"accuracy_rate"`       // 准确率
	ImprovementTrend  float64 `json:"improvement_trend"`   // 改进趋势
	ActiveCategories  int     `json:"active_categories"`   // 活跃分类数
}

// NewLearnerAgent 创建学习器智能体
func NewLearnerAgent() (*LearnerAgent, error) {
	db, err := storage.NewDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	cfg := config.Get()

	return &LearnerAgent{
		BaseAgent: NewBaseAgent("learner"),
		cfg:       cfg,
		db:        db,
		learningData: &LearningData{
			UserFeedbacks:    make([]UserFeedback, 0),
			CategoryPatterns: make(map[string]Pattern),
			LearningMetrics: LearningMetrics{
				TotalFeedbacks:   0,
				Corrections:      0,
				AccuracyRate:     0,
				ImprovementTrend: 0,
				ActiveCategories: 0,
			},
			LastUpdated: time.Now(),
		},
	}, nil
}

// Initialize 初始化学习器
func (l *LearnerAgent) Initialize(ctx context.Context, config map[string]interface{}) error {
	l.setStatus(StatusWorking)
	defer l.setStatus(StatusIdle)

	// 加载历史学习数据
	if err := l.loadLearningData(); err != nil {
		// 如果加载失败，使用空数据继续
		fmt.Printf("Warning: Failed to load learning data: %v\n", err)
	}

	return nil
}

// Execute 执行学习任务
func (l *LearnerAgent) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	startTime := time.Now()
	l.IncrementTaskCount()
	l.setStatus(StatusWorking)
	defer func() {
		l.setStatus(StatusComplete)
	}()

	// 解析任务数据
	taskType, _ := task.Data["task_type"].(string)

	var resultData map[string]interface{}
	var err error

	switch taskType {
	case "learn_from_results":
		resultData, err = l.learnFromResults(task)
	case "learn_from_feedback":
		resultData, err = l.learnFromFeedback(task)
	case "update_patterns":
		resultData, err = l.updateCategoryPatterns(task)
	default:
		err = fmt.Errorf("unknown task type: %s", taskType)
	}

	if err != nil {
		l.IncrementErrorCount()
		return CreateTaskResult(task.ID, false, nil, err, time.Since(startTime)), nil
	}

	duration := time.Since(startTime)
	return CreateTaskResult(task.ID, true, resultData, nil, duration), nil
}

// learnFromResults 从分类结果中学习
func (l *LearnerAgent) learnFromResults(task *Task) (map[string]interface{}, error) {
	results, ok := task.Data["classification_results"].([]classifier.Result)
	if !ok {
		return nil, fmt.Errorf("invalid task data: missing classification results")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	learnedCount := 0

	// 从高置信度结果中提取模式
	for _, result := range results {
		if result.Confidence >= 0.85 { // 只学习高置信度的结果
			l.updatePattern(result.Category, result.FileInfo.Name, result.Confidence)
			learnedCount++
		}
	}

	// 更新学习指标
	l.learningData.LearningMetrics.ActiveCategories = len(l.learningData.CategoryPatterns)
	l.learningData.LastUpdated = time.Now()

	// 保存学习数据
	if err := l.saveLearningData(); err != nil {
		return nil, fmt.Errorf("failed to save learning data: %w", err)
	}

	return map[string]interface{}{
		"learned_patterns": learnedCount,
		"total_patterns":   len(l.learningData.CategoryPatterns),
		"active_categories": l.learningData.LearningMetrics.ActiveCategories,
	}, nil
}

// learnFromFeedback 从用户反馈中学习
func (l *LearnerAgent) learnFromFeedback(task *Task) (map[string]interface{}, error) {
	feedback, ok := task.Data["feedback"].(UserFeedback)
	if !ok {
		return nil, fmt.Errorf("invalid task data: missing feedback")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 记录反馈
	l.learningData.UserFeedbacks = append(l.learningData.UserFeedbacks, feedback)
	l.learningData.LearningMetrics.TotalFeedbacks++

	// 如果是修正反馈，更新模式
	if feedback.FeedbackType == "correction" {
		l.learningData.LearningMetrics.Corrections++

		// 降低原分类的权重
		if pattern, exists := l.learningData.CategoryPatterns[feedback.OriginalCategory]; exists {
			pattern.SampleCount--
			l.learningData.CategoryPatterns[feedback.OriginalCategory] = pattern
		}

		// 增加正确分类的权重
		l.updatePattern(feedback.CorrectCategory, feedback.Filename, 1.0)
	}

	// 更新准确率
	if l.learningData.LearningMetrics.TotalFeedbacks > 0 {
		corrections := l.learningData.LearningMetrics.Corrections
		total := l.learningData.LearningMetrics.TotalFeedbacks
		l.learningData.LearningMetrics.AccuracyRate = 1.0 - (float64(corrections) / float64(total))
	}

	// 保存学习数据
	if err := l.saveLearningData(); err != nil {
		return nil, fmt.Errorf("failed to save learning data: %w", err)
	}

	return map[string]interface{}{
		"feedback_recorded": true,
		"total_feedbacks":   l.learningData.LearningMetrics.TotalFeedbacks,
		"accuracy_rate":     l.learningData.LearningMetrics.AccuracyRate,
	}, nil
}

// updateCategoryPatterns 更新分类模式
func (l *LearnerAgent) updateCategoryPatterns(task *Task) (map[string]interface{}, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 清理过期模式（30天未更新）
	expiredPatterns := 0
	cutoff := time.Now().AddDate(0, 0, -30)

	for category, pattern := range l.learningData.CategoryPatterns {
		if pattern.LastSeen.Before(cutoff) && pattern.SampleCount < 3 {
			delete(l.learningData.CategoryPatterns, category)
			expiredPatterns++
		}
	}

	// 保存更新后的数据
	if err := l.saveLearningData(); err != nil {
		return nil, fmt.Errorf("failed to save learning data: %w", err)
	}

	return map[string]interface{}{
		"expired_patterns_removed": expiredPatterns,
		"remaining_patterns":       len(l.learningData.CategoryPatterns),
	}, nil
}

// updatePattern 更新单个分类模式
func (l *LearnerAgent) updatePattern(category, filename string, confidence float64) {
	pattern, exists := l.learningData.CategoryPatterns[category]
	if !exists {
		pattern = Pattern{
			Category:       category,
			Keywords:       make([]string, 0),
			FileExtensions: make([]string, 0),
			SampleCount:    0,
			AvgConfidence:  0,
		}
	}

	// 更新样本统计
	pattern.SampleCount++
	pattern.AvgConfidence = (pattern.AvgConfidence*float64(pattern.SampleCount-1) + confidence) / float64(pattern.SampleCount)
	pattern.LastSeen = time.Now()

	// 提取并添加关键词
	keywords := l.extractKeywords(filename)
	for _, kw := range keywords {
		if !l.contains(pattern.Keywords, kw) {
			pattern.Keywords = append(pattern.Keywords, kw)
		}
	}

	// 添加文件扩展名
	ext := l.getFileExtension(filename)
	if ext != "" && !l.contains(pattern.FileExtensions, ext) {
		pattern.FileExtensions = append(pattern.FileExtensions, ext)
	}

	l.learningData.CategoryPatterns[category] = pattern
}

// extractKeywords 从文件名提取关键词
func (l *LearnerAgent) extractKeywords(filename string) []string {
	// 移除扩展名
	name := filename
	if idx := filepath.Ext(filename); idx != "" {
		name = filename[:len(filename)-len(idx)]
	}

	// 按常见分隔符分割
	keywords := make([]string, 0)
	for _, sep := range []string{"_", "-", " ", "."} {
		parts := splitString(name, sep)
		for _, part := range parts {
			if len(part) >= 2 { // 忽略太短的词
				keywords = append(keywords, part)
			}
		}
	}

	return keywords
}

// splitString 按分隔符分割字符串
func splitString(s, sep string) []string {
	result := make([]string, 0)
	current := ""
	for _, ch := range s {
		if string(ch) == sep {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// contains 检查切片是否包含元素
func (l *LearnerAgent) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// getFileExtension 获取文件扩展名
func (l *LearnerAgent) getFileExtension(filename string) string {
	ext := filepath.Ext(filename)
	if ext != "" {
		return ext[1:] // 移除前导点
	}
	return ""
}

// loadLearningData 加载学习数据
func (l *LearnerAgent) loadLearningData() error {
	dataPath := filepath.Join(l.cfg.DataDir, "learning_data.json")

	data, err := os.ReadFile(dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，使用默认数据
		}
		return err
	}

	return json.Unmarshal(data, l.learningData)
}

// saveLearningData 保存学习数据
func (l *LearnerAgent) saveLearningData() error {
	dataPath := filepath.Join(l.cfg.DataDir, "learning_data.json")

	data, err := json.MarshalIndent(l.learningData, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(dataPath, data, 0644)
}

// GetLearnedPatterns 获取学习到的模式
func (l *LearnerAgent) GetLearnedPatterns() map[string]Pattern {
	l.mu.RLock()
	defer l.mu.RUnlock()

	patterns := make(map[string]Pattern)
	for k, v := range l.learningData.CategoryPatterns {
		patterns[k] = v
	}

	return patterns
}

// GetLearningMetrics 获取学习指标
func (l *LearnerAgent) GetLearningMetrics() LearningMetrics {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.learningData.LearningMetrics
}

// GetMetrics 获取学习器指标
func (l *LearnerAgent) GetMetrics() map[string]interface{} {
	metrics := l.BaseAgent.GetMetrics()
	metrics["total_patterns"] = len(l.learningData.CategoryPatterns)
	metrics["total_feedbacks"] = l.learningData.LearningMetrics.TotalFeedbacks
	metrics["accuracy_rate"] = l.learningData.LearningMetrics.AccuracyRate
	return metrics
}

// Close 关闭学习器
func (l *LearnerAgent) Close() error {
	if l.db != nil {
		return l.db.Close()
	}
	return nil
}

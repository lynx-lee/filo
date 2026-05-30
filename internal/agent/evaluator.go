// Package agent 智能体模块 - 评估器
// 负责分类结果的质量评估、置信度校验和反馈学习
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/lynx-lee/filo/internal/classifier"
	"github.com/lynx-lee/filo/internal/config"
)

// EvaluatorAgent 评估器智能体
// 对分类结果进行质量评估，提供反馈和改进建议
type EvaluatorAgent struct {
	*BaseAgent
	cfg *config.Config
	evaluationHistory []EvaluationResult // 评估历史
}

// EvaluationResult 评估结果
type EvaluationResult struct {
	ResultID      string    // 结果 ID
	Filename      string    // 文件名
	Category      string    // 分类
	Confidence    float64   // 置信度
	QualityScore  float64   // 质量评分 (0-1)
	Issues        []string  // 发现的问题
	Suggestions   []string  // 改进建议
	EvaluatedAt   time.Time // 评估时间
}

// QualityMetrics 质量指标
type QualityMetrics struct {
	AverageConfidence float64   // 平均置信度
	LowConfidenceCount int      // 低置信度数量
	Distribution      map[string]int // 分类分布
	TopIssues         []string // 常见问题
	OverallScore      float64  // 总体评分
}

// NewEvaluatorAgent 创建评估器智能体
func NewEvaluatorAgent() *EvaluatorAgent {
	return &EvaluatorAgent{
		BaseAgent: NewBaseAgent(EvaluatorAgentType),
		cfg:       config.Get(),
		evaluationHistory: make([]EvaluationResult, 0),
	}
}

// Initialize 初始化评估器
func (e *EvaluatorAgent) Initialize(ctx context.Context, config map[string]interface{}) error {
	e.setStatus(StatusWorking)
	defer e.setStatus(StatusIdle)
	
	// 可以从配置加载阈值等参数
	if threshold, ok := config["confidence_threshold"].(float64); ok {
		e.cfg.ConfidenceThreshold = threshold
	}
	
	return nil
}

// Execute 执行评估任务
func (e *EvaluatorAgent) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	startTime := time.Now()
	e.IncrementTaskCount()
	e.setStatus(StatusWorking)
	defer func() {
		e.setStatus(StatusComplete)
	}()
	
	// 解析任务数据
	results, ok := task.Data["classification_results"].([]classifier.Result)
	if !ok {
		e.IncrementErrorCount()
		return CreateTaskResult(task.ID, false, nil, 
			fmt.Errorf("invalid task data: missing classification results"), time.Since(startTime)), nil
	}
	
	// 评估每个分类结果
	evaluations := make([]EvaluationResult, 0, len(results))
	for _, result := range results {
		eval := e.evaluateSingleResult(result)
		evaluations = append(evaluations, eval)
		e.evaluationHistory = append(e.evaluationHistory, eval)
	}
	
	// 计算整体质量指标
	metrics := e.calculateQualityMetrics(evaluations)
	
	resultData := map[string]interface{}{
		"evaluations": evaluations,
		"metrics": metrics,
		"total_evaluated": len(evaluations),
		"quality_score": metrics.OverallScore,
	}
	
	duration := time.Since(startTime)
	return CreateTaskResult(task.ID, true, resultData, nil, duration), nil
}

// evaluateSingleResult 评估单个分类结果
func (e *EvaluatorAgent) evaluateSingleResult(result classifier.Result) EvaluationResult {
	eval := EvaluationResult{
		ResultID:    fmt.Sprintf("%s_%d", result.FileInfo.Name, time.Now().UnixNano()),
		Filename:    result.FileInfo.Name,
		Category:    result.Category,
		Confidence:  result.Confidence,
		EvaluatedAt: time.Now(),
		Issues:      make([]string, 0),
		Suggestions: make([]string, 0),
	}
	
	// 检查置信度
	if result.Confidence < e.cfg.ConfidenceThreshold {
		eval.Issues = append(eval.Issues, "置信度低于阈值")
		eval.Suggestions = append(eval.Suggestions, "建议人工审核或使用 LLM 重新分类")
	}
	
	// 检查分类是否为"其他"
	if result.Category == "其他" || result.Category == "Other" {
		eval.Issues = append(eval.Issues, "分类过于宽泛")
		eval.Suggestions = append(eval.Suggestions, "考虑添加更具体的分类规则")
	}
	
	// 计算质量评分
	eval.QualityScore = e.calculateQualityScore(result, eval)
	
	return eval
}

// calculateQualityScore 计算质量评分
func (e *EvaluatorAgent) calculateQualityScore(result classifier.Result, eval EvaluationResult) float64 {
	score := result.Confidence // 基础分数为置信度
	
	// 如果有问题，降低分数
	score -= float64(len(eval.Issues)) * 0.1
	
	// 如果使用了记忆系统，加分
	if result.Source == "memory" {
		score += 0.1
	}
	
	// 确保分数在 0-1 之间
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	
	return score
}

// calculateQualityMetrics 计算整体质量指标
func (e *EvaluatorAgent) calculateQualityMetrics(evaluations []EvaluationResult) *QualityMetrics {
	metrics := &QualityMetrics{
		Distribution: make(map[string]int),
		TopIssues:    make([]string, 0),
	}
	
	if len(evaluations) == 0 {
		return metrics
	}
	
	totalConfidence := 0.0
	totalScore := 0.0
	issueCount := make(map[string]int)
	
	for _, eval := range evaluations {
		totalConfidence += eval.Confidence
		totalScore += eval.QualityScore
		metrics.Distribution[eval.Category]++
		
		if eval.Confidence < e.cfg.ConfidenceThreshold {
			metrics.LowConfidenceCount++
		}
		
		for _, issue := range eval.Issues {
			issueCount[issue]++
		}
	}
	
	metrics.AverageConfidence = totalConfidence / float64(len(evaluations))
	metrics.OverallScore = totalScore / float64(len(evaluations))
	
	// 找出最常见的问题
	for issue, count := range issueCount {
		if count > len(evaluations)/10 { // 超过 10% 的问题
			metrics.TopIssues = append(metrics.TopIssues, 
				fmt.Sprintf("%s (%d次)", issue, count))
		}
	}
	
	return metrics
}

// GetMetrics 获取评估器指标
func (e *EvaluatorAgent) GetMetrics() map[string]interface{} {
	metrics := e.BaseAgent.GetMetrics()
	metrics["total_evaluations"] = len(e.evaluationHistory)
	
	if len(e.evaluationHistory) > 0 {
		avgScore := 0.0
		for _, eval := range e.evaluationHistory {
			avgScore += eval.QualityScore
		}
		metrics["average_quality_score"] = avgScore / float64(len(e.evaluationHistory))
	}
	
	return metrics
}

// GetRecentEvaluations 获取最近的评估结果
func (e *EvaluatorAgent) GetRecentEvaluations(count int) []EvaluationResult {
	if count > len(e.evaluationHistory) {
		count = len(e.evaluationHistory)
	}
	
	start := len(e.evaluationHistory) - count
	return e.evaluationHistory[start:]
}

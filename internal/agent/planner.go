// Package agent 智能体模块 - 规划器
// 负责任务分解、策略制定和优先级排序
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lynx-lee/filo/internal/config"
	"github.com/lynx-lee/filo/internal/scanner"
)

// PlannerAgent 规划器智能体
// 分析文件集合，制定分类策略，分解任务
type PlannerAgent struct {
	*BaseAgent
	cfg *config.Config
}

// PlanningStrategy 分类策略
type PlanningStrategy struct {
	UseMemoryFirst    bool     // 是否优先使用记忆系统
	UseLLM            bool     // 是否使用 LLM
	BatchSize         int      // 批次大小
	PriorityFiles     []string // 高优先级文件列表
	ComplexityLevel   string   // 复杂度等级: simple, medium, complex
	EstimatedTime     time.Duration // 预估时间
}

// NewPlannerAgent 创建规划器智能体
func NewPlannerAgent() *PlannerAgent {
	return &PlannerAgent{
		BaseAgent: NewBaseAgent(PlannerAgentType),
		cfg:       config.Get(),
	}
}

// Initialize 初始化规划器
func (p *PlannerAgent) Initialize(ctx context.Context, config map[string]interface{}) error {
	p.setStatus(StatusWorking)
	defer p.setStatus(StatusIdle)
	
	// 可以从配置中加载自定义策略
	if batchSize, ok := config["batch_size"].(int); ok {
		p.cfg.BatchSize = batchSize
	}
	
	return nil
}

// Execute 执行规划任务
func (p *PlannerAgent) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	startTime := time.Now()
	p.IncrementTaskCount()
	p.setStatus(StatusWorking)
	defer func() {
		p.setStatus(StatusComplete)
	}()
	
	// 解析任务数据
	files, ok := task.Data["files"].([]scanner.FileInfo)
	if !ok {
		p.IncrementErrorCount()
		return CreateTaskResult(task.ID, false, nil, 
			fmt.Errorf("invalid task data: missing files"), time.Since(startTime)), nil
	}
	
	sourceDir, _ := task.Data["source_dir"].(string)
	targetDir, _ := task.Data["target_dir"].(string)
	
	// 分析文件集合并制定策略
	strategy := p.analyzeAndPlan(files, sourceDir, targetDir)
	
	resultData := map[string]interface{}{
		"strategy": strategy,
		"file_count": len(files),
		"complexity": strategy.ComplexityLevel,
		"estimated_time": strategy.EstimatedTime.String(),
		"batches": (len(files) + strategy.BatchSize - 1) / strategy.BatchSize,
	}
	
	duration := time.Since(startTime)
	return CreateTaskResult(task.ID, true, resultData, nil, duration), nil
}

// analyzeAndPlan 分析文件并制定分类策略
func (p *PlannerAgent) analyzeAndPlan(files []scanner.FileInfo, sourceDir, targetDir string) *PlanningStrategy {
	strategy := &PlanningStrategy{
		UseMemoryFirst: true,  // 默认优先使用记忆
		UseLLM: true,          // 默认启用 LLM
		BatchSize: p.cfg.BatchSize,
	}
	
	// 分析文件复杂度
	totalSize := int64(0)
	fileTypes := make(map[string]int)
	hasSubdirs := false
	
	for _, f := range files {
		totalSize += f.Size
		ext := getFileExtension(f.Name)
		fileTypes[ext]++
		
		// 检查是否有子目录
		if strings.Contains(f.Path[len(sourceDir):], "/") || 
		   strings.Contains(f.Path[len(sourceDir):], "\\") {
			hasSubdirs = true
		}
	}
	
	// 根据特征确定复杂度
	if len(files) < 50 && len(fileTypes) < 5 && !hasSubdirs {
		strategy.ComplexityLevel = "simple"
		strategy.EstimatedTime = time.Duration(len(files)) * 2 * time.Second
	} else if len(files) < 500 && len(fileTypes) < 15 {
		strategy.ComplexityLevel = "medium"
		strategy.EstimatedTime = time.Duration(len(files)) * 3 * time.Second
	} else {
		strategy.ComplexityLevel = "complex"
		strategy.EstimatedTime = time.Duration(len(files)) * 5 * time.Second
		// 复杂任务增大批次
		strategy.BatchSize = p.cfg.BatchSize * 2
	}
	
	// 识别高优先级文件（大文件、特殊类型）
	for _, f := range files {
		if f.Size > 10*1024*1024 { // > 10MB
			strategy.PriorityFiles = append(strategy.PriorityFiles, f.Path)
		}
	}
	
	// 如果文件类型很单一，可以跳过 LLM
	if len(fileTypes) <= 2 && len(files) < 100 {
		strategy.UseLLM = false
	}
	
	return strategy
}

// getFileExtension 获取文件扩展名
func getFileExtension(filename string) string {
	idx := strings.LastIndex(filename, ".")
	if idx == -1 {
		return ""
	}
	return strings.ToLower(filename[idx:])
}

// GetMetrics 获取规划器指标
func (p *PlannerAgent) GetMetrics() map[string]interface{} {
	metrics := p.BaseAgent.GetMetrics()
	metrics["avg_planning_time"] = "N/A" // 可以添加平均规划时间统计
	return metrics
}

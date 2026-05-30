// Package agent 智能体模块
// 实现基于 ReAct 模式的多智能体协作系统
// 提供规划、执行、评估三个核心智能体
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package agent

import (
	"context"
	"fmt"
	"time"
)

// AgentType 智能体类型
type AgentType string

const (
	PlannerAgentType  AgentType = "planner"  // 规划器
	ExecutorAgentType AgentType = "executor" // 执行器
	EvaluatorAgentType AgentType = "evaluator" // 评估器
)

// AgentStatus 智能体状态
type AgentStatus string

const (
	StatusIdle     AgentStatus = "idle"      // 空闲
	StatusWorking  AgentStatus = "working"   // 工作中
	StatusComplete AgentStatus = "complete"  // 完成
	StatusError    AgentStatus = "error"     // 错误
)

// Task 任务定义
type Task struct {
	ID          string                 // 任务 ID
	Type        string                 // 任务类型
	Priority    int                    // 优先级 (1-10, 10最高)
	Data        map[string]interface{} // 任务数据
	CreatedAt   time.Time              // 创建时间
	Deadline    time.Time              // 截止时间（可选）
}

// TaskResult 任务结果
type TaskResult struct {
	TaskID    string                 // 任务 ID
	Success   bool                   // 是否成功
	Data      map[string]interface{} // 结果数据
	Error     error                  // 错误信息
	Duration  time.Duration          // 执行耗时
	Metadata  map[string]string      // 元数据
}

// Agent 智能体接口
type Agent interface {
	// GetType 获取智能体类型
	GetType() AgentType
	
	// Initialize 初始化智能体
	Initialize(ctx context.Context, config map[string]interface{}) error
	
	// Execute 执行任务
	Execute(ctx context.Context, task *Task) (*TaskResult, error)
	
	// GetStatus 获取智能体状态
	GetStatus() AgentStatus
	
	// Reset 重置智能体状态
	Reset()
	
	// GetMetrics 获取性能指标
	GetMetrics() map[string]interface{}
}

// BaseAgent 智能体基类
type BaseAgent struct {
	agentType AgentType
	status    AgentStatus
	metrics   map[string]interface{}
	config    map[string]interface{}
	taskCount int
	errorCount int
}

// NewBaseAgent 创建基础智能体
func NewBaseAgent(agentType AgentType) *BaseAgent {
	return &BaseAgent{
		agentType: agentType,
		status:    StatusIdle,
		metrics:   make(map[string]interface{}),
		config:    make(map[string]interface{}),
		taskCount: 0,
		errorCount: 0,
	}
}

// GetType 获取智能体类型
func (b *BaseAgent) GetType() AgentType {
	return b.agentType
}

// GetStatus 获取智能体状态
func (b *BaseAgent) GetStatus() AgentStatus {
	return b.status
}

// setStatus 设置智能体状态（内部方法）
func (b *BaseAgent) setStatus(status AgentStatus) {
	b.status = status
}

// IncrementTaskCount 增加任务计数
func (b *BaseAgent) IncrementTaskCount() {
	b.taskCount++
}

// IncrementErrorCount 增加错误计数
func (b *BaseAgent) IncrementErrorCount() {
	b.errorCount++
}

// GetMetrics 获取性能指标
func (b *BaseAgent) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"type":         string(b.agentType),
		"status":       string(b.status),
		"task_count":   b.taskCount,
		"error_count":  b.errorCount,
		"success_rate": b.calculateSuccessRate(),
	}
}

// calculateSuccessRate 计算成功率
func (b *BaseAgent) calculateSuccessRate() float64 {
	if b.taskCount == 0 {
		return 0
	}
	successCount := b.taskCount - b.errorCount
	return float64(successCount) / float64(b.taskCount)
}

// Reset 重置智能体状态
func (b *BaseAgent) Reset() {
	b.status = StatusIdle
	b.taskCount = 0
	b.errorCount = 0
}

// CreateTask 创建任务辅助函数
func CreateTask(taskType string, priority int, data map[string]interface{}) *Task {
	return &Task{
		ID:        fmt.Sprintf("%s_%d", taskType, time.Now().UnixNano()),
		Type:      taskType,
		Priority:  priority,
		Data:      data,
		CreatedAt: time.Now(),
	}
}

// CreateTaskResult 创建任务结果辅助函数
func CreateTaskResult(taskID string, success bool, data map[string]interface{}, err error, duration time.Duration) *TaskResult {
	result := &TaskResult{
		TaskID:   taskID,
		Success:  success,
		Data:     data,
		Error:    err,
		Duration: duration,
		Metadata: make(map[string]string),
	}
	
	if err != nil {
		result.Metadata["error"] = err.Error()
	}
	
	return result
}

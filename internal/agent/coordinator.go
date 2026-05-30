// Package agent 智能体模块 - 协调器
// 负责任务分发、结果聚合和冲突解决
// 实现多智能体协作的核心逻辑
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lynx-lee/filo/internal/classifier"
	"github.com/lynx-lee/filo/internal/config"
	"github.com/lynx-lee/filo/internal/scanner"
	"github.com/lynx-lee/filo/internal/ui"
)

// Coordinator 智能体协调器
// 管理多个智能体的协作，实现完整的分类工作流
type Coordinator struct {
	planner    *PlannerAgent
	executor   *classifier.Classifier // 使用现有分类器作为执行器
	evaluator  *EvaluatorAgent
	optimizer  *OptimizerAgent
	learner    *LearnerAgent
	eventBus   *EventBus
	protocol   CommunicationProtocol
	cfg        *config.Config
	
	mu          sync.RWMutex
	taskQueue   []*Task
	results     []*TaskResult
	isRunning   bool
}

// NewCoordinator 创建协调器
func NewCoordinator() (*Coordinator, error) {
	cfg := config.Get()
	
	executor, err := classifier.NewClassifier()
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}
	
	learner, err := NewLearnerAgent()
	if err != nil {
		return nil, fmt.Errorf("failed to create learner: %w", err)
	}
	
	// 创建事件总线
	eventBus := NewEventBus(100)
	
	// 创建通信协议
	protocol := NewStandardProtocol(eventBus, 30*time.Second)
	
	c := &Coordinator{
		planner:    NewPlannerAgent(),
		executor:   executor,
		evaluator:  NewEvaluatorAgent(),
		optimizer:  NewOptimizerAgent(),
		learner:    learner,
		eventBus:   eventBus,
		protocol:   protocol,
		cfg:        cfg,
		taskQueue:  make([]*Task, 0),
		results:    make([]*TaskResult, 0),
	}
	
	// 设置智能体间通信
	c.setupCommunication()
	
	return c, nil
}

// ExecuteWorkflow 执行完整的智能体工作流
// 1. Planner 分析并制定策略
// 2. Executor 执行分类
// 3. Optimizer 优化结果
// 4. Evaluator 评估质量
// 5. Learner 从结果中学习
func (c *Coordinator) ExecuteWorkflow(ctx context.Context, files []scanner.FileInfo, 
	sourceDir, targetDir string, verbose bool) ([]classifier.Result, error) {
	
	c.isRunning = true
	defer func() {
		c.isRunning = false
	}()
	
	startTime := time.Now()
	
	if verbose {
		ui.Info("🤖 启动智能体工作流...")
	}
	
	// ========== 阶段1: 规划 ==========
	if verbose {
		ui.Info("📋 [Planner] 分析文件集合并制定策略...")
	}
	
	planningTask := CreateTask("planning", 10, map[string]interface{}{
		"files":      files,
		"source_dir": sourceDir,
		"target_dir": targetDir,
	})
	
	planningResult, err := c.planner.Execute(ctx, planningTask)
	if err != nil || !planningResult.Success {
		return nil, fmt.Errorf("planning failed: %v", err)
	}
	
	strategy := planningResult.Data["strategy"].(*PlanningStrategy)
	
	if verbose {
		ui.Info("✓ 复杂度: %s", strategy.ComplexityLevel)
		ui.Info("✓ 预估时间: %s", strategy.EstimatedTime.String())
		ui.Info("✓ 批次数: %d", planningResult.Data["batches"])
		ui.Info("✓ 使用记忆优先: %v", strategy.UseMemoryFirst)
		ui.Info("✓ 使用 LLM: %v", strategy.UseLLM)
	}
	
	// ========== 阶段2: 执行 ==========
	if verbose {
		ui.Info("\n⚙️  [Executor] 执行文件分类...")
	}
	
	// 使用现有的分类器执行
	classificationResults, err := c.executor.Classify(files, verbose)
	if err != nil {
		return nil, fmt.Errorf("classification failed: %w", err)
	}
	
	if verbose {
		ui.Info("✓ 分类完成: %d 个文件", len(classificationResults))
	}
	
	// ========== 阶段3: 优化 ==========
	if verbose {
		ui.Info("\n✨ [Optimizer] 优化分类结果...")
	}
	
	optimizationTask := CreateTask("optimization", 7, map[string]interface{}{
		"classification_results": classificationResults,
	})
	
	optimizationResult, err := c.optimizer.Execute(ctx, optimizationTask)
	if err != nil || !optimizationResult.Success {
		if verbose {
			ui.Warning("⚠️  优化失败: %v", err)
		}
	} else {
		// 应用优化后的结果
		if optimized, ok := optimizationResult.Data["optimized_results"].([]classifier.Result); ok {
			classificationResults = optimized
		}
		
		if verbose {
			improvementRate := optimizationResult.Data["improvement_rate"].(float64)
			issuesFound := optimizationResult.Data["issues_found"].(int)
			ui.Info("✓ 优化改进率: %.1f%%", improvementRate*100)
			ui.Info("✓ 发现问题: %d 个", issuesFound)
		}
	}
	
	// ========== 阶段4: 评估 ==========
	if verbose {
		ui.Info("\n🔍 [Evaluator] 评估分类质量...")
	}
	
	evaluationTask := CreateTask("evaluation", 8, map[string]interface{}{
		"classification_results": classificationResults,
	})
	
	evaluationResult, err := c.evaluator.Execute(ctx, evaluationTask)
	if err != nil || !evaluationResult.Success {
		// 评估失败不影响主流程，只记录警告
		if verbose {
			ui.Warning("⚠️  评估失败: %v", err)
		}
	} else {
		metrics := evaluationResult.Data["metrics"].(*QualityMetrics)
		
		if verbose {
			ui.Info("✓ 平均置信度: %.2f%%", metrics.AverageConfidence*100)
			ui.Info("✓ 总体质量评分: %.2f/1.00", metrics.OverallScore)
			ui.Info("✓ 低置信度文件: %d", metrics.LowConfidenceCount)
			
			if len(metrics.TopIssues) > 0 {
				ui.Info("\n📊 常见问题:")
				for _, issue := range metrics.TopIssues {
					ui.Info("  • %s", issue)
				}
			}
		}
		
		// 如果质量太低，给出建议
		if metrics.OverallScore < 0.7 {
			if verbose {
				ui.Warning("\n⚠️  分类质量较低，建议:")
				ui.Warning("  • 检查 Ollama 模型是否正常运行")
				ui.Warning("  • 调整配置文件中的阈值参数")
				ui.Warning("  • 添加更多自定义分类规则")
			}
		}
	}
	
	// ========== 阶段5: 学习 ==========
	if verbose {
		ui.Info("\n📚 [Learner] 从本次分类中学习...")
	}
	
	learningTask := CreateTask("learn_from_results", 5, map[string]interface{}{
		"task_type": "learn_from_results",
		"classification_results": classificationResults,
	})
	
	learningResult, err := c.learner.Execute(ctx, learningTask)
	if err != nil || !learningResult.Success {
		if verbose {
			ui.Warning("⚠️  学习失败: %v", err)
		}
	} else {
		if verbose {
			learnedPatterns := learningResult.Data["learned_patterns"].(int)
			totalPatterns := learningResult.Data["total_patterns"].(int)
			ui.Info("✓ 学习新模式: %d 个", learnedPatterns)
			ui.Info("✓ 总模式数: %d 个", totalPatterns)
		}
	}
	
	duration := time.Since(startTime)
	
	if verbose {
		ui.Info("\n✅ 智能体工作流完成 (耗时: %s)", duration.String())
		c.printAgentMetrics()
	}
	
	return classificationResults, nil
}

// printAgentMetrics 打印各智能体性能指标
func (c *Coordinator) printAgentMetrics() {
	ui.Info("\n📈 智能体性能指标:")
	
	plannerMetrics := c.planner.GetMetrics()
	ui.Info("  Planner: 任务数=%d, 成功率=%.1f%%", 
		plannerMetrics["task_count"], 
		plannerMetrics["success_rate"].(float64)*100)
	
	executorMetrics := c.executor.GetModelStats()
	ui.Info("  Executor: 处理文件=%d, 平均耗时=%dms",
		executorMetrics["file_count"],
		executorMetrics["avg_time_ms"])
	
	evaluatorMetrics := c.evaluator.GetMetrics()
	ui.Info("  Evaluator: 评估数=%d, 平均质量=%.2f",
		evaluatorMetrics["total_evaluations"],
		evaluatorMetrics["average_quality_score"])
}

// GetWorkflowStatus 获取工作流状态
func (c *Coordinator) GetWorkflowStatus() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return map[string]interface{}{
		"is_running": c.isRunning,
		"task_queue_length": len(c.taskQueue),
		"results_count": len(c.results),
		"planner_status": c.planner.GetStatus(),
		"evaluator_status": c.evaluator.GetStatus(),
	}
}

// Reset 重置协调器状态
func (c *Coordinator) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.taskQueue = make([]*Task, 0)
	c.results = make([]*TaskResult, 0)
	c.isRunning = false
	
	c.planner.Reset()
	c.evaluator.Reset()
}

// Shutdown 关闭协调器，释放资源
func (c *Coordinator) Shutdown() {
	c.Reset()
	c.executor.Close()
	if c.learner != nil {
		c.learner.Close()
	}
	if c.eventBus != nil {
		c.eventBus.Shutdown()
	}
}

// setupCommunication 设置智能体间通信
func (c *Coordinator) setupCommunication() {
	// Planner 完成任务后通知 Coordinator
	c.eventBus.Subscribe(PlannerAgentType, MsgTaskCompleted, func(msg *Message) error {
		fmt.Printf("[Comm] Planner completed task: %s\n", msg.Payload["task_id"])
		return nil
	})
	
	// Evaluator 提交评估结果
	c.eventBus.Subscribe(EvaluatorAgentType, MsgFeedback, func(msg *Message) error {
		fmt.Printf("[Comm] Evaluator submitted feedback\n")
		return nil
	})
	
	// Optimizer 提供优化建议
	c.eventBus.Subscribe("optimizer", MsgOptimization, func(msg *Message) error {
		fmt.Printf("[Comm] Optimizer provided suggestions\n")
		return nil
	})
	
	// Learner 更新学习数据
	c.eventBus.Subscribe("learner", MsgLearningUpdate, func(msg *Message) error {
		fmt.Printf("[Comm] Learner updated knowledge\n")
		return nil
	})
}

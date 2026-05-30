// Package agent 智能体模块 - 通信协议
// 定义智能体间的消息传递和协作机制
// 实现发布-订阅模式和事件驱动架构
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package agent

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MessageType 消息类型
type MessageType string

const (
	MsgTaskAssigned    MessageType = "task_assigned"    // 任务分配
	MsgTaskCompleted   MessageType = "task_completed"   // 任务完成
	MsgTaskFailed      MessageType = "task_failed"      // 任务失败
	MsgResultReady     MessageType = "result_ready"     // 结果就绪
	MsgFeedback        MessageType = "feedback"         // 反馈信息
	MsgMetricsUpdate   MessageType = "metrics_update"   // 指标更新
	MsgStateChange     MessageType = "state_change"     // 状态变更
	MsgOptimization    MessageType = "optimization"     // 优化建议
	MsgLearningUpdate  MessageType = "learning_update"  // 学习更新
)

// MessagePriority 消息优先级
type MessagePriority int

const (
	PriorityLow    MessagePriority = 1  // 低优先级
	PriorityNormal MessagePriority = 5  // 普通优先级
	PriorityHigh   MessagePriority = 8  // 高优先级
	PriorityUrgent MessagePriority = 10 // 紧急
)

// Message 智能体间消息
type Message struct {
	ID        string            `json:"id"`         // 消息 ID
	Type      MessageType       `json:"type"`       // 消息类型
	Priority  MessagePriority   `json:"priority"`   // 优先级
	Sender    AgentType         `json:"sender"`     // 发送者
	Receiver  AgentType         `json:"receiver"`   // 接收者（空表示广播）
	Payload   map[string]interface{} `json:"payload"` // 消息内容
	Timestamp time.Time         `json:"timestamp"`  // 时间戳
	TTL       time.Duration     `json:"ttl"`        // 存活时间
}

// MessageHandler 消息处理器
type MessageHandler func(msg *Message) error

// EventBus 事件总线
// 实现智能体间的异步通信
type EventBus struct {
	mu           sync.RWMutex
	subscribers  map[AgentType]map[MessageType][]MessageHandler
	messageQueue chan *Message
	isRunning    bool
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewEventBus 创建事件总线
func NewEventBus(bufferSize int) *EventBus {
	ctx, cancel := context.WithCancel(context.Background())

	bus := &EventBus{
		subscribers:  make(map[AgentType]map[MessageType][]MessageHandler),
		messageQueue: make(chan *Message, bufferSize),
		isRunning:    true,
		ctx:          ctx,
		cancel:       cancel,
	}

	// 启动消息处理循环
	go bus.processMessages()

	return bus
}

// Subscribe 订阅消息
func (bus *EventBus) Subscribe(agentType AgentType, msgType MessageType, handler MessageHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if _, exists := bus.subscribers[agentType]; !exists {
		bus.subscribers[agentType] = make(map[MessageType][]MessageHandler)
	}

	bus.subscribers[agentType][msgType] = append(bus.subscribers[agentType][msgType], handler)
}

// Publish 发布消息
func (bus *EventBus) Publish(msg *Message) error {
	if !bus.isRunning {
		return fmt.Errorf("event bus is not running")
	}

	msg.ID = fmt.Sprintf("msg_%d", time.Now().UnixNano())
	msg.Timestamp = time.Now()

	select {
	case bus.messageQueue <- msg:
		return nil
	case <-bus.ctx.Done():
		return fmt.Errorf("event bus is shutting down")
	default:
		return fmt.Errorf("message queue is full")
	}
}

// Broadcast 广播消息给所有智能体
func (bus *EventBus) Broadcast(msg *Message) error {
	msg.Receiver = "" // 空接收者表示广播
	return bus.Publish(msg)
}

// processMessages 处理消息队列
func (bus *EventBus) processMessages() {
	for {
		select {
		case msg := <-bus.messageQueue:
			bus.handleMessage(msg)
		case <-bus.ctx.Done():
			return
		}
	}
}

// handleMessage 处理单条消息
func (bus *EventBus) handleMessage(msg *Message) {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	// 检查消息是否过期
	if msg.TTL > 0 && time.Since(msg.Timestamp) > msg.TTL {
		return // 消息已过期，丢弃
	}

	// 查找接收者的处理器
	var handlers []MessageHandler

	if msg.Receiver != "" {
		// 定向消息
		if agentHandlers, exists := bus.subscribers[msg.Receiver]; exists {
			if typeHandlers, exists := agentHandlers[msg.Type]; exists {
				handlers = typeHandlers
			}
		}
	} else {
		// 广播消息：发送给所有订阅了该消息类型的智能体
		for _, agentHandlers := range bus.subscribers {
			if typeHandlers, exists := agentHandlers[msg.Type]; exists {
				handlers = append(handlers, typeHandlers...)
			}
		}
	}

	// 按优先级排序处理器（可选优化）
	// 执行所有处理器
	for _, handler := range handlers {
		if err := handler(msg); err != nil {
			// 记录错误但不中断其他处理器
			fmt.Printf("Message handler error: %v\n", err)
		}
	}
}

// Shutdown 关闭事件总线
func (bus *EventBus) Shutdown() {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if bus.isRunning {
		bus.isRunning = false
		bus.cancel()
		close(bus.messageQueue)
	}
}

// GetQueueLength 获取消息队列长度
func (bus *EventBus) GetQueueLength() int {
	return len(bus.messageQueue)
}

// CreateMessage 创建消息辅助函数
func CreateMessage(msgType MessageType, sender AgentType, receiver AgentType, 
	priority MessagePriority, payload map[string]interface{}) *Message {
	return &Message{
		Type:     msgType,
		Sender:   sender,
		Receiver: receiver,
		Priority: priority,
		Payload:  payload,
		TTL:      5 * time.Minute, // 默认 5 分钟 TTL
	}
}

// CommunicationProtocol 通信协议接口
// 定义智能体间标准化的交互方式
type CommunicationProtocol interface {
	// RequestClassification 请求分类服务
	RequestClassification(files []string) (*Task, error)
	
	// SubmitEvaluationResult 提交评估结果
	SubmitEvaluationResult(evaluation *EvaluationResult) error
	
	// RequestOptimization 请求优化建议
	RequestOptimization(results []interface{}) (*Task, error)
	
	// SubmitLearningData 提交学习数据
	SubmitLearningData(data interface{}) error
	
	// QueryMetrics 查询性能指标
	QueryMetrics(agentType AgentType) (map[string]interface{}, error)
}

// StandardProtocol 标准通信协议实现
type StandardProtocol struct {
	eventBus *EventBus
	timeout  time.Duration
}

// NewStandardProtocol 创建标准通信协议
func NewStandardProtocol(eventBus *EventBus, timeout time.Duration) *StandardProtocol {
	return &StandardProtocol{
		eventBus: eventBus,
		timeout:  timeout,
	}
}

// RequestClassification 请求分类服务
func (p *StandardProtocol) RequestClassification(files []string) (*Task, error) {
	task := CreateTask("classification", int(PriorityHigh), map[string]interface{}{
		"files": files,
	})

	// 发布任务分配消息
	msg := CreateMessage(MsgTaskAssigned, "coordinator", ExecutorAgentType, PriorityHigh, map[string]interface{}{
		"task": task,
	})

	if err := p.eventBus.Publish(msg); err != nil {
		return nil, err
	}

	return task, nil
}

// SubmitEvaluationResult 提交评估结果
func (p *StandardProtocol) SubmitEvaluationResult(evaluation *EvaluationResult) error {
	msg := CreateMessage(MsgFeedback, EvaluatorAgentType, "coordinator", PriorityNormal, map[string]interface{}{
		"evaluation": evaluation,
	})

	return p.eventBus.Publish(msg)
}

// RequestOptimization 请求优化建议
func (p *StandardProtocol) RequestOptimization(results []interface{}) (*Task, error) {
	task := CreateTask("optimization", int(PriorityNormal), map[string]interface{}{
		"results": results,
	})

	msg := CreateMessage(MsgOptimization, "coordinator", "optimizer", PriorityNormal, map[string]interface{}{
		"task": task,
	})

	if err := p.eventBus.Publish(msg); err != nil {
		return nil, err
	}

	return task, nil
}

// SubmitLearningData 提交学习数据
func (p *StandardProtocol) SubmitLearningData(data interface{}) error {
	msg := CreateMessage(MsgLearningUpdate, "coordinator", "learner", PriorityLow, map[string]interface{}{
		"data": data,
	})

	return p.eventBus.Publish(msg)
}

// QueryMetrics 查询性能指标
func (p *StandardProtocol) QueryMetrics(agentType AgentType) (map[string]interface{}, error) {
	// 这里可以实现同步查询或异步请求-响应模式
	// 简化实现：直接返回空，实际使用时需要实现回调机制
	return make(map[string]interface{}), nil
}

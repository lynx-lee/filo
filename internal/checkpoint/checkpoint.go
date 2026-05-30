// Package checkpoint 断点续传模块
// 提供分类任务的检查点保存和恢复功能
// 支持大批量文件处理时的中断恢复
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package checkpoint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lynx-lee/filo/internal/config"
)

// CheckpointData 检查点数据结构
type CheckpointData struct {
	Timestamp    time.Time              `json:"timestamp"`    // 检查点创建时间
	SourceDir    string                 `json:"source_dir"`   // 源目录
	TargetDir    string                 `json:"target_dir"`   // 目标目录
	TotalFiles   int                    `json:"total_files"`  // 总文件数
	ProcessedCount int                  `json:"processed_count"` // 已处理数量
	PendingCount   int                  `json:"pending_count"`   // 待处理数量
	BatchID      string                 `json:"batch_id"`     // 批次 ID
}

// Manager 检查点管理器
type Manager struct {
	cfg        *config.Config
	mu         sync.RWMutex
	checkpoint *CheckpointData
}

// NewManager 创建检查点管理器
func NewManager() *Manager {
	cfg := config.Get()
	return &Manager{
		cfg: cfg,
	}
}

// getCheckpointPath 获取检查点文件路径
func (m *Manager) getCheckpointPath() string {
	return filepath.Join(m.cfg.DataDir, "checkpoint.json")
}

// SaveCheckpoint 保存检查点
// 将当前处理进度保存到文件，支持断点续传
func (m *Manager) SaveCheckpoint(sourceDir, targetDir string, totalFiles int, 
	processedCount, pendingCount int, batchID string) error {
	
	m.mu.Lock()
	defer m.mu.Unlock()

	m.checkpoint = &CheckpointData{
		Timestamp:      time.Now(),
		SourceDir:      sourceDir,
		TargetDir:      targetDir,
		TotalFiles:     totalFiles,
		ProcessedCount: processedCount,
		PendingCount:   pendingCount,
		BatchID:        batchID,
	}

	data, err := json.MarshalIndent(m.checkpoint, "", "  ")
	if err != nil {
		return err
	}

	checkpointPath := m.getCheckpointPath()
	return os.WriteFile(checkpointPath, data, 0644)
}

// LoadCheckpoint 加载检查点
// 从文件恢复之前的处理进度
func (m *Manager) LoadCheckpoint() (*CheckpointData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	checkpointPath := m.getCheckpointPath()
	
	// 检查文件是否存在
	if _, err := os.Stat(checkpointPath); os.IsNotExist(err) {
		return nil, nil // 没有检查点
	}

	data, err := os.ReadFile(checkpointPath)
	if err != nil {
		return nil, err
	}

	var checkpoint CheckpointData
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, err
	}

	return &checkpoint, nil
}

// ClearCheckpoint 清除检查点
// 任务完成后调用，删除检查点文件
func (m *Manager) ClearCheckpoint() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	checkpointPath := m.getCheckpointPath()
	m.checkpoint = nil
	
	// 删除检查点文件
	if _, err := os.Stat(checkpointPath); err == nil {
		return os.Remove(checkpointPath)
	}
	return nil
}

// HasCheckpoint 检查是否存在未完成的检查点
func (m *Manager) HasCheckpoint() bool {
	checkpointPath := m.getCheckpointPath()
	_, err := os.Stat(checkpointPath)
	return err == nil
}

// GetProgress 获取处理进度
func (m *Manager) GetProgress() (processed, total int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.checkpoint == nil {
		return 0, 0
	}

	return m.checkpoint.ProcessedCount, m.checkpoint.TotalFiles
}

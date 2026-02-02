// Package config 配置管理模块
// 提供全局配置的加载、保存和管理功能
// 配置文件存储在 ~/.filo/config.json
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// 版本和作者信息常量
const (
	Version   = "2.0.0"                          // 程序版本号
	BuildDate = "2026"                           // 构建日期
	Author    = "lynx-lee"                       // 作者
	Homepage  = "https://github.com/lynx-lee/filo" // 项目主页
	License   = "MIT"                            // 开源许可
)

// Config 全局配置结构体
// 包含模型配置、学习配置和处理配置
type Config struct {
	// ==================== 模型配置 ====================
	LLMModel       string  `json:"llm_model"`       // LLM 模型名称（用于分类）
	EmbeddingModel string  `json:"embedding_model"` // 向量嵌入模型名称
	OllamaURL      string  `json:"ollama_url"`      // Ollama 服务地址
	Temperature    float64 `json:"temperature"`     // 模型温度（0-1，越低越确定）
	MaxTokens      int     `json:"max_tokens"`      // 最大生成 token 数

	// ==================== 学习配置 ====================
	EnableLearning      bool    `json:"enable_learning"`       // 是否启用学习功能
	SimilarityThreshold float64 `json:"similarity_threshold"`  // 相似度匹配阈值（0-1）
	ConfidenceThreshold float64 `json:"confidence_threshold"`  // 置信度阈值（0-1）
	MinSamplesForRule   int     `json:"min_samples_for_rule"`  // 生成规则所需的最小样本数

	// ==================== 处理配置 ====================
	BatchSize int `json:"batch_size"` // 批量处理大小（每批分类的文件数）

	// ==================== 内部路径（不序列化）====================
	DataDir string `json:"-"` // 数据目录路径 (~/.filo)
	DBPath  string `json:"-"` // 数据库文件路径 (~/.filo/memory.db)
}

// 单例模式相关变量
var (
	instance *Config   // 全局配置实例
	once     sync.Once // 确保只初始化一次
)

// Get 获取全局配置实例（单例模式）
// 首次调用时会初始化默认配置并尝试从文件加载
func Get() *Config {
	once.Do(func() {
		instance = defaultConfig() // 创建默认配置
		instance.initPaths()       // 初始化路径
		instance.Load()            // 从文件加载（如果存在）
	})
	return instance
}

// defaultConfig 创建默认配置
// 返回带有合理默认值的配置实例
func defaultConfig() *Config {
	return &Config{
		LLMModel:            "qwen3:8b",              // 默认使用 qwen3:8b 模型
		EmbeddingModel:      "nomic-embed-text",      // 默认嵌入模型
		OllamaURL:           "http://localhost:11434", // Ollama 默认地址
		Temperature:         0.3,                      // 较低温度保证输出稳定
		MaxTokens:           2048,                     // 最大 token 数
		EnableLearning:      true,                     // 默认启用学习
		SimilarityThreshold: 0.85,                     // 相似度阈值 85%
		ConfidenceThreshold: 0.7,                      // 置信度阈值 70%
		MinSamplesForRule:   3,                        // 至少3个样本才生成规则
		BatchSize:           15,                       // 每批处理15个文件
	}
}

// initPaths 初始化数据存储路径
// 创建 ~/.filo 目录（如果不存在）
func (c *Config) initPaths() {
	homeDir, _ := os.UserHomeDir()
	c.DataDir = filepath.Join(homeDir, ".filo")           // 数据目录
	c.DBPath = filepath.Join(c.DataDir, "memory.db")      // SQLite 数据库路径
	os.MkdirAll(c.DataDir, 0755)                          // 创建目录
}

// Load 从文件加载配置
// 配置文件路径: ~/.filo/config.json
func (c *Config) Load() error {
	configPath := filepath.Join(c.DataDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err // 文件不存在时返回错误，使用默认配置
	}
	return json.Unmarshal(data, c)
}

// Save 保存配置到文件
// 以格式化的 JSON 格式保存
func (c *Config) Save() error {
	configPath := filepath.Join(c.DataDir, "config.json")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

// SetModel 设置 LLM 模型
// 用于通过命令行参数临时切换模型
func (c *Config) SetModel(model string) {
	c.LLMModel = model
}

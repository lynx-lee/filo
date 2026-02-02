// Package embedding 向量嵌入模块
// 提供文本向量化功能，用于相似文件匹配
// 支持本地哈希嵌入和 Ollama 模型嵌入两种方式
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package embedding

import (
	"context"
	"hash/fnv"
	"math"
	"regexp"
	"strings"
	"time"

	"filo/internal/llm"
)

// ==================== 接口定义 ====================

// Embedder 嵌入器接口
// 定义文本向量化和相似度计算的标准方法
type Embedder interface {
	Embed(text string) []float64              // 将文本转换为向量
	Similarity(v1, v2 []float64) float64      // 计算两个向量的相似度
}

// ==================== 本地嵌入器 ====================

// LocalEmbedder 本地嵌入器
// 使用哈希算法将文本转换为向量，不依赖外部服务
// 速度快，适合大量文件的快速匹配
type LocalEmbedder struct {
	dimension int // 向量维度
}

// NewLocalEmbedder 创建本地嵌入器
// 默认使用256维向量
func NewLocalEmbedder() *LocalEmbedder {
	return &LocalEmbedder{dimension: 256}
}

// Embed 生成文本的嵌入向量
// 使用多种特征提取方法：字符级、词级、N-gram
func (e *LocalEmbedder) Embed(text string) []float64 {
	vec := make([]float64, e.dimension)
	text = strings.ToLower(text) // 转小写，忽略大小写差异

	// ========== 特征1: 字符级特征 ==========
	// 每个字符通过哈希映射到向量的某个位置
	// 位置越靠前的字符权重越高
	for i, char := range text {
		h := fnv.New64a()
		h.Write([]byte(string(char)))
		idx := int(h.Sum64() % uint64(e.dimension))
		vec[idx] += 1.0 / float64(i+1) // 权重递减
	}

	// ========== 特征2: 词级特征 ==========
	// 提取中文词、英文词、数字，权重更高
	re := regexp.MustCompile(`[\p{Han}]+|[a-zA-Z]+|\d+`)
	words := re.FindAllString(text, -1)
	for i, word := range words {
		h := fnv.New64a()
		h.Write([]byte(word))
		idx := int(h.Sum64() % uint64(e.dimension))
		vec[idx] += 2.0 / float64(i+1) // 词级特征权重为字符级的2倍
	}

	// ========== 特征3: N-gram 特征 ==========
	// 提取连续3个字符的组合，捕获局部模式
	for i := 0; i < len(text)-2; i++ {
		ngram := text[i : i+3]
		h := fnv.New64a()
		h.Write([]byte(ngram))
		idx := int(h.Sum64() % uint64(e.dimension))
		vec[idx] += 0.5 // 固定权重
	}

	return normalize(vec) // 归一化向量
}

// Similarity 计算两个向量的余弦相似度
// 返回值范围: 0（完全不同）到 1（完全相同）
func (e *LocalEmbedder) Similarity(v1, v2 []float64) float64 {
	if len(v1) != len(v2) {
		return 0 // 维度不同，无法比较
	}

	var dot, n1, n2 float64
	for i := range v1 {
		dot += v1[i] * v2[i]  // 点积
		n1 += v1[i] * v1[i]   // v1 的模的平方
		n2 += v2[i] * v2[i]   // v2 的模的平方
	}

	if n1 == 0 || n2 == 0 {
		return 0 // 零向量
	}

	// 余弦相似度 = 点积 / (|v1| * |v2|)
	return dot / (math.Sqrt(n1) * math.Sqrt(n2))
}

// normalize 向量归一化
// 将向量缩放为单位向量（模为1）
func normalize(vec []float64) []float64 {
	var sum float64
	for _, v := range vec {
		sum += v * v
	}
	norm := math.Sqrt(sum)
	if norm == 0 {
		return vec // 零向量不处理
	}

	result := make([]float64, len(vec))
	for i, v := range vec {
		result[i] = v / norm
	}
	return result
}

// ==================== Ollama 嵌入器 ====================

// OllamaEmbedder 使用 Ollama 模型的嵌入器
// 生成更高质量的语义向量，但速度较慢
type OllamaEmbedder struct {
	client   *llm.Client     // Ollama 客户端
	fallback *LocalEmbedder  // 失败时的后备方案
}

// NewOllamaEmbedder 创建 Ollama 嵌入器
func NewOllamaEmbedder() *OllamaEmbedder {
	return &OllamaEmbedder{
		client:   llm.NewClient(),
		fallback: NewLocalEmbedder(),
	}
}

// Embed 使用 Ollama 模型生成嵌入向量
// 如果调用失败，自动回退到本地嵌入器
func (e *OllamaEmbedder) Embed(text string) []float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	vec, err := e.client.Embed(ctx, text)
	if err != nil {
		// Ollama 不可用时回退到本地嵌入
		return e.fallback.Embed(text)
	}
	return vec
}

// Similarity 计算相似度
// 复用本地嵌入器的相似度计算方法
func (e *OllamaEmbedder) Similarity(v1, v2 []float64) float64 {
	return e.fallback.Similarity(v1, v2)
}

// ==================== 工厂函数 ====================

// NewEmbedder 创建嵌入器（自动选择）
// 默认使用本地嵌入（更快更稳定）
func NewEmbedder() Embedder {
	return NewLocalEmbedder()
}

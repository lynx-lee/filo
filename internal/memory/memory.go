// Package memory 记忆系统模块
// 实现文件分类的学习和记忆功能
// 支持规则匹配、向量匹配和历史匹配三种方式
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package memory

import (
	"path/filepath"
	"regexp"
	"strings"

	"filo/internal/config"
	"filo/internal/embedding"
	"filo/internal/storage"
)

// ==================== 常量定义 ====================

const (
	MaxVectorSearchLimit = 200 // 向量搜索最大数量
	MinKeywordLength     = 2   // 关键词最小长度
)

// 预编译的正则表达式（性能优化）
var (
	// 关键词提取正则：中文词、英文词（2+字符）、数字（4+位）
	keywordRegex = regexp.MustCompile(`[\p{Han}]+|[a-zA-Z]{2,}|\d{4,}`)
	// 文件名分词正则：中文词、英文词、数字
	tokenRegex = regexp.MustCompile(`[\p{Han}]+|[a-zA-Z]+|\d+`)
)

// ==================== 类型定义 ====================

// Match 记忆匹配结果
// 存储从记忆系统中查询到的分类信息
type Match struct {
	Category    string  // 主分类
	Subcategory string  // 子分类
	Confidence  float64 // 置信度（0-1）
	Source      string  // 来源: rule（规则）, vector（向量）, history（历史）
	Reasoning   string  // 匹配理由
}

// Memory 记忆系统
// 管理分类的学习和查询
type Memory struct {
	db       *storage.Database   // 数据库连接
	embedder embedding.Embedder  // 向量嵌入器
	cfg      *config.Config      // 配置
}

// ==================== 构造函数 ====================

// NewMemory 创建记忆系统
// 初始化数据库连接和嵌入器
func NewMemory() (*Memory, error) {
	db, err := storage.NewDatabase()
	if err != nil {
		return nil, err
	}

	return &Memory{
		db:       db,
		embedder: embedding.NewEmbedder(),
		cfg:      config.Get(),
	}, nil
}

// Close 关闭记忆系统
// 释放数据库连接
func (m *Memory) Close() error {
	return m.db.Close()
}

// ==================== 查询方法 ====================

// Query 查询文件的分类记忆
// 按优先级依次尝试: 规则匹配 -> 向量匹配 -> 历史匹配
// 返回置信度最高的匹配结果，如果都不满足阈值则返回 nil
func (m *Memory) Query(filename string) *Match {
	// 1. 规则匹配（最快，优先级最高）
	if match := m.matchRules(filename); match != nil {
		if match.Confidence >= m.cfg.SimilarityThreshold {
			return match
		}
	}

	// 2. 向量匹配（语义相似度）
	if match := m.matchVectors(filename); match != nil {
		if match.Confidence >= m.cfg.SimilarityThreshold {
			return match
		}
	}

	// 3. 历史匹配（关键词匹配）
	if match := m.matchHistory(filename); match != nil {
		if match.Confidence >= m.cfg.SimilarityThreshold {
			return match
		}
	}

	return nil // 无匹配结果
}

// matchRules 规则匹配
// 根据已学习的规则（关键词、扩展名）进行匹配
func (m *Memory) matchRules(filename string) *Match {
	keywords := extractKeywords(filename)
	ext := strings.ToLower(filepath.Ext(filename))

	// 从数据库获取匹配的规则
	rules, err := m.db.GetMatchingRules(filename, keywords, ext)
	if err != nil || len(rules) == 0 {
		return nil
	}

	best := rules[0] // 取最优规则

	// 计算置信度：基础分 + 命中次数加成
	conf := 0.6 + float64(best.HitCount)/50.0*0.35
	if conf > 0.95 {
		conf = 0.95 // 上限 95%
	}

	return &Match{
		Category:    best.Category,
		Subcategory: best.Subcategory,
		Confidence:  conf,
		Source:      "rule",
		Reasoning:   "匹配规则: " + best.PatternType + "「" + best.Pattern + "」",
	}
}

// matchVectors 向量匹配（优化版本）
// 通过向量相似度查找相似文件的分类
// 优化：使用分类预过滤减少比对数量
func (m *Memory) matchVectors(filename string) *Match {
	// 生成查询向量
	queryVec := m.embedder.Embed(filename)

	// 提取关键词和扩展名，用于预过滤
	keywords := extractKeywords(filename)
	ext := strings.ToLower(filepath.Ext(filename))

	// 获取候选分类（优化：预过滤）
	candidateCategories := m.db.GetCandidateCategories(keywords, ext)

	// 获取过滤后的向量（数量大大减少）
	var vectors []storage.VectorRecord
	var err error
	if len(candidateCategories) > 0 {
		// 有候选分类时，只搜索相关分类的向量
		vectors, err = m.db.SearchVectorsByCategories(candidateCategories, MaxVectorSearchLimit)
	} else {
		// 无候选分类时，按扩展名搜索
		vectors, err = m.db.SearchVectorsByExtension(ext, MaxVectorSearchLimit)
	}

	if err != nil || len(vectors) == 0 {
		return nil
	}

	// 查找最相似的向量
	var best struct {
		Filename    string
		Category    string
		Subcategory string
		Similarity  float64
	}

	for _, v := range vectors {
		sim := m.embedder.Similarity(queryVec, v.Vector)
		if sim > best.Similarity {
			best.Filename = v.Filename
			best.Category = v.Category
			best.Subcategory = v.Subcategory
			best.Similarity = sim
		}
	}

	// 检查是否达到阈值
	if best.Similarity < m.cfg.SimilarityThreshold {
		return nil
	}

	return &Match{
		Category:    best.Category,
		Subcategory: best.Subcategory,
		Confidence:  best.Similarity,
		Source:      "vector",
		Reasoning:   "相似文件: " + best.Filename,
	}
}

// matchHistory 历史匹配
// 根据关键词在历史分类记录中查找
func (m *Memory) matchHistory(filename string) *Match {
	keywords := extractKeywords(filename)
	if len(keywords) == 0 {
		return nil
	}

	// 查找相似的历史记录
	records, err := m.db.GetSimilarClassifications(keywords, 5)
	if err != nil || len(records) == 0 {
		return nil
	}

	best := records[0]
	// 计算文件名相似度
	sim := filenameSimilarity(filename, best.Filename)

	return &Match{
		Category:    best.Category,
		Subcategory: best.Subcategory,
		Confidence:  sim * 0.9, // 历史匹配置信度打9折
		Source:      "history",
		Reasoning:   "历史记录: " + best.Filename,
	}
}

// ==================== 学习方法 ====================

// Learn 从分类结果学习
// 将分类结果存入历史记录和向量库，用户确认时还会生成规则
func (m *Memory) Learn(filename, category, subcategory, source string, confidence float64, userConfirmed bool) error {
	ext := strings.ToLower(filepath.Ext(filename))
	keywords := extractKeywords(filename)

	// 添加到历史记录
	if _, err := m.db.AddClassification(filename, ext, category, subcategory, source, confidence, keywords, userConfirmed); err != nil {
		return err
	}

	// 添加到向量库
	vec := m.embedder.Embed(filename)
	if err := m.db.SaveVector(filename, category, subcategory, vec); err != nil {
		return err
	}

	// 用户确认时，学习规则
	if userConfirmed {
		m.learnRules(filename, category, subcategory)
	}

	return nil
}

// learnRules 从文件名学习规则
// 提取扩展名和关键词，生成分类规则
func (m *Memory) learnRules(filename, category, subcategory string) {
	ext := strings.ToLower(filepath.Ext(filename))
	keywords := extractKeywords(filename)

	// 学习扩展名规则（优先级较低）
	if ext != "" {
		m.db.AddOrUpdateRule(ext, "extension", category, subcategory, 5)
	}

	// 学习关键词规则（优先级较高）
	for _, kw := range keywords {
		if len(kw) >= MinKeywordLength {
			m.db.AddOrUpdateRule(strings.ToLower(kw), "keyword", category, subcategory, 10)
		}
	}
}

// LearnFromCorrection 从用户纠正中学习
// 当用户修改分类时调用，生成高优先级规则
func (m *Memory) LearnFromCorrection(filename, origCat, corrCat, origSub, corrSub string) error {
	// 记录用户反馈
	m.db.AddFeedback(filename, origCat, corrCat, origSub, corrSub)

	// 高优先级学习（用户纠正的权重更高）
	keywords := extractKeywords(filename)
	for _, kw := range keywords {
		if len(kw) >= MinKeywordLength {
			m.db.AddOrUpdateRule(strings.ToLower(kw), "keyword", corrCat, corrSub, 20) // 优先级20
		}
	}

	return nil
}

// ==================== 信息获取方法 ====================

// GetLearnedRules 获取已学习的规则
// 返回最常用的规则列表，用于提供给 LLM 参考
func (m *Memory) GetLearnedRules(limit int) []map[string]string {
	rules, err := m.db.GetTopRules(limit)
	if err != nil {
		return nil
	}

	// 转换为简化格式
	result := make([]map[string]string, 0, len(rules))
	for _, r := range rules {
		result = append(result, map[string]string{
			"pattern":     r["pattern"].(string),
			"category":    r["category"].(string),
			"subcategory": r["subcategory"].(string),
		})
	}
	return result
}

// GetStatistics 获取统计信息
// 返回记忆系统的各项统计数据
func (m *Memory) GetStatistics() (map[string]interface{}, error) {
	stats, err := m.db.GetStatistics()
	if err != nil {
		return nil, err
	}
	stats["learning_enabled"] = m.cfg.EnableLearning
	return stats, nil
}

// ==================== 辅助函数 ====================

// extractKeywords 从文件名提取关键词
// 提取中文词、英文词（至少2字符）、数字（至少4位）
// 使用预编译的正则表达式提升性能
func extractKeywords(filename string) []string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename)) // 去除扩展名
	return keywordRegex.FindAllString(name, -1)
}

// filenameSimilarity 计算两个文件名的相似度
// 使用 Jaccard 相似系数（交集/并集）
// 使用预编译的正则表达式提升性能
func filenameSimilarity(n1, n2 string) float64 {
	// 提取两个文件名的词集合
	w1 := make(map[string]bool)
	w2 := make(map[string]bool)

	for _, w := range tokenRegex.FindAllString(strings.ToLower(n1), -1) {
		w1[w] = true
	}
	for _, w := range tokenRegex.FindAllString(strings.ToLower(n2), -1) {
		w2[w] = true
	}

	if len(w1) == 0 || len(w2) == 0 {
		return 0
	}

	// 计算交集大小
	var intersection int
	for w := range w1 {
		if w2[w] {
			intersection++
		}
	}

	// Jaccard 系数 = 交集 / 并集
	union := len(w1) + len(w2) - intersection
	return float64(intersection) / float64(union)
}

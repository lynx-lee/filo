// Package storage 数据存储模块
// 提供 SQLite 数据库的封装，用于持久化存储分类历史、学习规则、用户反馈和向量数据
// 这是 Filo 学习系统的核心存储层，支持记忆功能和规则学习
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo
package storage

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	// 使用纯 Go 实现的 SQLite 驱动，无需 CGO
	_ "modernc.org/sqlite"

	"filo/internal/config"
)

// Database 数据库管理器
// 封装 SQLite 数据库连接，提供分类系统所需的所有数据操作接口
// 采用 WAL 模式提升并发性能，支持索引优化查询
type Database struct {
	db *sql.DB // SQLite 数据库连接实例
}

// ClassificationRecord 分类历史记录结构体
// 记录每次文件分类的完整信息，用于学习和统计分析
type ClassificationRecord struct {
	ID            int64     // 记录唯一标识符
	Filename      string    // 文件名（不含路径）
	Extension     string    // 文件扩展名（如 .pdf, .docx）
	Category      string    // 主分类（如 "工作文档"、"个人照片"）
	Subcategory   string    // 子分类（如 "会议纪要"、"旅行照片"）
	Confidence    float64   // 分类置信度（0.0 ~ 1.0）
	Keywords      []string  // 从文件名中提取的关键词列表
	UserConfirmed bool      // 是否经过用户确认（确认后用于学习）
	CreatedAt     time.Time // 记录创建时间
}

// LearnedRule 学习规则结构体
// 表示从用户确认的分类中学习到的规则模式
// 支持三种模式类型：关键词、扩展名、前缀
type LearnedRule struct {
	ID          int64   // 规则唯一标识符
	Pattern     string  // 匹配模式（如关键词 "invoice"、扩展名 ".pdf"）
	PatternType string  // 模式类型：keyword（关键词）、extension（扩展名）、prefix（前缀）
	Category    string  // 匹配后对应的主分类
	Subcategory string  // 匹配后对应的子分类
	Priority    int     // 规则优先级（数值越高优先级越高）
	HitCount    int     // 规则命中次数（用于统计和排序）
	SuccessRate float64 // 规则成功率（保留字段，暂未使用）
}

// NewDatabase 创建并初始化数据库连接
// 执行以下操作：
// 1. 从全局配置获取数据库路径
// 2. 打开 SQLite 数据库连接
// 3. 启用 WAL 模式和 NORMAL 同步模式以提升性能
// 4. 初始化所有必要的数据表和索引
//
// 返回值:
//   - *Database: 初始化完成的数据库管理器实例
//   - error: 如果打开数据库或初始化失败，返回相应错误
func NewDatabase() (*Database, error) {
	// 从全局配置获取数据库文件路径
	cfg := config.Get()
	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return nil, err
	}

	// 启用 WAL（Write-Ahead Logging）模式
	// WAL 模式可以显著提升读写并发性能，减少锁竞争
	db.Exec("PRAGMA journal_mode=WAL")

	// 设置同步模式为 NORMAL
	// NORMAL 模式在性能和数据安全性之间取得平衡
	db.Exec("PRAGMA synchronous=NORMAL")

	// 创建数据库管理器实例并初始化表结构
	d := &Database{db: db}
	if err := d.init(); err != nil {
		return nil, err
	}
	return d, nil
}

// init 初始化数据库表结构和索引
// 创建以下数据表：
// 1. classification_history - 分类历史记录表
// 2. learned_rules - 学习规则表
// 3. user_feedback - 用户反馈表
// 4. vectors - 向量存储表
// 以及相关的索引以优化查询性能
//
// 返回值:
//   - error: 如果任何表或索引创建失败，返回错误
func (d *Database) init() error {
	schemas := []string{
		// ========== 分类历史表 ==========
		// 记录每次文件分类的详细信息
		// 包含文件信息、分类结果、置信度、关键词等
		// user_confirmed 字段标记是否经过用户确认，用于学习
		`CREATE TABLE IF NOT EXISTS classification_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT NOT NULL,
			extension TEXT DEFAULT '',
			category TEXT NOT NULL,
			subcategory TEXT DEFAULT '',
			confidence REAL DEFAULT 0.5,
			keywords TEXT DEFAULT '[]',
			user_confirmed INTEGER DEFAULT 0,
			source TEXT DEFAULT 'llm',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// ========== 学习规则表 ==========
		// 存储从用户确认中学习到的分类规则
		// 支持关键词、扩展名、前缀三种模式类型
		// 通过 hit_count 和 success_count 统计规则效果
		`CREATE TABLE IF NOT EXISTS learned_rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pattern TEXT NOT NULL,
			pattern_type TEXT DEFAULT 'keyword',
			category TEXT NOT NULL,
			subcategory TEXT DEFAULT '',
			priority INTEGER DEFAULT 0,
			hit_count INTEGER DEFAULT 0,
			success_count INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(pattern, pattern_type, category)
		)`,

		// ========== 用户反馈表 ==========
		// 记录用户对分类结果的修正
		// 用于分析分类错误和改进规则
		`CREATE TABLE IF NOT EXISTS user_feedback (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT NOT NULL,
			original_category TEXT,
			corrected_category TEXT NOT NULL,
			original_subcategory TEXT DEFAULT '',
			corrected_subcategory TEXT DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// ========== 向量存储表 ==========
		// 存储文件名的向量嵌入表示
		// 用于基于语义相似度的分类匹配
		`CREATE TABLE IF NOT EXISTS vectors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT NOT NULL,
			category TEXT NOT NULL,
			subcategory TEXT DEFAULT '',
			vector BLOB NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// ========== 索引定义 ==========
		// 为常用查询字段创建索引，提升查询性能
		`CREATE INDEX IF NOT EXISTS idx_history_filename ON classification_history(filename)`,
		`CREATE INDEX IF NOT EXISTS idx_history_category ON classification_history(category)`,
		`CREATE INDEX IF NOT EXISTS idx_history_confirmed ON classification_history(user_confirmed)`,
		`CREATE INDEX IF NOT EXISTS idx_rules_pattern ON learned_rules(pattern)`,
		`CREATE INDEX IF NOT EXISTS idx_rules_category ON learned_rules(category)`,

		// ========== 操作日志表 ==========
		// 记录文件移动操作，支持撤销功能
		`CREATE TABLE IF NOT EXISTS operation_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id TEXT NOT NULL,
			source_path TEXT NOT NULL,
			dest_path TEXT NOT NULL,
			filename TEXT NOT NULL,
			category TEXT NOT NULL,
			subcategory TEXT DEFAULT '',
			status TEXT DEFAULT 'success',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// 向量表扩展索引：按分类和扩展名过滤
		`CREATE INDEX IF NOT EXISTS idx_vectors_category ON vectors(category)`,
		`CREATE INDEX IF NOT EXISTS idx_vectors_subcategory ON vectors(subcategory)`,
		`CREATE INDEX IF NOT EXISTS idx_operation_batch ON operation_logs(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_operation_time ON operation_logs(created_at)`,

		// ========== 模型性能统计表 ==========
		// 记录每个模型的执行性能和准确度
		// 用于自适应模型选择
		`CREATE TABLE IF NOT EXISTS model_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			model_name TEXT NOT NULL,
			batch_id TEXT NOT NULL,
			file_count INTEGER DEFAULT 0,
			total_time_ms INTEGER DEFAULT 0,
			avg_time_per_file_ms REAL DEFAULT 0,
			avg_confidence REAL DEFAULT 0,
			confirmed_count INTEGER DEFAULT 0,
			corrected_count INTEGER DEFAULT 0,
			accuracy_rate REAL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// 模型性能索引
		`CREATE INDEX IF NOT EXISTS idx_model_stats_name ON model_stats(model_name)`,
		`CREATE INDEX IF NOT EXISTS idx_model_stats_time ON model_stats(created_at)`,
	}

	// 依次执行所有 DDL 语句
	for _, schema := range schemas {
		if _, err := d.db.Exec(schema); err != nil {
			return err
		}
	}
	return nil
}

// Close 关闭数据库连接
// 释放数据库资源，应在程序退出前调用
//
// 返回值:
//   - error: 如果关闭失败，返回错误
func (d *Database) Close() error {
	return d.db.Close()
}

// ==================== 分类历史操作 ====================
// 以下方法用于管理分类历史记录的增删改查

// AddClassification 添加一条新的分类记录到历史表
// 将文件分类结果持久化存储，用于后续学习和统计
//
// 参数:
//   - filename: 文件名（不含路径）
//   - ext: 文件扩展名
//   - category: 主分类名称
//   - subcategory: 子分类名称
//   - source: 分类来源（"llm" 表示 AI 分类，"memory" 表示记忆匹配）
//   - confidence: 分类置信度（0.0 ~ 1.0）
//   - keywords: 从文件名提取的关键词列表
//   - confirmed: 是否已确认（用户确认后可用于学习）
//
// 返回值:
//   - int64: 新插入记录的 ID
//   - error: 如果插入失败，返回错误
func (d *Database) AddClassification(filename, ext, category, subcategory, source string, confidence float64, keywords []string, confirmed bool) (int64, error) {
	// 将关键词列表序列化为 JSON 字符串存储
	kw, _ := json.Marshal(keywords)
	result, err := d.db.Exec(`
		INSERT INTO classification_history (filename, extension, category, subcategory, confidence, keywords, user_confirmed, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, filename, ext, category, subcategory, confidence, string(kw), confirmed, source)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetSimilarClassifications 获取与给定关键词相似的历史分类记录
// 基于关键词在文件名中的模糊匹配，查找已确认的历史分类
// 用于"记忆优先"策略中的历史匹配
//
// 参数:
//   - keywords: 要匹配的关键词列表
//   - limit: 返回结果的最大数量
//
// 返回值:
//   - []ClassificationRecord: 匹配到的分类记录列表
//   - error: 如果查询失败，返回错误
func (d *Database) GetSimilarClassifications(keywords []string, limit int) ([]ClassificationRecord, error) {
	// 如果没有关键词，直接返回空结果
	if len(keywords) == 0 {
		return nil, nil
	}

	// 构建动态 OR 查询条件
	// 对每个关键词生成一个 LIKE 条件
	conditions := make([]string, 0, len(keywords))
	args := make([]interface{}, 0, len(keywords)+1)
	for _, kw := range keywords {
		// 忽略过短的关键词（少于2个字符）
		if len(kw) >= 2 {
			conditions = append(conditions, "LOWER(filename) LIKE ?")
			args = append(args, "%"+strings.ToLower(kw)+"%")
		}
	}

	// 如果所有关键词都被过滤掉，返回空结果
	if len(conditions) == 0 {
		return nil, nil
	}

	// 构建完整的 SQL 查询
	// 只查询已确认的记录，按创建时间倒序排列
	query := `
		SELECT id, filename, extension, category, subcategory, confidence, keywords, user_confirmed, created_at
		FROM classification_history
		WHERE user_confirmed = 1 AND (` + strings.Join(conditions, " OR ") + `)
		ORDER BY created_at DESC
		LIMIT ?
	`
	args = append(args, limit)

	// 执行查询
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 遍历结果集，构建返回数据
	var records []ClassificationRecord
	for rows.Next() {
		var r ClassificationRecord
		var kwJSON, createdAt string
		if err := rows.Scan(&r.ID, &r.Filename, &r.Extension, &r.Category, &r.Subcategory, &r.Confidence, &kwJSON, &r.UserConfirmed, &createdAt); err != nil {
			continue
		}
		// 反序列化关键词 JSON
		json.Unmarshal([]byte(kwJSON), &r.Keywords)
		// 解析时间字符串
		r.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		records = append(records, r)
	}
	return records, nil
}

// ConfirmClassification 确认分类记录
// 将指定 ID 的分类记录标记为已确认
// 确认后的记录将被用于规则学习
//
// 参数:
//   - id: 要确认的记录 ID
//
// 返回值:
//   - error: 如果更新失败，返回错误
func (d *Database) ConfirmClassification(id int64) error {
	_, err := d.db.Exec("UPDATE classification_history SET user_confirmed = 1 WHERE id = ?", id)
	return err
}

// ==================== 规则操作 ====================
// 以下方法用于管理学习规则的增删改查
// 规则是从用户确认的分类中自动提取的模式

// AddOrUpdateRule 添加或更新学习规则
// 如果规则已存在（相同模式、类型、分类），则增加命中次数并更新优先级
// 如果规则不存在，则创建新规则
// 使用"更新优先"策略减少数据库写入次数
//
// 参数:
//   - pattern: 匹配模式（如关键词 "report"、扩展名 ".pdf"）
//   - patternType: 模式类型（"keyword"、"extension"、"prefix"）
//   - category: 对应的主分类
//   - subcategory: 对应的子分类
//   - priority: 规则优先级
//
// 返回值:
//   - error: 如果操作失败，返回错误
func (d *Database) AddOrUpdateRule(pattern, patternType, category, subcategory string, priority int) error {
	// 统一转换为小写，确保匹配时大小写不敏感
	pattern = strings.ToLower(pattern)

	// 尝试更新已有规则
	// 如果存在相同的 pattern + pattern_type + category 组合，
	// 则增加命中次数，并取当前优先级和传入优先级的较大值
	result, err := d.db.Exec(`
		UPDATE learned_rules 
		SET hit_count = hit_count + 1, 
		    priority = MAX(priority, ?),
		    updated_at = CURRENT_TIMESTAMP
		WHERE pattern = ? AND pattern_type = ? AND category = ?
	`, priority, pattern, patternType, category)

	if err != nil {
		return err
	}

	// 检查是否有记录被更新
	affected, _ := result.RowsAffected()
	if affected == 0 {
		// 没有已存在的规则，插入新规则
		_, err = d.db.Exec(`
			INSERT INTO learned_rules (pattern, pattern_type, category, subcategory, priority, hit_count)
			VALUES (?, ?, ?, ?, ?, 1)
		`, pattern, patternType, category, subcategory, priority)
	}
	return err
}

// GetMatchingRules 获取与给定文件匹配的规则
// 根据文件名、关键词和扩展名查找匹配的学习规则
// 支持两种匹配方式：
// 1. 扩展名精确匹配
// 2. 关键词模糊匹配（模式包含在文件名中）
//
// 参数:
//   - filename: 文件名
//   - keywords: 从文件名提取的关键词列表
//   - ext: 文件扩展名
//
// 返回值:
//   - []LearnedRule: 匹配到的规则列表（已去重）
//   - error: 如果查询失败，返回错误
func (d *Database) GetMatchingRules(filename string, keywords []string, ext string) ([]LearnedRule, error) {
	var rules []LearnedRule
	// 统一转换为小写进行匹配
	filename = strings.ToLower(filename)
	ext = strings.ToLower(ext)

	// ===== 1. 扩展名匹配 =====
	// 查找与文件扩展名完全匹配的规则
	if ext != "" {
		rows, _ := d.db.Query(`
			SELECT id, pattern, pattern_type, category, subcategory, priority, hit_count
			FROM learned_rules 
			WHERE pattern_type = 'extension' AND pattern = ?
			ORDER BY priority DESC, hit_count DESC
			LIMIT 3
		`, ext)
		if rows != nil {
			rules = append(rules, d.scanRules(rows)...)
			rows.Close()
		}
	}

	// ===== 2. 关键词匹配 =====
	// 遍历每个关键词，查找文件名中包含该模式的规则
	for _, kw := range keywords {
		// 跳过过短的关键词
		if len(kw) < 2 {
			continue
		}
		rows, _ := d.db.Query(`
			SELECT id, pattern, pattern_type, category, subcategory, priority, hit_count
			FROM learned_rules 
			WHERE pattern_type = 'keyword' AND ? LIKE '%' || pattern || '%'
			ORDER BY priority DESC, hit_count DESC
			LIMIT 3
		`, filename)
		if rows != nil {
			rules = append(rules, d.scanRules(rows)...)
			rows.Close()
		}
	}

	// ===== 3. 结果去重 =====
	// 使用 pattern|category 作为唯一键去除重复规则
	seen := make(map[string]bool)
	unique := make([]LearnedRule, 0)
	for _, r := range rules {
		key := r.Pattern + "|" + r.Category
		if !seen[key] {
			seen[key] = true
			unique = append(unique, r)
		}
	}
	return unique, nil
}

// scanRules 从数据库行扫描规则数据
// 辅助方法，用于将 sql.Rows 转换为 LearnedRule 切片
//
// 参数:
//   - rows: 数据库查询结果集
//
// 返回值:
//   - []LearnedRule: 解析后的规则列表
func (d *Database) scanRules(rows *sql.Rows) []LearnedRule {
	var rules []LearnedRule
	for rows.Next() {
		var r LearnedRule
		if err := rows.Scan(&r.ID, &r.Pattern, &r.PatternType, &r.Category, &r.Subcategory, &r.Priority, &r.HitCount); err == nil {
			rules = append(rules, r)
		}
	}
	return rules
}

// GetTopRules 获取使用频率最高的规则
// 用于统计展示和规则分析
// 按命中次数和优先级排序
//
// 参数:
//   - limit: 返回结果的最大数量
//
// 返回值:
//   - []map[string]interface{}: 规则信息列表，每个元素包含 pattern、pattern_type、category 等字段
//   - error: 如果查询失败，返回错误
func (d *Database) GetTopRules(limit int) ([]map[string]interface{}, error) {
	rows, err := d.db.Query(`
		SELECT pattern, pattern_type, category, subcategory, hit_count, priority
		FROM learned_rules
		WHERE hit_count >= 1
		ORDER BY hit_count DESC, priority DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 使用 map 作为返回类型，便于 JSON 序列化和灵活访问
	var rules []map[string]interface{}
	for rows.Next() {
		var pattern, ptype, cat, sub string
		var hitCount, priority int
		if rows.Scan(&pattern, &ptype, &cat, &sub, &hitCount, &priority) == nil {
			rules = append(rules, map[string]interface{}{
				"pattern":      pattern,
				"pattern_type": ptype,
				"category":     cat,
				"subcategory":  sub,
				"hit_count":    hitCount,
				"priority":     priority,
			})
		}
	}
	return rules, nil
}

// ==================== 向量操作 ====================
// 以下方法用于管理向量嵌入数据
// 向量用于基于语义相似度的分类匹配

// SaveVector 保存文件名的向量嵌入
// 将文件名及其对应的分类信息和向量存储到数据库
// 向量可以是本地哈希生成或通过 Ollama API 生成
//
// 参数:
//   - filename: 文件名
//   - category: 对应的主分类
//   - subcategory: 对应的子分类
//   - vector: 向量嵌入数据（float64 数组）
//
// 返回值:
//   - error: 如果保存失败，返回错误
func (d *Database) SaveVector(filename, category, subcategory string, vector []float64) error {
	// 将向量序列化为 JSON 字符串存储
	vecJSON, _ := json.Marshal(vector)
	_, err := d.db.Exec(`
		INSERT INTO vectors (filename, category, subcategory, vector)
		VALUES (?, ?, ?, ?)
	`, filename, category, subcategory, vecJSON)
	return err
}

// VectorRecord 向量记录结构体
type VectorRecord struct {
	Filename    string    // 文件名
	Category    string    // 主分类
	Subcategory string    // 子分类
	Vector      []float64 // 向量嵌入数据
}

// SearchVectors 检索存储的向量数据
// 获取最近存储的向量记录，用于相似度计算
// 返回结果按创建时间倒序排列
//
// 参数:
//   - limit: 返回结果的最大数量
//
// 返回值:
//   - 向量记录切片，每个元素包含文件名、分类和向量数据
//   - error: 如果查询失败，返回错误
func (d *Database) SearchVectors(limit int) ([]VectorRecord, error) {
	rows, err := d.db.Query(`
		SELECT filename, category, subcategory, vector
		FROM vectors
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.scanVectorRows(rows)
}

// SearchVectorsByCategories 按分类过滤检索向量数据（优化版本）
// 通过预过滤减少需要比对的向量数量，提升大数据量下的性能
//
// 参数:
//   - categories: 候选分类列表（从规则或历史匹配中获取）
//   - limit: 每个分类返回的最大数量
//
// 返回值:
//   - 向量记录切片
//   - error: 如果查询失败，返回错误
func (d *Database) SearchVectorsByCategories(categories []string, limit int) ([]VectorRecord, error) {
	if len(categories) == 0 {
		// 无分类过滤时，使用默认搜索
		return d.SearchVectors(limit)
	}

	// 构建 IN 查询条件
	placeholders := make([]string, len(categories))
	args := make([]interface{}, len(categories)+1)
	for i, cat := range categories {
		placeholders[i] = "?"
		args[i] = cat
	}
	args[len(categories)] = limit

	query := `
		SELECT filename, category, subcategory, vector
		FROM vectors
		WHERE category IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.scanVectorRows(rows)
}

// SearchVectorsByExtension 按扩展名搜索相关向量
// 利用已有的分类历史，找出该扩展名常被归入的分类
//
// 参数:
//   - ext: 文件扩展名
//   - limit: 返回结果的最大数量
//
// 返回值:
//   - 向量记录切片
//   - error: 如果查询失败，返回错误
func (d *Database) SearchVectorsByExtension(ext string, limit int) ([]VectorRecord, error) {
	// 先查找该扩展名常见的分类
	rows, err := d.db.Query(`
		SELECT DISTINCT category
		FROM classification_history
		WHERE extension = ? AND user_confirmed = 1
		ORDER BY COUNT(*) DESC
		LIMIT 5
	`, ext)
	if err != nil {
		return d.SearchVectors(limit) // 失败时回退到普通搜索
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var cat string
		if rows.Scan(&cat) == nil {
			categories = append(categories, cat)
		}
	}

	if len(categories) == 0 {
		return d.SearchVectors(limit)
	}

	return d.SearchVectorsByCategories(categories, limit)
}

// GetCandidateCategories 获取候选分类列表
// 基于关键词和扩展名，从历史记录中找出可能的分类
// 用于向量搜索的预过滤
//
// 参数:
//   - keywords: 文件名关键词
//   - ext: 文件扩展名
//
// 返回值:
//   - 候选分类列表
func (d *Database) GetCandidateCategories(keywords []string, ext string) []string {
	categorySet := make(map[string]int)

	// 1. 从规则中获取可能的分类
	for _, kw := range keywords {
		if len(kw) < 2 {
			continue
		}
		rows, _ := d.db.Query(`
			SELECT category, hit_count
			FROM learned_rules
			WHERE pattern = ? OR pattern LIKE ?
			ORDER BY hit_count DESC
			LIMIT 3
		`, strings.ToLower(kw), "%"+strings.ToLower(kw)+"%")
		if rows != nil {
			for rows.Next() {
				var cat string
				var count int
				if rows.Scan(&cat, &count) == nil {
					categorySet[cat] += count
				}
			}
			rows.Close()
		}
	}

	// 2. 从扩展名规则获取分类
	if ext != "" {
		rows, _ := d.db.Query(`
			SELECT category, hit_count
			FROM learned_rules
			WHERE pattern_type = 'extension' AND pattern = ?
			ORDER BY hit_count DESC
			LIMIT 3
		`, ext)
		if rows != nil {
			for rows.Next() {
				var cat string
				var count int
				if rows.Scan(&cat, &count) == nil {
					categorySet[cat] += count * 2 // 扩展名权重更高
				}
			}
			rows.Close()
		}
	}

	// 3. 从历史记录中获取分类
	for _, kw := range keywords {
		if len(kw) < 2 {
			continue
		}
		rows, _ := d.db.Query(`
			SELECT category, COUNT(*) as cnt
			FROM classification_history
			WHERE user_confirmed = 1 AND LOWER(filename) LIKE ?
			GROUP BY category
			ORDER BY cnt DESC
			LIMIT 3
		`, "%"+strings.ToLower(kw)+"%")
		if rows != nil {
			for rows.Next() {
				var cat string
				var count int
				if rows.Scan(&cat, &count) == nil {
					categorySet[cat] += count
				}
			}
			rows.Close()
		}
	}

	// 转换为切片并按权重排序
	type catScore struct {
		category string
		score    int
	}
	var scored []catScore
	for cat, score := range categorySet {
		scored = append(scored, catScore{cat, score})
	}

	// 按分数降序排序
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// 返回前5个分类
	var result []string
	for i, cs := range scored {
		if i >= 5 {
			break
		}
		result = append(result, cs.category)
	}
	return result
}

// scanVectorRows 从数据库行扫描向量数据
// 辅助方法，用于将 sql.Rows 转换为 VectorRecord 切片
func (d *Database) scanVectorRows(rows *sql.Rows) ([]VectorRecord, error) {
	var results []VectorRecord
	for rows.Next() {
		var r VectorRecord
		var vecJSON string
		if rows.Scan(&r.Filename, &r.Category, &r.Subcategory, &vecJSON) == nil {
			json.Unmarshal([]byte(vecJSON), &r.Vector)
			results = append(results, r)
		}
	}
	return results, nil
}

// ==================== 反馈操作 ====================
// 以下方法用于管理用户反馈数据
// 用户反馈用于记录分类修正，帮助改进系统

// AddFeedback 添加用户反馈记录
// 当用户修正 AI 的分类结果时，记录原始分类和修正后的分类
// 这些数据可用于分析分类错误模式和改进算法
//
// 参数:
//   - filename: 文件名
//   - origCat: 原始主分类（AI 给出的分类）
//   - corrCat: 修正后的主分类（用户指定的分类）
//   - origSub: 原始子分类
//   - corrSub: 修正后的子分类
//
// 返回值:
//   - error: 如果保存失败，返回错误
func (d *Database) AddFeedback(filename, origCat, corrCat, origSub, corrSub string) error {
	_, err := d.db.Exec(`
		INSERT INTO user_feedback (filename, original_category, corrected_category, original_subcategory, corrected_subcategory)
		VALUES (?, ?, ?, ?, ?)
	`, filename, origCat, corrCat, origSub, corrSub)
	return err
}

// ==================== 统计操作 ====================
// 以下方法用于获取系统统计信息

// GetStatistics 获取系统整体统计信息
// 返回各种统计指标，包括：
// - 总记录数、确认记录数
// - 学习规则数、向量数、反馈数
// - 分类分布（Top 10 分类及其数量）
//
// 返回值:
//   - map[string]interface{}: 统计信息字典
//   - error: 如果查询失败，返回错误
func (d *Database) GetStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// ===== 1. 总记录数 =====
	// 统计 classification_history 表中的记录总数
	var total int
	d.db.QueryRow("SELECT COUNT(*) FROM classification_history").Scan(&total)
	stats["total_records"] = total

	// ===== 2. 已确认记录数 =====
	// 统计经用户确认的记录数量
	var confirmed int
	d.db.QueryRow("SELECT COUNT(*) FROM classification_history WHERE user_confirmed = 1").Scan(&confirmed)
	stats["confirmed_records"] = confirmed

	// ===== 3. 有效规则数 =====
	// 统计至少被命中一次的学习规则数量
	var rules int
	d.db.QueryRow("SELECT COUNT(*) FROM learned_rules WHERE hit_count > 0").Scan(&rules)
	stats["learned_rules"] = rules

	// ===== 4. 向量数 =====
	// 统计存储的向量嵌入数量
	var vectors int
	d.db.QueryRow("SELECT COUNT(*) FROM vectors").Scan(&vectors)
	stats["vector_count"] = vectors

	// ===== 5. 反馈数 =====
	// 统计用户反馈记录数量
	var feedback int
	d.db.QueryRow("SELECT COUNT(*) FROM user_feedback").Scan(&feedback)
	stats["feedback_count"] = feedback

	// ===== 6. 分类分布 =====
	// 统计各分类的记录数量，返回 Top 10
	rows, _ := d.db.Query(`
		SELECT category, COUNT(*) as cnt 
		FROM classification_history 
		GROUP BY category 
		ORDER BY cnt DESC 
		LIMIT 10
	`)
	if rows != nil {
		defer rows.Close()
		dist := make(map[string]int)
		for rows.Next() {
			var cat string
			var cnt int
			if rows.Scan(&cat, &cnt) == nil {
				dist[cat] = cnt
			}
		}
		stats["category_distribution"] = dist
	}

	return stats, nil
}

// ==================== 重置操作 ====================
// 以下方法用于清空数据库中的数据
// 提供细粒度的重置控制，可以单独重置某类数据或全部重置

// ResetHistory 重置分类历史记录
// 清空 classification_history 表中的所有数据
// 警告：此操作不可恢复，请谨慎使用
//
// 返回值:
//   - error: 如果删除失败，返回错误
func (d *Database) ResetHistory() error {
	_, err := d.db.Exec("DELETE FROM classification_history")
	return err
}

// ResetRules 重置学习规则
// 清空 learned_rules 表中的所有数据
// 这将导致系统失去所有学习到的分类规则
// 警告：此操作不可恢复，请谨慎使用
//
// 返回值:
//   - error: 如果删除失败，返回错误
func (d *Database) ResetRules() error {
	_, err := d.db.Exec("DELETE FROM learned_rules")
	return err
}

// ResetVectors 重置向量数据
// 清空 vectors 表中的所有数据
// 这将导致基于向量相似度的匹配无法使用
// 警告：此操作不可恢复，请谨慎使用
//
// 返回值:
//   - error: 如果删除失败，返回错误
func (d *Database) ResetVectors() error {
	_, err := d.db.Exec("DELETE FROM vectors")
	return err
}

// ResetAll 重置所有数据
// 清空所有数据表：
// - classification_history（分类历史）
// - learned_rules（学习规则）
// - user_feedback（用户反馈）
// - vectors（向量数据）
// - operation_logs（操作日志）
//
// 这将使系统恢复到初始状态，失去所有学习记忆
// 警告：此操作不可恢复，请谨慎使用
//
// 返回值:
//   - error: 如果任何表删除失败，返回错误
func (d *Database) ResetAll() error {
	// 需要清空的所有表
	tables := []string{"classification_history", "learned_rules", "user_feedback", "vectors", "operation_logs"}

	// 依次清空每个表
	for _, t := range tables {
		if _, err := d.db.Exec("DELETE FROM " + t); err != nil {
			return err
		}
	}
	return nil
}

// ==================== 操作日志 ====================
// 以下方法用于管理操作日志，支持撤销功能

// OperationLog 操作日志记录
type OperationLog struct {
	ID          int64     // 记录 ID
	BatchID     string    // 批次 ID（同一次整理操作的唯一标识）
	SourcePath  string    // 原始路径
	DestPath    string    // 目标路径
	Filename    string    // 文件名
	Category    string    // 分类
	Subcategory string    // 子分类
	Status      string    // 状态: success, failed, undone
	CreatedAt   time.Time // 创建时间
}

// AddOperationLog 添加操作日志
// 记录文件移动操作，用于撤销功能
//
// 参数:
//   - batchID: 批次 ID
//   - sourcePath: 原始文件路径
//   - destPath: 目标文件路径
//   - filename: 文件名
//   - category: 主分类
//   - subcategory: 子分类
//   - status: 操作状态
//
// 返回值:
//   - error: 如果插入失败，返回错误
func (d *Database) AddOperationLog(batchID, sourcePath, destPath, filename, category, subcategory, status string) error {
	_, err := d.db.Exec(`
		INSERT INTO operation_logs (batch_id, source_path, dest_path, filename, category, subcategory, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, batchID, sourcePath, destPath, filename, category, subcategory, status)
	return err
}

// GetRecentBatches 获取最近的操作批次
// 用于显示可撤销的操作列表
//
// 参数:
//   - limit: 返回结果的最大数量
//
// 返回值:
//   - 批次信息列表
//   - error: 如果查询失败，返回错误
func (d *Database) GetRecentBatches(limit int) ([]map[string]interface{}, error) {
	rows, err := d.db.Query(`
		SELECT batch_id, 
		       COUNT(*) as file_count, 
		       MIN(created_at) as created_at,
		       GROUP_CONCAT(DISTINCT category) as categories
		FROM operation_logs
		WHERE status = 'success'
		GROUP BY batch_id
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batches []map[string]interface{}
	for rows.Next() {
		var batchID, createdAt, categories string
		var fileCount int
		if rows.Scan(&batchID, &fileCount, &createdAt, &categories) == nil {
			batches = append(batches, map[string]interface{}{
				"batch_id":   batchID,
				"file_count": fileCount,
				"created_at": createdAt,
				"categories": categories,
			})
		}
	}
	return batches, nil
}

// GetBatchLogs 获取指定批次的所有操作日志
//
// 参数:
//   - batchID: 批次 ID
//
// 返回值:
//   - 操作日志列表
//   - error: 如果查询失败，返回错误
func (d *Database) GetBatchLogs(batchID string) ([]OperationLog, error) {
	rows, err := d.db.Query(`
		SELECT id, batch_id, source_path, dest_path, filename, category, subcategory, status, created_at
		FROM operation_logs
		WHERE batch_id = ? AND status = 'success'
		ORDER BY id ASC
	`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []OperationLog
	for rows.Next() {
		var log OperationLog
		var createdAt string
		if rows.Scan(&log.ID, &log.BatchID, &log.SourcePath, &log.DestPath, &log.Filename, &log.Category, &log.Subcategory, &log.Status, &createdAt) == nil {
			log.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
			logs = append(logs, log)
		}
	}
	return logs, nil
}

// MarkBatchUndone 标记批次为已撤销
//
// 参数:
//   - batchID: 批次 ID
//
// 返回值:
//   - error: 如果更新失败，返回错误
func (d *Database) MarkBatchUndone(batchID string) error {
	_, err := d.db.Exec(`
		UPDATE operation_logs
		SET status = 'undone'
		WHERE batch_id = ?
	`, batchID)
	return err
}

// GetLatestBatch 获取最近一次操作的批次 ID
//
// 返回值:
//   - 批次 ID，如果没有则返回空字符串
func (d *Database) GetLatestBatch() string {
	var batchID string
	d.db.QueryRow(`
		SELECT batch_id
		FROM operation_logs
		WHERE status = 'success'
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(&batchID)
	return batchID
}

// ==================== 模型性能统计 ====================
// 以下方法用于记录和分析模型执行性能

// ModelStats 模型性能统计记录
type ModelStats struct {
	ID               int64     // 记录 ID
	ModelName        string    // 模型名称
	BatchID          string    // 批次 ID
	FileCount        int       // 处理的文件数
	TotalTimeMs      int64     // 总耗时（毫秒）
	AvgTimePerFileMs float64   // 平均每文件耗时（毫秒）
	AvgConfidence    float64   // 平均置信度
	ConfirmedCount   int       // 用户确认数
	CorrectedCount   int       // 用户纠正数
	AccuracyRate     float64   // 准确率（确认数/(确认数+纠正数)）
	CreatedAt        time.Time // 创建时间
}

// ModelSummary 模型综合评估
type ModelSummary struct {
	ModelName        string  // 模型名称
	TotalBatches     int     // 总批次数
	TotalFiles       int     // 总处理文件数
	AvgTimePerFileMs float64 // 平均每文件耗时
	AvgConfidence    float64 // 平均置信度
	TotalConfirmed   int     // 总确认数
	TotalCorrected   int     // 总纠正数
	AccuracyRate     float64 // 综合准确率
	Score            float64 // 综合评分（用于排序推荐）
	LastUsed         string  // 最后使用时间
}

// AddModelStats 添加模型性能统计记录
func (d *Database) AddModelStats(modelName, batchID string, fileCount int, totalTimeMs int64, avgConfidence float64) error {
	avgTimePerFile := float64(0)
	if fileCount > 0 {
		avgTimePerFile = float64(totalTimeMs) / float64(fileCount)
	}

	_, err := d.db.Exec(`
		INSERT INTO model_stats (model_name, batch_id, file_count, total_time_ms, avg_time_per_file_ms, avg_confidence)
		VALUES (?, ?, ?, ?, ?, ?)
	`, modelName, batchID, fileCount, totalTimeMs, avgTimePerFile, avgConfidence)
	return err
}

// UpdateModelAccuracy 更新模型准确度统计
// 当用户确认或纠正分类时调用
func (d *Database) UpdateModelAccuracy(batchID string, confirmed, corrected int) error {
	accuracyRate := float64(0)
	total := confirmed + corrected
	if total > 0 {
		accuracyRate = float64(confirmed) / float64(total)
	}

	_, err := d.db.Exec(`
		UPDATE model_stats
		SET confirmed_count = confirmed_count + ?,
		    corrected_count = corrected_count + ?,
		    accuracy_rate = ?
		WHERE batch_id = ?
	`, confirmed, corrected, accuracyRate, batchID)
	return err
}

// GetModelSummaries 获取所有模型的综合性能统计
// 用于模型对比和推荐
func (d *Database) GetModelSummaries() ([]ModelSummary, error) {
	rows, err := d.db.Query(`
		SELECT 
			model_name,
			COUNT(*) as total_batches,
			SUM(file_count) as total_files,
			AVG(avg_time_per_file_ms) as avg_time_per_file,
			AVG(avg_confidence) as avg_confidence,
			SUM(confirmed_count) as total_confirmed,
			SUM(corrected_count) as total_corrected,
			MAX(created_at) as last_used
		FROM model_stats
		GROUP BY model_name
		ORDER BY total_files DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []ModelSummary
	for rows.Next() {
		var s ModelSummary
		var totalConfirmed, totalCorrected sql.NullInt64
		if rows.Scan(&s.ModelName, &s.TotalBatches, &s.TotalFiles, &s.AvgTimePerFileMs, &s.AvgConfidence, &totalConfirmed, &totalCorrected, &s.LastUsed) == nil {
			s.TotalConfirmed = int(totalConfirmed.Int64)
			s.TotalCorrected = int(totalCorrected.Int64)

			// 计算准确率
			total := s.TotalConfirmed + s.TotalCorrected
			if total > 0 {
				s.AccuracyRate = float64(s.TotalConfirmed) / float64(total)
			} else {
				s.AccuracyRate = s.AvgConfidence // 无反馈时用置信度估算
			}

			// 计算综合评分（准确率 * 0.5 + 置信度 * 0.3 + 速度分 * 0.2）
			// 速度分：基于平均耗时，越快分越高（假设 500ms 为基准）
			speedScore := 1.0 - (s.AvgTimePerFileMs / 1000.0)
			if speedScore < 0 {
				speedScore = 0
			}
			if speedScore > 1 {
				speedScore = 1
			}
			s.Score = s.AccuracyRate*0.5 + s.AvgConfidence*0.3 + speedScore*0.2

			summaries = append(summaries, s)
		}
	}

	// 按综合评分排序
	for i := 0; i < len(summaries)-1; i++ {
		for j := i + 1; j < len(summaries); j++ {
			if summaries[j].Score > summaries[i].Score {
				summaries[i], summaries[j] = summaries[j], summaries[i]
			}
		}
	}

	return summaries, nil
}

// GetBestModel 获取综合评分最高的模型
// 用于自适应模型选择
func (d *Database) GetBestModel() string {
	summaries, err := d.GetModelSummaries()
	if err != nil || len(summaries) == 0 {
		return ""
	}

	// 只有当模型处理过足够多的文件时才推荐
	for _, s := range summaries {
		if s.TotalFiles >= 10 { // 至少处理过10个文件
			return s.ModelName
		}
	}
	return ""
}

// GetModelRecentStats 获取指定模型的最近统计记录
func (d *Database) GetModelRecentStats(modelName string, limit int) ([]ModelStats, error) {
	rows, err := d.db.Query(`
		SELECT id, model_name, batch_id, file_count, total_time_ms, avg_time_per_file_ms, 
		       avg_confidence, confirmed_count, corrected_count, accuracy_rate, created_at
		FROM model_stats
		WHERE model_name = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, modelName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []ModelStats
	for rows.Next() {
		var s ModelStats
		var createdAt string
		if rows.Scan(&s.ID, &s.ModelName, &s.BatchID, &s.FileCount, &s.TotalTimeMs, &s.AvgTimePerFileMs,
			&s.AvgConfidence, &s.ConfirmedCount, &s.CorrectedCount, &s.AccuracyRate, &createdAt) == nil {
			s.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
			stats = append(stats, s)
		}
	}
	return stats, nil
}

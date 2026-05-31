// Package llm Ollama LLM 客户端模块
// 封装与 Ollama API 的交互，提供聊天、嵌入和文件分类功能
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/lynx-lee/filo/internal/config"
)

// ==================== 类型定义 ====================

// Client Ollama API 客户端
// 封装 HTTP 请求，提供与 Ollama 服务交互的方法
type Client struct {
	baseURL    string       // Ollama 服务地址
	model      string       // 当前使用的模型
	httpClient *http.Client // HTTP 客户端（带超时）
}

// ChatMessage 聊天消息结构
// 用于构建 LLM 对话上下文
type ChatMessage struct {
	Role    string `json:"role"`    // 角色: system/user/assistant
	Content string `json:"content"` // 消息内容
}

// ==================== 构造函数 ====================

// NewClient 创建 Ollama 客户端
// 从配置中获取服务地址和模型信息
func NewClient() *Client {
	cfg := config.Get()
	return &Client{
		baseURL: cfg.OllamaURL,
		model:   cfg.LLMModel,
		httpClient: &http.Client{
			Timeout: 180 * time.Second, // 3分钟超时（模型推理可能较慢）
		},
	}
}

// ==================== 服务检查方法 ====================

// IsAvailable 检查 Ollama 服务是否可用
// 通过访问 /api/tags 接口判断服务状态
func (c *Client) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// HasModel 检查指定模型是否已安装
// 在已安装的模型列表中查找
func (c *Client) HasModel(model string) bool {
	models, err := c.ListModels()
	if err != nil {
		return false
	}
	for _, m := range models {
		if m == model {
			return true
		}
	}
	return false
}

// ListModels 列出所有已安装的模型
// 调用 /api/tags 接口获取模型列表
func (c *Client) ListModels() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Models []struct {
			Name string `json:"name"` // 模型名称
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// 提取模型名称列表
	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}
	return models, nil
}

// ==================== 核心 API 方法 ====================

// Chat 发送聊天请求
// 支持多轮对话和 JSON 输出模式
func (c *Client) Chat(ctx context.Context, messages []ChatMessage, jsonMode bool) (string, error) {
	cfg := config.Get()

	// 构建请求体
	options := map[string]interface{}{
		"temperature": cfg.Temperature, // 使用配置的温度
		"num_predict": 4096,            // 最大生成 token 数
	}

	payload := map[string]interface{}{
		"model":    c.model,
		"messages": messages,
		"stream":   false, // 非流式输出（批量分类需要完整 JSON）
		"options":  options,
	}

	// JSON 模式：强制模型输出 JSON 格式
	if jsonMode {
		payload["format"] = "json"
		// 关闭 qwen3 的 thinking 模式，加速响应
		options["thinking"] = false
	}

	// 发送 POST 请求
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	startTime := time.Now() // 记录请求开始时间
	resp, err := c.httpClient.Do(req)
	elapsed := time.Since(startTime) // 计算耗时
	
	if err != nil {
		return "", fmt.Errorf("请求失败: %w (耗时: %v)", err, elapsed)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API错误 %d: %s (耗时: %v)", resp.StatusCode, string(body), elapsed)
	}

	// 解析响应
	var chatResp struct {
		Message ChatMessage `json:"message"` // 助手回复
	}
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", err
	}

	return chatResp.Message.Content, nil
}

// Embed 获取文本的向量嵌入
// 调用 /api/embeddings 接口生成文本向量
func (c *Client) Embed(ctx context.Context, text string) ([]float64, error) {
	cfg := config.Get()

	// 构建请求体
	payload := map[string]string{
		"model":  cfg.EmbeddingModel, // 使用嵌入模型
		"prompt": text,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("嵌入API错误: %d", resp.StatusCode)
	}

	// 解析响应
	var embResp struct {
		Embedding []float64 `json:"embedding"` // 向量数组
	}
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, err
	}

	return embResp.Embedding, nil
}

// ==================== 文件分类方法 ====================

// ClassifyFiles 批量分类文件
// 构建提示词让 LLM 对文件进行智能分类
func (c *Client) ClassifyFiles(ctx context.Context, files []map[string]interface{}, rules []map[string]string) (map[string]interface{}, error) {
	// 构建系统提示词和用户提示词
	systemPrompt := buildSystemPrompt(rules)
	userPrompt := buildUserPrompt(files)

	// 组装对话消息
	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt}, // 系统指令
		{Role: "user", Content: userPrompt},     // 用户请求
	}

	// 调用 LLM（启用 JSON 模式）
	response, err := c.Chat(ctx, messages, true)
	if err != nil {
		return nil, err
	}

	// 解析 JSON 响应
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// 尝试从响应中提取 JSON（处理模型可能添加的额外文字）
		re := regexp.MustCompile(`\{[\s\S]*\}`)
		if match := re.FindString(response); match != "" {
			if err := json.Unmarshal([]byte(match), &result); err != nil {
				return nil, fmt.Errorf("解析失败: %w", err)
			}
		} else {
			return nil, fmt.Errorf("无法解析响应")
		}
	}

	return result, nil
}

// ==================== 提示词构建函数 ====================

// buildSystemPrompt 构建系统提示词
// 定义分类规则和输出格式要求
func buildSystemPrompt(rules []map[string]string) string {
	prompt := `你是一个专业的文件分类专家。请根据文件名智能分类，理解文件的实际用途和语义。

【分类原则】
1. 基于文件名语义进行分类，不要仅看扩展名
2. 识别项目名、客户名、业务领域关键词
3. 注意日期、版本号、状态标识（如 draft, final, v2）
4. 相关文件应归入同一类别
5. 如果无法确定，使用"其他"分类

【常用分类体系】
- 工作文档：合同、报告、方案、会议纪要、简历
- 个人照片：自拍、旅行、聚会、证件照
- 设计素材：UI稿、图标、海报、Logo
- 视频资料：教程、录屏、会议记录、电影
- 音频文件：音乐、录音、播客、语音笔记
- 代码文件：源码、配置文件、脚本、数据库
- 压缩归档：备份、资料包、安装包
- 数据表格：Excel、CSV、数据库导出
- 演示文稿：PPT、Keynote、演讲稿
- 电子书籍：PDF书籍、技术文档、论文

【输出要求】
1. 必须返回有效的 JSON 格式
2. confidence 必须是 0.0-1.0 之间的浮点数
3. reasoning 用中文简要说明分类理由（不超过50字）
4. keywords 提取2-5个关键特征词
5. category 和 subcategory 要具体明确

【示例输出格式】
{
  "classifications": [
    {
      "filename": "2024年度财务报告.pdf",
      "category": "工作文档",
      "subcategory": "财务报告",
      "confidence": 0.95,
      "reasoning": "包含年份和业务类型关键词，属于正式财务文档",
      "keywords": ["年度", "财务", "报告"]
    }
  ]
}`

	// 如果有已学习的规则，添加到提示词中
	if len(rules) > 0 {
		prompt += "\n\n已学习的分类规则（优先参考）：\n"
		for i, r := range rules {
			if i >= 20 { // 最多包含20条规则
				break
			}
			prompt += fmt.Sprintf("- 「%s」→ %s/%s\n", r["pattern"], r["category"], r["subcategory"])
		}
	}

	return prompt
}

// buildUserPrompt 构建用户提示词
// 将文件列表格式化为 JSON，要求 LLM 返回分类结果
func buildUserPrompt(files []map[string]interface{}) string {
	filesJSON, _ := json.MarshalIndent(files, "", "  ")
	return fmt.Sprintf(`请对以下 %d 个文件进行分类：

%s

返回JSON格式：
{
  "classifications": [
    {
      "filename": "文件名",
      "category": "主分类",
      "subcategory": "子分类",
      "confidence": 0.95,
      "reasoning": "分类理由",
      "keywords": ["关键词"]
    }
  ]
}`, len(files), string(filesJSON))
}

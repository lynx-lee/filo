# Filo - 文件智理，越用越懂你

```
  ███████╗██╗██╗      ██████╗ 
  ██╔════╝██║██║     ██╔═══██╗
  █████╗  ██║██║     ██║   ██║
  ██╔══╝  ██║██║     ██║   ██║
  ██║     ██║███████╗╚██████╔╝
  ╚═╝     ╚═╝╚══════╝ ╚═════╝ 
```

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-macOS%20|%20Linux%20|%20Windows-lightgrey)](https://github.com/lynx-lee/filo/releases)
[![Version](https://img.shields.io/badge/Version-v2.1.0--alpha-blue)](https://github.com/lynx-lee/filo/releases)

**Filo** 是一款基于本地 AI 的智能文件整理工具，通过学习你的整理习惯，让文件归类越来越精准、越来越快速。

> **🎉 v2.1.0-alpha 新特性**：引入多智能体协作系统（Multi-Agent System），包含规划器、执行器、评估器、优化器和学习器五个智能体，实现更智能、更透明、更自主的文件分类！详见 [智能体架构文档](docs/AGENT_ARCHITECTURE.md)

## ✨ 特性

### 🤖 智能体系统（v2.1+）
- **多智能体协作** - Planner、Executor、Evaluator、Optimizer、Learner 五大智能体协同工作
- **智能任务规划** - 自动分析文件复杂度，制定最优分类策略
- **结果优化引擎** - 一致性检查、异常值检测、分类规范化
- **持续学习能力** - 从每次分类中学习模式，准确率持续提升
- **质量评估反馈** - 实时评估分类质量，提供改进建议
- **透明可解释** - 每步决策都有清晰日志，完全可追溯

### 🧠 核心功能
- **本地 AI 驱动** - 基于 Ollama，数据不出本地，隐私安全
- **混合推理** - 记忆优先，AI 兜底，速度与准确率兼得
- **语义理解** - 不只看扩展名，理解文件名含义进行分类
- **交互审查** - 支持预览、确认、纠正，让你完全掌控
- **智能选模型** - 自动追踪模型性能，推荐最优模型
- **断点续传** - 大批量处理中断后可恢复
- **支持撤销** - 整理操作可撤销，放心使用
- **零配置** - 开箱即用，一键安装向导

## 📦 安装

### 方式一：一键安装（推荐）

**macOS / Linux:**
```bash
curl -fsSL https://raw.githubusercontent.com/lynx-lee/filo/main/install.sh | bash
```

**Windows:**
```powershell
# 以管理员身份运行 PowerShell，然后执行：
Set-ExecutionPolicy RemoteSigned -Scope Process
Invoke-Expression (Invoke-WebRequest -Uri "https://raw.githubusercontent.com/lynx-lee/filo/main/install.ps1" -UseBasicParsing).Content
```

> **注意**：
> - 需要管理员权限才能安装到 `C:\Program Files\Filo` 并添加到系统 PATH
> - 如果 Windows Defender 拦截，请点击“仍要运行”
> - 支持 x86_64、ARM64 和 x86 架构
> - **当前版本提示**：由于尚未发布正式 Release，一键安装可能会失败。建议使用下面的源码编译方式或 Go 安装

### 方式二：源码编译

```bash
# 克隆仓库
git clone https://github.com/lynx-lee/filo.git
cd filo

# 编译安装
make build
make install
```

**Windows 用户：**
```powershell
# 在 PowerShell 中执行
git clone https://github.com/lynx-lee/filo.git
cd filo
make build  # 或者: go build -o filo.exe .
```

### 方式三：Go 安装

```bash
go install github.com/lynx-lee/filo@latest
```

## 🔧 前置依赖

Filo 依赖 [Ollama](https://ollama.com) 运行本地大语言模型：

```bash
# macOS
brew install ollama

# Linux
curl -fsSL https://ollama.com/install.sh | sh

# Windows
# 访问 https://ollama.com/download/windows 下载安装
```

## 🚀 快速开始

### 1. 运行安装向导

```bash
filo setup
```

安装向导会自动：
- 检查 Ollama 是否安装
- 启动 Ollama 服务
- 下载推荐模型 (qwen3:8b)

### 2. 预览整理效果

```bash
filo ~/Downloads -n
```

`-n` 参数表示预览模式，不会实际移动文件。

### 3. 执行整理

```bash
filo ~/Downloads
```

### 4. 交互式审查

```bash
filo ~/Downloads -i
```

对低置信度的分类进行人工确认或纠正，系统会学习你的选择。

## 📖 命令详解

```bash
filo [目录] [选项]      # 整理指定目录

选项:
  -n, --dry-run         预览模式，不执行实际操作
  -i, --interactive     交互式审查模式
  -r, --recursive       递归扫描子目录
  -t, --target <目录>   指定目标目录（默认: 源目录/已整理）
  -m, --model <模型>    指定使用的模型
  -v, --verbose         详细输出
  --no-learning         禁用学习功能

子命令:
  filo setup            运行安装向导
  filo stats            查看学习统计
  filo config           查看/修改配置
  filo scan <目录>      扫描目录统计
  filo models           查看可用模型
  filo reset            重置学习数据
  filo undo             撤销整理操作
  filo version          查看版本信息
```

## 📊 使用示例

### 传统模式（默认）
```bash
# 整理下载文件夹
filo ~/Downloads

# 预览整理效果
filo ~/Downloads -n

# 递归整理子目录
filo ~/Downloads -r

# 交互式审查，适合首次使用
filo ~/Downloads -i

# 指定目标目录
filo ~/Downloads -t ~/Documents/已整理

# 使用其他模型
filo ~/Downloads -m llama3.2:3b
```

### 🤖 智能体模式（v2.1+ 实验性）
```bash
# 启用智能体工作流
filo ~/Downloads --agents
# 或简写
filo ~/Downloads -a

# 智能体 + 详细输出
filo ~/Downloads -a -v

# 智能体 + 预览模式
filo ~/Downloads -a -n

# 智能体 + 递归扫描
filo ~/Downloads -a -r
```

**智能体模式输出示例：**
```
🤖 启动智能体工作流...

📋 [Planner] 分析文件集合并制定策略...
✓ 复杂度: medium
✓ 预估时间: 2m30s
✓ 批次数: 8
✓ 使用记忆优先: true
✓ 使用 LLM: true

⚙️  [Executor] 执行文件分类...
✓ 分类完成: 120 个文件

✨ [Optimizer] 优化分类结果...
✓ 优化改进率: 15.0%
✓ 发现问题: 3 个

🔍 [Evaluator] 评估分类质量...
✓ 平均置信度: 87.50%
✓ 总体质量评分: 0.92/1.00
✓ 低置信度文件: 3

📚 [Learner] 从本次分类中学习...
✓ 学习新模式: 8 个
✓ 总模式数: 45 个

✅ 智能体工作流完成 (耗时: 2m15s)

📈 智能体性能指标:
  Planner: 任务数=1, 成功率=100.0%
  Executor: 处理文件=120, 平均耗时=1125ms
  Evaluator: 评估数=120, 平均质量=0.92
  Optimizer: 优化规则数=4
  Learner: 总反馈数=0, 准确率=N/A
```

# 查看学习统计
filo stats

# 查看/修改配置
filo config
filo config --model qwen3:8b
filo config --threshold 0.8

# 扫描目录信息
filo scan ~/Downloads

# 模型管理
filo models                # 查看可用模型
filo models --stats        # 查看模型性能对比
filo models --recommend    # 查看推荐模型

# 撤销整理操作
filo undo                  # 撤销最近一次
filo undo --list           # 查看可撤销列表

# 重置所有学习数据
filo reset --all
```

## 🧠 工作原理

Filo 采用 **"记忆优先，AI 兜底"** 的混合推理策略：

```
文件输入
    │
    ▼
┌─────────────────┐
│  1. 规则匹配    │  ← 已学习的分类规则（最快）
└────────┬────────┘
         │ 未命中
         ▼
┌─────────────────┐
│  2. 向量检索    │  ← 相似文件名语义匹配
└────────┬────────┘
         │ 未命中
         ▼
┌─────────────────┐
│  3. 历史匹配    │  ← 历史分类记录
└────────┬────────┘
         │ 未命中
         ▼
┌─────────────────┐
│  4. LLM 推理    │  ← AI 智能分类（最准）
└────────┬────────┘
         │
         ▼
    分类结果 → 学习入库
```

### 学习机制

- **自动学习**: 每次整理自动记录分类结果
- **确认强化**: 用户确认的分类获得更高权重
- **纠正学习**: 用户纠正会生成高优先级规则
- **规则提取**: 从高频分类中自动提取关键词规则

## 📁 项目结构

```
filo/
├── main.go                      # 程序入口
├── go.mod / go.sum              # Go 模块依赖
├── Makefile                     # 构建脚本
├── install.sh                   # 一键安装脚本
├── cmd/
│   ├── root.go                  # 主命令（整理）
│   ├── setup.go                 # 安装向导
│   ├── stats.go                 # 学习统计
│   ├── config.go                # 配置管理
│   ├── scan.go                  # 文件扫描
│   ├── models.go                # 模型管理
│   ├── reset.go                 # 重置数据
│   ├── undo.go                  # 撤销操作
│   └── version.go               # 版本信息
└── internal/
    ├── config/config.go         # 配置管理
    ├── llm/ollama.go            # Ollama API 客户端
    ├── embedding/embedding.go   # 向量嵌入（本地/Ollama）
    ├── scanner/scanner.go       # 文件扫描器
    ├── classifier/classifier.go # 智能分类器
    ├── organizer/organizer.go   # 文件整理器
    ├── memory/memory.go         # 记忆系统
    ├── storage/database.go      # SQLite 数据存储
    └── ui/ui.go                 # 终端界面
```

## ⚙️ 配置

配置文件位于 `~/.filo/config.json`：

```json
{
  "llm_model": "qwen3:8b",
  "embedding_model": "nomic-embed-text",
  "ollama_url": "http://localhost:11434",
  "temperature": 0.3,
  "max_tokens": 2048,
  "enable_learning": true,
  "similarity_threshold": 0.85,
  "confidence_threshold": 0.7,
  "min_samples_for_rule": 3,
  "batch_size": 15
}
```

### 配置说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `llm_model` | `qwen3:8b` | 分类使用的 LLM 模型 |
| `embedding_model` | `nomic-embed-text` | 向量嵌入模型 |
| `ollama_url` | `http://localhost:11434` | Ollama 服务地址 |
| `temperature` | `0.3` | 模型温度（越低越确定） |
| `enable_learning` | `true` | 是否启用学习功能 |
| `similarity_threshold` | `0.85` | 相似度匹配阈值 |
| `confidence_threshold` | `0.7` | 置信度阈值 |
| `batch_size` | `15` | 批量分类大小 |

## 🗄️ 数据存储

学习数据存储在 `~/.filo/memory.db` (SQLite)：

- **classification_history** - 分类历史记录
- **learned_rules** - 学习到的规则
- **vectors** - 文件名向量嵌入
- **user_feedback** - 用户反馈记录
- **operation_logs** - 操作日志（支持撤销）
- **model_stats** - 模型性能统计（自适应选择）

## 🔌 推荐模型

| 模型 | 大小 | 特点 | 推荐场景 |
|------|------|------|----------|
| `qwen3:8b` | 4.9GB | 中英文优秀，推理准确 | **默认推荐** |
| `qwen3:4b` | 2.6GB | 轻量快速 | 内存受限 |
| `llama3.2:3b` | 2.0GB | 英文优秀 | 英文文件为主 |
| `gemma2:9b` | 5.4GB | 综合能力强 | 追求准确率 |

安装模型：
```bash
ollama pull qwen3:8b
```

## 🛠️ 开发

```bash
# 克隆仓库
git clone https://github.com/lynx-lee/filo.git
cd filo

# 安装依赖
make deps

# 开发运行
make run

# 构建
make build

# 构建所有平台（包括 Windows x86_64/ARM64/x86）
make build-all

# 运行测试
make test
```

**Windows 开发者注意：**
- 需要安装 [Make for Windows](https://gnuwin32.sourceforge.net/packages/make.htm) 或使用 WSL
- 或者直接使用 Go 命令：`go build -o filo.exe .`
- **交叉编译提示**：由于 SQLite 依赖限制，建议在 Windows 本机执行 `make build-windows` 进行构建

## 📝 Changelog

### v2.1.0-alpha (2026-05-30)
- 🤖 **引入多智能体系统** - Planner、Executor、Evaluator、Optimizer、Learner 五大智能体
- 📋 **智能任务规划** - 自动分析复杂度，制定最优策略
- ✨ **结果优化引擎** - 一致性检查、异常值检测、分类规范化
- 📚 **持续学习能力** - 从每次分类中学习模式，准确率持续提升
- 🔍 **质量评估反馈** - 实时评估分类质量，提供改进建议
- 🔄 **事件驱动通信** - EventBus 实现松耦合智能体协作
- 💾 **断点续传支持** - 大批量处理中断后可恢复
- 📊 **性能监控增强** - API 耗时监控、数据库连接池优化
- 🛡️ **配置验证** - 自动修正无效配置值
- 🎯 **提示词工程优化** - 更详细的分类体系和输出规范

### v2.0.1
- 🎉 新增 Windows 电脑安装脚本支持
- 🪟 完善 Windows 跨平台支持（x86_64、ARM64、x86）
- 🔧 修复 Go 模块路径冲突问题

### v2.0.0
- 🎉 首个公开版本
- 🧠 混合推理引擎（规则 + 向量 + 历史 + LLM）
- 📚 持续学习系统
- 💬 交互式审查模式
- 🎨 美化终端界面

## 📚 技术文档

- [智能体架构设计](docs/AGENT_ARCHITECTURE.md) - 多智能体系统详细说明
- [实施报告](docs/IMPLEMENTATION_REPORT.md) - 完整的技术实施方案和中长期规划

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送分支 (`git push origin feature/amazing-feature`)
5. 提交 Pull Request

## 📄 License

[MIT License](LICENSE) © 2024-2026 [lynx-lee](https://github.com/lynx-lee)

---

<p align="center">
  <b>Filo</b> - 文件智理，越用越懂你 🧠
</p>

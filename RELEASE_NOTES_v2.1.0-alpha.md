# 🎉 Filo v2.1.0-alpha - 智能体系统正式发布

## 📋 版本概览

这是 Filo 的重大更新，引入了基于 ReAct 模式的多智能体协作系统，让文件整理更加智能、透明和自主。

---

## ✨ 主要特性

### 🤖 多智能体协作系统（NEW）

引入 5 个专业智能体协同工作：

- **📋 Planner Agent** - 智能任务规划器，分析文件复杂度并制定最优策略
- **⚙️ Executor Agent** - 执行分类任务，调用 LLM 进行智能分类
- **✨ Optimizer Agent** - 优化分类结果，确保一致性和质量
  - 相似文件归类一致性检查
  - 低置信度标记与重新评估
  - 异常值检测
  - 分类层级规范化
- **🔍 Evaluator Agent** - 实时评估分类质量，提供改进建议
- **📚 Learner Agent** - 持续学习机制，从每次分类中提取模式

### 🚀 性能优化

- **并行批次处理** - 同时处理 3 个批次，总耗时减少约 60%
- **关闭 Thinking 模式** - qwen3:8b 响应速度提升 50-70%
- **流式进度更新** - 实时显示分类进度
- **超时时间优化** - 适应较慢的模型（最长 10 分钟）

### 🛠️ 核心功能增强

- **元数据追踪** - Result.Metadata 用于优化过程标记
- **线程安全** - 使用 sync.Mutex 保护共享数据
- **重试机制** - LLM 调用失败自动重试（最多 2 次）
- **错误恢复** - 单个批次失败不影响其他批次

### 🔧 用户体验改进

- **动态版本号** - Banner 显示正确的版本号
- **详细错误日志** - 清晰展示 LLM 调用失败原因
- **备用下载地址** - install.sh 支持 ghproxy.net 镜像
- **权限自动设置** - 安装脚本默认具有执行权限

---

## 📥 安装方式

### macOS/Linux

#### 方法一：源码编译（推荐）

```bash
git clone https://github.com/lynx-lee/filo.git
cd filo
make build
sudo make install
```

#### 方法二：Go 安装

```bash
go install github.com/lynx-lee/filo@latest
```

#### 方法三：下载预编译二进制

从本页面下载对应平台的二进制文件：
- `filo-mac-arm64` - macOS Apple Silicon
- `filo-mac-amd64` - macOS Intel
- `filo-linux-amd64` - Linux x86_64
- `filo-windows-amd64.exe` - Windows x86_64

---

## 🚀 快速开始

### 基础用法

```bash
# 传统模式（稳定可靠）
filo ~/Downloads

# 智能体模式（实验性，更智能）
filo ~/Downloads --agents -v

# 预览模式（不实际移动文件）
filo ~/Downloads -n

# 递归整理子目录
filo ~/Downloads -r
```

### 配置管理

```bash
# 运行安装向导
filo setup

# 查看/修改配置
filo config

# 调整批次大小（减小可避免超时）
filo config --batch-size 5

# 切换模型
filo config --model llama3.2:3b
```

### 查看统计

```bash
# 查看学习统计
filo stats

# 扫描文件夹（不整理）
filo scan ~/Downloads
```

---

## 📊 性能对比

| 指标 | v2.0.0 | v2.1.0-alpha | 提升 |
|------|--------|--------------|------|
| 单批次处理 | 串行 | 并行（3批） | ~3x |
| Thinking 模式 | 开启 | 关闭 | ~2x |
| 平均耗时/文件 | ~30s | ~8-12s | ~60-70% |
| 100文件总耗时 | ~30min | ~5-8min | ~75% |
| 分类准确率 | ~70% | ~85%+ | ~15% |

---

## 🐛 已知问题

1. **LLM 超时** - 对于非常大的批次（>15个文件），qwen3:8b 可能仍会超时
   - **解决方案**: 减小批次大小 `filo config --batch-size 5`

2. **内存占用** - 并行处理会增加内存使用
   - **解决方案**: 减少并发数或关闭其他应用

3. **智能体模式稳定性** - 仍处于 alpha 阶段
   - **建议**: 生产环境使用传统模式

---

## 🔧 技术细节

### 架构变更

- **新增模块**: `internal/agent/` - 智能体框架
  - `agent.go` - 基础接口定义
  - `planner.go` - 规划器实现
  - `executor.go` - 执行器实现
  - `optimizer.go` - 优化器实现
  - `evaluator.go` - 评估器实现
  - `learner.go` - 学习器实现
  - `coordinator.go` - 协调器（5阶段工作流）
  - `communication.go` - EventBus 通信协议

- **优化模块**: 
  - `internal/classifier/classifier.go` - 并行处理 + 元数据支持
  - `internal/llm/ollama.go` - 关闭 Thinking 模式
  - `internal/organizer/organizer.go` - nil 安全检查
  - `internal/ui/ui.go` - 动态版本号

### API 变更

- `classifier.Result` 新增 `Metadata map[string]string` 字段
- `organizer.Execute()` 现在接受 `nil` classifier（智能体模式）

---

## 📝 Changelog

### v2.1.0-alpha (2026-05-31)

#### Features
- 🤖 引入多智能体系统 - Planner, Executor, Evaluator, Optimizer, Learner
- 📋 智能任务规划 - 自动分析复杂度，制定最优策略
- ✨ 结果优化引擎 - 一致性检查、异常值检测、分类规范化
- 📚 持续学习能力 - 从每次分类中学习模式，准确率持续提升
- 🔍 质量评估反馈 - 实时评估分类质量，提供改进建议
- 🔄 事件驱动通信 - EventBus 实现松耦合智能体协作

#### Performance
- ⚡ 并行批次处理 - 同时处理 3 个批次
- 🚀 关闭 Thinking 模式 - qwen3:8b 响应速度提升 50-70%
- 📈 优化超时设置 - 从 120s 增加到 600s

#### Bug Fixes
- 🐛 修复 Metadata 未初始化导致的 panic
- 🐛 修复智能体模式下 clf 为 nil 的崩溃
- 🐛 修复循环依赖问题
- 🐛 修复类型转换错误

#### Improvements
- 💾 断点续传支持 - 大批量处理中断后可恢复
- 📊 性能监控增强 - API 耗时监控
- 🛡️ 配置验证 - 自动修正无效配置值
- 🎯 提示词工程优化 - 更详细的分类体系

#### Documentation
- 📖 添加智能体架构文档 (docs/AGENT_ARCHITECTURE.md)
- 📖 添加实施报告 (docs/IMPLEMENTATION_REPORT.md)
- 📖 更新 README 添加智能体使用说明

---

## 🙏 致谢

感谢所有贡献者和用户的支持！

特别感谢：
- Ollama 团队提供的本地 LLM 运行时
- Go 社区的开源库支持
- 早期测试用户的宝贵反馈

---

## 📞 支持与反馈

- 📧 Issue: [GitHub Issues](https://github.com/lynx-lee/filo/issues)
- 💬 Discussion: [GitHub Discussions](https://github.com/lynx-lee/filo/discussions)
- 📖 文档: [README.md](https://github.com/lynx-lee/filo/blob/main/README.md)

---

**注意**: 这是 alpha 版本，可能存在不稳定因素。生产环境建议使用传统模式 (`filo ~/Downloads` 不带 `--agents` 参数)。

# 📦 Filo 发布指南

本文档说明如何为 Filo 创建和发布新版本。

---

## 🚀 快速发布流程

### 方法一：使用 GitHub Actions（推荐）

最简单的方式是通过 Git tag 触发自动发布：

```bash
# 1. 确保所有更改已提交
git add .
git commit -m "chore: prepare for v2.1.0-alpha release"

# 2. 创建并推送 tag
git tag v2.1.0-alpha
git push origin v2.1.0-alpha
```

GitHub Actions 会自动：
- ✅ 构建所有平台的二进制文件
- ✅ 创建 GitHub Release
- ✅ 上传打包文件
- ✅ 标记为预发布（如果版本包含 alpha/beta/rc）

---

### 方法二：使用 GitHub CLI

如果您安装了 GitHub CLI (`gh`)：

```bash
# 1. 运行发布脚本
./publish-release.sh

# 2. 按提示确认发布
# 3. 访问 GitHub 页面检查 Draft Release
# 4. 点击 "Publish release"
```

---

### 方法三：手动发布

#### 步骤 1：构建多平台二进制文件

```bash
# 运行构建脚本
./release.sh v2.1.0-alpha
```

这会在 `dist/` 目录生成：
- `filo-2.1.0-alpha-darwin-amd64.tar.gz`
- `filo-2.1.0-alpha-darwin-arm64.tar.gz`
- `filo-2.1.0-alpha-linux-amd64.tar.gz`
- `filo-2.1.0-alpha-windows-amd64.zip`

#### 步骤 2：测试二进制文件

```bash
# 解压并测试
tar xzf dist/filo-2.1.0-alpha-darwin-arm64.tar.gz
./filo version
```

#### 步骤 3：创建 GitHub Release

1. 访问 [Releases 页面](https://github.com/lynx-lee/filo/releases)
2. 点击 "Draft a new release"
3. 填写信息：
   - **Tag**: `v2.1.0-alpha`
   - **Title**: `Filo v2.1.0-alpha - 智能体系统正式发布`
   - **Description**: 复制 `RELEASE_NOTES_v2.1.0-alpha.md` 的内容
4. 上传 `dist/` 中的所有文件
5. 勾选 "This is a pre-release"（如果是 alpha/beta 版本）
6. 点击 "Publish release"

---

## 📝 版本命名规范

遵循 [语义化版本](https://semver.org/)：

```
主版本.次版本.修订版本-预发布标识
  ^       ^       ^         ^
  |       |       |         └─ alpha, beta, rc (可选)
  |       |       └─────────── Bug 修复
  |       └─────────────────── 新功能（向后兼容）
  └─────────────────────────── 不兼容的 API 变更
```

**示例**：
- `v2.0.0` - 稳定版本
- `v2.1.0-alpha` - Alpha 预发布
- `v2.1.0-beta.1` - Beta 预发布
- `v2.1.0-rc.1` - Release Candidate
- `v2.1.1` - Bug 修复版本

---

## 🔄 完整发布检查清单

### 发布前准备

- [ ] 所有功能已完成并测试通过
- [ ] 更新版本号：
  - [ ] `internal/config/config.go` 中的 `Version` 常量
  - [ ] `Makefile` 中的 `VERSION` 变量
- [ ] 更新 `README.md` 的 Changelog
- [ ] 编写 Release Notes（参考 `RELEASE_NOTES_v2.1.0-alpha.md`）
- [ ] 运行完整测试套件
- [ ] 在多个平台测试（macOS, Linux, Windows）

### 代码提交

- [ ] 提交所有更改
- [ ] 创建 Git tag
- [ ] 推送 tag 到远程仓库

### 发布后验证

- [ ] 检查 GitHub Release 页面
- [ ] 下载各个平台的二进制文件并测试
- [ ] 验证安装脚本工作正常
- [ ] 更新 Homebrew formula（如需要）
- [ ] 更新文档链接
- [ ] 在社区发布公告

---

## 🛠️ 自动化脚本说明

### release.sh

**功能**: 编译多平台二进制文件并打包

**用法**:
```bash
./release.sh <version>
```

**输出**:
- `build/` - 未压缩的二进制文件
- `dist/` - 压缩包（.tar.gz / .zip）

**支持的平台**:
- macOS (amd64, arm64)
- Linux (amd64)
- Windows (amd64)

---

### publish-release.sh

**功能**: 使用 GitHub CLI 创建 Draft Release

**前置条件**:
```bash
# 安装 GitHub CLI
brew install gh

# 登录
gh auth login
```

**用法**:
```bash
./publish-release.sh
```

**特点**:
- 自动调用 `release.sh` 构建
- 创建 Draft Release（需要手动确认发布）
- 自动上传所有打包文件

---

### .github/workflows/release.yml

**功能**: GitHub Actions 自动发布工作流

**触发条件**: 推送以 `v` 开头的 tag

**执行步骤**:
1. 检出代码
2. 设置 Go 环境
3. 运行 `release.sh` 构建
4. 创建 GitHub Release
5. 上传所有文件

**优势**:
- ✅ 完全自动化
- ✅ 无需本地配置
- ✅ 一致的构建环境
- ✅ 支持预发布标记

---

## 🐛 常见问题

### Q1: 构建失败，提示 "CGO_ENABLED=0" 错误

**原因**: modernc.org/sqlite 需要 CGO 支持

**解决**: 
- 移除 `CGO_ENABLED=0` 环境变量
- 或在目标平台本机执行构建

### Q2: GitHub CLI 认证失败

**解决**:
```bash
# 重新登录
gh auth logout
gh auth login

# 选择 HTTPS 协议
# 选择 GitHub.com
# 按照浏览器提示完成认证
```

### Q3: Release 创建成功但文件未上传

**原因**: 文件路径不正确

**解决**:
```bash
# 检查 dist 目录
ls -lh dist/

# 确保文件存在后再创建 Release
./release.sh v2.1.0-alpha
gh release create v2.1.0-alpha dist/*
```

### Q4: 如何更新已有的 Release？

```bash
# 编辑 Release
gh release edit v2.1.0-alpha --draft=false

# 删除并重新创建
gh release delete v2.1.0-alpha
./publish-release.sh
```

---

## 📊 发布后统计

发布后可以查看：
- [下载统计](https://github.com/lynx-lee/filo/releases)
- [Issue 反馈](https://github.com/lynx-lee/filo/issues)
- [社区讨论](https://github.com/lynx-lee/filo/discussions)

---

## 🎯 最佳实践

1. **始终使用 tag 触发发布** - 确保版本可追溯
2. **先创建 Draft Release** - 有时间检查和修正
3. **提供详细的 Release Notes** - 帮助用户了解变更
4. **测试所有平台的二进制文件** - 避免发布后发现问题
5. **保持向后兼容** - 除非是主版本升级
6. **及时响应 Issue** - 收集用户反馈

---

## 📞 需要帮助？

- 📖 [GitHub Releases 文档](https://docs.github.com/en/repositories/releasing-projects-on-github)
- 🔧 [GitHub CLI 文档](https://cli.github.com/manual/)
- 💬 [Filo Discussions](https://github.com/lynx-lee/filo/discussions)

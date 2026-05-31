# 🚀 如何创建 GitHub Release

## 快速开始（3 步完成）

### 方法一：GitHub Actions（最简单）✨

```bash
# 1. 提交所有更改
git add .
git commit -m "chore: prepare for v2.1.0-alpha release"

# 2. 创建并推送 tag
git tag v2.1.0-alpha
git push origin v2.1.0-alpha
```

✅ **完成！** GitHub Actions 会自动构建并发布。

访问 https://github.com/lynx-lee/filo/releases 查看结果。

---

### 方法二：使用 GitHub CLI

```bash
# 1. 安装 GitHub CLI（如果未安装）
brew install gh

# 2. 登录
gh auth login

# 3. 运行发布脚本
./publish-release.sh
```

按照提示确认发布，然后访问 GitHub 页面点击 "Publish release"。

---

### 方法三：手动发布

#### 步骤 1：构建二进制文件

```bash
./release.sh v2.1.0-alpha
```

生成的文件在 `dist/` 目录：
- `filo-2.1.0-alpha-darwin-amd64.tar.gz`
- `filo-2.1.0-alpha-darwin-arm64.tar.gz`
- `filo-2.1.0-alpha-linux-amd64.tar.gz`
- `filo-2.1.0-alpha-windows-amd64.zip`

#### 步骤 2：创建 Release

1. 访问 https://github.com/lynx-lee/filo/releases/new
2. Tag: `v2.1.0-alpha`
3. Title: `Filo v2.1.0-alpha - 智能体系统正式发布`
4. Description: 复制 `RELEASE_NOTES_v2.1.0-alpha.md` 的内容
5. 上传 `dist/*` 中的所有文件
6. 勾选 "This is a pre-release"
7. 点击 "Publish release"

---

## 📋 已准备的文档

- ✅ `RELEASE_NOTES_v2.1.0-alpha.md` - 完整的 Release Notes
- ✅ `release.sh` - 多平台构建脚本
- ✅ `publish-release.sh` - GitHub CLI 发布脚本
- ✅ `.github/workflows/release.yml` - GitHub Actions 自动发布
- ✅ `docs/RELEASE_GUIDE.md` - 详细发布指南

---

## 💡 推荐流程

对于 **v2.1.0-alpha** 版本，推荐使用 **方法一（GitHub Actions）**：

```bash
# 确保代码已提交
git status

# 如果有未提交的更改
git add .
git commit -m "chore: prepare for v2.1.0-alpha release"

# 推送 tag
git tag v2.1.0-alpha
git push origin v2.1.0-alpha
```

等待 2-3 分钟，Release 就会自动出现在 GitHub 上！🎉

---

## 🔍 验证发布

发布完成后，测试下载和安装：

```bash
# macOS Apple Silicon
curl -fsSL https://github.com/lynx-lee/filo/releases/download/v2.1.0-alpha/filo-2.1.0-alpha-darwin-arm64.tar.gz | tar xz
./filo version

# 或直接用 Homebrew 安装（需要更新 formula）
# brew install filo
```

---

## ❓ 需要帮助？

查看详细文档：[docs/RELEASE_GUIDE.md](docs/RELEASE_GUIDE.md)

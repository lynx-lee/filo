// Package cmd 命令行入口模块
// 提供 filo 的所有命令行功能，包括文件整理、配置、统计等
//
// Copyright (c) 2024-2026 lynx-lee
// https://github.com/lynx-lee/filo

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/lynx-lee/filo/internal/agent"
	"github.com/lynx-lee/filo/internal/classifier"
	"github.com/lynx-lee/filo/internal/config"
	"github.com/lynx-lee/filo/internal/llm"
	"github.com/lynx-lee/filo/internal/organizer"
	"github.com/lynx-lee/filo/internal/scanner"
	"github.com/lynx-lee/filo/internal/storage"
	"github.com/lynx-lee/filo/internal/ui"
)

// 命令行参数变量
var (
	targetDir   string // 目标目录，整理后文件存放位置
	model       string // 指定使用的 LLM 模型
	dryRun      bool   // 预览模式，不实际移动文件
	verbose     bool   // 详细输出模式
	interactive bool   // 交互式审查模式
	noLearning  bool   // 禁用学习功能
	recursive   bool   // 递归扫描子目录
	useAgents   bool   // 使用智能体模式
)

// rootCmd 根命令定义
// 用于整理指定目录中的文件
var rootCmd = &cobra.Command{
	Use:   "filo [目录]",
	Short: "filo - 文件智理，越用越懂你",
	Long: ui.Cyan(`
  ███████╗██╗██╗      ██████╗ 
  ██╔════╝██║██║     ██╔═══██╗
  █████╗  ██║██║     ██║   ██║
  ██╔══╝  ██║██║     ██║   ██║
  ██║     ██║███████╗╚██████╔╝
  ╚═╝     ╚═╝╚══════╝ ╚═════╝ `) + `
  
  文件智理 · 越用越懂你  v` + config.Version + `

  🧠 本地AI，隐私安全
  📚 自动学习你的整理习惯
  🚀 越用越快，越用越准

示例:
  filo ~/Downloads              # 整理下载文件夹
  filo ~/Downloads -n           # 预览模式
  filo ~/Downloads -r           # 递归整理子目录
  filo ~/Downloads -i           # 交互式审查
  filo setup                    # 安装向导
  filo stats                    # 查看学习统计
  filo config                   # 查看/修改配置
`,
	Args: cobra.MaximumNArgs(1), // 最多接受一个参数（目录路径）
	Run:  runOrganize,           // 执行整理操作
}

// init 初始化命令行参数
func init() {
	// 注册命令行标志
	rootCmd.Flags().StringVarP(&targetDir, "target", "t", "", "目标目录")
	rootCmd.Flags().StringVarP(&model, "model", "m", "", "使用的模型")
	rootCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "预览模式")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "详细输出")
	rootCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "交互式审查")
	rootCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "递归扫描子目录")
	rootCmd.Flags().BoolVar(&noLearning, "no-learning", false, "禁用学习")
	rootCmd.Flags().BoolVarP(&useAgents, "agents", "a", false, "使用智能体模式（实验性）")
}

// Execute 执行根命令
// 这是程序的主入口，由 main.go 调用
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// runOrganize 执行文件整理的核心逻辑
// 整体流程：扫描 -> 分类 -> 生成计划 -> 审查（可选）-> 执行
func runOrganize(cmd *cobra.Command, args []string) {
	// 检查是否提供了目录参数
	if len(args) == 0 {
		cmd.Help()
		return
	}

	sourceDir := args[0]

	// 显示启动横幅
	ui.Banner()

	// 更新配置：应用命令行参数
	cfg := config.Get()
	if model != "" {
		cfg.SetModel(model) // 使用指定模型
	} else {
		// 自适应模型选择：基于历史性能推荐最优模型
		db, err := storage.NewDatabase()
		if err == nil {
			if bestModel := db.GetBestModel(); bestModel != "" && bestModel != cfg.LLMModel {
				ui.Info("推荐模型: %s (基于历史性能)", ui.Bold(bestModel))
				ui.Dim("使用 -m %s 切换，或 'filo models --stats' 查看对比", bestModel)
			}
			db.Close()
		}
	}
	if noLearning {
		cfg.EnableLearning = false // 禁用学习功能
	}

	// 设置默认目标目录
	if targetDir == "" {
		targetDir = filepath.Join(sourceDir, "已整理")
	}

	// 检查源目录是否存在
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		ui.Error("目录不存在: %s", sourceDir)
		return
	}

	// 检查 Ollama 服务状态
	client := llm.NewClient()
	if !client.IsAvailable() {
		ui.Error("Ollama 服务未运行")
		ui.Info("请先启动: ollama serve")
		ui.Info("或运行: filo setup")
		return
	}

	// 检查模型是否已安装
	if !client.HasModel(cfg.LLMModel) {
		ui.Error("模型 %s 未安装", cfg.LLMModel)
		ui.Info("运行 'filo setup' 安装模型")
		return
	}

	// ========== 步骤1: 扫描目录 ==========
	scanMode := "扫描"
	if recursive {
		scanMode = "递归扫描"
	}
	ui.Title("📂", fmt.Sprintf("%s: %s", scanMode, sourceDir))
	files, err := scanner.ScanDirectory(sourceDir, recursive)
	if err != nil {
		ui.Error("扫描失败: %v", err)
		return
	}

	// 统计文件数量（不包括目录）
	fileCount := 0
	for _, f := range files {
		if !f.IsDir {
			fileCount++
		}
	}
	ui.Success("找到 %d 个文件", fileCount)

	// 检查是否有文件需要整理
	if fileCount == 0 {
		ui.Warning("没有文件需要整理")
		return
	}

	// ========== 步骤2: 智能分类 ==========
	var results []classifier.Result
	var clf *classifier.Classifier
	
	if useAgents {
		// 使用智能体模式
		ui.Title("🤖", "智能体模式")
		coordinator, err := agent.NewCoordinator()
		if err != nil {
			ui.Error("初始化智能体协调器失败: %v", err)
			return
		}
		defer coordinator.Shutdown()
		
		results, err = coordinator.ExecuteWorkflow(cmd.Context(), files, sourceDir, targetDir, verbose)
		if err != nil {
			ui.Error("智能体工作流执行失败: %v", err)
			return
		}
	} else {
		// 使用传统模式
		var err error
		clf, err = classifier.NewClassifier()
		if err != nil {
			ui.Error("初始化分类器失败: %v", err)
			return
		}
		defer clf.Close() // 确保分类器资源被释放

		// 执行分类
		results, err = clf.Classify(files, verbose)
		if err != nil {
			ui.Error("分类失败: %v", err)
			return
		}
	}

	// ========== 步骤3: 生成整理计划 ==========
	plan := organizer.GeneratePlan(results, targetDir)
	organizer.PrintPlan(plan)

	// ========== 步骤4: 交互式审查（可选）==========
	if interactive && clf != nil {
		plan = organizer.InteractiveReview(plan, clf)
		organizer.PrintPlan(plan) // 显示修改后的计划
	}

	// ========== 步骤5: 执行整理 ==========
	if dryRun {
		// 预览模式：只显示计划，不执行
		ui.Warning("预览模式 - 未执行实际操作")
		ui.Dim("去掉 -n 参数执行实际整理")
	} else {
		// 确认后执行
		if organizer.Confirm("\n确认执行整理?") {
			organizer.Execute(plan, clf, verbose)
		} else {
			ui.Warning("已取消")
		}
	}
}

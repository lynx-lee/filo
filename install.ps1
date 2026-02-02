<#
.SYNOPSIS
    Filo 安装脚本（Windows 版）
.DESCRIPTION
    一键安装 Filo - 基于本地 AI 的智能文件整理工具
.EXAMPLE
    # 以管理员身份运行 PowerShell，然后执行：
    Set-ExecutionPolicy RemoteSigned -Scope Process
    .\install.ps1
#>

Clear-Host

Write-Host "`n"
Write-Host "  ███████╗██╗██╗      ██████╗ " -ForegroundColor Cyan
Write-Host "  ██╔════╝██║██║     ██╔═══██╗" -ForegroundColor Cyan
Write-Host "  █████╗  ██║██║     ██║   ██║" -ForegroundColor Cyan
Write-Host "  ██╔══╝  ██║██║     ██║   ██║" -ForegroundColor Cyan
Write-Host "  ██║     ██║███████╗╚██████╔╝" -ForegroundColor Cyan
Write-Host "  ╚═╝     ╚═╝╚══════╝ ╚═════╝ " -ForegroundColor Cyan
Write-Host "`n"
Write-Host "  文件智理 · 越用越懂你" -ForegroundColor Cyan
Write-Host "`n"
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Gray

# 检测系统架构
$Architecture = $env:PROCESSOR_ARCHITECTURE
if ($Architecture -eq "AMD64") {
    $Architecture = "amd64"
} elseif ($Architecture -eq "x86") {
    $Architecture = "x86"
} else {
    Write-Host "❌ 不支持的架构: $Architecture" -ForegroundColor Red
    exit 1
}

Write-Host "`n"
Write-Host "📍 系统: Windows $Architecture" -ForegroundColor Green

# 确定下载文件名
$Binary = "filo-windows.exe"

# 下载地址（替换为实际地址）
$DownloadUrl = "https://github.com/lynx-lee/filo/releases/latest/download/$Binary"

Write-Host "`n"
Write-Host "📥 下载 filo..." -ForegroundColor Green

# 设置安装目录
$InstallDir = "C:\Program Files\Filo"
$ExePath = "$InstallDir\filo.exe"
$TmpFile = "$env:TEMP\filo-windows.exe"

# 创建安装目录
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# 下载文件
try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $TmpFile -ErrorAction Stop
    Write-Host "✅ 下载成功" -ForegroundColor Green
} catch {
    Write-Host "❌ 下载失败: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# 复制到安装目录
try {
    Copy-Item -Path $TmpFile -Destination $ExePath -Force -ErrorAction Stop
    Write-Host "📦 安装到 $InstallDir..." -ForegroundColor Green
} catch {
    Write-Host "❌ 安装失败: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# 添加到 PATH 环境变量
$Path = [Environment]::GetEnvironmentVariable("PATH", "Machine")
if (-not $Path.Contains($InstallDir)) {
    Write-Host "🔧 添加到系统 PATH..." -ForegroundColor Green
    $NewPath = "$Path;$InstallDir"
    [Environment]::SetEnvironmentVariable("PATH", $NewPath, "Machine")
    # 通知用户需要重启终端
    Write-Host "⚠️  PATH 已更新，需要重启终端才能生效" -ForegroundColor Yellow
}

# 清理临时文件
Remove-Item -Path $TmpFile -Force -ErrorAction SilentlyContinue

Write-Host "`n"
Write-Host "✅ 安装完成！" -ForegroundColor Green
Write-Host "`n"
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Gray
Write-Host "`n"
Write-Host "下一步：" -ForegroundColor Cyan
Write-Host "`n"
Write-Host "  1. 运行安装向导: filo setup" -ForegroundColor White
Write-Host "  2. 预览整理效果: filo ~/Downloads -n" -ForegroundColor White
Write-Host "  3. 执行整理:     filo ~/Downloads" -ForegroundColor White
Write-Host "`n"

# 提示用户按任意键退出
Write-Host "按任意键退出..." -ForegroundColor Gray
$Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown") | Out-Null

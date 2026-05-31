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
} elseif ($Architecture -eq "ARM64") {
    $Architecture = "arm64"
} elseif ($Architecture -eq "x86") {
    $Architecture = "x86"
} else {
    Write-Host "❌ 不支持的架构: $Architecture" -ForegroundColor Red
    exit 1
}

Write-Host "`n"
Write-Host "📍 系统: Windows $Architecture" -ForegroundColor Green

# 确定下载文件名
if ($Architecture -eq "arm64") {
    $Binary = "filo-windows-arm64.exe"
} elseif ($Architecture -eq "x86") {
    $Binary = "filo-windows-x86.exe"
} else {
    $Binary = "filo-windows.exe"
}

# 下载地址（优先使用 GitHub Releases，失败则使用备用地址）
$DownloadUrl = "https://github.com/lynx-lee/filo/releases/latest/download/$Binary"
$BackupUrl = "https://ghproxy.net/https://github.com/lynx-lee/filo/releases/latest/download/$Binary"

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

# 下载文件（带重试机制）
$downloadSuccess = $false
$maxRetries = 3

for ($i = 1; $i -le $maxRetries; $i++) {
    try {
        Write-Host "  尝试下载 (第 $i/$maxRetries 次)..." -ForegroundColor Gray
        
        # 配置 TLS 1.2
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        
        # 创建 WebClient 并设置超时
        $webClient = New-Object System.Net.WebClient
        $webClient.Headers.Add("User-Agent", "PowerShell Install Script")
        
        # 尝试主地址
        $webClient.DownloadFile($DownloadUrl, $TmpFile)
        $downloadSuccess = $true
        Write-Host "✅ 下载成功" -ForegroundColor Green
        break
        
    } catch {
        Write-Host "  ⚠️  第 $i 次尝试失败: $($_.Exception.Message)" -ForegroundColor Yellow
        
        # 如果是最后一次尝试，尝试备用地址
        if ($i -eq $maxRetries) {
            Write-Host "  尝试备用下载地址..." -ForegroundColor Cyan
            try {
                $webClient.DownloadFile($BackupUrl, $TmpFile)
                $downloadSuccess = $true
                Write-Host "✅ 通过备用地址下载成功" -ForegroundColor Green
                break
            } catch {
                Write-Host "❌ 所有下载方式均失败" -ForegroundColor Red
            }
        }
        
        # 等待后重试
        if ($i -lt $maxRetries) {
            Start-Sleep -Seconds 2
        }
    } finally {
        if ($webClient) {
            $webClient.Dispose()
        }
    }
}

if (-not $downloadSuccess) {
    Write-Host "`n" -NoNewline
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Gray
    Write-Host "`n" -NoNewline
    Write-Host "❌ 无法自动下载安装包" -ForegroundColor Red
    Write-Host "`n" -NoNewline
    Write-Host "可能的原因：" -ForegroundColor Yellow
    Write-Host "  1. GitHub Releases 尚未发布正式版本" -ForegroundColor White
    Write-Host "  2. 网络连接问题或被防火墙拦截" -ForegroundColor White
    Write-Host "  3. 需要配置代理或加速器" -ForegroundColor White
    Write-Host "`n" -NoNewline
    Write-Host "解决方案：" -ForegroundColor Cyan
    Write-Host "`n" -NoNewline
    Write-Host "  方案一：手动下载（推荐）" -ForegroundColor Green
    Write-Host "    1. 访问: https://github.com/lynx-lee/filo/releases" -ForegroundColor White
    Write-Host "    2. 下载最新版本: $Binary" -ForegroundColor White
    Write-Host "    3. 将文件复制到: $InstallDir" -ForegroundColor White
    Write-Host "    4. 运行: filo setup" -ForegroundColor White
    Write-Host "`n" -NoNewline
    Write-Host "  方案二：使用 Go 安装" -ForegroundColor Green
    Write-Host "    go install github.com/lynx-lee/filo@latest" -ForegroundColor White
    Write-Host "`n" -NoNewline
    Write-Host "  方案三：源码编译" -ForegroundColor Green
    Write-Host "    git clone https://github.com/lynx-lee/filo.git" -ForegroundColor White
    Write-Host "    cd filo" -ForegroundColor White
    Write-Host "    go build -o filo.exe ." -ForegroundColor White
    Write-Host "    Copy-Item filo.exe '$InstallDir'" -ForegroundColor White
    Write-Host "`n" -NoNewline
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Gray
    Write-Host "`n" -NoNewline
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

<#
.SYNOPSIS
    Jiasine CLI 构建脚本
.DESCRIPTION
    支持本地构建、跨平台全量编译、清理等操作
.PARAMETER Target
    构建目标: local (默认), cross, clean, windows
.EXAMPLE
    .\build.ps1
    .\build.ps1 -Target cross
    .\build.ps1 -Target clean
#>
param(
    [ValidateSet("local", "cross", "clean", "windows")]
    [string]$Target = "local"
)

$ErrorActionPreference = "Stop"

# ===== 确保 Go 在 PATH 中 =====
$GoExe = Get-Command go -ErrorAction SilentlyContinue
if (-not $GoExe) {
    # 尝试从系统/用户 PATH 重新加载
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" +
                [System.Environment]::GetEnvironmentVariable("Path", "User")
    $GoExe = Get-Command go -ErrorAction SilentlyContinue
    if (-not $GoExe) {
        # 最后尝试默认安装路径
        $defaultGo = "C:\Program Files\Go\bin"
        if (Test-Path "$defaultGo\go.exe") {
            $env:Path += ";$defaultGo"
        } else {
            Write-Host "  [错误] 未找到 Go 编译器，请先安装 Go" -ForegroundColor Red
            exit 1
        }
    }
}
Write-Host "  Go: $(go version)" -ForegroundColor DarkGray

# ===== 项目信息 =====
$ModulePath  = "github.com/xiangjianhe-github/jiasinecli"
$CmdPkg      = "$ModulePath/cmd"
$DistDir     = "dist"
$Version     = "0.1.0-alpha.1"
$GitCommit   = git rev-parse --short HEAD 2>$null
if (-not $GitCommit) { $GitCommit = "none" }
$BuildDate   = Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ"
$LdFlags     = "-s -w -X '$CmdPkg.Version=$Version' -X '$CmdPkg.GitCommit=$GitCommit' -X '$CmdPkg.BuildDate=$BuildDate'"

# ===== 跨平台目标 (7 个) =====
$Targets = @(
    @{ GOOS = "windows"; GOARCH = "amd64"; Output = "jiasinecli-windows.exe" }
    @{ GOOS = "windows"; GOARCH = "arm64"; Output = "jiasinecli-windows-arm64.exe" }
    @{ GOOS = "linux";   GOARCH = "amd64"; Output = "jiasinecli-linux" }
    @{ GOOS = "linux";   GOARCH = "arm64"; Output = "jiasinecli-linux-arm64" }
    @{ GOOS = "linux";   GOARCH = "arm";   Output = "jiasinecli-raspi" }
    @{ GOOS = "darwin";  GOARCH = "arm64"; Output = "jiasinecli-macos-arm" }
    @{ GOOS = "darwin";  GOARCH = "amd64"; Output = "jiasinecli-macos-intel" }
)

# ===== 函数 =====
function Build-Single {
    param([string]$GOOS, [string]$GOARCH, [string]$OutputPath)

    $env:GOOS   = $GOOS
    $env:GOARCH = $GOARCH
    $env:CGO_ENABLED = "0"

    Write-Host "  编译 $GOOS/$GOARCH -> $OutputPath" -ForegroundColor Cyan
    go build -ldflags $LdFlags -o $OutputPath .
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  [失败] $GOOS/$GOARCH" -ForegroundColor Red
        return $false
    }

    $size = [math]::Round((Get-Item $OutputPath).Length / 1MB, 2)
    Write-Host "  [成功] $size MB" -ForegroundColor Green
    return $true
}

function Clean-Dist {
    Write-Host "  清理 $DistDir/ ..." -ForegroundColor Yellow
    if (Test-Path $DistDir) {
        Remove-Item "$DistDir\*" -Force -Recurse
    }
    # 清理根目录下的 exe
    Remove-Item "jiasinecli.exe" -ErrorAction SilentlyContinue
    Write-Host "  [完成] 清理完毕" -ForegroundColor Green
}

# ===== 主逻辑 =====
Write-Host ""

switch ($Target) {
    "clean" {
        Write-Host "========== 清理构建产物 ==========" -ForegroundColor Yellow
        Clean-Dist
    }

    "local" {
        Write-Host "========== 本地构建 ==========" -ForegroundColor Cyan
        # 本地构建不禁用 CGO，保留调试信息
        $env:CGO_ENABLED = "0"
        $output = "jiasinecli.exe"
        Write-Host "  编译 windows/amd64 -> $output" -ForegroundColor Cyan
        go build -ldflags $LdFlags -o $output .
        if ($LASTEXITCODE -eq 0) {
            $size = [math]::Round((Get-Item $output).Length / 1MB, 2)
            Write-Host "  [成功] $size MB" -ForegroundColor Green
        } else {
            Write-Host "  [失败] 本地构建出错" -ForegroundColor Red
            exit 1
        }
    }

    "windows" {
        Write-Host "========== Windows 构建 (含资源嵌入) ==========" -ForegroundColor Cyan

        # 检查 windres
        $windresPath = "C:\msys64\ucrt64\bin"
        if (-not (Get-Command windres -ErrorAction SilentlyContinue)) {
            if (Test-Path "$windresPath\windres.exe") {
                $env:Path += ";$windresPath"
            } else {
                Write-Host "  [警告] 未找到 windres，跳过资源嵌入" -ForegroundColor Yellow
            }
        }

        # 生成资源文件
        if (Get-Command windres -ErrorAction SilentlyContinue) {
            Write-Host "  生成 Windows 资源..." -ForegroundColor DarkGray
            go run internal/winres/generate.go
            windres -o rsrc_windows_amd64.syso assets/app.rc
            Write-Host "  [成功] 资源已嵌入 .syso" -ForegroundColor Green
        }

        if (-not (Test-Path $DistDir)) { New-Item -ItemType Directory -Path $DistDir -Force | Out-Null }
        Build-Single -GOOS "windows" -GOARCH "amd64" -OutputPath "$DistDir\jiasinecli-windows.exe"
    }

    "cross" {
        Write-Host "========== 跨平台全量编译 ==========" -ForegroundColor Cyan
        if (-not (Test-Path $DistDir)) { New-Item -ItemType Directory -Path $DistDir -Force | Out-Null }

        $success = 0
        $failed  = 0

        foreach ($t in $Targets) {
            $result = Build-Single -GOOS $t.GOOS -GOARCH $t.GOARCH -OutputPath "$DistDir\$($t.Output)"
            if ($result) { $success++ } else { $failed++ }
        }

        # 恢复本地环境变量
        $env:GOOS   = "windows"
        $env:GOARCH = "amd64"
        Remove-Item env:CGO_ENABLED -ErrorAction SilentlyContinue

        Write-Host ""
        Write-Host "========== 编译结果 ==========" -ForegroundColor Cyan
        Write-Host "  成功: $success / $($Targets.Count)" -ForegroundColor $(if ($failed -eq 0) { "Green" } else { "Yellow" })
        if ($failed -gt 0) {
            Write-Host "  失败: $failed" -ForegroundColor Red
        }

        # 显示产物列表
        Write-Host ""
        Get-ChildItem $DistDir | Select-Object Name, @{N='Size(MB)';E={[math]::Round($_.Length/1MB,2)}} | Format-Table -AutoSize
    }
}

Write-Host ""

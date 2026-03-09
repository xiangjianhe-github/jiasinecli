#!/usr/bin/env python3
"""
Jiasine CLI 跨平台构建脚本
支持 Windows / Linux / macOS 全平台编译
"""

import os
import sys
import subprocess
import shutil
from pathlib import Path
from datetime import datetime

# ===== 配置 =====
MODULE_PATH = "github.com/xiangjianhe-github/jiasinecli"
CMD_PKG = f"{MODULE_PATH}/cmd"
DIST_DIR = "dist"
VERSION = "0.2.0-alpha"

# 获取 Git 提交信息
try:
    GIT_COMMIT = subprocess.check_output(
        ["git", "rev-parse", "--short", "HEAD"],
        stderr=subprocess.DEVNULL
    ).decode().strip()
except:
    GIT_COMMIT = "none"

BUILD_DATE = datetime.now().strftime("%Y-%m-%dT%H:%M:%SZ")
LDFLAGS = f"-s -w -X '{CMD_PKG}.Version={VERSION}' -X '{CMD_PKG}.GitCommit={GIT_COMMIT}' -X '{CMD_PKG}.BuildDate={BUILD_DATE}'"

# ===== 跨平台目标 =====
TARGETS = [
    {"GOOS": "windows", "GOARCH": "amd64", "output": "jiasinecli-windows.exe"},
    {"GOOS": "windows", "GOARCH": "arm64", "output": "jiasinecli-windows-arm64.exe"},
    {"GOOS": "linux", "GOARCH": "amd64", "output": "jiasinecli-linux"},
    {"GOOS": "linux", "GOARCH": "arm64", "output": "jiasinecli-linux-arm64"},
    {"GOOS": "linux", "GOARCH": "arm", "output": "jiasinecli-raspi"},
    {"GOOS": "darwin", "GOARCH": "arm64", "output": "jiasinecli-macos-arm"},
    {"GOOS": "darwin", "GOARCH": "amd64", "output": "jiasinecli-macos-intel"},
]

# ===== 颜色输出 =====
class Color:
    CYAN = "\033[96m"
    GREEN = "\033[92m"
    YELLOW = "\033[93m"
    RED = "\033[91m"
    GRAY = "\033[90m"
    RESET = "\033[0m"
    BOLD = "\033[1m"

def print_colored(text, color=""):
    """彩色打印"""
    print(f"{color}{text}{Color.RESET}")

def print_header(text):
    """打印标题"""
    print()
    print_colored(f"========== {text} ==========", Color.CYAN + Color.BOLD)

def check_go():
    """检查 Go 是否安装"""
    try:
        result = subprocess.run(
            ["go", "version"],
            capture_output=True,
            text=True,
            check=True
        )
        print_colored(f"  Go: {result.stdout.strip()}", Color.GRAY)
        return True
    except (subprocess.CalledProcessError, FileNotFoundError):
        print_colored("  [错误] 未找到 Go 编译器，请先安装 Go", Color.RED)
        return False

def build_single(goos, goarch, output_path):
    """编译单个目标"""
    env = os.environ.copy()
    env["GOOS"] = goos
    env["GOARCH"] = goarch
    env["CGO_ENABLED"] = "0"

    print_colored(f"  编译 {goos}/{goarch} -> {output_path}", Color.CYAN)

    try:
        subprocess.run(
            ["go", "build", "-ldflags", LDFLAGS, "-o", output_path, "."],
            env=env,
            check=True,
            capture_output=True
        )

        size_mb = Path(output_path).stat().st_size / (1024 * 1024)
        print_colored(f"  [成功] {size_mb:.2f} MB", Color.GREEN)
        return True
    except subprocess.CalledProcessError as e:
        print_colored(f"  [失败] {goos}/{goarch}", Color.RED)
        if e.stderr:
            print_colored(f"  错误: {e.stderr.decode()}", Color.RED)
        return False

def clean_dist():
    """清理构建产物"""
    print_header("清理构建产物")

    if Path(DIST_DIR).exists():
        shutil.rmtree(DIST_DIR)
        print_colored(f"  已删除 {DIST_DIR}/", Color.YELLOW)

    # 清理本地可执行文件
    for pattern in ["jiasinecli", "jiasinecli.exe"]:
        for file in Path(".").glob(pattern):
            file.unlink()
            print_colored(f"  已删除 {file}", Color.YELLOW)

    print_colored("  [完成] 清理完毕", Color.GREEN)

def build_local():
    """本地构建"""
    print_header("本地构建")

    # 根据当前平台选择输出文件名
    if sys.platform == "win32":
        output = "jiasinecli.exe"
        goos = "windows"
    elif sys.platform == "darwin":
        output = "jiasinecli"
        goos = "darwin"
    else:
        output = "jiasinecli"
        goos = "linux"

    # 获取当前架构
    import platform
    machine = platform.machine().lower()
    if machine in ["x86_64", "amd64"]:
        goarch = "amd64"
    elif machine in ["aarch64", "arm64"]:
        goarch = "arm64"
    elif machine.startswith("arm"):
        goarch = "arm"
    else:
        goarch = "amd64"  # 默认

    return build_single(goos, goarch, output)

def build_cross():
    """跨平台全量编译"""
    print_header("跨平台全量编译")

    # 创建输出目录
    Path(DIST_DIR).mkdir(exist_ok=True)

    success = 0
    failed = 0

    for target in TARGETS:
        output_path = Path(DIST_DIR) / target["output"]
        if build_single(target["GOOS"], target["GOARCH"], str(output_path)):
            success += 1
        else:
            failed += 1

    # 显示结果
    print()
    print_header("编译结果")
    color = Color.GREEN if failed == 0 else Color.YELLOW
    print_colored(f"  成功: {success} / {len(TARGETS)}", color)
    if failed > 0:
        print_colored(f"  失败: {failed}", Color.RED)

    # 显示产物列表
    print()
    print_colored("  产物列表:", Color.CYAN)
    for file in sorted(Path(DIST_DIR).glob("*")):
        size_mb = file.stat().st_size / (1024 * 1024)
        print(f"    {file.name:<35} {size_mb:>8.2f} MB")

    return failed == 0

def main():
    """主函数"""
    import argparse

    parser = argparse.ArgumentParser(
        description="Jiasine CLI 跨平台构建脚本",
        formatter_class=argparse.RawDescriptionHelpFormatter
    )
    parser.add_argument(
        "target",
        nargs="?",
        default="local",
        choices=["local", "cross", "clean"],
        help="构建目标: local (默认), cross, clean"
    )

    args = parser.parse_args()

    # 检查 Go
    if args.target != "clean" and not check_go():
        return 1

    # 执行构建
    try:
        if args.target == "clean":
            clean_dist()
            return 0
        elif args.target == "local":
            return 0 if build_local() else 1
        elif args.target == "cross":
            return 0 if build_cross() else 1
    except KeyboardInterrupt:
        print()
        print_colored("  [取消] 用户中断", Color.YELLOW)
        return 130
    except Exception as e:
        print_colored(f"  [错误] {e}", Color.RED)
        return 1

if __name__ == "__main__":
    print()
    sys.exit(main())

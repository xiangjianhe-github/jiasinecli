// JiasineCli - Jiasine 系统统一 CLI 工具
// Go 作为胶水层，统一调用底层能力（动态库、独立服务）
package main

import (
	"fmt"
	"os"

	"github.com/xiangjianhe-github/jiasinecli/cmd"
	"github.com/xiangjianhe-github/jiasinecli/internal/shell"
	"github.com/xiangjianhe-github/jiasinecli/internal/version"
)

func main() {
	// Windows 双击启动时：重新在 PowerShell 中打开
	if shell.RelaunchInPowerShell() {
		return
	}

	// 双击时无论如何都保持窗口打开（LIFO：最后执行）
	defer shell.KeepWindowOpen()

	// panic 安全网：打印错误信息（LIFO：先于 KeepWindowOpen 执行）
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "\n\u274c 程序异常: %v\n", r)
		}
	}()

	// 后台异步检查更新（独立 recover 保护，防止 goroutine panic 导致整体崩溃）
	go func() {
		defer func() { recover() }()
		checkUpdateInBackground()
	}()

	enterInteractive := func() {
		shell.RunInteractive(func(args []string) error {
			return cmd.ExecuteArgs(args)
		})
	}

	// 无参数启动 → 直接进入 AI 对话模式
	if len(os.Args) <= 1 {
		cmd.EnterDefaultAIMode()
		return
	}

	// 检查是否显式指定了 --interactive / -i（进入传统命令 Shell）
	if hasFlag("-i", "--interactive") {
		enterInteractive()
		return
	}

	// 正常 CLI 模式
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// hasFlag 检查 os.Args 中是否包含指定的 flag
func hasFlag(flags ...string) bool {
	for _, arg := range os.Args[1:] {
		for _, flag := range flags {
			if arg == flag {
				return true
			}
		}
	}
	return false
}

// checkUpdateInBackground 后台异步检查更新
func checkUpdateInBackground() {
	// 仅在非 update 命令时检查
	if len(os.Args) > 1 && os.Args[1] == "update" {
		return
	}

	updater := version.NewUpdater()
	hasUpdate, remote, err := updater.CheckUpdate()

	// 静默失败（不干扰用户体验）
	if err != nil {
		return
	}

	if hasUpdate && remote != nil {
		fmt.Fprintf(os.Stderr, "\n💡 发现新版本 %s，运行 'jiasinecli update' 更新\n\n", remote.String())
	}
}

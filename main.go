// JiasineCli - Jiasine 系统统一 CLI 工具
// Go 作为胶水层，统一调用底层能力（动态库、独立服务）
package main

import (
	"os"

	"github.com/xiangjianhe-github/jiasinecli/cmd"
	"github.com/xiangjianhe-github/jiasinecli/internal/shell"
)

func main() {
	enterInteractive := func() {
		shell.RunInteractive(func(args []string) error {
			return cmd.ExecuteArgs(args)
		})
	}

	// Windows 双击启动时，通过 cmd.exe 重新启动以获得完整终端支持
	if len(os.Args) <= 1 && shell.IsDoubleClicked() {
		shell.RelaunchInCmd()
		return
	}

	// 无参数启动 → 进入交互式 Shell
	// (终端直接运行 jiasinecli)
	if len(os.Args) <= 1 {
		enterInteractive()
		return
	}

	// 检查是否显式指定了 --interactive / -i
	// 如果是，直接进入交互模式，跳过 cobra 默认帮助输出
	if hasFlag("-i", "--interactive") {
		enterInteractive()
		return
	}

	// 正常 CLI 模式
	if err := cmd.Execute(); err != nil {
		shell.KeepWindowOpen()
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

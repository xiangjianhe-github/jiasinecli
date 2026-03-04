// Package shell 提供交互式命令行 Shell
// 当用户双击 .exe 启动时，自动进入交互模式
package shell

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/xiangjianhe-github/jiasinecli/internal/banner"
)

// interactiveHelp 交互模式下的帮助信息
func interactiveHelp() string {
	c := banner.BrightCyan  // 标题色
	g := banner.BrightGreen // 命令色
	d := banner.Dim         // 说明色
	b := banner.Bold
	r := banner.Reset

	return fmt.Sprintf(`
%s%s可用命令%s
  %sversion%s        查看版本信息
  %stest%s           运行集成测试      %s--lang c|python|rust|csharp|js|ts|java|swift|objc|all%s
  %sbridge%s         桥接层管理        %slist | call%s
  %sservice%s        服务管理          %slist | call | health%s
  %splugin%s         插件管理          %sview | list | install | remove | <名称>%s
  %sai%s             AI 大模型交互    %schat | provider | agent | skill%s

%s%s内置命令%s
  %shelp%s           显示此帮助
  %sclear%s / %scls%s    清屏
  %sbanner%s         显示 Logo
  %sjiasine%s        回到初始欢迎界面
  %sexit%s / %squit%s    退出

%s提示%s: 输入任意命令按 Enter 执行，如 %splugin view%s
`,
		b, c, r,
		g, r,
		g, r, d, r,
		g, r, d, r,
		g, r, d, r,
		g, r, d, r,
		g, r, d, r,
		b, c, r,
		g, r,
		g, r, g, r,
		g, r,
		g, r,
		g, r, g, r,
		d, r, c, r,
	)
}

// RunInteractive 启动交互式 Shell 模式
// 显示欢迎屏幕，循环等待用户输入命令
func RunInteractive(executeFunc func(args []string) error) {
	// 在 Windows 上启用 ANSI 转义序列
	enableVirtualTerminal()

	// 显示欢迎屏幕
	fmt.Print(banner.WelcomeScreen())

	scanner := bufio.NewScanner(os.Stdin)
	prompt := fmt.Sprintf("%s%sjiasinecli%s%s > %s",
		banner.Bold, banner.BrightCyan,
		banner.Reset, banner.Dim, banner.Reset,
	)

	for {
		fmt.Print(prompt)
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		// 去除可能的 BOM 和不可见字符
		input = strings.TrimLeft(input, "\uFEFF\u200B\u00A0")
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// 内置命令
		switch strings.ToLower(input) {
		case "exit", "quit", "q":
			fmt.Print(banner.Farewell())
			return
		case "clear", "cls":
			clearScreen()
			fmt.Print(banner.ShortBanner())
			fmt.Println()
			continue
		case "banner":
			fmt.Println(banner.Logo())
			continue
		case "help", "?":
			fmt.Print(interactiveHelp())
			continue
		case "jiasine", "jiasinecli":
			// 回到初始状态，重新显示欢迎屏幕
			clearScreen()
			fmt.Print(banner.WelcomeScreen())
			continue
		}

		// 解析参数并执行 cobra 命令
		args := parseArgs(input)
		if err := executeFunc(args); err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "unknown command") {
				fmt.Printf("  %s未知命令%s: %s，输入 %shelp%s 查看可用命令\n",
					banner.Bold+banner.BrightCyan, banner.Reset,
					args[0],
					banner.BrightGreen, banner.Reset)
			} else {
				fmt.Printf("  %s错误%s: %v\n",
					banner.Bold+banner.BrightCyan, banner.Reset, err)
			}
		}
		fmt.Println() // 命令间留空行
	}
}

// parseArgs 简单解析命令行参数（支持引号）
func parseArgs(input string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(input); i++ {
		ch := input[i]
		switch {
		case !inQuote && (ch == '"' || ch == '\''):
			inQuote = true
			quoteChar = ch
		case inQuote && ch == quoteChar:
			inQuote = false
		case !inQuote && ch == ' ':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// clearScreen 清空终端屏幕
func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

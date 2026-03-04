// Package tui 提供终端交互 UI 组件
//
// 支持上下箭头键选择的交互式列表
package tui

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// ANSI 颜色/控制码
const (
	reset      = "\033[0m"
	bold       = "\033[1m"
	dim        = "\033[2m"
	brightCyan = "\033[96m"
	brightGreen = "\033[92m"
	hideCursor = "\033[?25l"
	showCursor = "\033[?25h"
)

// SelectOption 选择项
type SelectOption struct {
	Label       string // 主标签 (如 "DeepSeek")
	Description string // 描述 (如 "deepseek-chat")
	Active      bool   // 是否为当前激活项
}

// Select 显示交互式选择菜单，支持上下箭头键选择
//
// 参数:
//   - prompt: 提示文字 (如 "选择要连接的 AI 模型")
//   - options: 选项列表
//   - defaultIdx: 默认选中项索引 (0-based)
//
// 返回选中的索引，取消时返回 -1
func Select(prompt string, options []SelectOption, defaultIdx int) (int, error) {
	if len(options) == 0 {
		return -1, fmt.Errorf("无可选项")
	}

	if defaultIdx < 0 || defaultIdx >= len(options) {
		defaultIdx = 0
	}

	// 尝试进入 raw 模式
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// raw 模式失败，回退到数字输入
		return selectFallback(prompt, options, defaultIdx)
	}
	defer term.Restore(fd, oldState)

	selected := defaultIdx

	// 隐藏光标
	write(hideCursor)
	defer write(showCursor)

	// 打印提示
	writeln(fmt.Sprintf("\r%s%s %s%s", bold+brightCyan, "?", prompt, reset))
	writeln(fmt.Sprintf("\r%s  ↑/↓ 选择, Enter 确认, Esc 取消%s", dim, reset))
	writeln("")

	// 渲染初始菜单
	renderSelect(options, selected)

	// 读取按键
	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return -1, err
		}

		switch {
		case n == 1 && (buf[0] == 13 || buf[0] == 10): // Enter
			clearLines(len(options))
			renderSelectFinal(options, selected)
			return selected, nil

		case n == 1 && buf[0] == 3: // Ctrl+C
			clearLines(len(options))
			writeln(fmt.Sprintf("\r%s已取消%s", dim, reset))
			return -1, fmt.Errorf("已取消")

		case n == 1 && buf[0] == 27: // 可能是 Esc 或 ANSI 转义序列的开始
			// 尝试读取更多字节（非阻塞检查）
			extra := make([]byte, 2)
			os.Stdin.Read(extra)
			if extra[0] == '[' {
				switch extra[1] {
				case 'A': // Up
					if selected > 0 {
						selected--
					}
				case 'B': // Down
					if selected < len(options)-1 {
						selected++
					}
				}
			} else if extra[0] == 0 {
				// 纯 Esc 键
				clearLines(len(options))
				writeln(fmt.Sprintf("\r%s已取消%s", dim, reset))
				return -1, fmt.Errorf("已取消")
			}

		case n == 3 && buf[0] == 27 && buf[1] == '[': // ANSI 序列
			switch buf[2] {
			case 'A': // Up
				if selected > 0 {
					selected--
				}
			case 'B': // Down
				if selected < len(options)-1 {
					selected++
				}
			}

		case n == 2 && (buf[0] == 0xe0 || buf[0] == 0x00): // Windows 扫描码
			switch buf[1] {
			case 0x48: // Up
				if selected > 0 {
					selected--
				}
			case 0x50: // Down
				if selected < len(options)-1 {
					selected++
				}
			}

		case n == 1 && buf[0] == 'k': // Vim 风格
			if selected > 0 {
				selected--
			}
		case n == 1 && buf[0] == 'j':
			if selected < len(options)-1 {
				selected++
			}
		}

		// 重绘
		clearLines(len(options))
		renderSelect(options, selected)
	}
}

// renderSelect 渲染选择菜单
func renderSelect(options []SelectOption, selected int) {
	for i, opt := range options {
		if i == selected {
			// 选中项: > 高亮显示
			active := ""
			if opt.Active {
				active = fmt.Sprintf(" %s(当前)%s", brightGreen, reset)
			}
			line := fmt.Sprintf("\r  %s❯ %s%s", brightCyan, opt.Label, reset)
			if opt.Description != "" {
				line += fmt.Sprintf("  %s%s%s", dim, opt.Description, reset)
			}
			line += active
			writeln(line)
		} else {
			// 未选中: 缩进
			active := ""
			if opt.Active {
				active = fmt.Sprintf(" %s(当前)%s", dim, reset)
			}
			line := fmt.Sprintf("\r    %s%s%s", dim, opt.Label, reset)
			if opt.Description != "" {
				line += fmt.Sprintf("  %s%s%s", dim, opt.Description, reset)
			}
			line += active
			writeln(line)
		}
	}
}

// renderSelectFinal 渲染最终选择结果
func renderSelectFinal(options []SelectOption, selected int) {
	opt := options[selected]
	line := fmt.Sprintf("\r  %s✓ %s%s", brightGreen, opt.Label, reset)
	if opt.Description != "" {
		line += fmt.Sprintf("  %s%s%s", dim, opt.Description, reset)
	}
	writeln(line)
}

// clearLines 清除 n 行内容（光标上移并清行）
func clearLines(n int) {
	for i := 0; i < n; i++ {
		write("\033[1A") // 上移一行
		write("\033[2K") // 清除整行
	}
}

// write 写入到 stdout（raw 模式下必须直接写）
func write(s string) {
	os.Stdout.WriteString(s)
}

// writeln 写入一行（raw 模式下 \n 不做 \r，需要 \r\n）
func writeln(s string) {
	os.Stdout.WriteString(s + "\r\n")
}

// selectFallback 当 raw 模式不可用时的数字输入回退
func selectFallback(prompt string, options []SelectOption, defaultIdx int) (int, error) {
	fmt.Printf("\n%s%s %s%s\n", bold+brightCyan, "?", prompt, reset)
	fmt.Println(strings.Repeat("─", 50))

	for i, opt := range options {
		activeTag := ""
		if opt.Active {
			activeTag = fmt.Sprintf(" %s(当前)%s", brightGreen, reset)
		}
		fmt.Printf("  %s%d%s. %s", brightCyan, i+1, reset, opt.Label)
		if opt.Description != "" {
			fmt.Printf("  %s%s%s", dim, opt.Description, reset)
		}
		fmt.Printf("%s\n", activeTag)
	}

	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("\n请输入编号 (1-%d) [%d]: ", len(options), defaultIdx+1)

	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultIdx, nil
	}

	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(options) {
		return -1, fmt.Errorf("无效选择: %s", input)
	}

	return choice - 1, nil
}

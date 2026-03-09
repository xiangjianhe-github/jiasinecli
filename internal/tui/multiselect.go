package tui

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// MultiSelectOption 多选项
type MultiSelectOption struct {
	Label       string // 主标签
	Description string // 描述
	Checked     bool   // 是否选中
}

// MultiSelect 显示交互式复选框菜单，支持上下箭头和空格选择
//
// 参数:
//   - prompt: 提示文字 (如 "选择要启用的 Skills")
//   - options: 选项列表
//
// 返回每个选项的选中状态（与输入 options 对应），取消时返回 nil
func MultiSelect(prompt string, options []MultiSelectOption) ([]bool, error) {
	if len(options) == 0 {
		return nil, fmt.Errorf("无可选项")
	}

	// 尝试进入 raw 模式
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return multiSelectFallback(prompt, options)
	}
	defer term.Restore(fd, oldState)

	// 复制初始选中状态
	checked := make([]bool, len(options))
	for i, opt := range options {
		checked[i] = opt.Checked
	}

	cursor := 0

	// 隐藏光标
	write(hideCursor)
	defer write(showCursor)

	// 打印提示
	writeln(fmt.Sprintf("\r%s%s %s%s", bold+brightCyan, "?", prompt, reset))
	writeln(fmt.Sprintf("\r%s  ↑/↓ 移动, Space 选择/取消, Enter 确认, Esc 取消%s", dim, reset))
	writeln("")

	renderMultiSelect(options, checked, cursor)

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return nil, err
		}

		switch {
		case n == 1 && (buf[0] == 13 || buf[0] == 10): // Enter
			clearLines(len(options))
			renderMultiSelectFinal(options, checked)
			return checked, nil

		case n == 1 && buf[0] == ' ': // Space - 切换选中
			checked[cursor] = !checked[cursor]

		case n == 1 && buf[0] == 3: // Ctrl+C
			clearLines(len(options))
			writeln(fmt.Sprintf("\r%s已取消%s", dim, reset))
			return nil, fmt.Errorf("已取消")

		case n == 1 && buf[0] == 27: // Esc 或 ANSI 序列开始
			extra := make([]byte, 2)
			en, _ := os.Stdin.Read(extra)
			if en >= 1 && extra[0] == '[' {
				var arrow byte
				if en >= 2 {
					arrow = extra[1]
				} else {
					ab := make([]byte, 1)
					if an, _ := os.Stdin.Read(ab); an == 1 {
						arrow = ab[0]
					}
				}
				switch arrow {
				case 'A': // Up
					if cursor > 0 {
						cursor--
					}
				case 'B': // Down
					if cursor < len(options)-1 {
						cursor++
					}
				}
			} else {
				clearLines(len(options))
				writeln(fmt.Sprintf("\r%s已取消%s", dim, reset))
				return nil, fmt.Errorf("已取消")
			}

		case n == 3 && buf[0] == 27 && buf[1] == '[': // ANSI
			switch buf[2] {
			case 'A':
				if cursor > 0 {
					cursor--
				}
			case 'B':
				if cursor < len(options)-1 {
					cursor++
				}
			}

		case n == 2 && (buf[0] == 0xe0 || buf[0] == 0x00): // Windows 扫描码
			switch buf[1] {
			case 0x48:
				if cursor > 0 {
					cursor--
				}
			case 0x50:
				if cursor < len(options)-1 {
					cursor++
				}
			}

		case n == 1 && buf[0] == 'k':
			if cursor > 0 {
				cursor--
			}
		case n == 1 && buf[0] == 'j':
			if cursor < len(options)-1 {
				cursor++
			}
		case n == 1 && buf[0] == 'x': // x 切换选中
			checked[cursor] = !checked[cursor]
		}

		clearLines(len(options))
		renderMultiSelect(options, checked, cursor)
	}
}

func renderMultiSelect(options []MultiSelectOption, checked []bool, cursor int) {
	for i, opt := range options {
		check := "☐"
		if checked[i] {
			check = fmt.Sprintf("%s☑%s", brightGreen, reset)
		}

		if i == cursor {
			line := fmt.Sprintf("\r  %s❯%s %s %s%s%s", brightCyan, reset, check, bold, opt.Label, reset)
			if opt.Description != "" {
				line += fmt.Sprintf("  %s%s%s", dim, opt.Description, reset)
			}
			writeln(line)
		} else {
			line := fmt.Sprintf("\r    %s %s%s%s", check, dim, opt.Label, reset)
			if opt.Description != "" {
				line += fmt.Sprintf("  %s%s%s", dim, opt.Description, reset)
			}
			writeln(line)
		}
	}
}

func renderMultiSelectFinal(options []MultiSelectOption, checked []bool) {
	var selected []string
	for i, opt := range options {
		if checked[i] {
			selected = append(selected, opt.Label)
		}
	}
	if len(selected) == 0 {
		writeln(fmt.Sprintf("\r  %s✓ 未选择任何项%s", dim, reset))
	} else {
		for _, name := range selected {
			writeln(fmt.Sprintf("\r  %s✓ %s%s", brightGreen, name, reset))
		}
	}
}

func multiSelectFallback(prompt string, options []MultiSelectOption) ([]bool, error) {
	fmt.Printf("\n%s%s %s%s\n", bold+brightCyan, "?", prompt, reset)
	fmt.Println("输入编号切换选中 (如: 1 3 5), 输入 'done' 确认:")

	checked := make([]bool, len(options))
	for i, opt := range options {
		checked[i] = opt.Checked
	}

	for {
		for i, opt := range options {
			check := "[ ]"
			if checked[i] {
				check = fmt.Sprintf("[%s✓%s]", brightGreen, reset)
			}
			fmt.Printf("  %s%d%s. %s %s", brightCyan, i+1, reset, check, opt.Label)
			if opt.Description != "" {
				fmt.Printf("  %s%s%s", dim, opt.Description, reset)
			}
			fmt.Println()
		}

		fmt.Print("\n> ")
		var input string
		fmt.Scanln(&input)

		if input == "done" || input == "" {
			return checked, nil
		}

		var idx int
		if _, err := fmt.Sscanf(input, "%d", &idx); err == nil && idx >= 1 && idx <= len(options) {
			checked[idx-1] = !checked[idx-1]
		}
	}
}

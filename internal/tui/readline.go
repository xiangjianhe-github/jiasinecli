package tui

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

// commandDescs 命令描述（用于智能提示）
var commandDescs = map[string]string{
	"/help":           "显示帮助",
	"/model":          "切换 AI 模型",
	"/connect":        "切换 AI 模型",
	"/web":            "切换联网搜索",
	"/search":         "切换联网搜索",
	"/memory":         "查看记忆状态",
	"/memory on":      "开启记忆",
	"/memory off":     "关闭记忆",
	"/mem":            "查看记忆状态",
	"/mem on":         "开启记忆",
	"/mem off":        "关闭记忆",
	"/mem clear":      "清空所有记忆",
	"/new":            "开始新会话",
	"/clear":          "清空对话历史",
	"/reset":          "清空对话历史",
	"/compact":        "压缩上下文",
	"/context":        "查看上下文",
	"/agent":          "选择 Agent",
	"/theme":          "切换主题",
	"/skills":         "管理 Skills",
	"/history":        "查看历史会话",
	"/status":         "查看当前状态",
	"/version":        "版本信息",
	"/setup":          "配置系统环境",
	"/update":         "检查更新",
	"/plugin list":    "查看插件",
	"/plugin install": "安装插件",
	"/plugin remove":  "删除插件",
	"/exit":           "退出",
	"/quit":           "退出",
}

// slashCommands 所有可补全的斜杠命令（按显示优先级排序）
var slashCommands = []string{
	"/help", "/model", "/web", "/memory", "/mem",
	"/mem on", "/mem off", "/mem clear",
	"/memory on", "/memory off",
	"/new", "/clear", "/reset",
	"/compact", "/context",
	"/agent", "/theme", "/skills",
	"/history", "/status",
	"/connect", "/search",
	"/version", "/setup", "/update",
	"/plugin list", "/plugin install", "/plugin remove",
	"/exit", "/quit",
}

// webSearchMode 当前是否处于联网模式（影响提示符显示）
var webSearchMode bool

// ReadLine 在 raw 模式下读取一行输入，支持:
//   - 左右箭头移动光标、Home/End、Backspace/Delete
//   - Tab 补全 /命令
//   - 输入 / 开头时自动显示匹配命令列表
//   - Ctrl+C 返回 false，Enter 提交
//
// webSearch 控制提示符是否显示 🌐 图标
func ReadLine(webSearch bool) (string, bool) {
	webSearchMode = webSearch

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		writePrompt()
		return readLineFallback()
	}
	defer term.Restore(fd, oldState)

	writePrompt()

	var line []rune
	pos := 0

	buf := make([]byte, 64)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			clearSuggestions()
			return "", false
		}

		i := 0
		for i < n {
			b := buf[i]

			switch {
			case b == 13 || b == 10: // Enter
				clearSuggestions()
				os.Stdout.WriteString("\r\n")
				return string(line), true

			case b == 3: // Ctrl+C
				clearSuggestions()
				os.Stdout.WriteString("\r\n")
				return "", false

			case b == 4: // Ctrl+D (EOF)
				if len(line) == 0 {
					clearSuggestions()
					os.Stdout.WriteString("\r\n")
					return "", false
				}
				i++
				continue

			case b == 9: // Tab — 自动补全
				line, pos = handleTabCompletion(line, pos)
				updateSuggestions(line, pos)
				i++
				continue

			case b == 127 || b == 8: // Backspace
				if pos > 0 {
					line = append(line[:pos-1], line[pos:]...)
					pos--
					redrawLine(line, pos)
					updateSuggestions(line, pos)
				}
				i++
				continue

			case b == 21: // Ctrl+U — 清除整行
				line = nil
				pos = 0
				redrawLine(line, pos)
				updateSuggestions(line, pos)
				i++
				continue

			case b == 23: // Ctrl+W — 删除前一个单词
				if pos > 0 {
					newPos := pos - 1
					for newPos > 0 && line[newPos-1] == ' ' {
						newPos--
					}
					for newPos > 0 && line[newPos-1] != ' ' {
						newPos--
					}
					line = append(line[:newPos], line[pos:]...)
					pos = newPos
					redrawLine(line, pos)
					updateSuggestions(line, pos)
				}
				i++
				continue

			case b == 1: // Ctrl+A — 行首
				pos = 0
				redrawLine(line, pos)
				i++
				continue

			case b == 5: // Ctrl+E — 行尾
				pos = len(line)
				redrawLine(line, pos)
				i++
				continue

			case b == 27: // ESC — 可能是方向键序列
				if i+2 < n && buf[i+1] == '[' {
					switch buf[i+2] {
					case 'A': // Up
						i += 3
						continue
					case 'B': // Down
						i += 3
						continue
					case 'C': // Right
						if pos < len(line) {
							pos++
							redrawLine(line, pos)
						}
						i += 3
						continue
					case 'D': // Left
						if pos > 0 {
							pos--
							redrawLine(line, pos)
						}
						i += 3
						continue
					case '3': // Delete key (ESC [ 3 ~)
						if i+3 < n && buf[i+3] == '~' {
							if pos < len(line) {
								line = append(line[:pos], line[pos+1:]...)
								redrawLine(line, pos)
								updateSuggestions(line, pos)
							}
							i += 4
							continue
						}
					case 'H': // Home
						pos = 0
						redrawLine(line, pos)
						i += 3
						continue
					case 'F': // End
						pos = len(line)
						redrawLine(line, pos)
						i += 3
						continue
					}
					i += 3
					continue
				} else if i+1 < n && buf[i+1] == '[' {
					// ESC [ 收到但第三字节还没到
					extra := make([]byte, 1)
					en, _ := os.Stdin.Read(extra)
					if en == 1 {
						switch extra[0] {
						case 'C':
							if pos < len(line) {
								pos++
								redrawLine(line, pos)
							}
						case 'D':
							if pos > 0 {
								pos--
								redrawLine(line, pos)
							}
						case '3':
							del := make([]byte, 1)
							os.Stdin.Read(del)
							if pos < len(line) {
								line = append(line[:pos], line[pos+1:]...)
								redrawLine(line, pos)
							updateSuggestions(line, pos)
							}
						}
					}
					i += 2
					continue
				}
				i++
				continue

			case b == 0xe0 || b == 0x00: // Windows 扫描码
				if i+1 < n {
					switch buf[i+1] {
					case 0x4B: // Left
						if pos > 0 {
							pos--
							redrawLine(line, pos)
						}
					case 0x4D: // Right
						if pos < len(line) {
							pos++
							redrawLine(line, pos)
						}
					case 0x53: // Delete
						if pos < len(line) {
							line = append(line[:pos], line[pos+1:]...)
							redrawLine(line, pos)
							updateSuggestions(line, pos)
						}
					case 0x47: // Home
						pos = 0
						redrawLine(line, pos)
					case 0x4F: // End
						pos = len(line)
						redrawLine(line, pos)
					}
					i += 2
					continue
				}
				i++
				continue

			default:
				// 普通字符（可能是 UTF-8 多字节）
				remaining := buf[i:n]
				r, size := utf8.DecodeRune(remaining)
				if r != utf8.RuneError {
					line = append(line[:pos], append([]rune{r}, line[pos:]...)...)
					pos++
					redrawLine(line, pos)
					updateSuggestions(line, pos)
					i += size
				} else {
					i++
				}
				continue
			}

			i++
		}
	}
}

// writePrompt 输出提示符
func writePrompt() {
	if webSearchMode {
		os.Stdout.WriteString("\033[96m┃\033[0m \033[96m🌐\033[0m \033[96m❯\033[0m ")
	} else {
		os.Stdout.WriteString("\033[96m┃\033[0m \033[96m❯\033[0m ")
	}
}

// handleTabCompletion 处理 Tab 补全
func handleTabCompletion(line []rune, pos int) ([]rune, int) {
	input := string(line[:pos])
	if !strings.HasPrefix(input, "/") {
		return line, pos
	}

	inputLower := strings.ToLower(input)
	var matches []string
	for _, cmd := range slashCommands {
		if strings.HasPrefix(strings.ToLower(cmd), inputLower) {
			matches = append(matches, cmd)
		}
	}

	if len(matches) == 0 {
		return line, pos
	}

	if len(matches) == 1 {
		// 唯一匹配 — 补全
		newLine := []rune(matches[0])
		newPos := len(newLine)
		redrawLine(newLine, newPos)
		return newLine, newPos
	}

	// 多个匹配 — 补全公共前缀
	common := longestCommonPrefix(matches)
	commonRunes := []rune(common)
	if len(commonRunes) > pos {
		newLine := append(commonRunes, line[pos:]...)
		newPos := len(commonRunes)
		redrawLine(newLine, newPos)
		return newLine, newPos
	}

	// 公共前缀等于当前输入，无法继续补全（提示已在下方显示）
	return line, pos
}

// longestCommonPrefix 计算字符串列表的最长公共前缀
func longestCommonPrefix(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	prefix := ss[0]
	for _, s := range ss[1:] {
		for !strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix)) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				return ""
			}
		}
	}
	return prefix
}

// redrawLine 清除当前行并重绘（含提示符）
func redrawLine(line []rune, pos int) {
	s := string(line)
	if webSearchMode {
		os.Stdout.WriteString(fmt.Sprintf("\r\033[2K\033[96m┃\033[0m \033[96m🌐\033[0m \033[96m❯\033[0m %s", s))
	} else {
		os.Stdout.WriteString(fmt.Sprintf("\r\033[2K\033[96m┃\033[0m \033[96m❯\033[0m %s", s))
	}
	if pos < len(line) {
		moveBack := len(line) - pos
		os.Stdout.WriteString(fmt.Sprintf("\033[%dD", moveBack))
	}
}

// updateSuggestions 根据当前输入内容在行下方显示匹配的命令提示
// 使用相对光标移动（\033[nA）而非保存/恢复，避免终端滚动时光标位置异常
func updateSuggestions(line []rune, pos int) {
	// 移到下一行并清除到屏幕底部（删除旧的提示）
	os.Stdout.WriteString("\r\n\033[J")

	count := 0
	input := string(line)
	if strings.HasPrefix(input, "/") && len(input) > 0 {
		inputLower := strings.ToLower(input)
		var matches []string
		for _, cmd := range slashCommands {
			if strings.HasPrefix(strings.ToLower(cmd), inputLower) {
				matches = append(matches, cmd)
			}
		}
		maxShow := 10
		if len(matches) > maxShow {
			matches = matches[:maxShow]
		}
		for _, m := range matches {
			desc := commandDescs[m]
			if desc != "" {
				os.Stdout.WriteString(fmt.Sprintf("  \033[96m%-18s\033[0m \033[90m%s\033[0m\r\n", m, desc))
			} else {
				os.Stdout.WriteString(fmt.Sprintf("  \033[96m%s\033[0m\r\n", m))
			}
			count++
		}
	}

	// 用相对移动回到输入行：初始 \r\n 下移 1 行 + count 行提示
	os.Stdout.WriteString(fmt.Sprintf("\033[%dA", 1+count))
	// 重绘输入行以修正光标位置
	redrawLine(line, pos)
}

// clearSuggestions 清除行下方的命令提示
func clearSuggestions() {
	os.Stdout.WriteString("\r\n\033[J\033[1A")
}

// readLineFallback raw 模式不可用时的回退
func readLineFallback() (string, bool) {
	buf := make([]byte, 8192)
	n, err := os.Stdin.Read(buf)
	if err != nil {
		return "", false
	}
	text := strings.TrimRight(string(buf[:n]), "\r\n")
	return text, true
}

package render

import (
	"fmt"
	"regexp"
	"strings"
)

// Markdown 将 Markdown 文本渲染为带 ANSI 转义码的终端输出
// 支持: 标题(#), 代码块(```), 行内代码(`), 粗体(**), 斜体(*), 列表(- / 1.), 分隔线(---)
func Markdown(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	var codeBlock []string
	var codeLang string
	inCodeBlock := false

	for _, line := range lines {
		// 代码块检测
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			if !inCodeBlock {
				// 代码块开始
				trimmed := strings.TrimSpace(line)
				codeLang = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
				inCodeBlock = true
				codeBlock = nil
				continue
			}
			// 代码块结束
			inCodeBlock = false
			rendered := renderCodeBlock(codeLang, strings.Join(codeBlock, "\n"))
			result = append(result, rendered)
			codeBlock = nil
			codeLang = ""
			continue
		}

		if inCodeBlock {
			codeBlock = append(codeBlock, line)
			continue
		}

		// 普通 Markdown 行
		result = append(result, renderMarkdownLine(line))
	}

	// 如果代码块未关闭，仍然渲染
	if inCodeBlock && len(codeBlock) > 0 {
		rendered := renderCodeBlock(codeLang, strings.Join(codeBlock, "\n"))
		result = append(result, rendered)
	}

	return strings.Join(result, "\n")
}

// renderCodeBlock 渲染代码块：语言标签 + 语法高亮 + 边框
func renderCodeBlock(lang, code string) string {
	if code == "" {
		return ""
	}

	var sb strings.Builder

	langLabel := lang
	if langLabel == "" {
		langLabel = "code"
	}

	// 固定框宽度: 49 个可见字符 (含左右边框字符)
	const boxWidth = 49

	// 顶部: ┌─── lang ─────...┐
	// "┌─── " = 5 chars, " " after lang = 1, "┐" = 1
	padLen := boxWidth - 5 - len(langLabel) - 1 - 1
	if padLen < 3 {
		padLen = 3
	}
	sb.WriteString(fmt.Sprintf("\033[2m┌─── \033[0m\033[96m%s\033[0m\033[2m %s┐\033[0m\n",
		langLabel, strings.Repeat("─", padLen)))

	// 语法高亮代码行: │ NNN code
	highlighted := HighlightCode(lang, code)
	codeLines := strings.Split(highlighted, "\n")
	for i, cl := range codeLines {
		lineNum := fmt.Sprintf("%3d", i+1)
		sb.WriteString(fmt.Sprintf("\033[2m│\033[0m \033[2m%s\033[0m %s%s%s\n",
			lineNum, ansiBgCode, cl, ansiReset+ansiBgReset))
	}

	// 底部: └─────...─────┘  (宽度与顶部一致)
	sb.WriteString(fmt.Sprintf("\033[2m└%s┘\033[0m", strings.Repeat("─", boxWidth-2)))

	return sb.String()
}

// inline 样式正则
var (
	boldRegex       = regexp.MustCompile(`\*\*(.+?)\*\*`)
	italicRegex     = regexp.MustCompile(`(?:^|[^*])\*([^*]+?)\*(?:[^*]|$)`)
	inlineCodeRegex = regexp.MustCompile("`([^`]+)`")
	linkRegex       = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

// renderMarkdownLine 渲染单行 Markdown
func renderMarkdownLine(line string) string {
	trimmed := strings.TrimSpace(line)

	// 空行
	if trimmed == "" {
		return ""
	}

	// 水平分隔线
	if isHorizontalRule(trimmed) {
		return "\033[2m────────────────────────────────────────────────\033[0m"
	}

	// 标题
	if strings.HasPrefix(trimmed, "#") {
		return renderHeading(trimmed)
	}

	// 无序列表
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ") {
		indent := countLeadingSpaces(line)
		content := trimmed[2:]
		prefix := strings.Repeat("  ", indent/2) + "\033[96m•\033[0m "
		return prefix + renderInline(content)
	}

	// 有序列表
	if matched, _ := regexp.MatchString(`^\d+\.\s`, trimmed); matched {
		idx := strings.Index(trimmed, ".")
		num := trimmed[:idx]
		content := strings.TrimSpace(trimmed[idx+1:])
		indent := countLeadingSpaces(line)
		prefix := strings.Repeat("  ", indent/2) + "\033[96m" + num + ".\033[0m "
		return prefix + renderInline(content)
	}

	// 引用块
	if strings.HasPrefix(trimmed, "> ") {
		content := trimmed[2:]
		return "\033[2m│\033[0m \033[3m" + renderInline(content) + "\033[0m"
	}

	// 普通段落
	return renderInline(line)
}

// renderHeading 渲染标题
func renderHeading(line string) string {
	level := 0
	for _, ch := range line {
		if ch == '#' {
			level++
		} else {
			break
		}
	}
	content := strings.TrimSpace(line[level:])

	switch level {
	case 1:
		return fmt.Sprintf("\n\033[1m\033[96m█ %s\033[0m\n", content)
	case 2:
		return fmt.Sprintf("\n\033[1m\033[94m▌ %s\033[0m\n", content)
	case 3:
		return fmt.Sprintf("\033[1m\033[95m  ▸ %s\033[0m", content)
	default:
		return fmt.Sprintf("\033[1m%s%s\033[0m", strings.Repeat("  ", level-1), content)
	}
}

// renderInline 渲染行内 Markdown 元素
func renderInline(text string) string {
	// 行内代码 (必须在 bold/italic 之前处理)
	text = inlineCodeRegex.ReplaceAllStringFunc(text, func(m string) string {
		code := inlineCodeRegex.FindStringSubmatch(m)[1]
		return "\033[48;5;236m\033[96m " + code + " \033[0m"
	})

	// 粗体
	text = boldRegex.ReplaceAllStringFunc(text, func(m string) string {
		inner := boldRegex.FindStringSubmatch(m)[1]
		return "\033[1m" + inner + "\033[0m"
	})

	// 链接 [text](url)
	text = linkRegex.ReplaceAllStringFunc(text, func(m string) string {
		parts := linkRegex.FindStringSubmatch(m)
		return fmt.Sprintf("\033[4m\033[94m%s\033[0m \033[2m(%s)\033[0m", parts[1], parts[2])
	})

	return text
}

func isHorizontalRule(line string) bool {
	clean := strings.ReplaceAll(line, " ", "")
	if len(clean) < 3 {
		return false
	}
	allDash := true
	allStar := true
	allUnder := true
	for _, ch := range clean {
		if ch != '-' {
			allDash = false
		}
		if ch != '*' {
			allStar = false
		}
		if ch != '_' {
			allUnder = false
		}
	}
	return allDash || allStar || allUnder
}

func countLeadingSpaces(s string) int {
	count := 0
	for _, ch := range s {
		if ch == ' ' {
			count++
		} else if ch == '\t' {
			count += 2
		} else {
			break
		}
	}
	return count
}

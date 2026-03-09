package render

import (
	"fmt"
	"regexp"
	"strings"
)

// ANSI 转义序列（markdown 格式化专用 - VS Code 暗色主题风格）
// 基础格式化常量（ansiReset, ansiBold, ansiDim, ansiItalic）定义在 highlight.go
const (
	ansiUnderline = "\033[4m"
	ansiBlue      = "\033[38;5;75m"  // 亮蓝色 (链接)
	ansiCyan      = "\033[38;5;87m"  // 亮青色 (高亮)
	ansiGreen     = "\033[38;5;120m" // 亮绿色 (一级标题)
	ansiYellow    = "\033[38;5;228m" // 亮黄色 (二级标题)
	ansiMagenta   = "\033[38;5;213m" // 亮紫色 (三级标题)
	ansiBgGray    = "\033[48;5;235m" // 深灰色背景 (代码块)
	ansiCodeText  = "\033[38;5;215m" // 行内代码文字颜色 (浅橙)
)

// Markdown 将 Markdown 文本渲染为带 ANSI 转义码的终端输出
// 支持: 标题(#), 代码块(```), 行内代码(`), 列表(- / *), 链接([]()), 水平线(---), 粗体(**), 斜体(*)
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

// renderCodeBlock 渲染代码块：深灰背景 + 语法高亮
func renderCodeBlock(lang, code string) string {
	if code == "" {
		return ""
	}

	var sb strings.Builder

	// 顶部标签（如果有语言标识）
	if lang != "" {
		sb.WriteString(fmt.Sprintf("\n%s%s┌─ %s %s%s\n", ansiDim, ansiBgGray, lang, strings.Repeat("─", 36), ansiReset))
	} else {
		sb.WriteString(fmt.Sprintf("\n%s%s┌%s%s\n", ansiDim, ansiBgGray, strings.Repeat("─", 42), ansiReset))
	}

	// 使用语法高亮渲染（如果支持该语言）
	highlighted := HighlightCode(lang, code)
	codeLines := strings.Split(highlighted, "\n")

	// 每行设置深灰背景
	for _, line := range codeLines {
		if line == "" {
			sb.WriteString(fmt.Sprintf("%s%s%s\n", ansiBgGray, strings.Repeat(" ", 44), ansiReset))
		} else {
			// 代码行前添加 │ 符号
			sb.WriteString(fmt.Sprintf("%s%s│%s %s%s\n", ansiDim, ansiBgGray, ansiReset+ansiBgGray, line, ansiReset))
		}
	}

	// 底部边框
	sb.WriteString(fmt.Sprintf("%s%s└%s%s\n", ansiDim, ansiBgGray, strings.Repeat("─", 42), ansiReset))

	return sb.String()
}

// inline 样式正则（处理顺序：行内代码 → 粗体 → 斜体 → 链接）
var (
	inlineCodeRegex   = regexp.MustCompile("`([^`]+)`")
	boldTextRegex     = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	italicStarRegex   = regexp.MustCompile(`\*([^*\s][^*]*?)\*`)  // *text* 但不匹配 **
	italicUnderRegex  = regexp.MustCompile(`_([^_]+)_`)
	linkRegex         = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
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
		return ansiDim + strings.Repeat("─", 50) + ansiReset
	}

	// 标题
	if strings.HasPrefix(trimmed, "#") {
		return renderHeading(trimmed)
	}

	// 无序列表
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ") {
		indent := countLeadingSpaces(line)
		content := trimmed[2:]
		prefix := strings.Repeat("  ", indent/2) + "• "
		return prefix + renderInline(content)
	}

	// 有序列表
	if matched, _ := regexp.MatchString(`^\d+\.\s`, trimmed); matched {
		idx := strings.Index(trimmed, ".")
		num := trimmed[:idx]
		content := strings.TrimSpace(trimmed[idx+1:])
		indent := countLeadingSpaces(line)
		prefix := strings.Repeat("  ", indent/2) + num + ". "
		return prefix + renderInline(content)
	}

	// 引用块
	if strings.HasPrefix(trimmed, "> ") {
		content := trimmed[2:]
		return ansiDim + "│ " + ansiReset + renderInline(content)
	}

	// 普通段落
	return renderInline(line)
}

// renderHeading 渲染标题（加粗 + 颜色区分）
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

	// 不同级别标题用不同颜色和样式
	switch level {
	case 1:
		// 一级标题：亮绿色 + 加粗 + 双分隔线
		return fmt.Sprintf("\n%s%s═══ %s ═══%s\n", ansiBold, ansiGreen, content, ansiReset)
	case 2:
		// 二级标题：亮黄色 + 加粗 + 箭头
		return fmt.Sprintf("\n%s%s▸ %s%s\n", ansiBold, ansiYellow, content, ansiReset)
	case 3:
		// 三级标题：亮紫色 + 加粗 + 点
		return fmt.Sprintf("\n%s%s  • %s%s", ansiBold, ansiMagenta, content, ansiReset)
	case 4:
		// 四级标题：亮蓝色 + 加粗
		return fmt.Sprintf("\n%s%s    › %s%s", ansiBold, ansiBlue, content, ansiReset)
	default:
		// 更深层级：青色 + 缩进
		return fmt.Sprintf("%s%s%s%s%s", ansiBold, ansiCyan, strings.Repeat("  ", level-1), content, ansiReset)
	}
}

// renderInline 渲染行内 Markdown 元素
// 处理顺序很重要：行内代码 → 链接 → 粗体 → 斜体
func renderInline(text string) string {
	// 1. 行内代码：深灰背景 + 浅橙文字（确保黑色外围背景）
	text = inlineCodeRegex.ReplaceAllStringFunc(text, func(m string) string {
		code := inlineCodeRegex.FindStringSubmatch(m)[1]
		return fmt.Sprintf("%s%s %s %s", ansiBgCode, ansiCodeText, code, ansiReset)
	})

	// 2. 链接：蓝色下划线文字
	text = linkRegex.ReplaceAllStringFunc(text, func(m string) string {
		parts := linkRegex.FindStringSubmatch(m)
		displayText := parts[1]
		url := parts[2]
		return fmt.Sprintf("%s%s%s%s %s(%s)%s", ansiUnderline, ansiBlue, displayText, ansiReset, ansiDim, url, ansiReset)
	})

	// 3. 粗体：**text**（必须在斜体前处理，避免 ** 被拆成两个 *）
	text = boldTextRegex.ReplaceAllStringFunc(text, func(m string) string {
		content := boldTextRegex.FindStringSubmatch(m)[1]
		return fmt.Sprintf("%s%s%s", ansiBold, content, ansiReset)
	})

	// 4. 斜体：*text* 或 _text_
	text = italicStarRegex.ReplaceAllStringFunc(text, func(m string) string {
		content := italicStarRegex.FindStringSubmatch(m)[1]
		return fmt.Sprintf("%s%s%s", ansiItalic, content, ansiReset)
	})

	text = italicUnderRegex.ReplaceAllStringFunc(text, func(m string) string {
		content := italicUnderRegex.FindStringSubmatch(m)[1]
		return fmt.Sprintf("%s%s%s", ansiItalic, content, ansiReset)
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

// Package theme 提供 CLI 主题系统
// 支持 Dark 和 Light 两种现代风格主题
package theme

import "fmt"

// ThemeName 主题名称
type ThemeName string

const (
	Dark  ThemeName = "dark"
	Light ThemeName = "light"
)

// Theme 主题配色方案
type Theme struct {
	Name ThemeName
	// 文本样式
	Bold      string
	Dim       string
	Italic    string
	Underline string
	Reset     string
	// 主色
	Primary   string // 主色调 (标题、提示符)
	Secondary string // 次要色调 (链接)
	Accent    string // 强调色 (成功、选中)
	Warning   string // 警告色
	Error     string // 错误色
	Info      string // 信息色
	// 前景色系列
	Cyan          string
	Blue          string
	Green         string
	Yellow        string
	Red           string
	Magenta       string
	White         string
	BrightCyan    string
	BrightBlue    string
	BrightGreen   string
	BrightYellow  string
	BrightRed     string
	BrightMagenta string
	// 灰色系
	Gray      string
	LightGray string
	// 背景色
	BgCode    string // 代码块背景
	BgReset   string
	BgDefault string // 全局默认背景色 ANSI 序列
}

// current 当前激活的主题
var current *Theme

func init() {
	current = darkTheme()
}

// Current 返回当前主题
func Current() *Theme {
	return current
}

// Set 切换主题
func Set(name ThemeName) {
	switch name {
	case Light:
		current = lightTheme()
	default:
		current = darkTheme()
	}
}

// CurrentName 返回当前主题名称
func CurrentName() ThemeName {
	return current.Name
}

// Available 返回所有可用主题
func Available() []ThemeName {
	return []ThemeName{Dark, Light}
}

// darkTheme 暗色主题 — VS Code / Copilot CLI 风格
func darkTheme() *Theme {
	return &Theme{
		Name:      Dark,
		Bold:      "\033[1m",
		Dim:       "\033[2m",
		Italic:    "\033[3m",
		Underline: "\033[4m",
		Reset:     "\033[0m",
		// 主色
		Primary:   "\033[38;5;87m",  // 亮青色
		Secondary: "\033[38;5;111m", // 亮蓝色
		Accent:    "\033[38;5;120m", // 亮绿色
		Warning:   "\033[38;5;228m", // 亮黄色
		Error:     "\033[38;5;210m", // 亮红色
		Info:      "\033[38;5;80m",  // 柔和青色
		// 前景色
		Cyan:          "\033[38;5;80m",
		Blue:          "\033[38;5;75m",
		Green:         "\033[38;5;114m",
		Yellow:        "\033[38;5;221m",
		Red:           "\033[38;5;204m",
		Magenta:       "\033[38;5;176m",
		White:         "\033[38;5;231m",
		BrightCyan:    "\033[38;5;87m",
		BrightBlue:    "\033[38;5;111m",
		BrightGreen:   "\033[38;5;120m",
		BrightYellow:  "\033[38;5;228m",
		BrightRed:     "\033[38;5;210m",
		BrightMagenta: "\033[38;5;213m",
		// 灰色
		Gray:      "\033[38;5;240m",
		LightGray: "\033[38;5;250m",
		// 背景
		BgCode:    "\033[48;5;235m",
		BgReset:   "\033[49m",
		BgDefault: "\033[48;5;0m",
	}
}

// lightTheme 明亮主题 — 现代化浅色风格
func lightTheme() *Theme {
	return &Theme{
		Name:      Light,
		Bold:      "\033[1m",
		Dim:       "\033[2m",
		Italic:    "\033[3m",
		Underline: "\033[4m",
		Reset:     "\033[0m",
		// 主色 (较深，在浅色背景下可读)
		Primary:   "\033[38;5;31m",  // 深青色
		Secondary: "\033[38;5;25m",  // 深蓝色
		Accent:    "\033[38;5;28m",  // 深绿色
		Warning:   "\033[38;5;136m", // 深黄/橙色
		Error:     "\033[38;5;160m", // 深红色
		Info:      "\033[38;5;30m",  // 深青色
		// 前景色 (较深)
		Cyan:          "\033[38;5;30m",
		Blue:          "\033[38;5;25m",
		Green:         "\033[38;5;28m",
		Yellow:        "\033[38;5;136m",
		Red:           "\033[38;5;160m",
		Magenta:       "\033[38;5;133m",
		White:         "\033[38;5;232m",
		BrightCyan:    "\033[38;5;31m",
		BrightBlue:    "\033[38;5;33m",
		BrightGreen:   "\033[38;5;34m",
		BrightYellow:  "\033[38;5;172m",
		BrightRed:     "\033[38;5;196m",
		BrightMagenta: "\033[38;5;163m",
		// 灰色
		Gray:      "\033[38;5;243m",
		LightGray: "\033[38;5;248m",
		// 背景
		BgCode:    "\033[48;5;254m",
		BgReset:   "\033[49m",
		BgDefault: "\033[48;5;231m",
	}
}

// Prompt 生成格式化的提示符字符串
func (t *Theme) Prompt(label string) string {
	return fmt.Sprintf("%s%s%s%s%s > %s",
		t.Bold, t.Primary, label, t.Reset, t.Dim, t.Reset)
}

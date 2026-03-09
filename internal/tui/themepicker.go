package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── bubbletea 主题选择器 ─────────────────────────────────────────────────────

// ThemeInfo 传入主题选择器的主题描述
type ThemeInfo struct {
	Key         string // "dark" / "light"
	Icon        string // 显示图标
	Label       string // 显示名称
	Description string // 简短描述
	// 预览色板：颜色 ANSI 字符串 (用于 ansi256 预览)
	PrimaryHex   string // lipgloss 颜色 (hex)
	AccentHex    string
	BgHex        string
	TextHex      string
	CommentHex   string
	SuccessHex   string
	WarningHex   string
	ErrorHex     string
}

// DarkThemeInfo 暗色主题描述
func DarkThemeInfo() ThemeInfo {
	return ThemeInfo{
		Key: "dark", Icon: "🌙", Label: "暗色主题",
		Description: "VS Code 暗色调  ·  护眼  ·  夜间友好",
		BgHex: "#1e1e2e", TextHex: "#cdd6f4",
		PrimaryHex: "#89dceb", AccentHex: "#a6e3a1",
		CommentHex: "#6c7086", SuccessHex: "#a6e3a1",
		WarningHex: "#f9e2af", ErrorHex: "#f38ba8",
	}
}

// LightThemeInfo 亮色主题描述
func LightThemeInfo() ThemeInfo {
	return ThemeInfo{
		Key: "light", Icon: "☀️", Label: "亮色主题",
		Description: "现代浅色调  ·  办公室友好  ·  高对比",
		BgHex: "#eff1f5", TextHex: "#4c4f69",
		PrimaryHex: "#209fb5", AccentHex: "#40a02b",
		CommentHex: "#9ca0b0", SuccessHex: "#40a02b",
		WarningHex: "#df8e1d", ErrorHex: "#d20f39",
	}
}

// themePickerModel bubbletea 模型
type themePickerModel struct {
	themes   []ThemeInfo
	cursor   int
	selected int // -1 = 取消
	quit     bool
}

// Init 初始化
func (m themePickerModel) Init() tea.Cmd { return nil }

// Update 按键处理
func (m themePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k", "left", "h":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j", "right", "l":
			if m.cursor < len(m.themes)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = m.cursor
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			m.selected = -1
			m.quit = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View 渲染
func (m themePickerModel) View() string {
	var sb strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#89dceb")).
		Margin(0, 0, 1, 2)

	sb.WriteString(titleStyle.Render("🎨 选择主题"))
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6c7086")).
		MarginLeft(2).
		Render("↑/↓  选择    Enter 确认    Esc 取消"))
	sb.WriteString("\n\n")

	for i, th := range m.themes {
		card := renderThemeCard(th, i == m.cursor)
		sb.WriteString(lipgloss.NewStyle().MarginLeft(2).Render(card))
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderThemeCard 渲染单张主题卡片
func renderThemeCard(th ThemeInfo, active bool) string {
	bgColor := lipgloss.Color(th.BgHex)
	textColor := lipgloss.Color(th.TextHex)

	borderColor := lipgloss.Color("#6c7086")
	if active {
		borderColor = lipgloss.Color(th.PrimaryHex)
	}

	// 色板行
	swatches := renderSwatches(th)

	// 示例内容预览
	userLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.PrimaryHex)).
		Background(bgColor).
		Render("  ❯ 你好，今天天气怎么样？")

	aiLine := lipgloss.NewStyle().
		Foreground(textColor).
		Background(bgColor).
		Render("  🤖 今天天气晴朗，适合出门。")

	successLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.SuccessHex)).
		Background(bgColor).
		Render("  ✓ 主题应用成功")

	warnLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.WarningHex)).
		Background(bgColor).
		Render("  ⚠ 警告示例文字")

	// 标题行
	marker := "  "
	if active {
		marker = "❯ "
	}
	titleLine := lipgloss.NewStyle().
		Bold(active).
		Foreground(lipgloss.Color(th.PrimaryHex)).
		Background(bgColor).
		Padding(0, 1).
		Render(fmt.Sprintf("%s%s %s", marker, th.Icon, th.Label))

	descLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.CommentHex)).
		Background(bgColor).
		Padding(0, 1).
		Render(th.Description)

	// 组合内部内容
	inner := lipgloss.JoinVertical(lipgloss.Left,
		titleLine,
		descLine,
		lipgloss.NewStyle().Background(bgColor).Render(""),
		swatches,
		lipgloss.NewStyle().Background(bgColor).Render(""),
		userLine,
		aiLine,
		successLine,
		warnLine,
		lipgloss.NewStyle().Background(bgColor).Render(""),
	)

	// 外框
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Background(bgColor).
		Width(58).
		Padding(0, 1)

	return boxStyle.Render(inner)
}

// renderSwatches 渲染颜色色板行
func renderSwatches(th ThemeInfo) string {
	bg := lipgloss.Color(th.BgHex)
	swatch := func(hex, label string) string {
		dot := lipgloss.NewStyle().
			Background(lipgloss.Color(hex)).
			Foreground(lipgloss.Color(hex)).
			Render("███")
		lbl := lipgloss.NewStyle().
			Foreground(lipgloss.Color(th.CommentHex)).
			Background(bg).
			Render(" " + label + "  ")
		return lipgloss.NewStyle().Background(bg).Render(dot + lbl)
	}

	row1 := lipgloss.JoinHorizontal(lipgloss.Top,
		swatch(th.PrimaryHex, "主色"),
		swatch(th.AccentHex, "强调"),
		swatch(th.SuccessHex, "成功"),
	)
	row2 := lipgloss.JoinHorizontal(lipgloss.Top,
		swatch(th.WarningHex, "警告"),
		swatch(th.ErrorHex, "错误"),
		swatch(th.CommentHex, "注释"),
	)

	lineStyle := lipgloss.NewStyle().Background(bg).MarginLeft(1)
	return lineStyle.Render(row1) + "\n" + lineStyle.Render(row2)
}

// PickTheme 启动 bubbletea 主题选择器
// 返回选中的主题 Key（"dark"/"light"），取消时返回 ""
func PickTheme(currentKey string) string {
	themes := []ThemeInfo{DarkThemeInfo(), LightThemeInfo()}

	cursor := 0
	for i, th := range themes {
		if th.Key == currentKey {
			cursor = i
			break
		}
	}

	m := themePickerModel{themes: themes, cursor: cursor, selected: -1}

	p := tea.NewProgram(m, tea.WithOutput(os.Stdout))
	result, err := p.Run()
	if err != nil {
		return ""
	}

	final := result.(themePickerModel)
	if final.selected < 0 {
		return ""
	}
	return themes[final.selected].Key
}

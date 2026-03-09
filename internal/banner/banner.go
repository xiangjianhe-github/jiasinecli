// Package banner 提供 CLI 启动时的酷炫 ASCII 艺术字体横幅
package banner

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/xiangjianhe-github/jiasinecli/internal/theme"
	"github.com/xiangjianhe-github/jiasinecli/internal/version"
)

// 色彩快捷访问 — 从当前主题获取
// 这些变量函数避免了调用方到处写 theme.Current().XXX
func t() *theme.Theme { return theme.Current() }

// 导出颜色属性（兼容已有代码的引用方式）
// 通过函数返回当前主题的颜色，实现主题切换后即时生效
var (
	ForceBlackBg  = "" // 已废弃，保留空值兼容
)

// 以下颜色属性通过函数动态获取，确保跟随主题切换
// 为兼容已有代码的常量风格引用，使用包级变量
// 注意: 这些在 init/主题切换后需调用 RefreshColors() 更新
var (
	Reset         string
	Bold          string
	Dim           string
	Italic        string
	Underline     string
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
	BgDarkGray    string
	BgReset       string
	Gray          string
	LightGray     string
	BgDefault     string
)

func init() {
	RefreshColors()
}

// RefreshColors 从当前主题刷新所有颜色变量
// 在主题切换后调用此函数
func RefreshColors() {
	th := t()
	Reset = th.Reset
	Bold = th.Bold
	Dim = th.Dim
	Italic = th.Italic
	Underline = th.Underline
	Cyan = th.Cyan
	Blue = th.Blue
	Green = th.Green
	Yellow = th.Yellow
	Red = th.Red
	Magenta = th.Magenta
	White = th.White
	BrightCyan = th.BrightCyan
	BrightBlue = th.BrightBlue
	BrightGreen = th.BrightGreen
	BrightYellow = th.BrightYellow
	BrightRed = th.BrightRed
	BrightMagenta = th.BrightMagenta
	BgDarkGray = th.BgCode
	BgReset = th.BgReset
	Gray = th.Gray
	LightGray = th.LightGray
	BgDefault = th.BgDefault
}

// asciiArt 是 jiasinecli 的 ANSI 渐变 ASCII 艺术字
// 从左到右：青色 → 蓝色 → 绿色 渐变
func asciiArt() string {
	lines := []string{
		`                   ╦╦╔═╗╔═╗╦╔╗╔╔═╗    ╔═╗╦  ╦`,
		`                   ║║╠═╣╚═╗║║║║║╣  ══ ║  ║  ║`,
		`                 ╚═╝╩╩ ╩╚═╝╩╝╚╝╚═╝    ╚═╝╩═╝╩`,
	}

	colors := []string{
		Bold + BrightCyan,
		Bold + BrightBlue,
		Bold + BrightGreen,
	}

	var sb strings.Builder
	for i, line := range lines {
		sb.WriteString(colors[i%len(colors)])
		sb.WriteString(line)
		sb.WriteString(Reset)
		sb.WriteString("\n")
	}
	return sb.String()
}

// Logo 返回酷炫的 ASCII 艺术字 logo
func Logo() string {
	art := asciiArt()
	box := fmt.Sprintf(
		"%s  ╔══════════════════════════════════════════════════════════════════╗\n"+
			"  ║  %s⚡ Jiasine CLI%s  — Cross-platform multi-language support system  %s║\n"+
			"  ╚══════════════════════════════════════════════════════════════════╝%s",
		Dim+BrightBlue,
		Bold+BrightGreen, Dim+BrightBlue, Dim+BrightBlue,
		Reset,
	)
	return art + box
}

// ShortBanner 返回简短的带颜色横幅
func ShortBanner() string {
	ver := version.Current
	return fmt.Sprintf("%s▸ jiasinecli %s%s  %s%s/%s%s",
		Bold+BrightCyan, ver.String(), Reset,
		Dim, runtime.GOOS, runtime.GOARCH, Reset,
	)
}

// WelcomeScreen 返回完整的欢迎屏幕（双击运行时显示）
func WelcomeScreen() string {
	ver := version.Current
	return fmt.Sprintf(`%s
%s  版本  %s
  平台  %s/%s
  Go    %s
%s
%s  输入命令开始使用，输入 %s/help%s 查看帮助，%s/exit%s 退出%s
`,
		Logo(),
		Dim+White, Bold+BrightGreen+ver.String()+Reset,
		runtime.GOOS, runtime.GOARCH,
		runtime.Version(),
		Reset,
		Dim, BrightCyan, Dim, BrightCyan, Dim, Reset,
	)
}

// Farewell 退出时显示
func Farewell() string {
	return fmt.Sprintf("\n%s  👋 再见！%s\n", Dim+BrightCyan, Reset)
}

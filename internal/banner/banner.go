// Package banner 提供 CLI 启动时的酷炫 ASCII 艺术字体横幅
package banner

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/xiangjianhe-github/jiasinecli/internal/version"
)

// ANSI 色彩常量 — 配色参考 icon.json Lottie 动画
// 青色系: #5ef8db → BrightCyan  蓝色系: #8dadf2 → BrightBlue  青蓝: #31c9e3 → Cyan
const (
	Reset       = "\033[0m"
	Bold        = "\033[1m"
	Dim         = "\033[2m"
	Italic      = "\033[3m"
	Cyan        = "\033[36m"
	Blue        = "\033[34m"
	Green       = "\033[32m"
	Yellow      = "\033[33m"
	White       = "\033[97m"
	BrightCyan  = "\033[96m"
	BrightBlue  = "\033[94m"
	BrightGreen = "\033[92m"
)

// asciiArt 是 jiasinecli 的 ANSI 渐变 ASCII 艺术字
// 从左到右：青色 → 蓝色 → 绿色 渐变
func asciiArt() string {
	// 使用 block/modern 风格字体
	lines := []string{
		`     ╦╦╔═╗╔═╗╦╔╗╔╔═╗   ╔═╗╦  ╦`,
		`     ║║╠═╣╚═╗║║║║║╣    ║  ║  ║`,
		`    ╚╝╩╩ ╩╚═╝╩╝╚╝╚═╝   ╚═╝╩═╝╩`,
	}

	// 渐变色序列
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
		"%s     ╔══════════════════════════════════════════════════════════════════╗\n"+
			"     ║  %s⚡ Jiasine CLI%s  — Cross-platform multi-language support system  %s║\n"+
			"     ╚══════════════════════════════════════════════════════════════════╝%s",
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
%s  版本  %s%s
  平台  %s/%s
  Go    %s
%s
%s  输入命令开始使用，输入 %shelp%s 查看帮助，%sexit%s 退出%s
`,
		Logo(),
		Dim+White, Bold+BrightGreen+ver.String()+Reset, Reset,
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

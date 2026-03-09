package render_test

import (
	"fmt"
	"testing"
	"github.com/xiangjianhe-github/jiasinecli/internal/render"
)

func TestMarkdownRendering(t *testing.T) {
	markdown := `# 测试标题

这是普通文本，包含 **粗体文字** 和 *斜体文字*。

行内代码示例：` + "`const name = \"JiasineCLI\"`" + `

代码块：

` + "```go" + `
func main() {
    fmt.Println("Hello, World!")
}
` + "```" + `

列表示例：
- 第一项
- **粗体项**
- ` + "`代码项`" + `

链接：[JiasineCLI](https://github.com/xiangjianhe-github/jiasinecli)

> 引用文本示例
> 包含 **粗体** 和 *斜体*

---

完成测试！`

	result := render.Markdown(markdown)
	fmt.Println("\n========== Markdown 渲染效果 ==========")
	fmt.Println(result)
	fmt.Println("========== 测试完成 ==========")
}

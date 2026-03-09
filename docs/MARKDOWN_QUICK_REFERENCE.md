# 快速参考：Markdown 渲染示例

## 支持的 Markdown 语法

### 1. 文本格式化
```markdown
**粗体文本**
*斜体文本*
_斜体文本_
```

### 2. 代码
````markdown
行内代码：`const value = 42`

代码块：
```go
func main() {
    fmt.Println("Hello, World!")
}
```
````

### 3. 列表
```markdown
无序列表：
- 项目 1
- 项目 2
  - 嵌套项

有序列表：
1. 首先
2. 然后
3. 最后
```

### 4. 链接
```markdown
[显示文本](https://example.com)
```

### 5. 引用
```markdown
> 这是引用文本
> 可以多行
```

### 6. 标题
```markdown
# 一级标题
## 二级标题
### 三级标题
```

### 7. 分隔线
```markdown
---
或
***
或
___
```

## 颜色参考

### 文字颜色
- **亮青色** (BrightCyan): 标题、强调
- **亮蓝色** (BrightBlue): 关键词、模型名
- **亮绿色** (BrightGreen): 成功信息
- **浅橙色** (CodeText): 行内代码文字

### 背景颜色
- **深灰** (235): 代码块背景
- **稍浅灰** (236): 行内代码背景

### 辅助颜色
- **深灰** (240): 注释、次要信息
- **浅灰** (250): 元数据

## 在 AI 对话中使用

当您与 AI 对话时，AI 的回复会自动使用 Markdown 渲染：

```bash
jiasinecli ai

┃ ❯ 请用 Markdown 格式回复一个示例
```

AI 回复会自动显示为美化的格式，包括：
- 粗体和斜体
- 彩色代码块
- 格式化的列表
- 可点击的链接（在支持的终端中）

## 测试命令

查看渲染效果：
```bash
# 编译
python build.py

# 运行单元测试
go test -v ./internal/render -run TestMarkdownRendering

# 进入 AI 对话模式测试
.\jiasinecli.exe ai
```

package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/xiangjianhe-github/jiasinecli/internal/logger"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// ToolCall AI 模型发起的工具调用
type ToolCall struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// ToolResult 工具执行结果
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// ToolExecutor MCP 工具执行器
// 负责在本地执行 AI 请求的工具调用
type ToolExecutor struct {
	workDir string // 工作目录
}

// NewToolExecutor 创建工具执行器
func NewToolExecutor() *ToolExecutor {
	wd, _ := os.Getwd()
	return &ToolExecutor{workDir: wd}
}

// Execute 执行工具调用，返回结果
func (e *ToolExecutor) Execute(call ToolCall) ToolResult {
	logger.Debug("执行工具调用", "tool", call.Name, "input", call.Input)

	var result string
	var err error

	switch call.Name {
	case "list_directory", "list_dir":
		result, err = e.listDirectory(call.Input)
	case "read_file", "read_code":
		result, err = e.readFile(call.Input)
	case "search_code", "grep_search":
		result, err = e.searchCode(call.Input)
	case "run_command", "execute":
		result, err = e.runCommand(call.Input)
	case "git_log", "git_log_search":
		result, err = e.gitLog(call.Input)
	case "git_diff", "git_diff_search":
		result, err = e.gitDiff(call.Input)
	case "git_blame":
		result, err = e.gitBlame(call.Input)
	case "write_file":
		result, err = e.writeFile(call.Input)
	default:
		err = fmt.Errorf("未知工具: %s", call.Name)
	}

	if err != nil {
		return ToolResult{
			ToolUseID: call.ID,
			Content:   fmt.Sprintf("错误: %s", err.Error()),
			IsError:   true,
		}
	}

	// 截断过大的结果
	if len(result) > 50000 {
		result = result[:50000] + "\n\n... (结果已截断，共 " + fmt.Sprintf("%d", len(result)) + " 字节)"
	}

	return ToolResult{
		ToolUseID: call.ID,
		Content:   result,
	}
}

// listDirectory 列出目录内容
func (e *ToolExecutor) listDirectory(input map[string]interface{}) (string, error) {
	path := e.getStringInput(input, "path", "directory", "dir")
	if path == "" {
		path = e.workDir
	}
	path = e.resolvePath(path)

	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("无法读取目录 %s: %w", path, err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("目录: %s\n\n", path))
	for _, entry := range entries {
		info, _ := entry.Info()
		if entry.IsDir() {
			sb.WriteString(fmt.Sprintf("📁 %-40s <DIR>\n", entry.Name()))
		} else if info != nil {
			size := info.Size()
			sb.WriteString(fmt.Sprintf("📄 %-40s %s\n", entry.Name(), formatSize(size)))
		} else {
			sb.WriteString(fmt.Sprintf("📄 %s\n", entry.Name()))
		}
	}
	sb.WriteString(fmt.Sprintf("\n共 %d 项", len(entries)))
	return sb.String(), nil
}

// readFile 读取文件内容
func (e *ToolExecutor) readFile(input map[string]interface{}) (string, error) {
	path := e.getStringInput(input, "file_path", "path", "file")
	if path == "" {
		return "", fmt.Errorf("缺少 file_path 参数")
	}
	path = e.resolvePath(path)

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("无法读取文件 %s: %w", path, err)
	}

	content := string(data)

	// 处理行号范围
	startLine := e.getIntInput(input, "start_line")
	endLine := e.getIntInput(input, "end_line")
	if startLine > 0 || endLine > 0 {
		lines := strings.Split(content, "\n")
		if startLine < 1 {
			startLine = 1
		}
		if endLine < 1 || endLine > len(lines) {
			endLine = len(lines)
		}
		if startLine > len(lines) {
			startLine = len(lines)
		}
		content = strings.Join(lines[startLine-1:endLine], "\n")
		return fmt.Sprintf("文件: %s (行 %d-%d / 共 %d 行)\n\n%s", path, startLine, endLine, len(lines), content), nil
	}

	return fmt.Sprintf("文件: %s (%d 字节)\n\n%s", path, len(data), content), nil
}

// searchCode 在代码中搜索
func (e *ToolExecutor) searchCode(input map[string]interface{}) (string, error) {
	query := e.getStringInput(input, "query", "pattern", "keyword")
	if query == "" {
		return "", fmt.Errorf("缺少 query 参数")
	}

	searchPath := e.getStringInput(input, "path", "dir", "directory")
	if searchPath == "" {
		searchPath = e.workDir
	}
	searchPath = e.resolvePath(searchPath)

	// 使用系统 grep/findstr 搜索
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("findstr", "/S", "/N", "/I", query, filepath.Join(searchPath, "*"))
	} else {
		cmd = exec.Command("grep", "-rn", "--include=*", "-i", query, searchPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// grep 返回码 1 表示没找到
		if len(output) == 0 {
			return fmt.Sprintf("在 %s 中未找到 \"%s\"", searchPath, query), nil
		}
	}

	result := decodeOutput(output)
	lines := strings.Split(result, "\n")
	if len(lines) > 100 {
		result = strings.Join(lines[:100], "\n") + fmt.Sprintf("\n\n... (共 %d 条结果，显示前 100 条)", len(lines))
	}

	return fmt.Sprintf("搜索 \"%s\" in %s:\n\n%s", query, searchPath, result), nil
}

// runCommand 执行命令
func (e *ToolExecutor) runCommand(input map[string]interface{}) (string, error) {
	command := e.getStringInput(input, "command", "cmd")
	if command == "" {
		return "", fmt.Errorf("缺少 command 参数")
	}

	// 安全检查 — 拒绝危险命令
	dangerousPatterns := []string{"rm -rf /", "format ", "del /s /q", "rmdir /s", "shutdown", "reboot"}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(strings.ToLower(command), pattern) {
			return "", fmt.Errorf("安全限制: 拒绝执行危险命令 \"%s\"", command)
		}
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// 使用 chcp 65001 切换到 UTF-8 代码页，避免中文输出乱码
		cmd = exec.Command("cmd", "/C", "chcp 65001 >nul 2>&1 & "+command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Dir = e.workDir

	output, err := cmd.CombinedOutput()
	result := decodeOutput(output)
	if err != nil {
		result += fmt.Sprintf("\n\n(命令退出码: %s)", err.Error())
	}
	return result, nil
}

// gitLog 查看 git 日志
func (e *ToolExecutor) gitLog(input map[string]interface{}) (string, error) {
	args := []string{"log", "--oneline", "--no-color"}

	limit := e.getIntInput(input, "limit", "n")
	if limit > 0 {
		args = append(args, fmt.Sprintf("-n%d", limit))
	} else {
		args = append(args, "-n20")
	}

	author := e.getStringInput(input, "author")
	if author != "" {
		args = append(args, fmt.Sprintf("--author=%s", author))
	}

	query := e.getStringInput(input, "query", "search", "keyword")
	if query != "" {
		args = append(args, fmt.Sprintf("--grep=%s", query))
	}

	path := e.getStringInput(input, "path", "file")
	if path != "" {
		args = append(args, "--", e.resolvePath(path))
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = e.workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git log 失败: %s\n%s", err, decodeOutput(output))
	}
	return decodeOutput(output), nil
}

// gitDiff 查看 git diff
func (e *ToolExecutor) gitDiff(input map[string]interface{}) (string, error) {
	args := []string{"diff", "--no-color"}

	commit := e.getStringInput(input, "commit", "ref")
	if commit != "" {
		args = append(args, commit)
	}

	path := e.getStringInput(input, "path", "file")
	if path != "" {
		args = append(args, "--", e.resolvePath(path))
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = e.workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git diff 失败: %s\n%s", err, decodeOutput(output))
	}
	return decodeOutput(output), nil
}

// gitBlame 查看 git blame
func (e *ToolExecutor) gitBlame(input map[string]interface{}) (string, error) {
	path := e.getStringInput(input, "path", "file")
	if path == "" {
		return "", fmt.Errorf("缺少 path 参数")
	}

	args := []string{"blame", "--no-color", e.resolvePath(path)}
	cmd := exec.Command("git", args...)
	cmd.Dir = e.workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git blame 失败: %s\n%s", err, decodeOutput(output))
	}
	return decodeOutput(output), nil
}

// writeFile 写入文件
func (e *ToolExecutor) writeFile(input map[string]interface{}) (string, error) {
	path := e.getStringInput(input, "file_path", "path", "file")
	if path == "" {
		return "", fmt.Errorf("缺少 file_path 参数")
	}
	content := e.getStringInput(input, "content", "data", "text")
	if content == "" {
		return "", fmt.Errorf("缺少 content 参数")
	}
	path = e.resolvePath(path)

	// 确保目录存在
	os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("写入失败: %w", err)
	}
	return fmt.Sprintf("已写入文件: %s (%d 字节)", path, len(content)), nil
}

// === 辅助函数 ===

// getStringInput 从 input map 中获取字符串值（支持多个候选 key）
func (e *ToolExecutor) getStringInput(input map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := input[key]; ok {
			if s, ok := val.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// getIntInput 从 input map 中获取整数值
func (e *ToolExecutor) getIntInput(input map[string]interface{}, keys ...string) int {
	for _, key := range keys {
		if val, ok := input[key]; ok {
			switch v := val.(type) {
			case float64:
				return int(v)
			case int:
				return v
			case json.Number:
				n, _ := v.Int64()
				return int(n)
			}
		}
	}
	return 0
}

// resolvePath 解析路径（相对路径 → 绝对路径）
func (e *ToolExecutor) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(e.workDir, path)
}

// formatSize 格式化文件大小
func formatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.1f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.1f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.1f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// MCPToolDefs 生成 Claude API 格式的 tools 定义
// 将 Skill 中的 MCP 工具定义转换为 Claude function-calling 格式
func MCPToolDefs(skills []*Skill) []map[string]interface{} {
	var tools []map[string]interface{}
	seen := make(map[string]bool)

	for _, skill := range skills {
		if skill.MCP == nil {
			continue
		}
		for _, t := range skill.MCP.Tools {
			if seen[t.Name] {
				continue
			}
			seen[t.Name] = true

			tool := map[string]interface{}{
				"name":        t.Name,
				"description": t.Description,
			}
			if t.InputSchema != nil {
				tool["input_schema"] = t.InputSchema
			} else {
				// 默认 schema
				tool["input_schema"] = map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				}
			}
			tools = append(tools, tool)
		}
	}

	// 始终提供基础工具
	builtinTools := []map[string]interface{}{
		{
			"name":        "list_directory",
			"description": "列出指定目录下的文件和文件夹",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "要列出的目录路径 (绝对路径)",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			"name":        "read_file",
			"description": "读取文件内容，可指定行号范围",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "文件路径",
					},
					"start_line": map[string]interface{}{
						"type":        "integer",
						"description": "起始行号 (可选)",
					},
					"end_line": map[string]interface{}{
						"type":        "integer",
						"description": "结束行号 (可选)",
					},
				},
				"required": []string{"file_path"},
			},
		},
		{
			"name":        "search_code",
			"description": "在代码库中搜索关键字",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "搜索关键字或模式",
					},
					"path": map[string]interface{}{
						"type":        "string",
						"description": "搜索路径 (可选)",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "run_command",
			"description": "在终端执行命令并返回输出",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "要执行的命令",
					},
				},
				"required": []string{"command"},
			},
		},
		{
			"name":        "git_log",
			"description": "查看 git 提交历史",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "文件/目录路径 (可选)",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "显示条数限制 (默认 20)",
					},
					"query": map[string]interface{}{
						"type":        "string",
						"description": "搜索提交消息中的关键字 (可选)",
					},
				},
			},
		},
		{
			"name":        "write_file",
			"description": "创建或覆盖写入文件",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "文件路径",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "文件内容",
					},
				},
				"required": []string{"file_path", "content"},
			},
		},
	}

	for _, bt := range builtinTools {
		if !seen[bt["name"].(string)] {
			tools = append(tools, bt)
		}
	}

	return tools
}

// decodeOutput 将命令输出转换为 UTF-8
// 在 Windows 上，cmd.exe 默认使用 GBK (code page 936) 编码输出中文
// 需要先尝试按 GBK 解码，如果失败则保持原样
func decodeOutput(raw []byte) string {
	if runtime.GOOS != "windows" {
		return string(raw)
	}
	// 检查是否已经是有效 UTF-8
	if utf8.Valid(raw) {
		return string(raw)
	}
	// 尝试 GBK → UTF-8
	decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(raw)
	if err != nil {
		return string(raw) // 解码失败，返回原始字符串
	}
	return string(decoded)
}

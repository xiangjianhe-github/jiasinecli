package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xiangjianhe-github/jiasinecli/internal/logger"
)

// Skill AI 技能定义
// Skills 是 Agent 的能力模块，可独立安装、组合使用
// 支持 MCP (Model Context Protocol) 协议
type Skill struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Version     string            `json:"version" yaml:"version"`
	Author      string            `json:"author" yaml:"author"`
	Prompt      string            `json:"prompt" yaml:"prompt"`           // 技能的提示词模板
	Examples    []string          `json:"examples" yaml:"examples"`       // 使用示例
	Parameters  map[string]string `json:"parameters" yaml:"parameters"`   // 参数说明
	Tags        []string          `json:"tags" yaml:"tags"`               // 分类标签
	// MCP 协议支持
	MCP *MCPConfig `json:"mcp,omitempty" yaml:"mcp,omitempty"` // MCP 工具/资源配置
}

// MCPConfig MCP (Model Context Protocol) 配置
// 支持 tools、resources、prompts 三种 MCP 原语
type MCPConfig struct {
	// Tools MCP 工具定义 — AI 可主动调用的函数
	Tools []MCPTool `json:"tools,omitempty" yaml:"tools,omitempty"`
	// Resources MCP 资源定义 — AI 可读取的上下文数据
	Resources []MCPResource `json:"resources,omitempty" yaml:"resources,omitempty"`
	// ServerURL MCP 服务端地址(可选，支持远程 MCP Server)
	ServerURL string `json:"server_url,omitempty" yaml:"server_url,omitempty"`
	// Transport 传输方式: stdio | sse | streamable-http
	Transport string `json:"transport,omitempty" yaml:"transport,omitempty"`
}

// MCPTool MCP 工具定义
type MCPTool struct {
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	InputSchema map[string]interface{} `json:"input_schema,omitempty" yaml:"input_schema,omitempty"` // JSON Schema
	Command     string                 `json:"command,omitempty" yaml:"command,omitempty"`           // 本地执行命令
	Args        []string               `json:"args,omitempty" yaml:"args,omitempty"`                // 命令参数
}

// MCPResource MCP 资源定义
type MCPResource struct {
	URI         string `json:"uri" yaml:"uri"`                   // 资源 URI (如 file:///path 或 custom://resource)
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	MimeType    string `json:"mime_type,omitempty" yaml:"mime_type,omitempty"`
}

// SkillConfig Skills 配置
type SkillConfig struct {
	Dir    string           `yaml:"dir" mapstructure:"dir"`       // Skill 文件目录
	Skills map[string]Skill `yaml:"skills" mapstructure:"skills"` // 内置 Skill
}

// SkillManager Skill 管理器
type SkillManager struct {
	skills   map[string]*Skill
	skillDir string
}

// NewSkillManager 创建 Skill 管理器
func NewSkillManager(cfg SkillConfig) *SkillManager {
	skillDir := cfg.Dir
	if skillDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			skillDir = filepath.Join(home, ".jiasine", "skills")
		}
	}

	mgr := &SkillManager{
		skills:   make(map[string]*Skill),
		skillDir: skillDir,
	}

	// 加载配置中的 Skills
	for name, skill := range cfg.Skills {
		s := skill
		s.Name = name
		mgr.skills[name] = &s
	}

	// 从目录加载
	if mgr.skillDir != "" {
		mgr.loadFromDir(mgr.skillDir)
	}

	// 注册内置 Skills
	mgr.registerBuiltinSkills()

	// 将内置 Skill 写入磁盘（如目录下不存在）
	mgr.ensureDefaults()

	return mgr
}

// List 列出所有 Skills
func (m *SkillManager) List() []SkillInfo {
	var result []SkillInfo
	for name, s := range m.skills {
		result = append(result, SkillInfo{
			Name:        name,
			Description: s.Description,
			Version:     s.Version,
			Author:      s.Author,
			Tags:        s.Tags,
		})
	}
	return result
}

// Get 获取指定 Skill
func (m *SkillManager) Get(name string) (*Skill, error) {
	skill, ok := m.skills[name]
	if !ok {
		return nil, fmt.Errorf("Skill '%s' 不存在 (可用: %s)", name, m.availableNames())
	}
	return skill, nil
}

// Install 安装 Skill（从 JSON 文件或目录）
func (m *SkillManager) Install(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("读取 Skill 路径失败: %w", err)
	}

	// 如果是目录，查找 SKILL.md 或 skill.json
	if info.IsDir() {
		return m.installFromDir(path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取 Skill 文件失败: %w", err)
	}

	var skill Skill
	if err := json.Unmarshal(data, &skill); err != nil {
		return fmt.Errorf("解析 Skill 文件失败: %w", err)
	}

	if skill.Name == "" {
		skill.Name = strings.TrimSuffix(filepath.Base(path), ".json")
	}

	// 复制到 skills 目录
	if m.skillDir != "" {
		destDir := filepath.Join(m.skillDir, skill.Name)
		os.MkdirAll(destDir, 0755)
		destPath := filepath.Join(destDir, "skill.json")
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return fmt.Errorf("安装 Skill 失败: %w", err)
		}
	}

	m.skills[skill.Name] = &skill
	logger.Info("Skill 已安装", "name", skill.Name)
	return nil
}

// installFromDir 从目录安装 Skill (支持 SKILL.md 格式)
func (m *SkillManager) installFromDir(dir string) error {
	name := filepath.Base(dir)

	// 尝试读取 skill.json
	jsonPath := filepath.Join(dir, "skill.json")
	if data, err := os.ReadFile(jsonPath); err == nil {
		var skill Skill
		if err := json.Unmarshal(data, &skill); err == nil {
			if skill.Name == "" {
				skill.Name = name
			}
			// 复制整个目录到 skills
			if m.skillDir != "" {
				destDir := filepath.Join(m.skillDir, skill.Name)
				copyDir(dir, destDir)
			}
			m.skills[skill.Name] = &skill
			logger.Info("Skill 已安装 (从目录)", "name", skill.Name)
			return nil
		}
	}

	// 尝试读取 SKILL.md — 将 Markdown 内容作为 Prompt
	mdPath := filepath.Join(dir, "SKILL.md")
	if data, err := os.ReadFile(mdPath); err == nil {
		skill := &Skill{
			Name:    name,
			Prompt:  string(data),
			Version: "1.0.0",
			Author:  "local",
		}
		// 从第一行提取描述
		lines := strings.SplitN(string(data), "\n", 3)
		for _, line := range lines {
			trimmed := strings.TrimSpace(strings.TrimLeft(line, "#"))
			if trimmed != "" {
				skill.Description = trimmed
				break
			}
		}
		// 安装到 skills 目录
		if m.skillDir != "" {
			destDir := filepath.Join(m.skillDir, name)
			copyDir(dir, destDir)
		}
		m.skills[name] = skill
		logger.Info("Skill 已安装 (从 SKILL.md)", "name", name)
		return nil
	}

	return fmt.Errorf("目录 '%s' 中未找到 skill.json 或 SKILL.md", dir)
}

// Remove 卸载 Skill
func (m *SkillManager) Remove(name string) error {
	if _, ok := m.skills[name]; !ok {
		return fmt.Errorf("Skill '%s' 不存在", name)
	}

	// 删除文件/目录
	if m.skillDir != "" {
		// 新格式: skills/<name>/ 目录
		dirPath := filepath.Join(m.skillDir, name)
		if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
			os.RemoveAll(dirPath)
		}
		// 旧格式: skills/<name>.json
		jsonPath := filepath.Join(m.skillDir, name+".json")
		os.Remove(jsonPath)
	}

	delete(m.skills, name)
	logger.Info("Skill 已卸载", "name", name)
	return nil
}

// BuildContext 为 Agent 构建 Skills 上下文
func (m *SkillManager) BuildContext(skillNames []string) string {
	var parts []string
	for _, name := range skillNames {
		skill, ok := m.skills[name]
		if !ok {
			continue
		}
		part := fmt.Sprintf("### %s\n%s\n\n%s", skill.Name, skill.Description, skill.Prompt)
		if len(skill.Examples) > 0 {
			part += "\n\n示例:\n"
			for _, ex := range skill.Examples {
				part += "- " + ex + "\n"
			}
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "\n\n---\n\n")
}

// SkillInfo Skill 展示信息
type SkillInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
}

// registerBuiltinSkills 注册内置 Skills (MCP 兼容)
func (m *SkillManager) registerBuiltinSkills() {
	builtins := map[string]*Skill{
		"prompt-analysis": {
			Name:        "prompt-analysis",
			Description: "分析 AI 提示模式和接受率",
			Version:     "1.0.0",
			Author:      "jiasine",
			Prompt: `你是一个 AI Prompt 分析专家。你的任务是分析用户与 AI 的交互模式，优化提示词质量。

## 能力
1. **提示词分析** — 评估提示词的清晰度、完整性、有效性
2. **模式识别** — 识别用户的提示模式（指令型、对话型、链式推理等）
3. **接受率分析** — 统计 AI 回复被用户接受/修改/拒绝的比率
4. **优化建议** — 提供具体的提示词改进方案

## 输出格式
- 分析结果使用结构化 Markdown 输出
- 包含评分（1-10）、优缺点、改进建议
- 给出优化后的提示词对比示例

## 分析维度
| 维度 | 说明 |
|------|------|
| 清晰度 | 意图是否明确，无歧义 |
| 完整性 | 上下文/约束/示例是否充分 |
| 可执行性 | AI 是否能直接执行 |
| 效率 | 是否最少 token 达到最优效果 |`,
			Examples: []string{
				"分析我上一段对话的提示词质量",
				"帮我优化这个 prompt: ...",
				"为什么 AI 没有按我的要求回答？",
			},
			Tags: []string{"prompt", "analysis", "optimization", "mcp"},
			MCP: &MCPConfig{
				Tools: []MCPTool{
					{
						Name:        "analyze_prompt",
						Description: "分析提示词质量并给出评分",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"prompt": map[string]interface{}{
									"type":        "string",
									"description": "要分析的提示词文本",
								},
								"context": map[string]interface{}{
									"type":        "string",
									"description": "对话上下文（可选）",
								},
							},
							"required": []string{"prompt"},
						},
					},
					{
						Name:        "suggest_improvement",
						Description: "给出提示词优化建议",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"original_prompt": map[string]interface{}{
									"type":        "string",
									"description": "原始提示词",
								},
								"goal": map[string]interface{}{
									"type":        "string",
									"description": "期望达成的目标",
								},
							},
							"required": []string{"original_prompt"},
						},
					},
				},
			},
		},
		"ask": {
			Name:        "ask",
			Description: "探索代码库 — 向代码作者提问理解代码",
			Version:     "1.0.0",
			Author:      "jiasine",
			Prompt: `你是一个代码探索助手。当用户在探索代码库时，你扮演代码作者的角色，帮助用户理解代码。

## 角色
你就像是编写这段代码的工程师。用户可以问你：
- 这段代码做什么？为什么这样写？
- 这个函数/类/模块的设计意图是什么？
- 为什么选择这种实现方式而不是其他？
- 这里有什么隐含的假设或约束？
- 这个架构决策的权衡是什么？

## 回答准则
1. **具体** — 引用代码中的具体行/函数/变量名
2. **深入** — 不仅解释 what，更要解释 why
3. **坦诚** — 如果代码有缺陷或技术债务，直说
4. **上下文** — 解释代码在整体架构中的位置和作用
5. **示例** — 用类比或场景说明复杂概念

## 交互模式
- 用户给出代码片段 → 你解释其作用和设计意图
- 用户问 "为什么" → 你解释决策背后的原因
- 用户问 "怎么修改" → 你给出修改方案并解释影响`,
			Examples: []string{
				"这个函数为什么要用 goroutine？",
				"为什么这里用 map 而不是 slice？",
				"这个设计模式的优势是什么？",
			},
			Tags: []string{"code", "exploration", "understanding", "mcp"},
			MCP: &MCPConfig{
				Tools: []MCPTool{
					{
						Name:        "read_code",
						Description: "读取指定文件或目录的代码内容",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"path": map[string]interface{}{
									"type":        "string",
									"description": "要读取的文件或目录路径",
								},
								"line_start": map[string]interface{}{
									"type":        "integer",
									"description": "起始行号（可选）",
								},
								"line_end": map[string]interface{}{
									"type":        "integer",
									"description": "结束行号（可选）",
								},
							},
							"required": []string{"path"},
						},
					},
					{
						Name:        "search_code",
						Description: "搜索代码中的符号、函数、类引用",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"query": map[string]interface{}{
									"type":        "string",
									"description": "搜索关键词或正则表达式",
								},
								"scope": map[string]interface{}{
									"type":        "string",
									"description": "搜索范围（文件/目录路径）",
								},
							},
							"required": []string{"query"},
						},
					},
				},
				Resources: []MCPResource{
					{
						URI:         "file://workspace",
						Name:        "workspace",
						Description: "当前工作区的代码文件",
					},
				},
			},
		},
		"git-ai-search": {
			Name:        "git-ai-search",
			Description: "从 Git 历史搜索并恢复 AI 对话上下文",
			Version:     "1.0.0",
			Author:      "jiasine",
			Prompt: `你是一个 Git 历史中 AI 对话上下文搜索专家。你的任务是帮助用户从 Git 提交历史中找到之前的 AI 交互记录和决策上下文。

## 能力
1. **搜索 Git 历史** — 按关键词、日期范围、作者搜索 commit 中的 AI 相关上下文
2. **恢复对话上下文** — 从 commit message、diff、comment 中提取 AI 辅助编码的记录
3. **追溯决策** — 找到某段代码是在什么 AI 对话中产生的
4. **关联分析** — 将多个 commit 中的 AI 交互串联成完整的对话流

## 搜索策略
- 搜索 commit message 中的 AI 相关标记（如 "AI:", "Copilot:", "Generated by" 等）
- 分析 diff 中的大段新增代码（可能是 AI 生成的）
- 检查 .ai-context、.copilot-history 等上下文文件
- 支持时间范围、文件范围、分支范围过滤

## 输出格式
- 搜索结果按相关度排序
- 包含 commit hash、日期、作者、摘要
- 高亮匹配的关键词和上下文片段`,
			Examples: []string{
				"搜索上周关于 API 设计的 AI 对话",
				"这段代码是什么时候用 AI 生成的？",
				"找到上次重构数据库的 AI 上下文",
			},
			Tags: []string{"git", "search", "context", "history", "mcp"},
			MCP: &MCPConfig{
				Tools: []MCPTool{
					{
						Name:        "git_log_search",
						Description: "搜索 Git 提交历史",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"query": map[string]interface{}{
									"type":        "string",
									"description": "搜索关键词",
								},
								"since": map[string]interface{}{
									"type":        "string",
									"description": "起始日期 (YYYY-MM-DD)",
								},
								"until": map[string]interface{}{
									"type":        "string",
									"description": "结束日期 (YYYY-MM-DD)",
								},
								"author": map[string]interface{}{
									"type":        "string",
									"description": "作者过滤",
								},
								"path": map[string]interface{}{
									"type":        "string",
									"description": "文件路径过滤",
								},
							},
							"required": []string{"query"},
						},
						Command: "git",
						Args:    []string{"log", "--all", "--oneline", "--grep"},
					},
					{
						Name:        "git_diff_search",
						Description: "搜索 Git diff 中的代码变更",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"query": map[string]interface{}{
									"type":        "string",
									"description": "搜索的代码片段或关键词",
								},
								"commit": map[string]interface{}{
									"type":        "string",
									"description": "指定 commit hash (可选)",
								},
							},
							"required": []string{"query"},
						},
						Command: "git",
						Args:    []string{"log", "-p", "-S"},
					},
					{
						Name:        "git_blame",
						Description: "查看文件每行的最后修改信息",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"file": map[string]interface{}{
									"type":        "string",
									"description": "文件路径",
								},
								"line_start": map[string]interface{}{
									"type":        "integer",
									"description": "起始行号",
								},
								"line_end": map[string]interface{}{
									"type":        "integer",
									"description": "结束行号",
								},
							},
							"required": []string{"file"},
						},
						Command: "git",
						Args:    []string{"blame"},
					},
				},
			},
		},
	}

	for name, skill := range builtins {
		m.skills[name] = skill
	}
}

// ensureDefaults 将内置 Skill 定义写入磁盘（仅当 skill.json 不存在时）
func (m *SkillManager) ensureDefaults() {
	if m.skillDir == "" {
		return
	}
	os.MkdirAll(m.skillDir, 0755)

	for name, skill := range m.skills {
		dirPath := filepath.Join(m.skillDir, name)
		jsonPath := filepath.Join(dirPath, "skill.json")

		// 如果 skill.json 已存在，跳过
		if _, err := os.Stat(jsonPath); err == nil {
			continue
		}

		os.MkdirAll(dirPath, 0755)
		data, err := json.MarshalIndent(skill, "", "  ")
		if err != nil {
			continue
		}
		os.WriteFile(jsonPath, data, 0644)
		logger.Debug("写入默认 Skill 定义", "name", name, "path", jsonPath)
	}
}

// loadFromDir 从目录加载 Skill 文件
// 支持两种格式：
//  1. <dir>/<name>.json — 单文件 JSON 格式
//  2. <dir>/<name>/skill.json — 子目录格式
//  3. <dir>/<name>/SKILL.md — Markdown 格式
func (m *SkillManager) loadFromDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		logger.Warn("读取 Skill 目录失败", "dir", dir, "error", err)
		return
	}

	for _, entry := range entries {
		// 子目录格式 — 查找 skill.json 或 SKILL.md
		if entry.IsDir() {
			name := entry.Name()
			subDir := filepath.Join(dir, name)

			// 优先 skill.json
			jsonPath := filepath.Join(subDir, "skill.json")
			if data, err := os.ReadFile(jsonPath); err == nil {
				var skill Skill
				if err := json.Unmarshal(data, &skill); err == nil {
					if skill.Name == "" {
						skill.Name = name
					}
					m.skills[name] = &skill
					continue
				}
			}

			// 其次 SKILL.md
			mdPath := filepath.Join(subDir, "SKILL.md")
			if data, err := os.ReadFile(mdPath); err == nil {
				skill := &Skill{
					Name:    name,
					Prompt:  string(data),
					Version: "1.0.0",
					Author:  "local",
				}
				lines := strings.SplitN(string(data), "\n", 3)
				for _, line := range lines {
					trimmed := strings.TrimSpace(strings.TrimLeft(line, "#"))
					if trimmed != "" {
						skill.Description = trimmed
						break
					}
				}
				m.skills[name] = skill
			}
			continue
		}

		// 单文件 JSON 格式
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var skill Skill
		if err := json.Unmarshal(data, &skill); err != nil {
			logger.Warn("解析 Skill 文件失败", "path", path, "error", err)
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".json")
		if skill.Name == "" {
			skill.Name = name
		}
		m.skills[name] = &skill
	}
}

// copyDir 递归复制目录
func copyDir(src, dst string) error {
	os.MkdirAll(dst, 0755)
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *SkillManager) availableNames() string {
	names := make([]string, 0, len(m.skills))
	for name := range m.skills {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

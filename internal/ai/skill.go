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
type Skill struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Version     string            `json:"version" yaml:"version"`
	Author      string            `json:"author" yaml:"author"`
	Prompt      string            `json:"prompt" yaml:"prompt"`           // 技能的提示词模板
	Examples    []string          `json:"examples" yaml:"examples"`       // 使用示例
	Parameters  map[string]string `json:"parameters" yaml:"parameters"`   // 参数说明
	Tags        []string          `json:"tags" yaml:"tags"`               // 分类标签
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
	mgr := &SkillManager{
		skills:   make(map[string]*Skill),
		skillDir: cfg.Dir,
	}

	// 加载配置中的 Skills
	for name, skill := range cfg.Skills {
		s := skill
		s.Name = name
		mgr.skills[name] = &s
	}

	// 从目录加载
	if cfg.Dir != "" {
		mgr.loadFromDir(cfg.Dir)
	}

	// 注册内置 Skills
	mgr.registerBuiltinSkills()

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

// Install 安装 Skill（从 JSON 文件）
func (m *SkillManager) Install(path string) error {
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
		os.MkdirAll(m.skillDir, 0755)
		destPath := filepath.Join(m.skillDir, skill.Name+".json")
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return fmt.Errorf("安装 Skill 失败: %w", err)
		}
	}

	m.skills[skill.Name] = &skill
	logger.Info("Skill 已安装", "name", skill.Name)
	return nil
}

// Remove 卸载 Skill
func (m *SkillManager) Remove(name string) error {
	if _, ok := m.skills[name]; !ok {
		return fmt.Errorf("Skill '%s' 不存在", name)
	}

	// 删除文件
	if m.skillDir != "" {
		path := filepath.Join(m.skillDir, name+".json")
		os.Remove(path)
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

// registerBuiltinSkills 注册内置 Skills
func (m *SkillManager) registerBuiltinSkills() {
	builtins := map[string]*Skill{
		"code-review": {
			Name:        "code-review",
			Description: "代码审查 — 分析代码质量、安全性、性能",
			Version:     "1.0.0",
			Author:      "jiasine",
			Prompt: `当用户提供代码时，请从以下维度进行审查：
1. **正确性** — 逻辑是否正确，是否有 bug
2. **安全性** — 是否有安全漏洞（注入、XSS、缓冲区溢出等）
3. **性能** — 是否有性能瓶颈或不必要的资源消耗
4. **可读性** — 命名、注释、代码结构是否清晰
5. **最佳实践** — 是否遵循语言/框架的最佳实践
给出具体的改进建议和修改后的代码示例。`,
			Examples: []string{"review this Go function", "检查这段 Python 代码的安全性"},
			Tags:     []string{"code", "review", "quality"},
		},
		"sql-expert": {
			Name:        "sql-expert",
			Description: "SQL 专家 — SQL 编写、优化、调试",
			Version:     "1.0.0",
			Author:      "jiasine",
			Prompt: `你是一个 SQL 数据库专家。擅长：
- 复杂查询编写（JOIN、子查询、窗口函数、CTE）
- 查询性能优化（索引策略、执行计划分析）
- 数据库设计（范式、分区、分表策略）
- 支持 MySQL、PostgreSQL、SQLite、SQL Server`,
			Examples: []string{"优化这个慢查询", "设计一个用户权限表"},
			Tags:     []string{"sql", "database", "optimization"},
		},
		"api-designer": {
			Name:        "api-designer",
			Description: "API 设计师 — RESTful/GraphQL/gRPC API 设计",
			Version:     "1.0.0",
			Author:      "jiasine",
			Prompt: `你是一个 API 设计专家。遵循以下原则：
- RESTful 最佳实践：合理使用 HTTP 方法、状态码、URI 设计
- 版本管理策略
- 认证和授权方案
- 分页、过滤、排序规范
- 错误响应格式统一
- OpenAPI/Swagger 文档规范`,
			Examples: []string{"设计一个用户管理 API", "设计 GraphQL schema"},
			Tags:     []string{"api", "rest", "design"},
		},
		"git-helper": {
			Name:        "git-helper",
			Description: "Git 助手 — 分支管理、冲突解决、工作流",
			Version:     "1.0.0",
			Author:      "jiasine",
			Prompt: `你是一个 Git 专家助手。擅长：
- Git 命令使用和最佳实践
- 分支策略（Git Flow、GitHub Flow、Trunk-Based）
- 合并冲突解决
- Commit Message 规范（Conventional Commits）
- Rebase、Cherry-pick、Bisect 等高级操作
- CI/CD 与 Git 集成`,
			Examples: []string{"如何回退到上一个版本", "解决这个合并冲突"},
			Tags:     []string{"git", "vcs", "workflow"},
		},
		"doc-writer": {
			Name:        "doc-writer",
			Description: "文档写手 — 技术文档、README、API 文档生成",
			Version:     "1.0.0",
			Author:      "jiasine",
			Prompt: `你是一个技术文档专家。擅长：
- README.md 编写（项目介绍、安装指南、使用说明）
- API 文档（接口描述、参数说明、响应示例）
- 架构文档（系统设计、流程图描述）
- 用户手册和操作指南
文档应结构清晰、信息完整、语言简洁。`,
			Examples: []string{"为这个项目写一个 README", "为这个 API 生成文档"},
			Tags:     []string{"documentation", "writing", "readme"},
		},
	}

	for name, skill := range builtins {
		if _, exists := m.skills[name]; !exists {
			m.skills[name] = skill
		}
	}
}

// loadFromDir 从目录加载 Skill JSON 文件
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
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
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

func (m *SkillManager) availableNames() string {
	names := make([]string, 0, len(m.skills))
	for name := range m.skills {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

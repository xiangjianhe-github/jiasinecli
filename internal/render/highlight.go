// Package render 提供终端 Markdown 渲染和语法高亮
package render

import (
	"regexp"
	"strings"
)

// ANSI 色彩常量（语法高亮专用）
const (
	// 基础重置：恢复默认前景色 + 强制黑色背景 (修复 PowerShell 蓝色背景问题)
	ansiReset   = "\033[0m\033[40m\033[97m" // 重置 + 黑色背景 + 亮白前景
	ansiBold    = "\033[1m"
	ansiDim     = "\033[2m"
	ansiItalic  = "\033[3m"
	ansiComment = "\033[38;5;114m"  // 注释: 柔和绿色
	ansiKeyword = "\033[38;5;176m"  // 关键字: 紫色
	ansiString  = "\033[38;5;215m"  // 字符串: 浅橙色
	ansiNumber  = "\033[38;5;117m"  // 数字: 亮青色
	ansiType    = "\033[38;5;111m"  // 类型: 亮蓝色
	ansiFunc    = "\033[38;5;228m"  // 函数名: 亮黄色
	ansiOp      = "\033[38;5;250m"  // 操作符: 浅灰色
	ansiBgCode  = "\033[48;5;235m"  // 代码背景: 深灰 (更暗，避免干扰)
	ansiBgReset = "\033[49m\033[40m" // 背景重置 + 强制黑色背景
)

// langDef 语言定义
type langDef struct {
	keywords       []string
	types          []string
	lineComment    string
	blockComStart  string
	blockComEnd    string
	stringChars    []byte // " ' `
	hashComment    bool   // # 开头注释
}

var languages = map[string]*langDef{
	"go": {
		keywords:      []string{"func", "return", "if", "else", "for", "range", "switch", "case", "default", "var", "const", "type", "struct", "interface", "map", "chan", "go", "defer", "select", "break", "continue", "package", "import", "fallthrough", "goto"},
		types:         []string{"string", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64", "bool", "byte", "rune", "error", "nil", "true", "false", "iota", "any"},
		lineComment:   "//",
		blockComStart: "/*",
		blockComEnd:   "*/",
		stringChars:   []byte{'"', '\'', '`'},
	},
	"python": {
		keywords:    []string{"def", "class", "return", "if", "elif", "else", "for", "while", "in", "not", "and", "or", "is", "import", "from", "as", "try", "except", "finally", "raise", "with", "yield", "lambda", "pass", "break", "continue", "global", "nonlocal", "assert", "del", "async", "await"},
		types:       []string{"int", "float", "str", "bool", "list", "dict", "tuple", "set", "None", "True", "False", "self", "cls"},
		hashComment: true,
		stringChars: []byte{'"', '\''},
	},
	"javascript": {
		keywords:      []string{"function", "return", "if", "else", "for", "while", "do", "switch", "case", "default", "break", "continue", "var", "let", "const", "class", "extends", "new", "this", "super", "import", "export", "from", "try", "catch", "finally", "throw", "async", "await", "yield", "typeof", "instanceof", "in", "of", "delete", "void"},
		types:         []string{"undefined", "null", "true", "false", "NaN", "Infinity", "Array", "Object", "String", "Number", "Boolean", "Promise", "Map", "Set", "Symbol", "BigInt"},
		lineComment:   "//",
		blockComStart: "/*",
		blockComEnd:   "*/",
		stringChars:   []byte{'"', '\'', '`'},
	},
	"typescript": {
		keywords:      []string{"function", "return", "if", "else", "for", "while", "do", "switch", "case", "default", "break", "continue", "var", "let", "const", "class", "extends", "implements", "new", "this", "super", "import", "export", "from", "try", "catch", "finally", "throw", "async", "await", "yield", "typeof", "instanceof", "in", "of", "delete", "void", "type", "interface", "enum", "namespace", "abstract", "declare", "readonly", "as", "keyof", "infer"},
		types:         []string{"undefined", "null", "true", "false", "NaN", "Infinity", "string", "number", "boolean", "any", "unknown", "never", "void", "object", "Array", "Promise", "Map", "Set", "Record", "Partial", "Required", "Readonly"},
		lineComment:   "//",
		blockComStart: "/*",
		blockComEnd:   "*/",
		stringChars:   []byte{'"', '\'', '`'},
	},
	"java": {
		keywords:      []string{"public", "private", "protected", "static", "final", "abstract", "class", "interface", "extends", "implements", "new", "return", "if", "else", "for", "while", "do", "switch", "case", "default", "break", "continue", "try", "catch", "finally", "throw", "throws", "import", "package", "void", "this", "super", "synchronized", "volatile", "transient", "native", "enum", "instanceof", "assert"},
		types:         []string{"int", "long", "short", "byte", "float", "double", "char", "boolean", "String", "Object", "Integer", "Long", "Double", "Float", "Boolean", "List", "Map", "Set", "Array", "null", "true", "false"},
		lineComment:   "//",
		blockComStart: "/*",
		blockComEnd:   "*/",
		stringChars:   []byte{'"', '\''},
	},
	"rust": {
		keywords:      []string{"fn", "let", "mut", "const", "static", "if", "else", "match", "for", "while", "loop", "break", "continue", "return", "struct", "enum", "impl", "trait", "pub", "mod", "use", "crate", "self", "super", "as", "in", "ref", "move", "async", "await", "unsafe", "where", "type", "dyn", "extern"},
		types:         []string{"i8", "i16", "i32", "i64", "i128", "u8", "u16", "u32", "u64", "u128", "f32", "f64", "bool", "char", "str", "String", "Vec", "Option", "Result", "Box", "Rc", "Arc", "None", "Some", "Ok", "Err", "true", "false", "Self", "usize", "isize"},
		lineComment:   "//",
		blockComStart: "/*",
		blockComEnd:   "*/",
		stringChars:   []byte{'"'},
	},
	"c": {
		keywords:      []string{"if", "else", "for", "while", "do", "switch", "case", "default", "break", "continue", "return", "goto", "typedef", "struct", "union", "enum", "sizeof", "static", "extern", "register", "volatile", "const", "inline", "restrict", "#include", "#define", "#ifdef", "#ifndef", "#endif", "#if", "#else", "#pragma"},
		types:         []string{"int", "long", "short", "char", "float", "double", "void", "unsigned", "signed", "size_t", "NULL", "true", "false", "bool", "uint8_t", "uint16_t", "uint32_t", "uint64_t", "int8_t", "int16_t", "int32_t", "int64_t"},
		lineComment:   "//",
		blockComStart: "/*",
		blockComEnd:   "*/",
		stringChars:   []byte{'"', '\''},
	},
	"cpp": {
		keywords:      []string{"if", "else", "for", "while", "do", "switch", "case", "default", "break", "continue", "return", "goto", "typedef", "struct", "union", "enum", "sizeof", "static", "extern", "register", "volatile", "const", "inline", "class", "public", "private", "protected", "virtual", "override", "final", "new", "delete", "try", "catch", "throw", "namespace", "using", "template", "typename", "auto", "decltype", "constexpr", "noexcept", "#include", "#define"},
		types:         []string{"int", "long", "short", "char", "float", "double", "void", "unsigned", "signed", "bool", "string", "vector", "map", "set", "pair", "shared_ptr", "unique_ptr", "nullptr", "true", "false", "size_t", "std"},
		lineComment:   "//",
		blockComStart: "/*",
		blockComEnd:   "*/",
		stringChars:   []byte{'"', '\''},
	},
	"bash": {
		keywords:    []string{"if", "then", "else", "elif", "fi", "for", "while", "do", "done", "case", "esac", "in", "function", "return", "exit", "local", "export", "source", "alias", "unalias", "echo", "printf", "read", "set", "unset", "shift", "trap", "eval", "exec", "test"},
		types:       []string{"true", "false", "null"},
		hashComment: true,
		stringChars: []byte{'"', '\''},
	},
	"json": {
		stringChars: []byte{'"'},
		types:       []string{"true", "false", "null"},
	},
	"yaml": {
		hashComment: true,
		types:       []string{"true", "false", "null", "yes", "no", "on", "off"},
	},
	"sql": {
		keywords:    []string{"SELECT", "FROM", "WHERE", "INSERT", "INTO", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER", "TABLE", "INDEX", "VIEW", "JOIN", "LEFT", "RIGHT", "INNER", "OUTER", "ON", "AND", "OR", "NOT", "IN", "BETWEEN", "LIKE", "ORDER", "BY", "GROUP", "HAVING", "LIMIT", "OFFSET", "AS", "SET", "VALUES", "DISTINCT", "COUNT", "SUM", "AVG", "MAX", "MIN", "UNION", "ALL", "EXISTS", "NULL", "IS", "CASE", "WHEN", "THEN", "ELSE", "END", "PRIMARY", "KEY", "FOREIGN", "REFERENCES", "CONSTRAINT", "DEFAULT", "CHECK", "UNIQUE", "ASC", "DESC", "BEGIN", "COMMIT", "ROLLBACK", "TRANSACTION", "CASCADE", "GRANT", "REVOKE", "TRIGGER", "PROCEDURE", "FUNCTION", "DECLARE", "CURSOR", "FETCH", "EXEC", "EXECUTE", "select", "from", "where", "insert", "into", "update", "delete", "create", "drop", "alter", "table", "index", "view", "join", "left", "right", "inner", "outer", "on", "and", "or", "not", "in", "between", "like", "order", "by", "group", "having", "limit", "offset", "as", "set", "values", "distinct", "count", "sum", "avg", "max", "min", "union", "all", "exists", "null", "is", "case", "when", "then", "else", "end", "primary", "key", "foreign", "references", "constraint", "default", "check", "unique", "asc", "desc", "begin", "commit", "rollback", "transaction"},
		types:       []string{"INT", "INTEGER", "VARCHAR", "TEXT", "BOOLEAN", "DATE", "TIMESTAMP", "FLOAT", "DOUBLE", "DECIMAL", "BLOB", "CHAR", "BIGINT", "SMALLINT", "SERIAL", "TRUE", "FALSE", "int", "integer", "varchar", "text", "boolean", "date", "timestamp", "float", "double", "decimal", "blob", "char", "bigint", "smallint", "serial", "true", "false"},
		lineComment: "--",
		stringChars: []byte{'\'', '"'},
	},
	"csharp": {
		keywords:      []string{"public", "private", "protected", "internal", "static", "readonly", "const", "class", "struct", "interface", "enum", "abstract", "sealed", "virtual", "override", "new", "return", "if", "else", "for", "foreach", "while", "do", "switch", "case", "default", "break", "continue", "try", "catch", "finally", "throw", "using", "namespace", "void", "this", "base", "var", "async", "await", "yield", "ref", "out", "in", "params", "is", "as", "typeof", "lock", "delegate", "event", "get", "set", "value", "where", "select", "from", "orderby", "group", "into", "join", "let", "on", "equals", "partial", "record"},
		types:         []string{"int", "long", "short", "byte", "float", "double", "decimal", "char", "bool", "string", "object", "dynamic", "void", "null", "true", "false", "String", "Int32", "Int64", "Boolean", "Double", "List", "Dictionary", "Task", "IEnumerable", "Action", "Func"},
		lineComment:   "//",
		blockComStart: "/*",
		blockComEnd:   "*/",
		stringChars:   []byte{'"', '\''},
	},
	"swift": {
		keywords:      []string{"func", "let", "var", "return", "if", "else", "guard", "for", "while", "repeat", "switch", "case", "default", "break", "continue", "class", "struct", "enum", "protocol", "extension", "import", "init", "deinit", "self", "super", "nil", "throw", "throws", "try", "catch", "as", "is", "in", "where", "typealias", "associatedtype", "static", "override", "final", "public", "private", "internal", "fileprivate", "open", "lazy", "weak", "unowned", "mutating", "async", "await", "@objc", "inout", "defer"},
		types:         []string{"Int", "String", "Bool", "Double", "Float", "Array", "Dictionary", "Set", "Optional", "Any", "AnyObject", "Void", "Character", "Error", "Result", "true", "false", "nil", "Self", "Type", "UInt", "Int8", "Int16", "Int32", "Int64"},
		lineComment:   "//",
		blockComStart: "/*",
		blockComEnd:   "*/",
		stringChars:   []byte{'"'},
	},
}

// 语言别名映射
var langAliases = map[string]string{
	"golang":     "go",
	"py":         "python",
	"js":         "javascript",
	"ts":         "typescript",
	"jsx":        "javascript",
	"tsx":        "typescript",
	"sh":         "bash",
	"shell":      "bash",
	"zsh":        "bash",
	"c++":        "cpp",
	"cs":         "csharp",
	"c#":         "csharp",
	"objc":       "c",
	"objective-c": "c",
	"rb":         "python", // Ruby 语法类似 Python
	"ruby":       "python",
	"yml":        "yaml",
	"jsonc":      "json",
	"mysql":      "sql",
	"postgresql": "sql",
	"postgres":   "sql",
	"sqlite":     "sql",
	"plsql":      "sql",
}

// HighlightCode 对代码进行语法高亮着色
// lang: 语言标识（如 go, python, javascript）
// code: 源代码文本
// 返回带 ANSI 转义码的着色代码
func HighlightCode(lang, code string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))

	// 别名解析
	if alias, ok := langAliases[lang]; ok {
		lang = alias
	}

	def, ok := languages[lang]
	if !ok {
		// 未识别的语言 — 仅做基本着色（字符串 + 数字）
		return highlightBasic(code)
	}

	return highlightWithDef(code, def)
}

// highlightWithDef 使用语言定义进行高亮
func highlightWithDef(code string, def *langDef) string {
	lines := strings.Split(code, "\n")
	var result []string
	inBlockComment := false

	for _, line := range lines {
		if inBlockComment {
			// 在块注释中
			if def.blockComEnd != "" {
				if idx := strings.Index(line, def.blockComEnd); idx >= 0 {
					// 块注释结束
					commentPart := line[:idx+len(def.blockComEnd)]
					rest := line[idx+len(def.blockComEnd):]
					result = append(result, ansiComment+commentPart+ansiReset+highlightLine(rest, def))
					inBlockComment = false
					continue
				}
			}
			result = append(result, ansiComment+line+ansiReset)
			continue
		}

		highlighted := highlightLine(line, def)
		// Check if this line starts a block comment that doesn't end
		if def.blockComStart != "" {
			stripped := strings.TrimSpace(line)
			if strings.Contains(stripped, def.blockComStart) && !strings.Contains(stripped, def.blockComEnd) {
				inBlockComment = true
				result = append(result, ansiComment+line+ansiReset)
				continue
			}
		}
		result = append(result, highlighted)
	}

	return strings.Join(result, "\n")
}

// highlightLine 高亮单行代码
func highlightLine(line string, def *langDef) string {
	// 空行直接返回
	if strings.TrimSpace(line) == "" {
		return line
	}

	// 检查行注释
	trimmed := strings.TrimSpace(line)
	if def.lineComment != "" && strings.HasPrefix(trimmed, def.lineComment) {
		return ansiComment + line + ansiReset
	}
	if def.hashComment && strings.HasPrefix(trimmed, "#") {
		return ansiComment + line + ansiReset
	}

	// 检查块注释在同一行
	if def.blockComStart != "" && def.blockComEnd != "" {
		if strings.Contains(trimmed, def.blockComStart) && strings.Contains(trimmed, def.blockComEnd) {
			return ansiComment + line + ansiReset
		}
	}

	// 逐 token 高亮
	return highlightTokens(line, def)
}

// numberRegex 匹配数字常量
var numberRegex = regexp.MustCompile(`\b(0[xX][0-9a-fA-F]+|0[oO][0-7]+|0[bB][01]+|\d+\.?\d*([eE][+-]?\d+)?)\b`)

// highlightTokens 对一行代码进行 token 级别高亮
func highlightTokens(line string, def *langDef) string {
	var result strings.Builder
	runes := []rune(line)
	i := 0

	for i < len(runes) {
		ch := runes[i]

		// 字符串
		if def.stringChars != nil && isStringChar(byte(ch), def.stringChars) {
			str := extractString(runes, i)
			result.WriteString(ansiString + str + ansiReset)
			i += len([]rune(str))
			continue
		}

		// 行注释（在行内出现）
		if def.lineComment != "" && i+len(def.lineComment) <= len(runes) {
			if string(runes[i:i+len([]rune(def.lineComment))]) == def.lineComment {
				result.WriteString(ansiComment + string(runes[i:]) + ansiReset)
				return result.String()
			}
		}
		if def.hashComment && ch == '#' {
			result.WriteString(ansiComment + string(runes[i:]) + ansiReset)
			return result.String()
		}

		// 标识符 / 关键字
		if isIdentStart(ch) {
			word := extractIdent(runes, i)
			if isKeyword(word, def.keywords) {
				result.WriteString(ansiKeyword + ansiBold + word + ansiReset)
			} else if isType(word, def.types) {
				result.WriteString(ansiType + word + ansiReset)
			} else {
				// 检查是否后跟 ( → 函数调用
				nextIdx := i + len([]rune(word))
				if nextIdx < len(runes) && runes[nextIdx] == '(' {
					result.WriteString(ansiFunc + word + ansiReset)
				} else {
					result.WriteString(word)
				}
			}
			i += len([]rune(word))
			continue
		}

		// 数字
		if ch >= '0' && ch <= '9' {
			num := extractNumber(runes, i)
			result.WriteString(ansiNumber + num + ansiReset)
			i += len([]rune(num))
			continue
		}

		// 其他字符原样输出
		result.WriteRune(ch)
		i++
	}

	return result.String()
}

// highlightBasic 基本高亮（无语言定义时）
func highlightBasic(code string) string {
	lines := strings.Split(code, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 简单的行注释检测
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "--") {
			result = append(result, ansiComment+line+ansiReset)
			continue
		}

		// 数字高亮
		highlighted := numberRegex.ReplaceAllStringFunc(line, func(s string) string {
			return ansiNumber + s + ansiReset
		})

		// 字符串高亮
		highlighted = highlightBasicStrings(highlighted)

		result = append(result, highlighted)
	}
	return strings.Join(result, "\n")
}

// highlightBasicStrings 简单字符串高亮
func highlightBasicStrings(line string) string {
	var result strings.Builder
	runes := []rune(line)
	i := 0
	for i < len(runes) {
		if runes[i] == '"' || runes[i] == '\'' {
			str := extractString(runes, i)
			result.WriteString(ansiString + str + ansiReset)
			i += len([]rune(str))
		} else {
			result.WriteRune(runes[i])
			i++
		}
	}
	return result.String()
}

func isStringChar(ch byte, chars []byte) bool {
	for _, c := range chars {
		if ch == c {
			return true
		}
	}
	return false
}

func extractString(runes []rune, start int) string {
	quote := runes[start]
	i := start + 1

	// 反引号字符串可以多行，但我们只处理单行
	for i < len(runes) {
		if runes[i] == '\\' && i+1 < len(runes) {
			i += 2 // 跳过转义
			continue
		}
		if runes[i] == quote {
			return string(runes[start : i+1])
		}
		i++
	}
	// 未闭合，返回到行尾
	return string(runes[start:])
}

func isIdentStart(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' || ch == '@' || ch == '#'
}

func isIdentChar(ch rune) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9')
}

func extractIdent(runes []rune, start int) string {
	i := start
	for i < len(runes) && isIdentChar(runes[i]) {
		i++
	}
	return string(runes[start:i])
}

func extractNumber(runes []rune, start int) string {
	i := start
	// 0x, 0b, 0o 前缀
	if i+1 < len(runes) && runes[i] == '0' {
		next := runes[i+1]
		if next == 'x' || next == 'X' || next == 'b' || next == 'B' || next == 'o' || next == 'O' {
			i += 2
			for i < len(runes) && isHexChar(runes[i]) {
				i++
			}
			return string(runes[start:i])
		}
	}
	// 十进制
	for i < len(runes) && (runes[i] >= '0' && runes[i] <= '9') {
		i++
	}
	// 小数点
	if i < len(runes) && runes[i] == '.' {
		i++
		for i < len(runes) && (runes[i] >= '0' && runes[i] <= '9') {
			i++
		}
	}
	// 科学计数法
	if i < len(runes) && (runes[i] == 'e' || runes[i] == 'E') {
		i++
		if i < len(runes) && (runes[i] == '+' || runes[i] == '-') {
			i++
		}
		for i < len(runes) && (runes[i] >= '0' && runes[i] <= '9') {
			i++
		}
	}
	return string(runes[start:i])
}

func isHexChar(ch rune) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func isKeyword(word string, keywords []string) bool {
	for _, kw := range keywords {
		if word == kw {
			return true
		}
	}
	return false
}

func isType(word string, types []string) bool {
	for _, t := range types {
		if word == t {
			return true
		}
	}
	return false
}

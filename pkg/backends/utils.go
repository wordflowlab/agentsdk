package backends

import (
	"fmt"
	"strings"
)

// 常量定义
const (
	// EmptyContentWarning 空文件内容警告
	EmptyContentWarning = "System reminder: File exists but has empty contents"

	// MaxLineLength 单行最大长度,超过此长度将分块显示
	MaxLineLength = 10000

	// LineNumberWidth 行号宽度(用于对齐)
	LineNumberWidth = 6

	// ToolResultTokenLimit 工具结果 token 限制(与驱逐阈值一致)
	ToolResultTokenLimit = 20000

	// TruncationGuidance 截断提示信息
	TruncationGuidance = "... [results truncated, try being more specific with your parameters]"
)

// SanitizeToolCallID 清理 tool_call_id,防止路径遍历攻击
//
// 将危险字符 (., /, \) 替换为下划线
//
// 参考: deepagents/backends/utils.py:29-35
func SanitizeToolCallID(toolCallID string) string {
	sanitized := strings.ReplaceAll(toolCallID, ".", "_")
	sanitized = strings.ReplaceAll(sanitized, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")
	return sanitized
}

// FormatContentWithLineNumbers 格式化文件内容,添加行号(cat -n 风格)
//
// 超长行会自动分块,使用延续标记(如 5.1, 5.2)
//
// 参数:
//   - content: 文件内容(字符串或行数组)
//   - startLine: 起始行号(默认 1)
//
// 返回: 带行号和延续标记的格式化内容
//
// 参考: deepagents/backends/utils.py:38-81
func FormatContentWithLineNumbers(content string, startLine int) string {
	lines := strings.Split(content, "\n")
	// 去掉末尾的空行(如果有)
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	resultLines := make([]string, 0, len(lines))

	for i, line := range lines {
		lineNum := i + startLine

		if len(line) <= MaxLineLength {
			// 正常长度的行
			resultLines = append(resultLines, fmt.Sprintf("%*d\t%s", LineNumberWidth, lineNum, line))
		} else {
			// 超长行:分块处理
			numChunks := (len(line) + MaxLineLength - 1) / MaxLineLength
			for chunkIdx := 0; chunkIdx < numChunks; chunkIdx++ {
				start := chunkIdx * MaxLineLength
				end := start + MaxLineLength
				if end > len(line) {
					end = len(line)
				}
				chunk := line[start:end]

				if chunkIdx == 0 {
					// 第一个分块:使用正常行号
					resultLines = append(resultLines, fmt.Sprintf("%*d\t%s", LineNumberWidth, lineNum, chunk))
				} else {
					// 延续分块:使用小数标记 (如 5.1, 5.2)
					continuationMarker := fmt.Sprintf("%d.%d", lineNum, chunkIdx)
					resultLines = append(resultLines, fmt.Sprintf("%*s\t%s", LineNumberWidth, continuationMarker, chunk))
				}
			}
		}
	}

	return strings.Join(resultLines, "\n")
}

// CheckEmptyContent 检查内容是否为空,返回警告信息
//
// 参数:
//   - content: 要检查的内容
//
// 返回: 如果为空返回警告信息,否则返回空字符串
//
// 参考: deepagents/backends/utils.py:84-95
func CheckEmptyContent(content string) string {
	if content == "" || strings.TrimSpace(content) == "" {
		return EmptyContentWarning
	}
	return ""
}

// TruncateIfTooLong 如果内容过长则截断,并添加提示信息
//
// 参数:
//   - result: 要检查的结果字符串
//   - limit: token 限制(默认使用 ToolResultTokenLimit)
//
// 返回: 可能被截断的结果(包含截断提示)
//
// 参考: deepagents/backends/utils.py 的截断逻辑
func TruncateIfTooLong(result string, limit int) string {
	if limit <= 0 {
		limit = ToolResultTokenLimit
	}

	// 简单估算: 1 token ≈ 4 chars
	charLimit := limit * 4

	if len(result) > charLimit {
		return result[:charLimit] + "\n" + TruncationGuidance
	}

	return result
}

// ExtractPreview 提取内容的前几行作为预览
//
// 参数:
//   - content: 完整内容
//   - numLines: 提取的行数(默认 10)
//
// 返回: 带行号的预览内容
func ExtractPreview(content string, numLines int) string {
	if numLines <= 0 {
		numLines = 10
	}

	lines := strings.Split(content, "\n")
	if len(lines) > numLines {
		lines = lines[:numLines]
	}

	return FormatContentWithLineNumbers(strings.Join(lines, "\n"), 1)
}

// NormalizePath 规范化路径
//
// 确保路径:
// 1. 以 / 开头
// 2. 不包含连续的 //
// 3. 不以 / 结尾(除非是根路径)
func NormalizePath(path string) string {
	// 去掉前后空格
	path = strings.TrimSpace(path)

	// 确保以 / 开头
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// 去掉连续的 //
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}

	// 去掉尾部的 / (除非是根路径)
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

	return path
}

// JoinPath 安全地拼接路径
//
// 确保结果路径规范化
func JoinPath(base, rel string) string {
	base = NormalizePath(base)
	rel = strings.TrimPrefix(rel, "/")

	if rel == "" {
		return base
	}

	if base == "/" {
		return "/" + rel
	}

	return NormalizePath(base + "/" + rel)
}

// FormatFileSize 格式化文件大小为人类可读格式
//
// 参数:
//   - bytes: 文件大小(字节)
//
// 返回: 格式化后的字符串(如 "1.5 KB", "2.3 MB")
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// IsTextFile 简单判断是否为文本文件(基于扩展名)
//
// 参数:
//   - path: 文件路径
//
// 返回: true 如果是常见的文本文件扩展名
func IsTextFile(path string) bool {
	textExtensions := map[string]bool{
		".txt": true, ".md": true, ".go": true, ".py": true,
		".js": true, ".ts": true, ".jsx": true, ".tsx": true,
		".json": true, ".yaml": true, ".yml": true, ".toml": true,
		".xml": true, ".html": true, ".css": true, ".scss": true,
		".sh": true, ".bash": true, ".zsh": true,
		".c": true, ".cpp": true, ".h": true, ".hpp": true,
		".java": true, ".kt": true, ".rs": true,
		".sql": true, ".graphql": true, ".proto": true,
		".dockerfile": true, ".gitignore": true,
	}

	for ext := range textExtensions {
		if strings.HasSuffix(strings.ToLower(path), ext) {
			return true
		}
	}

	return false
}

// -------- Grep 结构化助手函数 (Phase 6B-2) --------

// FormatGrepResults 格式化 Grep 匹配结果
//
// 支持三种输出模式:
//   - "files_with_matches": 只返回包含匹配的文件路径列表
//   - "content": 返回完整的匹配内容(文件:行号:内容)
//   - "count": 返回每个文件的匹配数量统计
//
// 参数:
//   - matches: Grep 匹配结果列表
//   - mode: 输出模式
//
// 返回: 格式化后的字符串
func FormatGrepResults(matches []GrepMatch, mode string) string {
	if len(matches) == 0 {
		return "(no matches)"
	}

	switch mode {
	case "files_with_matches":
		// 去重文件路径
		fileSet := make(map[string]bool)
		for _, m := range matches {
			fileSet[m.Path] = true
		}

		var files []string
		for file := range fileSet {
			files = append(files, file)
		}

		return strings.Join(files, "\n")

	case "count":
		// 统计每个文件的匹配数
		countMap := make(map[string]int)
		for _, m := range matches {
			countMap[m.Path]++
		}

		var result strings.Builder
		for file, count := range countMap {
			result.WriteString(fmt.Sprintf("%s: %d matches\n", file, count))
		}

		return strings.TrimSuffix(result.String(), "\n")

	default: // "content"
		var result strings.Builder
		for _, m := range matches {
			result.WriteString(fmt.Sprintf("%s:%d:%s\n", m.Path, m.LineNumber, m.Line))
		}

		return strings.TrimSuffix(result.String(), "\n")
	}
}

// GroupGrepMatches 将匹配结果按文件分组
//
// 参数:
//   - matches: Grep 匹配结果列表
//
// 返回: 按文件分组的匹配结果 map[文件路径][]匹配行
func GroupGrepMatches(matches []GrepMatch) map[string][]GrepMatch {
	grouped := make(map[string][]GrepMatch)
	for _, m := range matches {
		grouped[m.Path] = append(grouped[m.Path], m)
	}
	return grouped
}

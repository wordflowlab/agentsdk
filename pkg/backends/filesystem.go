package backends

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/wordflowlab/agentsdk/pkg/sandbox"
)

// FilesystemBackend 真实文件系统后端
// 直接操作磁盘文件,通过 Sandbox 接口进行安全访问
// 适用于需要与实际工作空间文件交互的场景
type FilesystemBackend struct {
	fs sandbox.SandboxFS
}

// NewFilesystemBackend 创建 FilesystemBackend 实例
func NewFilesystemBackend(fs sandbox.SandboxFS) *FilesystemBackend {
	return &FilesystemBackend{
		fs: fs,
	}
}

// ListInfo 实现 BackendProtocol.ListInfo
func (b *FilesystemBackend) ListInfo(ctx context.Context, path string) ([]FileInfo, error) {
	if path == "" {
		path = "."
	}

	// 解析为绝对路径
	absPath := b.fs.Resolve(path)

	// 检查路径是否在沙箱内
	if !b.fs.IsInside(absPath) {
		return nil, fmt.Errorf("path outside sandbox: %s", path)
	}

	// 读取目录
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	var results []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		results = append(results, FileInfo{
			Path:         filepath.Join(path, entry.Name()),
			IsDirectory:  entry.IsDir(),
			Size:         info.Size(),
			ModifiedTime: info.ModTime(),
			CreatedTime:  info.ModTime(), // Go 的 FileInfo 不提供创建时间
		})
	}

	return results, nil
}

// Read 实现 BackendProtocol.Read
func (b *FilesystemBackend) Read(ctx context.Context, path string, offset, limit int) (string, error) {
	// 使用 SandboxFS 读取文件
	content, err := b.fs.Read(ctx, path)
	if err != nil {
		return "", err
	}

	// 分割成行
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// 处理 offset
	if offset < 0 {
		offset = 0
	}
	if offset >= totalLines {
		return "", nil
	}

	// 处理 limit
	endLine := totalLines
	if limit > 0 {
		endLine = offset + limit
		if endLine > totalLines {
			endLine = totalLines
		}
	}

	selectedLines := lines[offset:endLine]
	return strings.Join(selectedLines, "\n"), nil
}

// Write 实现 BackendProtocol.Write
func (b *FilesystemBackend) Write(ctx context.Context, path, content string) (*WriteResult, error) {
	// 使用 SandboxFS 写入文件
	if err := b.fs.Write(ctx, path, content); err != nil {
		return nil, err
	}

	return &WriteResult{
		Error:        "", // 空字符串表示成功
		Path:         path,
		BytesWritten: int64(len(content)),
		FilesUpdate:  nil, // FilesystemBackend 不需要返回状态更新
	}, nil
}

// Edit 实现 BackendProtocol.Edit
func (b *FilesystemBackend) Edit(ctx context.Context, path, oldStr, newStr string, replaceAll bool) (*EditResult, error) {
	// 读取文件内容
	content, err := b.fs.Read(ctx, path)
	if err != nil {
		return &EditResult{
			Error: fmt.Sprintf("failed to read file: %v", err),
			Path:  path,
		}, nil
	}

	// 执行替换
	var newContent string
	var count int

	if replaceAll {
		count = strings.Count(content, oldStr)
		newContent = strings.ReplaceAll(content, oldStr, newStr)
	} else {
		if strings.Contains(content, oldStr) {
			newContent = strings.Replace(content, oldStr, newStr, 1)
			count = 1
		} else {
			newContent = content
			count = 0
		}
	}

	// 写回文件
	if count > 0 {
		if err := b.fs.Write(ctx, path, newContent); err != nil {
			return &EditResult{
				Error: fmt.Sprintf("failed to write file: %v", err),
				Path:  path,
			}, nil
		}
	}

	return &EditResult{
		Error:            "", // 空字符串表示成功
		Path:             path,
		ReplacementsMade: count,
	}, nil
}

// GrepRaw 实现 BackendProtocol.GrepRaw
func (b *FilesystemBackend) GrepRaw(ctx context.Context, pattern, path, glob string) ([]GrepMatch, error) {
	// 编译正则表达式
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	if path == "" {
		path = "."
	}

	// 使用 Glob 查找匹配的文件
	var searchPattern string
	if glob != "" {
		searchPattern = filepath.Join(path, "**", glob)
	} else {
		searchPattern = filepath.Join(path, "**", "*")
	}

	files, err := b.fs.Glob(ctx, searchPattern, &sandbox.GlobOptions{
		Dot: true,
	})
	if err != nil {
		return nil, fmt.Errorf("glob files: %w", err)
	}

	var matches []GrepMatch

	for _, filePath := range files {
		// 检查是否是文件
		stat, err := b.fs.Stat(ctx, filePath)
		if err != nil || stat.IsDir {
			continue
		}

		// 读取文件内容
		content, err := b.fs.Read(ctx, filePath)
		if err != nil {
			continue
		}

		// 搜索每一行
		lines := strings.Split(content, "\n")
		for lineNum, line := range lines {
			if re.MatchString(line) {
				matches = append(matches, GrepMatch{
					Path:       filePath,
					LineNumber: lineNum + 1, // 1-based
					Line:       line,
					Match:      re.FindString(line),
				})
			}
		}
	}

	return matches, nil
}

// GlobInfo 实现 BackendProtocol.GlobInfo
func (b *FilesystemBackend) GlobInfo(ctx context.Context, pattern, path string) ([]FileInfo, error) {
	if path == "" {
		path = "."
	}

	// 使用 SandboxFS 的 Glob 功能
	searchPattern := filepath.Join(path, pattern)
	files, err := b.fs.Glob(ctx, searchPattern, &sandbox.GlobOptions{
		Dot: true,
	})
	if err != nil {
		return nil, fmt.Errorf("glob files: %w", err)
	}

	var results []FileInfo
	for _, filePath := range files {
		stat, err := b.fs.Stat(ctx, filePath)
		if err != nil {
			continue
		}

		results = append(results, FileInfo{
			Path:         filePath,
			IsDirectory:  stat.IsDir,
			Size:         stat.Size,
			ModifiedTime: stat.ModTime,
			CreatedTime:  stat.ModTime,
		})
	}

	return results, nil
}

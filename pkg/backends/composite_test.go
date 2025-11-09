package backends

import (
	"context"
	"strings"
	"testing"
)

// TestCompositeBackend_PrefixStripping 测试前缀剥离功能
func TestCompositeBackend_PrefixStripping(t *testing.T) {
	ctx := context.Background()

	// 创建三个后端
	defaultBackend := NewStateBackend()
	memoryBackend := NewStateBackend()
	workspaceBackend := NewStateBackend()

	// 创建 CompositeBackend
	composite := NewCompositeBackend(defaultBackend, []RouteConfig{
		{Prefix: "/memories/", Backend: memoryBackend},
		{Prefix: "/workspace/", Backend: workspaceBackend},
	})

	// 测试 1: Write 操作应该剥离前缀
	t.Run("Write strips prefix", func(t *testing.T) {
		_, err := composite.Write(ctx, "/memories/test.txt", "memory content")
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		// 验证内容在 memoryBackend 中,路径是剥离后的
		content, err := memoryBackend.Read(ctx, "/test.txt", 0, 0)
		if err != nil {
			t.Fatalf("Read from memoryBackend failed: %v", err)
		}

		if content != "memory content" {
			t.Errorf("Expected 'memory content', got '%s'", content)
		}

		// 确认不在默认后端
		_, err = defaultBackend.Read(ctx, "/memories/test.txt", 0, 0)
		if err == nil {
			t.Error("Content should not be in defaultBackend with full path")
		}
	})

	// 测试 2: Read 操作应该使用剥离后的路径
	t.Run("Read strips prefix", func(t *testing.T) {
		// 直接在 workspaceBackend 写入剥离后的路径
		_, err := workspaceBackend.Write(ctx, "/file.go", "package main")
		if err != nil {
			t.Fatalf("Write to workspaceBackend failed: %v", err)
		}

		// 通过 composite 读取,使用完整路径
		content, err := composite.Read(ctx, "/workspace/file.go", 0, 0)
		if err != nil {
			t.Fatalf("Read from composite failed: %v", err)
		}

		if content != "package main" {
			t.Errorf("Expected 'package main', got '%s'", content)
		}
	})

	// 测试 3: Edit 操作应该剥离前缀
	t.Run("Edit strips prefix", func(t *testing.T) {
		// 先写入
		_, err := composite.Write(ctx, "/memories/edit.txt", "old text")
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		// 编辑
		result, err := composite.Edit(ctx, "/memories/edit.txt", "old", "new", false)
		if err != nil {
			t.Fatalf("Edit failed: %v", err)
		}

		if result.ReplacementsMade != 1 {
			t.Errorf("Expected 1 replacement, got %d", result.ReplacementsMade)
		}

		// 验证编辑结果
		content, _ := memoryBackend.Read(ctx, "/edit.txt", 0, 0)
		if content != "new text" {
			t.Errorf("Expected 'new text', got '%s'", content)
		}
	})

	// 测试 4: ListInfo 应该加回前缀
	t.Run("ListInfo adds prefix back", func(t *testing.T) {
		// 在不同后端写入文件
		memoryBackend.Write(ctx, "/note1.txt", "note1")
		memoryBackend.Write(ctx, "/note2.txt", "note2")
		workspaceBackend.Write(ctx, "/main.go", "package main")
		defaultBackend.Write(ctx, "/readme.md", "readme")

		// 列出所有文件
		files, err := composite.ListInfo(ctx, "/")
		if err != nil {
			t.Fatalf("ListInfo failed: %v", err)
		}

		// 验证路径包含前缀
		pathsFound := make(map[string]bool)
		for _, file := range files {
			pathsFound[file.Path] = true
		}

		expectedPaths := []string{
			"/memories/note1.txt",
			"/memories/note2.txt",
			"/workspace/main.go",
			"/readme.md",
		}

		for _, expected := range expectedPaths {
			if !pathsFound[expected] {
				t.Errorf("Expected to find path '%s' in ListInfo results", expected)
			}
		}
	})

	// 测试 5: GlobInfo 应该加回前缀
	t.Run("GlobInfo adds prefix back", func(t *testing.T) {
		// 搜索所有 .txt 文件
		files, err := composite.GlobInfo(ctx, "*.txt", "/")
		if err != nil {
			t.Fatalf("GlobInfo failed: %v", err)
		}

		// 验证路径包含前缀
		foundMemoryFiles := false
		for _, file := range files {
			if strings.HasPrefix(file.Path, "/memories/") {
				foundMemoryFiles = true
			}
		}

		if !foundMemoryFiles {
			t.Error("Expected to find files with /memories/ prefix in GlobInfo results")
		}
	})

	// 测试 6: GrepRaw 应该加回前缀
	t.Run("GrepRaw adds prefix back", func(t *testing.T) {
		// 写入包含特定模式的文件
		memoryBackend.Write(ctx, "/search.txt", "TODO: important task")
		workspaceBackend.Write(ctx, "/code.go", "// TODO: fix this")

		// 搜索 "TODO"
		matches, err := composite.GrepRaw(ctx, "TODO", "/", "")
		if err != nil {
			t.Fatalf("GrepRaw failed: %v", err)
		}

		// 验证路径包含前缀
		foundPaths := make(map[string]bool)
		for _, match := range matches {
			foundPaths[match.Path] = true
		}

		if !foundPaths["/memories/search.txt"] {
			t.Error("Expected to find /memories/search.txt in GrepRaw results")
		}

		if !foundPaths["/workspace/code.go"] {
			t.Error("Expected to find /workspace/code.go in GrepRaw results")
		}
	})

	// 测试 7: 嵌套路径处理
	t.Run("Nested paths", func(t *testing.T) {
		_, err := composite.Write(ctx, "/memories/subfolder/deep.txt", "deep content")
		if err != nil {
			t.Fatalf("Write nested path failed: %v", err)
		}

		// 验证在子后端中路径正确
		content, err := memoryBackend.Read(ctx, "/subfolder/deep.txt", 0, 0)
		if err != nil {
			t.Fatalf("Read nested path failed: %v", err)
		}

		if content != "deep content" {
			t.Errorf("Expected 'deep content', got '%s'", content)
		}
	})

	// 测试 8: 默认后端处理
	t.Run("Default backend", func(t *testing.T) {
		// 不匹配任何路由前缀的路径应该使用默认后端
		_, err := composite.Write(ctx, "/default.txt", "default content")
		if err != nil {
			t.Fatalf("Write to default backend failed: %v", err)
		}

		// 验证在默认后端中
		content, err := defaultBackend.Read(ctx, "/default.txt", 0, 0)
		if err != nil {
			t.Fatalf("Read from default backend failed: %v", err)
		}

		if content != "default content" {
			t.Errorf("Expected 'default content', got '%s'", content)
		}
	})
}

// TestCompositeBackend_EdgeCases 测试边界情况
func TestCompositeBackend_EdgeCases(t *testing.T) {
	ctx := context.Background()

	defaultBackend := NewStateBackend()
	specialBackend := NewStateBackend()

	composite := NewCompositeBackend(defaultBackend, []RouteConfig{
		{Prefix: "/special/", Backend: specialBackend},
	})

	// 测试根路径 "/"
	t.Run("Root path", func(t *testing.T) {
		_, err := composite.Write(ctx, "/special/", "root content")
		if err != nil {
			t.Fatalf("Write to root failed: %v", err)
		}

		// 应该在 specialBackend 中作为 "/"
		content, err := specialBackend.Read(ctx, "/", 0, 0)
		if err != nil {
			t.Fatalf("Read root from specialBackend failed: %v", err)
		}

		if content != "root content" {
			t.Errorf("Expected 'root content', got '%s'", content)
		}
	})

	// 测试前缀不以 / 结尾的情况(虽然不推荐)
	t.Run("Path without leading slash after strip", func(t *testing.T) {
		backend2 := NewStateBackend()
		composite2 := NewCompositeBackend(defaultBackend, []RouteConfig{
			{Prefix: "/prefix", Backend: backend2}, // 注意没有尾部 /
		})

		_, err := composite2.Write(ctx, "/prefixfile.txt", "content")
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		// 应该在 backend2 中作为 "/file.txt"
		content, err := backend2.Read(ctx, "/file.txt", 0, 0)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}

		if content != "content" {
			t.Errorf("Expected 'content', got '%s'", content)
		}
	})
}

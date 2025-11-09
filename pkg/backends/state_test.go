package backends

import (
	"context"
	"strings"
	"testing"
)

func TestStateBackend(t *testing.T) {
	ctx := context.Background()
	backend := NewStateBackend()

	t.Run("Write and Read", func(t *testing.T) {
		content := "line1\nline2\nline3"
		result, err := backend.Write(ctx, "/test.txt", content)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		if result.Error != "" {
			t.Fatalf("Write should succeed, got error: %s", result.Error)
		}

		if result.BytesWritten != int64(len(content)) {
			t.Errorf("Expected bytes written %d, got %d", len(content), result.BytesWritten)
		}

		// 读取全部内容
		readContent, err := backend.Read(ctx, "/test.txt", 0, 0)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}

		if readContent != content {
			t.Errorf("Expected content %q, got %q", content, readContent)
		}
	})

	t.Run("Read with offset and limit", func(t *testing.T) {
		content := "line1\nline2\nline3\nline4\nline5"
		backend.Write(ctx, "/test2.txt", content)

		// 读取第2-3行 (offset=1, limit=2)
		readContent, err := backend.Read(ctx, "/test2.txt", 1, 2)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}

		expected := "line2\nline3"
		if readContent != expected {
			t.Errorf("Expected content %q, got %q", expected, readContent)
		}
	})

	t.Run("Edit - Replace first", func(t *testing.T) {
		content := "hello world\nhello again\nhello there"
		backend.Write(ctx, "/test3.txt", content)

		result, err := backend.Edit(ctx, "/test3.txt", "hello", "hi", false)
		if err != nil {
			t.Fatalf("Edit failed: %v", err)
		}

		if result.Error != "" {
			t.Fatalf("Edit should succeed, got error: %s", result.Error)
		}

		if result.ReplacementsMade != 1 {
			t.Errorf("Expected 1 replacement, got %d", result.ReplacementsMade)
		}

		// 验证内容
		newContent, _ := backend.Read(ctx, "/test3.txt", 0, 0)
		if !strings.HasPrefix(newContent, "hi world") {
			t.Errorf("Expected content to start with 'hi world', got %q", newContent)
		}
	})

	t.Run("Edit - Replace all", func(t *testing.T) {
		content := "hello world\nhello again\nhello there"
		backend.Write(ctx, "/test4.txt", content)

		result, err := backend.Edit(ctx, "/test4.txt", "hello", "hi", true)
		if err != nil {
			t.Fatalf("Edit failed: %v", err)
		}

		if result.ReplacementsMade != 3 {
			t.Errorf("Expected 3 replacements, got %d", result.ReplacementsMade)
		}

		// 验证内容
		newContent, _ := backend.Read(ctx, "/test4.txt", 0, 0)
		if strings.Contains(newContent, "hello") {
			t.Errorf("Content should not contain 'hello' after replace all: %q", newContent)
		}
	})

	t.Run("GrepRaw", func(t *testing.T) {
		backend.Write(ctx, "/file1.txt", "error: something wrong\ninfo: all good\nerror: another issue")
		backend.Write(ctx, "/file2.txt", "warning: be careful\nerror: bad thing")

		matches, err := backend.GrepRaw(ctx, "error:", "", "")
		if err != nil {
			t.Fatalf("GrepRaw failed: %v", err)
		}

		if len(matches) != 3 {
			t.Errorf("Expected 3 matches, got %d", len(matches))
		}

		// 验证第一个匹配
		if matches[0].Match != "error:" {
			t.Errorf("Expected match 'error:', got %q", matches[0].Match)
		}
	})

	t.Run("GlobInfo", func(t *testing.T) {
		backend.Write(ctx, "/src/main.go", "package main")
		backend.Write(ctx, "/src/utils.go", "package utils")
		backend.Write(ctx, "/test/test.go", "package test")
		backend.Write(ctx, "/README.md", "# Readme")

		// 匹配所有 .go 文件
		results, err := backend.GlobInfo(ctx, "*.go", "/")
		if err != nil {
			t.Fatalf("GlobInfo failed: %v", err)
		}

		// 应该找到至少1个 .go 文件
		if len(results) == 0 {
			t.Error("Expected to find .go files")
		}
	})

	t.Run("ListInfo", func(t *testing.T) {
		backend.Write(ctx, "/dir1/file1.txt", "content1")
		backend.Write(ctx, "/dir1/file2.txt", "content2")
		backend.Write(ctx, "/dir2/file3.txt", "content3")

		// 列出根目录
		results, err := backend.ListInfo(ctx, "/")
		if err != nil {
			t.Fatalf("ListInfo failed: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected to find items in root")
		}
	})

	t.Run("Read non-existent file", func(t *testing.T) {
		_, err := backend.Read(ctx, "/nonexistent.txt", 0, 0)
		if err == nil {
			t.Fatal("Expected error when reading non-existent file")
		}
	})
}

func TestStateBackend_LoadFiles(t *testing.T) {
	backend := NewStateBackend()
	ctx := context.Background()

	// 写入一些数据
	backend.Write(ctx, "/file1.txt", "content1")
	backend.Write(ctx, "/file2.txt", "content2")

	// 获取文件数据
	files := backend.GetFiles()
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	// 创建新的 backend 并加载数据
	newBackend := NewStateBackend()
	newBackend.LoadFiles(files)

	// 验证数据已恢复
	content, err := newBackend.Read(ctx, "/file1.txt", 0, 0)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if content != "content1" {
		t.Errorf("Expected content 'content1', got %q", content)
	}
}

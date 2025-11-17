package builtin

import (
	"os"
	"testing"
)

func TestNewWriteTool(t *testing.T) {
	tool, err := NewWriteTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Write tool: %v", err)
	}

	if tool.Name() != "Write" {
		t.Errorf("Expected tool name 'Write', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Tool description should not be empty")
	}
}

func TestWriteTool_CreateNewFile(t *testing.T) {
	tool, err := NewWriteTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Write tool: %v", err)
	}

	helper := NewTestHelper(t)
	defer helper.CleanupAll()

	filePath := helper.TmpDir + "/new_file.txt"
	content := "This is new content"

	input := map[string]interface{}{
		"file_path": filePath,
		"content":   content,
	}

	result := ExecuteToolWithRealFS(t, tool, input)
	result = AssertToolSuccess(t, result)

	// 验证文件被创建
	AssertFileExists(t, filePath)
	AssertFileContent(t, filePath, content)

	// 验证响应字段
	if result["file_path"] != filePath {
		t.Error("file_path should be echoed back")
	}

	if fileSize, exists := result["file_size"]; !exists {
		t.Error("Result should contain 'file_size' field")
	} else if fileSizeInt, ok := fileSize.(int); !ok || fileSizeInt != len(content) {
		t.Errorf("file_size should be %d, got %v", len(content), fileSize)
	}
}

func TestWriteTool_OverwriteExisting(t *testing.T) {
	tool, err := NewWriteTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Write tool: %v", err)
	}

	helper := NewTestHelper(t)
	defer helper.CleanupAll()

	// 先创建一个文件
	filePath := helper.CreateTempFile("existing.txt", "Original content")

	newContent := "Overwritten content"
	input := map[string]interface{}{
		"file_path": filePath,
		"content":   newContent,
	}

	result := ExecuteToolWithRealFS(t, tool, input)
	result = AssertToolSuccess(t, result)

	// 验证文件被覆盖
	AssertFileContent(t, filePath, newContent)
}

func TestWriteTool_AppendMode(t *testing.T) {
	tool, err := NewWriteTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Write tool: %v", err)
	}

	helper := NewTestHelper(t)
	defer helper.CleanupAll()

	// 先创建一个文件
	originalContent := "Original line\n"
	filePath := helper.CreateTempFile("append.txt", originalContent)

	appendContent := "Appended line"
	input := map[string]interface{}{
		"file_path": filePath,
		"content":   appendContent,
		"append":    true,
	}

	result := ExecuteToolWithRealFS(t, tool, input)
	result = AssertToolSuccess(t, result)

	// 验证内容被追加
	expectedContent := originalContent + appendContent
	AssertFileContent(t, filePath, expectedContent)
}

func TestWriteTool_AutoCreateDirectory(t *testing.T) {
	tool, err := NewWriteTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Write tool: %v", err)
	}

	helper := NewTestHelper(t)
	defer helper.CleanupAll()

	// 使用不存在的目录
	filePath := helper.TmpDir + "/new/subdir/file.txt"
	content := "Content in auto-created directory"

	input := map[string]interface{}{
		"file_path": filePath,
		"content":   content,
	}

	result := ExecuteToolWithRealFS(t, tool, input)
	result = AssertToolSuccess(t, result)

	// 验证目录和文件都被创建
	AssertFileExists(t, filePath)
	AssertFileContent(t, filePath, content)
}

func BenchmarkWriteTool_SmallFile(b *testing.B) {
	tool, err := NewWriteTool(nil)
	if err != nil {
		b.Fatalf("Failed to create Write tool: %v", err)
	}

	helper := NewTestHelper(&testing.T{})
	defer helper.CleanupAll()

	filePath := helper.TmpDir + "/benchmark.txt"
	content := "Benchmark content"

	input := map[string]interface{}{
		"file_path": filePath,
		"content":   content,
	}

	// 清理文件
	defer os.Remove(filePath)

	BenchmarkTool(b, tool, input)
}
package builtin

import (
	"strings"
	"testing"
)

func TestNewEditTool(t *testing.T) {
	tool, err := NewEditTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Edit tool: %v", err)
	}

	if tool.Name() != "Edit" {
		t.Errorf("Expected tool name 'Edit', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Tool description should not be empty")
	}
}

func TestEditTool_InputSchema(t *testing.T) {
	tool, err := NewEditTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Edit tool: %v", err)
	}

	schema := tool.InputSchema()
	if schema == nil {
		t.Fatal("Input schema should not be nil")
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Properties should be a map")
	}

	// 验证必需字段存在
	requiredFields := []string{"file_path", "old_string", "new_string"}
	for _, field := range requiredFields {
		if _, exists := properties[field]; !exists {
			t.Errorf("Required field '%s' should exist in properties", field)
		}
	}

	// 验证可选字段存在
	optionalFields := []string{"replace_all", "preserve_indentation"}
	for _, field := range optionalFields {
		if _, exists := properties[field]; !exists {
			t.Errorf("Optional field '%s' should exist in properties", field)
		}
	}

	// 验证required字段
	required := schema["required"]
	var requiredArray []interface{}
	switch v := required.(type) {
	case []interface{}:
		requiredArray = v
	case []string:
		requiredArray = make([]interface{}, len(v))
		for i, s := range v {
			requiredArray[i] = s
		}
	default:
		t.Fatal("Required should be an array")
	}

	if len(requiredArray) != 3 {
		t.Errorf("Expected 3 required fields, got %d", len(requiredArray))
	}
}

func TestEditTool_SimpleEdit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file modification test in short mode")
	}

	tool, err := NewEditTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Edit tool: %v", err)
	}

	// 创建测试文件
	helper := NewTestHelper(t)
	testFile := helper.CreateTempFile("test.txt", "Hello World\nThis is a test\nGoodbye")
	defer helper.CleanupAll()

	input := map[string]interface{}{
		"file_path": testFile,
		"old_string": "Hello World",
		"new_string": "Hello Universe",
	}

	result := ExecuteToolWithRealFS(t, tool, input)
	result = AssertToolSuccess(t, result)

	// 验证响应字段
	if success, exists := result["success"]; !exists || !success.(bool) {
		t.Error("Edit operation should succeed")
	}

	if changes, exists := result["changes_made"]; !exists || changes.(int) != 1 {
		t.Error("Should indicate 1 change was made")
	}

	// 验证文件内容确实被修改
	content := helper.ReadFile(testFile)
	if !strings.Contains(content, "Hello Universe") {
		t.Error("File should contain the new string")
	}
	if strings.Contains(content, "Hello World") {
		t.Error("File should not contain the old string")
	}
}

func TestEditTool_MultipleEdit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file modification test in short mode")
	}

	tool, err := NewEditTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Edit tool: %v", err)
	}

	// 创建测试文件
	helper := NewTestHelper(t)
	testFile := helper.CreateTempFile("test.txt", "Hello World\nHello World\nHello World")
	defer helper.CleanupAll()

	input := map[string]interface{}{
		"file_path":   testFile,
		"old_string":  "Hello World",
		"new_string":  "Hello Universe",
		"replace_all": true,
	}

	result := ExecuteToolWithRealFS(t, tool, input)
	result = AssertToolSuccess(t, result)

	// 验证多个修改
	if changes, exists := result["changes_made"]; !exists || changes.(int) != 3 {
		t.Errorf("Should indicate 3 changes were made, got %v", result["changes_made"])
	}

	// 验证所有实例都被替换
	content := helper.ReadFile(testFile)
	if strings.Count(content, "Hello Universe") != 3 {
		t.Error("All instances should be replaced")
	}
	if strings.Contains(content, "Hello World") {
		t.Error("No old strings should remain")
	}
}

func TestEditTool_MissingString(t *testing.T) {
	tool, err := NewEditTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Edit tool: %v", err)
	}

	input := map[string]interface{}{
		"file_path": "/tmp/test.txt",
		// 缺少 old_string
		"new_string": "new content",
	}

	result := ExecuteToolWithInput(t, tool, input)

	// 应该返回错误
	errMsg := AssertToolError(t, result)
	if !strings.Contains(strings.ToLower(errMsg), "required") {
		t.Errorf("Expected required field error, got: %s", errMsg)
	}
}

func TestEditTool_NonExistentFile(t *testing.T) {
	tool, err := NewEditTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Edit tool: %v", err)
	}

	input := map[string]interface{}{
		"file_path":  "/nonexistent/file.txt",
		"old_string": "old content",
		"new_string": "new content",
	}

	result := ExecuteToolWithInput(t, tool, input)

	// 应该返回文件不存在的错误
	errMsg := AssertToolError(t, result)
	if !strings.Contains(strings.ToLower(errMsg), "file") &&
		!strings.Contains(strings.ToLower(errMsg), "not found") &&
		!strings.Contains(strings.ToLower(errMsg), "no such file") {
		t.Errorf("Expected file not found error, got: %s", errMsg)
	}
}

func TestEditTool_SameString(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file modification test in short mode")
	}

	tool, err := NewEditTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Edit tool: %v", err)
	}

	// 创建测试文件
	helper := NewTestHelper(t)
	testFile := helper.CreateTempFile("test.txt", "Hello World")
	defer helper.CleanupAll()

	input := map[string]interface{}{
		"file_path":  testFile,
		"old_string": "Hello World",
		"new_string": "Hello World", // 相同的内容
	}

	result := ExecuteToolWithRealFS(t, tool, input)
	result = AssertToolSuccess(t, result)

	// 应该显示没有修改
	if changes, exists := result["changes_made"]; !exists || changes.(int) != 0 {
		t.Error("Should indicate 0 changes were made")
	}
}

func BenchmarkEditTool_SimpleEdit(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	tool, err := NewEditTool(nil)
	if err != nil {
		b.Fatalf("Failed to create Edit tool: %v", err)
	}

	helper := NewTestHelper(&testing.T{})
	testFile := helper.CreateTempFile("benchmark.txt", "old content line\nold content line\nold content line")
	defer helper.CleanupAll()

	input := map[string]interface{}{
		"file_path":  testFile,
		"old_string": "old content",
		"new_string": "new content",
	}

	BenchmarkTool(b, tool, input)
}
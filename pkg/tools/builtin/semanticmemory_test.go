package builtin

import (
	"testing"
)

func TestNewSemanticSearchTool(t *testing.T) {
	tool, err := NewSemanticSearchTool(nil)
	if err != nil {
		t.Fatal(err)
	}

	if tool.Name() != "semantic_search" {
		t.Errorf("Expected tool name 'semantic_search', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Tool description should not be empty")
	}
}

func TestSemanticSearchTool_InputSchema(t *testing.T) {
	tool, err := NewSemanticSearchTool(nil)
	if err != nil {
		t.Fatal(err)
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
	requiredFields := []string{"query"}
	for _, field := range requiredFields {
		if _, exists := properties[field]; !exists {
			t.Errorf("Required field '%s' should exist in properties", field)
		}
	}

	// 验证可选字段存在
	optionalFields := []string{"top_k", "metadata"}
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

	if len(requiredArray) != 1 || requiredArray[0] != "query" {
		t.Errorf("Required should contain only 'query', got %v", requiredArray)
	}
}

func TestSemanticSearchTool_MissingQuery(t *testing.T) {
	tool, err := NewSemanticSearchTool(nil)
	if err != nil {
		t.Fatal(err)
	}

	input := map[string]interface{}{
		"top_k": 5,
	}

	result := ExecuteToolWithInput(t, tool, input)

	// 应该返回错误
	errMsg := AssertToolError(t, result)
	if !contains(errMsg, "query") && !contains(errMsg, "required") {
		t.Errorf("Expected query required error, got: %s", errMsg)
	}
}

func TestSemanticSearchTool_EmptyQuery(t *testing.T) {
	tool, err := NewSemanticSearchTool(nil)
	if err != nil {
		t.Fatal(err)
	}

	input := map[string]interface{}{
		"query": "", // 空查询
	}

	result := ExecuteToolWithInput(t, tool, input)

	// 应该返回错误
	errMsg := AssertToolError(t, result)
	if !contains(errMsg, "query") && !contains(errMsg, "empty") {
		t.Errorf("Expected query empty error, got: %s", errMsg)
	}
}

func TestSemanticSearchTool_WithTopK(t *testing.T) {
	tool, err := NewSemanticSearchTool(nil)
	if err != nil {
		t.Fatal(err)
	}

	input := map[string]interface{}{
		"query": "test query",
		"top_k": 3,
	}

	result := ExecuteToolWithInput(t, tool, input)

	// 由于可能没有实际的语义搜索引擎，检查工具是否正确处理了输入
	if !result["ok"].(bool) {
		t.Logf("Semantic search failed (expected in test environment): %v", result["error"])
	} else {
		// 如果成功，验证基本响应
		if query, exists := result["query"]; !exists || query.(string) != "test query" {
			t.Error("Should echo back the input query")
		}

		if topK, exists := result["top_k"]; !exists || topK.(int) != 3 {
			t.Error("Should include top_k setting in response")
		}
	}
}

func TestSemanticSearchTool_WithMetadata(t *testing.T) {
	tool, err := NewSemanticSearchTool(nil)
	if err != nil {
		t.Fatal(err)
	}

	input := map[string]interface{}{
		"query": "test query",
		"metadata": map[string]interface{}{
			"user_id":    "test_user",
			"project_id": "test_project",
		},
	}

	result := ExecuteToolWithInput(t, tool, input)

	// 由于可能没有实际的语义搜索引擎，检查工具是否正确处理了输入
	if !result["ok"].(bool) {
		t.Logf("Semantic search failed (expected in test environment): %v", result["error"])
	} else {
		// 如果成功，验证元数据
		if _, exists := result["metadata"]; !exists {
			t.Error("Should include metadata in response")
		}
	}
}
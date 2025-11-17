package builtin

import (
	"context"
	"fmt"

	"github.com/wordflowlab/agentsdk/pkg/sandbox"
)

// MockSandbox 简单的沙箱模拟实现
type MockSandbox struct {
	sandbox.MockSandbox
}

// NewMockSandbox 创建新的模拟沙箱
func NewMockSandbox() *MockSandbox {
	return &MockSandbox{
		MockSandbox: sandbox.MockSandbox{},
	}
}

// SafeExecute 模拟安全执行
func (m *MockSandbox) SafeExecute(ctx context.Context, cmd string) error {
	// 简单的安全检查模拟
	dangerousCommands := []string{"rm -rf", "dd if=", "mkfs", "format"}
	for _, dangerous := range dangerousCommands {
		if contains(cmd, dangerous) {
			return fmt.Errorf("security error: command contains dangerous patterns: %s", dangerous)
		}
	}
	return nil
}

// contains 辅助函数检查字符串包含
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
			 s[len(s)-len(substr):] == substr ||
			 indexOf(s, substr) >= 0)))
}

// indexOf 简单的字符串查找
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
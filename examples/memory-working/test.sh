#!/bin/bash
# Working Memory 功能验证脚本

set -e

echo "=== Working Memory 功能验证 ==="
echo ""

echo "1. 运行 Working Memory 单元测试..."
cd /Users/coso/Documents/dev/ai/wordflowlab/agentsdk/pkg/memory
go test -v -run TestWorkingMemoryManager 2>&1 | grep -E "(PASS|FAIL|ok|=== RUN)"

echo ""
echo "2. 运行 Working Memory 示例测试..."
cd /Users/coso/Documents/dev/ai/wordflowlab/agentsdk/examples/memory-working
go test -v verify_test.go 2>&1 | grep -E "(PASS|FAIL|ok|=== RUN|✓)"

echo ""
echo "=== 验证完成 ==="

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/wordflowlab/agentsdk/pkg/backends"
	"github.com/wordflowlab/agentsdk/pkg/middleware"
	"github.com/wordflowlab/agentsdk/pkg/tools"
)

// 本示例演示如何使用 AgentMemoryMiddleware 提供的高级记忆能力:
// - 基于文件 + grep 的长期记忆 (/memories/*.md)
// - memory_write: 向记忆文件追加/覆盖 Markdown 笔记
// - memory_search: 在记忆目录中进行全文搜索
//
// 注意: 这里直接通过 Tool 接口调用 memory_* 工具, 真实 Agent 中这些工具会通过
// AgentConfig.Middlewares 自动注入并由 LLM 调用。
func main() {
	ctx := context.Background()

	// 1. 构建 Backend:
	//    - 默认使用 StateBackend (内存临时文件)
	//    - /memories/ 路径映射到本地 ./memories 目录, 用于长期记忆
	stateBackend := backends.NewStateBackend()
	localMemBackend := backends.NewLocalBackend("./memories")

	memoryBackend := backends.NewCompositeBackend(
		stateBackend,
		[]backends.RouteConfig{
			{
				Prefix:  "/memories/",
				Backend: localMemBackend,
			},
		},
	)

	// 2. 创建 Filesystem + AgentMemory 中间件
	fsMiddleware := middleware.NewFilesystemMiddleware(&middleware.FilesystemMiddlewareConfig{
		Backend: memoryBackend,
	})

	memoryMW, err := middleware.NewAgentMemoryMiddleware(&middleware.AgentMemoryMiddlewareConfig{
		Backend:    memoryBackend,
		MemoryPath: "/memories/",
	})
	if err != nil {
		log.Fatalf("create AgentMemoryMiddleware failed: %v", err)
	}

	// 3. 组装中间件栈并收集工具
	stack := middleware.NewStack([]middleware.Middleware{
		fsMiddleware,
		memoryMW,
	})

	allTools := stack.Tools()
	fmt.Printf("✅ 中间件栈已创建, 工具总数: %d\n\n", len(allTools))

	var memoryWriteTool tools.Tool
	var memorySearchTool tools.Tool

	for _, t := range allTools {
		fmt.Printf("- 工具: %-16s 描述: %s\n", t.Name(), t.Description())
		switch t.Name() {
		case "memory_write":
			memoryWriteTool = t
		case "memory_search":
			memorySearchTool = t
		}
	}
	fmt.Println()

	if memoryWriteTool == nil || memorySearchTool == nil {
		log.Fatalf("memory_write 或 memory_search 工具未找到, 请检查 AgentMemoryMiddleware 初始化是否成功")
	}

	toolCtx := &tools.ToolContext{} // 本例中不需要 Sandbox, 传空即可

	// 4. 使用 memory_write 追加一条长期记忆
	fmt.Println("=== 使用 memory_write 追加记忆 ===")
	writeInput := map[string]interface{}{
		"file":    "user/alice.md",
		"mode":    "append",
		"title":   "初次见面",
		"content": "Alice 喜欢 grep 风格的记忆系统, 并偏好简洁的代码 diff。",
	}

	writeResult, err := memoryWriteTool.Execute(ctx, writeInput, toolCtx)
	if err != nil {
		log.Fatalf("memory_write 执行失败: %v", err)
	}
	fmt.Printf("memory_write 结果: %+v\n\n", writeResult)

	// 5. 使用 memory_search 在记忆中搜索关键字
	fmt.Println("=== 使用 memory_search 搜索记忆 ===")
	searchInput := map[string]interface{}{
		"query":       "Alice",
		"glob":        "user/*.md",
		"max_results": 10,
	}

	searchResult, err := memorySearchTool.Execute(ctx, searchInput, toolCtx)
	if err != nil {
		log.Fatalf("memory_search 执行失败: %v", err)
	}
	fmt.Printf("memory_search 结果: %+v\n", searchResult)
}


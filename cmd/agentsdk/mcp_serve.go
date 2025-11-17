package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/wordflowlab/agentsdk/pkg/mcpserver"
	"github.com/wordflowlab/agentsdk/pkg/tools"
)

func runMCPServe(args []string) error {
	fs := flag.NewFlagSet("mcp-serve", flag.ExitOnError)
	addr := fs.String("addr", ":8090", "MCP HTTP listen address")
	docsDir := fs.String("docs", "./docs/content", "Docs directory")

	if err := fs.Parse(args); err != nil {
		return err
	}

	registry := tools.NewRegistry()

	// Echo tool
	registry.Register("echo", func(cfg map[string]interface{}) (tools.Tool, error) {
		return &EchoTool{}, nil
	})

	// Docs tools
	if docsGet, err := mcpserver.NewDocsGetTool(*docsDir); err == nil {
		registry.Register(docsGet.Name(), func(cfg map[string]interface{}) (tools.Tool, error) {
			return docsGet, nil
		})
	}

	if docsSearch, err := mcpserver.NewDocsSearchTool(*docsDir); err == nil {
		registry.Register(docsSearch.Name(), func(cfg map[string]interface{}) (tools.Tool, error) {
			return docsSearch, nil
		})
	}

	srv, err := mcpserver.New(&mcpserver.Config{Registry: registry})
	if err != nil {
		return fmt.Errorf("create MCP server: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/mcp", srv.Handler())

	fmt.Printf("🚀 MCP Server at http://localhost%s/mcp\n", *addr)

	return (&http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}).ListenAndServe()
}

type EchoTool struct{}

func (t *EchoTool) Name() string        { return "echo" }
func (t *EchoTool) Description() string { return "Echo text" }
func (t *EchoTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"text": map[string]interface{}{"type": "string"},
		},
		"required": []string{"text"},
	}
}
func (t *EchoTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
	return input["text"], nil
}
func (t *EchoTool) Prompt() string { return "Echo tool" }

package workflow

import (
	"context"
	"fmt"
	"iter"
	"sync"

	"golang.org/x/sync/errgroup"
	"github.com/wordflowlab/agentsdk/pkg/session"
)

// ParallelAgent 并行执行多个子 Agent
// 参考 Google ADK-Go 的 ParallelAgent 设计
//
// 使用场景:
// - 同时运行不同算法进行比较
// - 生成多个候选响应供后续评估
// - 并行处理独立的任务
type ParallelAgent struct {
	name      string
	subAgents []Agent
	mu        sync.RWMutex
}

// Agent 接口定义
// 简化版本，实际应该从主 Agent 包导入
type Agent interface {
	Name() string
	Execute(ctx context.Context, message string) iter.Seq2[*session.Event, error]
}

// ParallelConfig ParallelAgent 配置
type ParallelConfig struct {
	// Name Agent 名称
	Name string

	// SubAgents 子 Agent 列表
	SubAgents []Agent

	// MaxConcurrent 最大并发数（0 表示无限制）
	MaxConcurrent int
}

// NewParallelAgent 创建并行 Agent
func NewParallelAgent(cfg ParallelConfig) (*ParallelAgent, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("agent name is required")
	}

	if len(cfg.SubAgents) == 0 {
		return nil, fmt.Errorf("at least one sub-agent is required")
	}

	return &ParallelAgent{
		name:      cfg.Name,
		subAgents: cfg.SubAgents,
	}, nil
}

// Name 返回 Agent 名称
func (a *ParallelAgent) Name() string {
	return a.name
}

// Execute 并行执行所有子 Agent
func (a *ParallelAgent) Execute(ctx context.Context, message string) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		var (
			eg, egCtx = errgroup.WithContext(ctx)
			resultsCh = make(chan result, len(a.subAgents)*10) // 缓冲通道
			doneCh    = make(chan struct{})
		)

		// 启动所有子 Agent
		for i, subAgent := range a.subAgents {
			sa := subAgent
			branch := fmt.Sprintf("%s.%s", a.name, sa.Name())
			index := i

			eg.Go(func() error {
				return a.runSubAgent(egCtx, sa, branch, index, message, resultsCh, doneCh)
			})
		}

		// 等待所有子 Agent 完成
		go func() {
			_ = eg.Wait() // 错误已通过 resultsCh 传递
			close(resultsCh)
		}()

		// 流式返回结果
		defer close(doneCh)

		for res := range resultsCh {
			if !yield(res.event, res.err) {
				return // 客户端取消
			}
		}
	}
}

// runSubAgent 运行单个子 Agent
func (a *ParallelAgent) runSubAgent(
	ctx context.Context,
	agent Agent,
	branch string,
	index int,
	message string,
	results chan<- result,
	done <-chan struct{},
) error {
	for event, err := range agent.Execute(ctx, message) {
		select {
		case <-done:
			return nil // 客户端取消
		case <-ctx.Done():
			select {
			case <-done:
			case results <- result{err: ctx.Err()}:
			}
			return ctx.Err()
		case results <- result{
			event: a.enrichEvent(event, branch, index),
			err:   err,
		}:
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// enrichEvent 丰富事件信息，添加 branch 和元数据
func (a *ParallelAgent) enrichEvent(event *session.Event, branch string, index int) *session.Event {
	if event == nil {
		return nil
	}

	// 更新 Branch 信息
	event.Branch = branch

	// 添加并行执行的元数据
	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}
	event.Metadata["parallel_index"] = index
	event.Metadata["parallel_agent"] = a.name

	return event
}

// AddSubAgent 动态添加子 Agent
func (a *ParallelAgent) AddSubAgent(agent Agent) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.subAgents = append(a.subAgents, agent)
}

// SubAgents 返回所有子 Agent
func (a *ParallelAgent) SubAgents() []Agent {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return append([]Agent{}, a.subAgents...)
}

// result 内部结果类型
type result struct {
	event *session.Event
	err   error
}

package workflow

import (
	"context"
	"fmt"
	"iter"

	"github.com/wordflowlab/agentsdk/pkg/session"
)

// SequentialAgent 顺序执行子 Agent
// 参考 Google ADK-Go 的 SequentialAgent 设计
//
// 实际上是 LoopAgent 的特例（MaxIterations=1）
//
// 使用场景:
// - 多步骤工作流（分析 -> 规划 -> 执行）
// - 流水线处理（预处理 -> 处理 -> 后处理）
// - 阶段性任务（收集信息 -> 分析 -> 决策）
type SequentialAgent struct {
	*LoopAgent
}

// SequentialConfig SequentialAgent 配置
type SequentialConfig struct {
	// Name Agent 名称
	Name string

	// SubAgents 子 Agent 列表（严格按顺序执行一次）
	SubAgents []Agent

	// StopOnError 遇到错误时是否停止（默认 true）
	StopOnError bool
}

// NewSequentialAgent 创建顺序 Agent
func NewSequentialAgent(cfg SequentialConfig) (*SequentialAgent, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("agent name is required")
	}

	if len(cfg.SubAgents) == 0 {
		return nil, fmt.Errorf("at least one sub-agent is required")
	}

	// SequentialAgent 是 LoopAgent 迭代 1 次的特例
	loopAgent, err := NewLoopAgent(LoopConfig{
		Name:          cfg.Name,
		SubAgents:     cfg.SubAgents,
		MaxIterations: 1,
		StopCondition: func(event *session.Event) bool {
			// Sequential 不依赖 Escalate，总是执行完所有子 Agent
			return false
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create sequential agent: %w", err)
	}

	return &SequentialAgent{
		LoopAgent: loopAgent,
	}, nil
}

// Execute 顺序执行所有子 Agent（仅一次）
func (a *SequentialAgent) Execute(ctx context.Context, message string) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		// 顺序执行所有子 Agent
		for i, subAgent := range a.subAgents {
			branch := fmt.Sprintf("%s.%s", a.name, subAgent.Name())

			for event, err := range subAgent.Execute(ctx, message) {
				// 丰富事件信息
				enrichedEvent := a.enrichSequentialEvent(event, branch, i)

				// 传递事件
				if !yield(enrichedEvent, err) {
					return // 客户端取消
				}

				// 检查错误
				if err != nil {
					return // 遇到错误停止
				}
			}

			// 检查上下文取消
			if ctx.Err() != nil {
				yield(nil, ctx.Err())
				return
			}
		}
	}
}

// enrichSequentialEvent 丰富顺序执行事件信息
func (a *SequentialAgent) enrichSequentialEvent(event *session.Event, branch string, index int) *session.Event {
	if event == nil {
		return nil
	}

	// 更新 Branch 信息
	event.Branch = branch

	// 添加顺序执行的元数据
	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}
	event.Metadata["sequential_step"] = index + 1
	event.Metadata["sequential_agent"] = a.name
	event.Metadata["total_steps"] = len(a.subAgents)

	return event
}

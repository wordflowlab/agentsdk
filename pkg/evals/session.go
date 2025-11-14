package evals

import (
	"fmt"
	"strings"

	"github.com/wordflowlab/agentsdk/pkg/session"
	"github.com/wordflowlab/agentsdk/pkg/types"
)

// BuildTextEvalInputFromEvents 根据一组 Session 事件构建 TextEvalInput。
//
// 约定:
// - 默认将最后一个 assistant 消息视为 Answer。
// - 将之前的 user / assistant 消息串联为 Context,用于评估时参考。
// - Reference 由调用方自行填充(例如从标注数据集中读取)。
func BuildTextEvalInputFromEvents(events []session.Event) *TextEvalInput {
	if len(events) == 0 {
		return &TextEvalInput{}
	}

	var answer string
	var context []string

	for idx, e := range events {
		msg := e.Content
		text := extractMessageText(&msg)

		isLast := idx == len(events)-1
		if isLast && msg.Role == types.MessageRoleAssistant {
			answer = text
			continue
		}

		// 其他消息作为上下文
		context = append(context, fmt.Sprintf("%s: %s", msg.Role, text))
	}

	return &TextEvalInput{
		Answer:  answer,
		Context: context,
	}
}

// extractMessageText 将 types.Message 中的文本内容展开为单一字符串。
func extractMessageText(msg *types.Message) string {
	if msg == nil {
		return ""
	}

	// 如果有 ContentBlocks，提取其中的文本
	if len(msg.ContentBlocks) > 0 {
		var blocks []string
		for _, block := range msg.ContentBlocks {
			if tb, ok := block.(*types.TextBlock); ok {
				blocks = append(blocks, tb.Text)
			}
		}
		return strings.TrimSpace(strings.Join(blocks, "\n"))
	}

	// 向后兼容：直接返回 Content
	return strings.TrimSpace(msg.Content)
}


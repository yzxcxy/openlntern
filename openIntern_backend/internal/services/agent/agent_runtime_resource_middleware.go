package agent

import (
	"context"
	"log"
	"strings"
	"time"

	kbsvc "openIntern/internal/services/kb"
	memorysvc "openIntern/internal/services/memory"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

const agentResourceInjectedRunLocalKey = "openintern_agent_resource_injected"
const agentMemoryRetrieveTimeout = 1200 * time.Millisecond

// agentResourceInjectionMiddleware injects agent-bound KB and memory context once per run.
type agentResourceInjectionMiddleware struct {
	adk.BaseChatModelAgentMiddleware

	knowledgeBaseNames []string
	memoryEnabled      bool
	retriever          MemoryRetriever
}

func newAgentResourceInjectionMiddleware(knowledgeBaseNames []string, memoryEnabled bool, retriever MemoryRetriever) adk.ChatModelAgentMiddleware {
	if len(knowledgeBaseNames) == 0 && (!memoryEnabled || retriever == nil) {
		return nil
	}
	return &agentResourceInjectionMiddleware{
		knowledgeBaseNames: append([]string{}, knowledgeBaseNames...),
		memoryEnabled:      memoryEnabled,
		retriever:          retriever,
	}
}

func (m *agentResourceInjectionMiddleware) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, _ *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	if state == nil {
		return ctx, state, nil
	}
	if _, found, err := adk.GetRunLocalValue(ctx, agentResourceInjectedRunLocalKey); err != nil {
		return ctx, nil, err
	} else if found {
		return ctx, state, nil
	}

	query := latestUserTextFromSchemaMessages(state.Messages)
	if err := adk.SetRunLocalValue(ctx, agentResourceInjectedRunLocalKey, true); err != nil {
		return ctx, nil, err
	}
	if query == "" {
		return ctx, state, nil
	}

	injections := make([]*schema.Message, 0, 2)
	if len(m.knowledgeBaseNames) > 0 {
		content, err := kbsvc.KnowledgeBase.BuildContextMessage(ctx, query, m.knowledgeBaseNames, knowledgeSearchLimitPerKB)
		if err != nil {
			log.Printf("RunAgent skip KB context because retrieval failed err=%v", err)
		} else if strings.TrimSpace(content) != "" {
			injections = append(injections, &schema.Message{
				Role:    schema.System,
				Content: content,
			})
		}
	}
	if m.memoryEnabled && m.retriever != nil {
		memoryCtx, cancel := context.WithTimeout(ctx, agentMemoryRetrieveTimeout)
		memories, err := m.retriever.Retrieve(memoryCtx, ownerIDFromContext(ctx), query)
		cancel()
		if err != nil {
			log.Printf("RunAgent skip memory context because retrieval failed err=%v", err)
		} else {
			content := memorysvc.BuildContextMessageContent(memories)
			if strings.TrimSpace(content) != "" {
				injections = append(injections, &schema.Message{
					Role:    schema.System,
					Content: content,
				})
			}
		}
	}
	if len(injections) == 0 {
		return ctx, state, nil
	}

	next := *state
	next.Messages = injectSchemaMessagesBeforeLastUser(state.Messages, injections...)
	return ctx, &next, nil
}

// latestUserTextFromSchemaMessages extracts the latest plain-text user message from schema messages.
func latestUserTextFromSchemaMessages(messages []*schema.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		if message == nil || message.Role != schema.User {
			continue
		}
		if content := strings.TrimSpace(message.Content); content != "" {
			return content
		}
		if content := strings.TrimSpace(concatSchemaInputText(message.UserInputMultiContent)); content != "" {
			return content
		}
	}
	return ""
}

// injectSchemaMessagesBeforeLastUser inserts system context immediately before the latest user turn.
func injectSchemaMessagesBeforeLastUser(messages []*schema.Message, injections ...*schema.Message) []*schema.Message {
	if len(messages) == 0 || len(injections) == 0 {
		return append([]*schema.Message{}, messages...)
	}
	userIndex := len(messages)
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i] != nil && messages[i].Role == schema.User {
			userIndex = i
			break
		}
	}
	if userIndex == len(messages) {
		result := append([]*schema.Message{}, messages...)
		return append(result, injections...)
	}

	result := make([]*schema.Message, 0, len(messages)+len(injections))
	result = append(result, messages[:userIndex]...)
	result = append(result, injections...)
	result = append(result, messages[userIndex:]...)
	return result
}

func concatSchemaInputText(parts []schema.MessageInputPart) string {
	var builder strings.Builder
	for _, part := range parts {
		if part.Type != schema.ChatMessagePartTypeText || part.Text == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(part.Text)
	}
	return builder.String()
}

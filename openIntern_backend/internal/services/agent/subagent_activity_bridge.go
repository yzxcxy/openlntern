package agent

import (
	"io"
	"strings"

	"openIntern/internal/services/agent/agui"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

const subAgentPanelActivityType = "sub-agent-panel"

// subAgentActivityBridge 将子 agent 的内部事件收敛为独立 activity 面板，避免混入主 assistant 文本流。
type subAgentActivityBridge struct {
	sender           *agui.AccumulatingSender
	panelsByToolCall map[string]*subAgentPanelState
	pendingPanels    []*subAgentPanelState
}

type subAgentPanelState struct {
	MessageID     string
	RootToolCall  string
	RootToolName  string
	AgentName     string
	Status        string
	OutputText    string
	ProcessItems  []map[string]any
	toolIndexByID map[string]int
	toolArgsByID  map[string]string
}

func newSubAgentActivityBridge(sender *agui.AccumulatingSender) *subAgentActivityBridge {
	return &subAgentActivityBridge{
		sender:           sender,
		panelsByToolCall: make(map[string]*subAgentPanelState),
	}
}

func (b *subAgentActivityBridge) OnToolCallStart(toolCallID, toolName string) {
	if !isSubAgentToolName(toolName) {
		return
	}
	if _, exists := b.panelsByToolCall[toolCallID]; exists {
		return
	}
	panel := &subAgentPanelState{
		MessageID:     uuid.NewString(),
		RootToolCall:  toolCallID,
		RootToolName:  toolName,
		Status:        "in_progress",
		toolIndexByID: make(map[string]int),
		toolArgsByID:  make(map[string]string),
	}
	b.panelsByToolCall[toolCallID] = panel
	b.pendingPanels = append(b.pendingPanels, panel)
}

func (b *subAgentActivityBridge) OnToolCallResult(toolCallID, content string) (bool, error) {
	panel := b.panelsByToolCall[toolCallID]
	if panel == nil {
		return false, nil
	}
	panel.OutputText = content
	panel.Status = "completed"
	return true, b.publish(panel)
}

func (b *subAgentActivityBridge) HandleNestedEvent(event *adk.AgentEvent) error {
	if event == nil || event.Output == nil || event.Output.MessageOutput == nil {
		return nil
	}
	panel := b.ensurePanel(event)
	if panel == nil {
		return nil
	}
	mv := event.Output.MessageOutput
	if mv.IsStreaming {
		return b.handleNestedStream(panel, mv)
	}
	msg, err := mv.GetMessage()
	if err != nil {
		return err
	}
	if msg == nil {
		return nil
	}
	return b.applyNestedMessage(panel, msg)
}

func (b *subAgentActivityBridge) ensurePanel(event *adk.AgentEvent) *subAgentPanelState {
	agentName := topLevelSubAgentName(event)
	if agentName == "" {
		return nil
	}
	for _, panel := range b.pendingPanels {
		if panel == nil || panel.Status == "completed" {
			continue
		}
		if panel.AgentName == agentName {
			return panel
		}
	}
	for _, panel := range b.pendingPanels {
		if panel == nil || panel.Status == "completed" {
			continue
		}
		if strings.TrimSpace(panel.AgentName) == "" {
			panel.AgentName = agentName
			return panel
		}
	}
	panel := &subAgentPanelState{
		MessageID:     uuid.NewString(),
		AgentName:     agentName,
		Status:        "in_progress",
		toolIndexByID: make(map[string]int),
		toolArgsByID:  make(map[string]string),
	}
	b.pendingPanels = append(b.pendingPanels, panel)
	return panel
}

func (b *subAgentActivityBridge) handleNestedStream(panel *subAgentPanelState, mv *adk.MessageVariant) error {
	if mv == nil || mv.MessageStream == nil {
		return nil
	}
	defer mv.MessageStream.Close()
	switch mv.Role {
	case schema.Assistant:
		toolCallStarted := make(map[string]bool)
		for {
			msg, err := mv.MessageStream.Recv()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			if msg == nil {
				continue
			}
			if err := b.applyNestedAssistantChunk(panel, msg, toolCallStarted); err != nil {
				return err
			}
		}
	case schema.Tool:
		var contentBuilder strings.Builder
		toolCallID := ""
		for {
			msg, err := mv.MessageStream.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			if msg == nil {
				continue
			}
			if toolCallID == "" && strings.TrimSpace(msg.ToolCallID) != "" {
				toolCallID = strings.TrimSpace(msg.ToolCallID)
			}
			if msg.Content != "" {
				contentBuilder.WriteString(msg.Content)
			}
		}
		if contentBuilder.Len() == 0 {
			return nil
		}
		b.appendToolResult(panel, toolCallID, contentBuilder.String())
		return b.publish(panel)
	default:
		for {
			_, err := mv.MessageStream.Recv()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
		}
	}
}

func (b *subAgentActivityBridge) applyNestedMessage(panel *subAgentPanelState, msg *schema.Message) error {
	if panel == nil || msg == nil {
		return nil
	}
	panel.Status = "in_progress"
	if msg.ReasoningContent != "" {
		b.appendReasoning(panel, msg.ReasoningContent)
	}
	if len(msg.ToolCalls) > 0 {
		for _, call := range msg.ToolCalls {
			b.upsertToolCall(panel, call.ID, call.Function.Name, call.Function.Arguments, true)
		}
	}
	if msg.Role == schema.Tool && msg.Content != "" {
		b.appendToolResult(panel, msg.ToolCallID, msg.Content)
	}
	return b.publish(panel)
}

func (b *subAgentActivityBridge) applyNestedAssistantChunk(panel *subAgentPanelState, msg *schema.Message, toolCallStarted map[string]bool) error {
	if panel == nil || msg == nil {
		return nil
	}
	panel.Status = "in_progress"
	if msg.ReasoningContent != "" {
		b.appendReasoning(panel, msg.ReasoningContent)
	}
	if len(msg.ToolCalls) > 0 {
		for _, call := range msg.ToolCalls {
			callID := strings.TrimSpace(call.ID)
			if callID == "" && call.Function.Name == "" {
				var onlyStartedID string
				if len(toolCallStarted) == 1 {
					for id := range toolCallStarted {
						onlyStartedID = id
						break
					}
				}
				if onlyStartedID != "" {
					callID = onlyStartedID
				}
			}
			if callID == "" {
				if call.Function.Name == "" {
					continue
				}
				callID = uuid.NewString()
			}
			if call.Function.Name != "" {
				toolCallStarted[callID] = true
			}
			if !toolCallStarted[callID] {
				continue
			}
			b.upsertToolCall(panel, callID, call.Function.Name, call.Function.Arguments, false)
		}
	}
	return b.publish(panel)
}

func (b *subAgentActivityBridge) appendReasoning(panel *subAgentPanelState, delta string) {
	if panel == nil || delta == "" {
		return
	}
	lastIndex := len(panel.ProcessItems) - 1
	if lastIndex >= 0 && panel.ProcessItems[lastIndex]["type"] == "reasoning" {
		currentSummary, _ := panel.ProcessItems[lastIndex]["summary"].([]map[string]any)
		if len(currentSummary) == 0 {
			currentSummary = []map[string]any{{"type": "summary_text", "text": delta}}
		} else {
			text, _ := currentSummary[0]["text"].(string)
			currentSummary[0]["text"] = text + delta
		}
		panel.ProcessItems[lastIndex]["summary"] = currentSummary
		panel.ProcessItems[lastIndex]["status"] = "in_progress"
		return
	}
	panel.ProcessItems = append(panel.ProcessItems, map[string]any{
		"id":      uuid.NewString(),
		"type":    "reasoning",
		"status":  "in_progress",
		"summary": []map[string]any{{"type": "summary_text", "text": delta}},
	})
}

func (b *subAgentActivityBridge) upsertToolCall(panel *subAgentPanelState, toolCallID, name, arguments string, replaceArgs bool) {
	if panel == nil {
		return
	}
	if toolCallID != "" {
		if index, exists := panel.toolIndexByID[toolCallID]; exists {
			item := panel.ProcessItems[index]
			if name != "" {
				item["name"] = name
			}
			if arguments != "" {
				if replaceArgs {
					item["arguments"] = arguments
				} else {
					prev := panel.toolArgsByID[toolCallID]
					nextArgs := arguments
					if prev != "" && strings.HasPrefix(arguments, prev) {
						nextArgs = arguments
					} else if prev != "" {
						nextArgs = prev + arguments
					}
					item["arguments"] = nextArgs
					panel.toolArgsByID[toolCallID] = nextArgs
				}
			}
			item["status"] = "in_progress"
			return
		}
	}
	item := map[string]any{
		"id":        firstNonEmpty(toolCallID, uuid.NewString()),
		"type":      "function_call",
		"call_id":   toolCallID,
		"name":      name,
		"arguments": arguments,
		"status":    "in_progress",
	}
	panel.ProcessItems = append(panel.ProcessItems, item)
	if toolCallID != "" {
		panel.toolIndexByID[toolCallID] = len(panel.ProcessItems) - 1
		panel.toolArgsByID[toolCallID] = arguments
	}
}

func (b *subAgentActivityBridge) appendToolResult(panel *subAgentPanelState, toolCallID, text string) {
	if panel == nil || strings.TrimSpace(text) == "" {
		return
	}
	if toolCallID != "" {
		if index, exists := panel.toolIndexByID[toolCallID]; exists {
			panel.ProcessItems[index]["status"] = "completed"
		}
	}
	panel.ProcessItems = append(panel.ProcessItems, map[string]any{
		"id":     uuid.NewString(),
		"type":   "tool_result_text",
		"text":   text,
		"status": "completed",
	})
}

func (b *subAgentActivityBridge) publish(panel *subAgentPanelState) error {
	if b == nil || b.sender == nil || panel == nil {
		return nil
	}
	return b.sender.ActivitySnapshot(panel.MessageID, subAgentPanelActivityType, map[string]any{
		"agentName":    panel.AgentName,
		"status":       panel.Status,
		"outputText":   panel.OutputText,
		"processItems": panel.ProcessItems,
	})
}

func topLevelSubAgentName(event *adk.AgentEvent) string {
	if event == nil || len(event.RunPath) < 2 {
		return ""
	}
	return strings.TrimSpace(event.RunPath[1].String())
}

func isSubAgentToolName(name string) bool {
	return strings.HasPrefix(strings.TrimSpace(name), "sub_agent_")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func concatAssistantOutputText(parts []schema.MessageOutputPart) string {
	if len(parts) == 0 {
		return ""
	}
	var builder strings.Builder
	for _, part := range parts {
		if part.Type == schema.ChatMessagePartTypeText && part.Text != "" {
			builder.WriteString(part.Text)
		}
	}
	return builder.String()
}

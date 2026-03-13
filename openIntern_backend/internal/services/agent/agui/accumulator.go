package agui

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/google/uuid"
)

type AccumulatedMessage struct {
	MsgID        string
	Type         string
	Role         string
	Content      any
	ActivityType string
	ToolCallID   string
	ToolName     string
	Metadata     map[string]any
}

type Accumulator struct {
	threadID     string
	runID        string
	messages     map[string]*AccumulatedMessage
	order        []string
	state        any
	customEvents []AccumulatedMessage
	rawEvents    []AccumulatedMessage
	errorEvents  []AccumulatedMessage
}

func NewAccumulator(threadID string) *Accumulator {
	return &Accumulator{
		threadID: threadID,
		messages: map[string]*AccumulatedMessage{},
	}
}

func (a *Accumulator) OnRunStarted(threadID, runID string) {
	if threadID != "" {
		a.threadID = threadID
	}
	if runID != "" {
		a.runID = runID
	}
}

func (a *Accumulator) OnRunFinished(threadID, runID string) {
	if threadID != "" {
		a.threadID = threadID
	}
	if runID != "" {
		a.runID = runID
	}
}

func (a *Accumulator) OnTextMessageStart(msgID, role string) {
	if msgID == "" {
		msgID = a.newEventID()
	}
	msg := a.ensureMessage(msgID, "text")
	msg.Role = role
}

func (a *Accumulator) OnTextMessageContent(msgID, delta string) {
	if msgID == "" {
		msgID = a.newEventID()
	}
	msg := a.ensureMessage(msgID, "text")
	if msg.Content == nil {
		msg.Content = ""
	}
	if s, ok := msg.Content.(string); ok {
		msg.Content = s + delta
	} else {
		msg.Content = fmt.Sprintf("%v%v", msg.Content, delta)
	}
}

func (a *Accumulator) OnTextMessageEnd(msgID string) {
	if msgID == "" {
		return
	}
	a.ensureMessage(msgID, "text")
}

func (a *Accumulator) OnReasoningStart(messageID string) {
	_ = messageID
}

func (a *Accumulator) OnReasoningEnd(messageID string) {
	_ = messageID
}

func (a *Accumulator) OnReasoningMessageStart(messageID, role string) {
	if messageID == "" {
		messageID = a.newEventID()
	}
	msg := a.ensureMessage(messageID, "reasoning_message")
	msg.Role = role
}

func (a *Accumulator) OnReasoningMessageContent(messageID, delta string) {
	if messageID == "" {
		messageID = a.newEventID()
	}
	msg := a.ensureMessage(messageID, "reasoning_message")
	if msg.Content == nil {
		msg.Content = ""
	}
	if s, ok := msg.Content.(string); ok {
		msg.Content = s + delta
	} else {
		msg.Content = fmt.Sprintf("%v%v", msg.Content, delta)
	}
}

func (a *Accumulator) OnReasoningMessageEnd(messageID string) {
	if messageID == "" {
		return
	}
	a.ensureMessage(messageID, "reasoning_message")
}

func (a *Accumulator) OnToolCallStart(toolCallID, toolName string) {
	if toolCallID == "" {
		toolCallID = a.newEventID()
	}
	msg := a.ensureMessage(toolCallID, "tool_call")
	msg.ToolCallID = toolCallID
	msg.ToolName = toolName
}

func (a *Accumulator) OnToolCallArgs(toolCallID, delta string) {
	if toolCallID == "" {
		toolCallID = a.newEventID()
	}
	msg := a.ensureMessage(toolCallID, "tool_call")
	msg.ToolCallID = toolCallID
	if msg.Content == nil {
		msg.Content = ""
	}
	if s, ok := msg.Content.(string); ok {
		msg.Content = s + delta
	} else {
		msg.Content = fmt.Sprintf("%v%v", msg.Content, delta)
	}
}

func (a *Accumulator) OnToolCallEnd(toolCallID string) {
	if toolCallID == "" {
		return
	}
	a.ensureMessage(toolCallID, "tool_call")
}

func (a *Accumulator) OnToolCallResult(msgID, toolCallID, content string) {
	if msgID == "" {
		msgID = a.newEventID()
	}
	msg := a.ensureMessage(msgID, "tool_result")
	msg.ToolCallID = toolCallID
	msg.Content = content
}

func (a *Accumulator) OnStateSnapshot(snapshot any) {
	a.state = deepCopy(snapshot)
}

func (a *Accumulator) OnStateUpdate(delta []events.JSONPatchOperation) {
	a.state = applyPatch(a.state, delta)
}

func (a *Accumulator) OnActivitySnapshot(messageID, activityType string, content any) {
	if messageID == "" {
		messageID = a.newEventID()
	}
	msg := a.ensureMessage(messageID, "activity")
	msg.ActivityType = activityType
	msg.Content = deepCopy(content)
}

func (a *Accumulator) OnActivityUpdate(messageID, activityType string, patch []events.JSONPatchOperation) {
	if messageID == "" {
		messageID = a.newEventID()
	}
	msg := a.ensureMessage(messageID, "activity")
	msg.ActivityType = activityType
	msg.Content = applyPatch(msg.Content, patch)
}

func (a *Accumulator) OnCustom(name string, value any) {
	event := AccumulatedMessage{
		MsgID:    a.newEventID(),
		Type:     "custom",
		Content:  value,
		Metadata: map[string]any{"name": name},
	}
	a.customEvents = append(a.customEvents, event)
}

func (a *Accumulator) OnRaw(event any, source string) {
	eventMsg := AccumulatedMessage{
		MsgID:    a.newEventID(),
		Type:     "raw",
		Content:  event,
		Metadata: map[string]any{},
	}
	if source != "" {
		eventMsg.Metadata["source"] = source
	}
	a.rawEvents = append(a.rawEvents, eventMsg)
}

func (a *Accumulator) OnRunError(message string, code string) {
	event := AccumulatedMessage{
		MsgID:    a.newEventID(),
		Type:     "error",
		Content:  message,
		Metadata: map[string]any{},
	}
	if code != "" {
		event.Metadata["code"] = code
	}
	a.errorEvents = append(a.errorEvents, event)
}

func (a *Accumulator) Flush() []AccumulatedMessage {
	var result []AccumulatedMessage
	for _, id := range a.order {
		msg := a.messages[id]
		if msg == nil {
			continue
		}
		result = append(result, *msg)
	}
	if a.state != nil {
		result = append(result, AccumulatedMessage{
			MsgID:   a.newEventID(),
			Type:    "state",
			Content: a.state,
		})
	}
	result = append(result, a.customEvents...)
	result = append(result, a.rawEvents...)
	result = append(result, a.errorEvents...)
	return result
}

func (a *Accumulator) ensureMessage(msgID, msgType string) *AccumulatedMessage {
	if msgID == "" {
		msgID = a.newEventID()
	}
	if msg, ok := a.messages[msgID]; ok && msg != nil {
		if msg.Type == "" {
			msg.Type = msgType
		}
		return msg
	}
	msg := &AccumulatedMessage{
		MsgID: msgID,
		Type:  msgType,
	}
	a.messages[msgID] = msg
	a.order = append(a.order, msgID)
	return msg
}

func (a *Accumulator) newEventID() string {
	return uuid.NewString()
}

func deepCopy(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return v
	}
	return out
}

func applyPatch(doc any, ops []events.JSONPatchOperation) any {
	out := doc
	for _, op := range ops {
		out = applySinglePatch(out, op)
	}
	return out
}

func applySinglePatch(doc any, op events.JSONPatchOperation) any {
	if op.Path == "" || op.Path == "/" {
		switch op.Op {
		case "add", "replace":
			return op.Value
		case "remove":
			return nil
		}
		return doc
	}

	if doc == nil {
		doc = map[string]any{}
	}

	parts := splitJSONPointer(op.Path)
	if len(parts) == 0 {
		return doc
	}
	return applyPatchAt(doc, parts, op)
}

func splitJSONPointer(path string) []string {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return nil
	}
	parts := strings.Split(path, "/")
	for i := range parts {
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(parts[i], "~1", "/"), "~0", "~")
	}
	return parts
}

func applyPatchAt(target any, parts []string, op events.JSONPatchOperation) any {
	if len(parts) == 0 {
		return target
	}
	if len(parts) == 1 {
		return applyOp(target, parts[0], op)
	}
	switch cur := target.(type) {
	case map[string]any:
		child := cur[parts[0]]
		if child == nil {
			if isSliceKey(parts[1]) {
				child = []any{}
			} else {
				child = map[string]any{}
			}
		}
		cur[parts[0]] = applyPatchAt(child, parts[1:], op)
		return cur
	case []any:
		idx, err := strconv.Atoi(parts[0])
		if err != nil || idx < 0 || idx >= len(cur) {
			return target
		}
		cur[idx] = applyPatchAt(cur[idx], parts[1:], op)
		return cur
	default:
		return target
	}
}

func applyOp(target any, key string, op events.JSONPatchOperation) any {
	switch cur := target.(type) {
	case map[string]any:
		switch op.Op {
		case "add", "replace":
			cur[key] = deepCopy(op.Value)
		case "remove":
			delete(cur, key)
		}
		return cur
	case []any:
		if key == "-" && op.Op == "add" {
			return append(cur, deepCopy(op.Value))
		}
		idx, err := strconv.Atoi(key)
		if err != nil {
			return target
		}
		switch op.Op {
		case "add":
			if idx >= 0 && idx <= len(cur) {
				return append(cur[:idx], append([]any{deepCopy(op.Value)}, cur[idx:]...)...)
			}
		case "replace":
			if idx >= 0 && idx < len(cur) {
				cur[idx] = deepCopy(op.Value)
			}
		case "remove":
			if idx >= 0 && idx < len(cur) {
				return append(cur[:idx], cur[idx+1:]...)
			}
		}
		return cur
	default:
		return target
	}
}

func isSliceKey(key string) bool {
	if key == "-" {
		return true
	}
	_, err := strconv.Atoi(key)
	return err == nil
}

package agui

import (
	"context"
	"io"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/encoding/sse"
)

// Sender provides a convenient interface for Eino framework to send AG-UI events
type Sender struct {
	w        io.Writer
	sse      *sse.SSEWriter
	ctx      context.Context
	runID    string
	threadID string
}

func (s *Sender) ThreadID() string {
	return s.threadID
}

func (s *Sender) RunID() string {
	return s.runID
}

// NewSender creates a new Sender instance with injected context, writer, runID and threadID
func NewSender(ctx context.Context, w io.Writer, runID, threadID string) *Sender {
	return &Sender{
		w:        w,
		sse:      sse.NewSSEWriter(),
		ctx:      ctx,
		runID:    runID,
		threadID: threadID,
	}
}

// NewSenderWithThreadID creates a new Sender instance with injected context, writer, runID and threadID
func NewSenderWithThreadID(ctx context.Context, w io.Writer, threadID string) *Sender {

	runID := events.GenerateRunID()

	return &Sender{
		w:        w,
		sse:      sse.NewSSEWriter(),
		ctx:      ctx,
		runID:    runID,
		threadID: threadID,
	}
}

// Start sends a RunStarted event using the injected runID and threadID
func (s *Sender) Start() error {
	evt := events.NewRunStartedEvent(s.threadID, s.runID)
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// StartMessage sends a TextMessageStart event
func (s *Sender) StartMessage(msgID string, role string) error {
	evt := events.NewTextMessageStartEvent(msgID, events.WithRole(role))
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// SendContent sends a TextMessageContent event (delta)
func (s *Sender) SendContent(msgID, delta string) error {
	evt := events.NewTextMessageContentEvent(msgID, delta)
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// EndMessage sends a TextMessageEnd event
func (s *Sender) EndMessage(msgID string) error {
	evt := events.NewTextMessageEndEvent(msgID)
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// StartThinking sends a ThinkingStart event
func (s *Sender) StartThinking(title string) error {
	evt := events.NewThinkingStartEvent().WithTitle(title)
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// EndThinking sends a ThinkingEnd event
func (s *Sender) EndThinking() error {
	evt := events.NewThinkingEndEvent()
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// StartThinkingMessage sends a ThinkingTextMessageStart event
func (s *Sender) StartThinkingMessage() error {
	evt := events.NewThinkingTextMessageStartEvent()
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// ThinkingContent sends a ThinkingTextMessageContent event
func (s *Sender) ThinkingContent(delta string) error {
	evt := events.NewThinkingTextMessageContentEvent(delta)
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// EndThinkingMessage sends a ThinkingTextMessageEnd event
func (s *Sender) EndThinkingMessage() error {
	evt := events.NewThinkingTextMessageEndEvent()
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// StartToolCall sends a ToolCallStart event
func (s *Sender) StartToolCall(toolCallID, toolName string) error {
	evt := events.NewToolCallStartEvent(toolCallID, toolName)
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// ToolCallArgs sends a ToolCallArgs event
func (s *Sender) ToolCallArgs(toolCallID, delta string) error {
	evt := events.NewToolCallArgsEvent(toolCallID, delta)
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// EndToolCall sends a ToolCallEnd event
func (s *Sender) EndToolCall(toolCallID string) error {
	evt := events.NewToolCallEndEvent(toolCallID)
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// ToolCallResult sends a ToolCallResult event
func (s *Sender) ToolCallResult(msgID, toolCallID, content string) error {
	evt := events.NewToolCallResultEvent(msgID, toolCallID, content)
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// StateSnapshot sends a StateSnapshot event
func (s *Sender) StateSnapshot(snapshot any) error {
	evt := events.NewStateSnapshotEvent(snapshot)
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// StateUpdate sends a StateDelta event
func (s *Sender) StateUpdate(delta []events.JSONPatchOperation) error {
	evt := events.NewStateDeltaEvent(delta)
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// ActivitySnapshot sends an ActivitySnapshot event
func (s *Sender) ActivitySnapshot(messageID, activityType string, content any) error {
	evt := events.NewActivitySnapshotEvent(messageID, activityType, content)
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// ActivityUpdate sends an ActivityDelta event
func (s *Sender) ActivityUpdate(messageID, activityType string, patch []events.JSONPatchOperation) error {
	evt := events.NewActivityDeltaEvent(messageID, activityType, patch)
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// SendA2UI sends an A2UI message using ActivitySnapshot event with "a2ui-surface" activity type
func (s *Sender) SendA2UI(messageID string, content any) error {
	return s.ActivitySnapshot(messageID, "a2ui-surface", content)
}

// UpdateA2UI sends an A2UI update using ActivityDelta event with "a2ui-surface" activity type
func (s *Sender) UpdateA2UI(messageID string, patch []events.JSONPatchOperation) error {
	return s.ActivityUpdate(messageID, "a2ui-surface", patch)
}

// Custom sends a Custom event
func (s *Sender) Custom(name string, value any) error {
	opts := []events.CustomEventOption{}
	if value != nil {
		opts = append(opts, events.WithValue(value))
	}
	evt := events.NewCustomEvent(name, opts...)
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// Raw sends a Raw event
func (s *Sender) Raw(event any, source string) error {
	opts := []events.RawEventOption{}
	if source != "" {
		opts = append(opts, events.WithSource(source))
	}
	evt := events.NewRawEvent(event, opts...)
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// Finish sends a RunFinished event using the injected runID and threadID
func (s *Sender) Finish() error {
	evt := events.NewRunFinishedEvent(s.threadID, s.runID)
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

// Error sends a RunError event using the injected runID
func (s *Sender) Error(message string, code string) error {
	opts := []events.RunErrorOption{}
	if code != "" {
		opts = append(opts, events.WithErrorCode(code))
	}
	// Always use the injected runID
	if s.runID != "" {
		opts = append(opts, events.WithRunID(s.runID))
	}

	evt := events.NewRunErrorEvent(message, opts...)
	return s.sse.WriteEvent(s.ctx, s.w, evt)
}

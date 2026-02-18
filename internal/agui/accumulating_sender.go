package agui

import "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"

type AccumulatingSender struct {
	base *Sender
	acc  *Accumulator
}

func NewAccumulatingSender(base *Sender, acc *Accumulator) *AccumulatingSender {
	return &AccumulatingSender{
		base: base,
		acc:  acc,
	}
}

func (s *AccumulatingSender) Start() error {
	if s.acc != nil {
		s.acc.OnRunStarted(s.base.ThreadID(), s.base.RunID())
	}
	return s.base.Start()
}

func (s *AccumulatingSender) StartMessage(msgID string, role string) error {
	if s.acc != nil {
		s.acc.OnTextMessageStart(msgID, role)
	}
	return s.base.StartMessage(msgID, role)
}

func (s *AccumulatingSender) SendContent(msgID, delta string) error {
	if s.acc != nil {
		s.acc.OnTextMessageContent(msgID, delta)
	}
	return s.base.SendContent(msgID, delta)
}

func (s *AccumulatingSender) EndMessage(msgID string) error {
	if s.acc != nil {
		s.acc.OnTextMessageEnd(msgID)
	}
	return s.base.EndMessage(msgID)
}

func (s *AccumulatingSender) StartThinking(title string) error {
	if s.acc != nil {
		s.acc.OnThinkingStart(title)
	}
	return s.base.StartThinking(title)
}

func (s *AccumulatingSender) EndThinking() error {
	if s.acc != nil {
		s.acc.OnThinkingEnd()
	}
	return s.base.EndThinking()
}

func (s *AccumulatingSender) StartThinkingMessage() error {
	if s.acc != nil {
		s.acc.OnThinkingMessageStart()
	}
	return s.base.StartThinkingMessage()
}

func (s *AccumulatingSender) ThinkingContent(delta string) error {
	if s.acc != nil {
		s.acc.OnThinkingMessageContent(delta)
	}
	return s.base.ThinkingContent(delta)
}

func (s *AccumulatingSender) EndThinkingMessage() error {
	if s.acc != nil {
		s.acc.OnThinkingMessageEnd()
	}
	return s.base.EndThinkingMessage()
}

func (s *AccumulatingSender) StartToolCall(toolCallID, toolName string) error {
	if s.acc != nil {
		s.acc.OnToolCallStart(toolCallID, toolName)
	}
	return s.base.StartToolCall(toolCallID, toolName)
}

func (s *AccumulatingSender) ToolCallArgs(toolCallID, delta string) error {
	if s.acc != nil {
		s.acc.OnToolCallArgs(toolCallID, delta)
	}
	return s.base.ToolCallArgs(toolCallID, delta)
}

func (s *AccumulatingSender) EndToolCall(toolCallID string) error {
	if s.acc != nil {
		s.acc.OnToolCallEnd(toolCallID)
	}
	return s.base.EndToolCall(toolCallID)
}

func (s *AccumulatingSender) ToolCallResult(msgID, toolCallID, content string) error {
	if s.acc != nil {
		s.acc.OnToolCallResult(msgID, toolCallID, content)
	}
	return s.base.ToolCallResult(msgID, toolCallID, content)
}

func (s *AccumulatingSender) StateSnapshot(snapshot any) error {
	if s.acc != nil {
		s.acc.OnStateSnapshot(snapshot)
	}
	return s.base.StateSnapshot(snapshot)
}

func (s *AccumulatingSender) StateUpdate(delta []events.JSONPatchOperation) error {
	if s.acc != nil {
		s.acc.OnStateUpdate(delta)
	}
	return s.base.StateUpdate(delta)
}

func (s *AccumulatingSender) ActivitySnapshot(messageID, activityType string, content any) error {
	if s.acc != nil {
		s.acc.OnActivitySnapshot(messageID, activityType, content)
	}
	return s.base.ActivitySnapshot(messageID, activityType, content)
}

func (s *AccumulatingSender) ActivityUpdate(messageID, activityType string, patch []events.JSONPatchOperation) error {
	if s.acc != nil {
		s.acc.OnActivityUpdate(messageID, activityType, patch)
	}
	return s.base.ActivityUpdate(messageID, activityType, patch)
}

func (s *AccumulatingSender) SendA2UI(messageID string, content any) error {
	return s.ActivitySnapshot(messageID, "a2ui-surface", content)
}

func (s *AccumulatingSender) UpdateA2UI(messageID string, patch []events.JSONPatchOperation) error {
	return s.ActivityUpdate(messageID, "a2ui-surface", patch)
}

func (s *AccumulatingSender) Custom(name string, value any) error {
	if s.acc != nil {
		s.acc.OnCustom(name, value)
	}
	return s.base.Custom(name, value)
}

func (s *AccumulatingSender) Raw(event any, source string) error {
	if s.acc != nil {
		s.acc.OnRaw(event, source)
	}
	return s.base.Raw(event, source)
}

func (s *AccumulatingSender) Finish() error {
	if s.acc != nil {
		s.acc.OnRunFinished(s.base.ThreadID(), s.base.RunID())
	}
	return s.base.Finish()
}

func (s *AccumulatingSender) Error(message string, code string) error {
	if s.acc != nil {
		s.acc.OnRunError(message, code)
	}
	return s.base.Error(message, code)
}

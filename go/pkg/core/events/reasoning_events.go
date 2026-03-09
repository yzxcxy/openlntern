package events

import (
	"encoding/json"
	"fmt"
)

type ReasoningStartEvent struct {
	*BaseEvent
	MessageID string `json:"messageId"`
}

func NewReasoningStartEvent(messageID string) *ReasoningStartEvent {
	return &ReasoningStartEvent{
		BaseEvent: NewBaseEvent(EventTypeReasoningStart),
		MessageID: messageID,
	}
}

func (e *ReasoningStartEvent) Validate() error {
	if err := e.BaseEvent.Validate(); err != nil {
		return err
	}
	if e.MessageID == "" {
		return fmt.Errorf("ReasoningStartEvent validation failed: messageId field is required")
	}
	return nil
}

func (e *ReasoningStartEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

type ReasoningEndEvent struct {
	*BaseEvent
	MessageID string `json:"messageId"`
}

func NewReasoningEndEvent(messageID string) *ReasoningEndEvent {
	return &ReasoningEndEvent{
		BaseEvent: NewBaseEvent(EventTypeReasoningEnd),
		MessageID: messageID,
	}
}

func (e *ReasoningEndEvent) Validate() error {
	if err := e.BaseEvent.Validate(); err != nil {
		return err
	}
	if e.MessageID == "" {
		return fmt.Errorf("ReasoningEndEvent validation failed: messageId field is required")
	}
	return nil
}

func (e *ReasoningEndEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

type ReasoningMessageStartEvent struct {
	*BaseEvent
	MessageID string  `json:"messageId"`
	Role      *string `json:"role,omitempty"`
}

type ReasoningMessageStartOption func(*ReasoningMessageStartEvent)

func WithReasoningMessageRole(role string) ReasoningMessageStartOption {
	return func(e *ReasoningMessageStartEvent) {
		e.Role = &role
	}
}

func NewReasoningMessageStartEvent(messageID string, options ...ReasoningMessageStartOption) *ReasoningMessageStartEvent {
	event := &ReasoningMessageStartEvent{
		BaseEvent: NewBaseEvent(EventTypeReasoningMessageStart),
		MessageID: messageID,
	}
	for _, opt := range options {
		opt(event)
	}
	return event
}

func (e *ReasoningMessageStartEvent) Validate() error {
	if err := e.BaseEvent.Validate(); err != nil {
		return err
	}
	if e.MessageID == "" {
		return fmt.Errorf("ReasoningMessageStartEvent validation failed: messageId field is required")
	}
	return nil
}

func (e *ReasoningMessageStartEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

type ReasoningMessageContentEvent struct {
	*BaseEvent
	MessageID string `json:"messageId"`
	Delta     string `json:"delta"`
}

func NewReasoningMessageContentEvent(messageID, delta string) *ReasoningMessageContentEvent {
	return &ReasoningMessageContentEvent{
		BaseEvent: NewBaseEvent(EventTypeReasoningMessageContent),
		MessageID: messageID,
		Delta:     delta,
	}
}

func (e *ReasoningMessageContentEvent) Validate() error {
	if err := e.BaseEvent.Validate(); err != nil {
		return err
	}
	if e.MessageID == "" {
		return fmt.Errorf("ReasoningMessageContentEvent validation failed: messageId field is required")
	}
	if e.Delta == "" {
		return fmt.Errorf("ReasoningMessageContentEvent validation failed: delta field is required")
	}
	return nil
}

func (e *ReasoningMessageContentEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

type ReasoningMessageEndEvent struct {
	*BaseEvent
	MessageID string `json:"messageId"`
}

func NewReasoningMessageEndEvent(messageID string) *ReasoningMessageEndEvent {
	return &ReasoningMessageEndEvent{
		BaseEvent: NewBaseEvent(EventTypeReasoningMessageEnd),
		MessageID: messageID,
	}
}

func (e *ReasoningMessageEndEvent) Validate() error {
	if err := e.BaseEvent.Validate(); err != nil {
		return err
	}
	if e.MessageID == "" {
		return fmt.Errorf("ReasoningMessageEndEvent validation failed: messageId field is required")
	}
	return nil
}

func (e *ReasoningMessageEndEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

type ReasoningMessageChunkEvent struct {
	*BaseEvent
	MessageID *string `json:"messageId,omitempty"`
	Delta     *string `json:"delta,omitempty"`
}

func NewReasoningMessageChunkEvent(messageID, delta *string) *ReasoningMessageChunkEvent {
	return &ReasoningMessageChunkEvent{
		BaseEvent: NewBaseEvent(EventTypeReasoningMessageChunk),
		MessageID: messageID,
		Delta:     delta,
	}
}

func (e *ReasoningMessageChunkEvent) Validate() error {
	if err := e.BaseEvent.Validate(); err != nil {
		return err
	}
	if e.MessageID == nil && e.Delta == nil {
		return fmt.Errorf("ReasoningMessageChunkEvent validation failed: at least one of messageId or delta must be present")
	}
	return nil
}

func (e *ReasoningMessageChunkEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

type ReasoningEncryptedValueEvent struct {
	*BaseEvent
	Subtype        string `json:"subtype"`
	EntityID       string `json:"entityId"`
	EncryptedValue string `json:"encryptedValue"`
}

func NewReasoningEncryptedValueEvent(subtype, entityID, encryptedValue string) *ReasoningEncryptedValueEvent {
	return &ReasoningEncryptedValueEvent{
		BaseEvent:      NewBaseEvent(EventTypeReasoningEncryptedValue),
		Subtype:        subtype,
		EntityID:       entityID,
		EncryptedValue: encryptedValue,
	}
}

func (e *ReasoningEncryptedValueEvent) Validate() error {
	if err := e.BaseEvent.Validate(); err != nil {
		return err
	}
	if e.Subtype == "" {
		return fmt.Errorf("ReasoningEncryptedValueEvent validation failed: subtype field is required")
	}
	if e.Subtype != "message" && e.Subtype != "tool-call" {
		return fmt.Errorf("ReasoningEncryptedValueEvent validation failed: subtype must be message or tool-call")
	}
	if e.EntityID == "" {
		return fmt.Errorf("ReasoningEncryptedValueEvent validation failed: entityId field is required")
	}
	if e.EncryptedValue == "" {
		return fmt.Errorf("ReasoningEncryptedValueEvent validation failed: encryptedValue field is required")
	}
	return nil
}

func (e *ReasoningEncryptedValueEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

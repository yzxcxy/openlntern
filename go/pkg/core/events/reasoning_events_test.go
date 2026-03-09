package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReasoningEvents(t *testing.T) {
	t.Run("ReasoningStartEvent", func(t *testing.T) {
		messageID := "reasoning-1"
		event := NewReasoningStartEvent(messageID)

		assert.Equal(t, EventTypeReasoningStart, event.Type())
		assert.Equal(t, messageID, event.MessageID)
		assert.NoError(t, event.Validate())

		event.MessageID = ""
		assert.Error(t, event.Validate())
	})

	t.Run("ReasoningEndEvent", func(t *testing.T) {
		messageID := "reasoning-1"
		event := NewReasoningEndEvent(messageID)

		assert.Equal(t, EventTypeReasoningEnd, event.Type())
		assert.Equal(t, messageID, event.MessageID)
		assert.NoError(t, event.Validate())

		event.MessageID = ""
		assert.Error(t, event.Validate())
	})

	t.Run("ReasoningMessageStartEvent", func(t *testing.T) {
		messageID := "msg-1"
		role := "assistant"

		event := NewReasoningMessageStartEvent(messageID, WithReasoningMessageRole(role))

		assert.Equal(t, EventTypeReasoningMessageStart, event.Type())
		assert.Equal(t, messageID, event.MessageID)
		require.NotNil(t, event.Role)
		assert.Equal(t, role, *event.Role)
		assert.NoError(t, event.Validate())

		event.MessageID = ""
		assert.Error(t, event.Validate())
	})

	t.Run("ReasoningMessageContentEvent", func(t *testing.T) {
		messageID := "msg-1"
		delta := "思考中"

		event := NewReasoningMessageContentEvent(messageID, delta)

		assert.Equal(t, EventTypeReasoningMessageContent, event.Type())
		assert.Equal(t, messageID, event.MessageID)
		assert.Equal(t, delta, event.Delta)
		assert.NoError(t, event.Validate())

		event.MessageID = ""
		assert.Error(t, event.Validate())

		event.MessageID = messageID
		event.Delta = ""
		assert.Error(t, event.Validate())
	})

	t.Run("ReasoningMessageEndEvent", func(t *testing.T) {
		messageID := "msg-1"
		event := NewReasoningMessageEndEvent(messageID)

		assert.Equal(t, EventTypeReasoningMessageEnd, event.Type())
		assert.Equal(t, messageID, event.MessageID)
		assert.NoError(t, event.Validate())

		event.MessageID = ""
		assert.Error(t, event.Validate())
	})

	t.Run("ReasoningMessageChunkEvent", func(t *testing.T) {
		messageID := "msg-1"
		delta := ""
		event := NewReasoningMessageChunkEvent(&messageID, &delta)

		assert.Equal(t, EventTypeReasoningMessageChunk, event.Type())
		require.NotNil(t, event.MessageID)
		require.NotNil(t, event.Delta)
		assert.Equal(t, messageID, *event.MessageID)
		assert.Equal(t, delta, *event.Delta)
		assert.NoError(t, event.Validate())

		event.MessageID = nil
		event.Delta = nil
		assert.Error(t, event.Validate())
	})

	t.Run("ReasoningEncryptedValueEvent", func(t *testing.T) {
		event := NewReasoningEncryptedValueEvent("message", "msg-1", "encrypted")

		assert.Equal(t, EventTypeReasoningEncryptedValue, event.Type())
		assert.Equal(t, "message", event.Subtype)
		assert.Equal(t, "msg-1", event.EntityID)
		assert.Equal(t, "encrypted", event.EncryptedValue)
		assert.NoError(t, event.Validate())

		event.Subtype = "invalid"
		assert.Error(t, event.Validate())
	})
}

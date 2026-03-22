package agent

import (
	"context"
	"fmt"
	"strings"
)

type runtimeContextKey string

const ownerIDRuntimeContextKey runtimeContextKey = "openintern_agent_owner_id"

// WithOwnerID attaches the authenticated owner identifier to the runtime context.
func WithOwnerID(ctx context.Context, ownerID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ownerIDRuntimeContextKey, strings.TrimSpace(ownerID))
}

// ownerIDFromContext extracts the authenticated owner identifier for agent mode resolution.
func ownerIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value := ctx.Value(ownerIDRuntimeContextKey)
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", value))
}

// isAgentConversationMode returns whether the current runtime config requested agent mode.
func isAgentConversationMode(runtimeConfig *AgentRuntimeConfig) bool {
	return runtimeConfig != nil && strings.EqualFold(strings.TrimSpace(runtimeConfig.Conversation.Mode), "agent")
}

// selectedAgentIDFromRuntimeConfig extracts the selected agent id from forwarded features.
func selectedAgentIDFromRuntimeConfig(runtimeConfig *AgentRuntimeConfig) string {
	if runtimeConfig == nil || len(runtimeConfig.Features) == 0 {
		return ""
	}
	value, ok := runtimeConfig.Features["selectedAgentId"]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", value))
}

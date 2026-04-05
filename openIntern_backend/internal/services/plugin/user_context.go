package plugin

import (
	"context"
	"strings"

	builtinTool "openIntern/internal/services/builtin_tool"
)

// userIDFromContext extracts the authenticated user identifier for user-scoped plugin runtime resolution.
func userIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	userID, _ := ctx.Value(builtinTool.ContextKeyUserID).(string)
	return strings.TrimSpace(userID)
}

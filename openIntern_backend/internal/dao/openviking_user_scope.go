package dao

import (
	"context"
	"errors"
	"strings"
)

type openVikingUserContextKey string

const openVikingUserIDContextKey openVikingUserContextKey = "openviking_user_id"

// WithOpenVikingUserID stores the authenticated openIntern user id for downstream OpenViking path builders.
func WithOpenVikingUserID(ctx context.Context, userID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, openVikingUserIDContextKey, strings.TrimSpace(userID))
}

// OpenVikingUserIDFromContext returns the authenticated openIntern user id required for user-scoped OpenViking paths.
func OpenVikingUserIDFromContext(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", errors.New("openviking user id is required")
	}
	value, _ := ctx.Value(openVikingUserIDContextKey).(string)
	userID := strings.TrimSpace(value)
	if userID == "" {
		return "", errors.New("openviking user id is required")
	}
	return userID, nil
}

// UserMemoryRootURI returns the user-private long-term memory root.
func UserMemoryRootURI(userID string) string {
	return "viking://user/" + strings.TrimSpace(userID) + "/memories/"
}

// UserSkillRootURI returns the user-private skills root.
func UserSkillRootURI(userID string) string {
	return "viking://user/" + strings.TrimSpace(userID) + "/skills"
}

// UserKnowledgeBaseRootURI returns the user-private knowledge base root.
func UserKnowledgeBaseRootURI(userID string) string {
	return "viking://resources/users/" + strings.TrimSpace(userID) + "/kbs/"
}

// UserKnowledgeBaseURI returns one user-private knowledge base root URI.
func UserKnowledgeBaseURI(userID, kbName string) string {
	return strings.TrimRight(UserKnowledgeBaseRootURI(userID), "/") + "/" + strings.Trim(kbName, "/") + "/"
}

// UserKnowledgeBaseInnerURI returns the imported content root inside one user-private knowledge base.
func UserKnowledgeBaseInnerURI(userID, kbName string) string {
	return strings.TrimRight(UserKnowledgeBaseURI(userID, kbName), "/") + "/" + strings.Trim(kbName, "/") + "/"
}

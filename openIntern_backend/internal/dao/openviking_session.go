package dao

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"openIntern/internal/database"
)

// OpenVikingSessionDAO provides persistence methods for OpenViking sessions.
type OpenVikingSessionDAO struct{}

// OpenVikingSession is the shared DAO singleton.
var OpenVikingSession = new(OpenVikingSessionDAO)

// Configured reports whether the OpenViking context store is available for session APIs.
func (d *OpenVikingSessionDAO) Configured() bool {
	return contextStoreReady()
}

// AddMessage appends a plain-text user or assistant message to an OpenViking session.
func (d *OpenVikingSessionDAO) AddMessage(ctx context.Context, sessionID, role, content string) error {
	sessionID = strings.TrimSpace(sessionID)
	role = strings.TrimSpace(role)
	content = strings.TrimSpace(content)
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if role == "" {
		return fmt.Errorf("role is required")
	}
	if content == "" {
		return fmt.Errorf("content is required")
	}
	path := "/api/v1/sessions/" + url.PathEscape(sessionID) + "/messages"
	body, err := database.Context.Post(ctx, path, map[string]any{
		"role":    role,
		"content": content,
	})
	if err != nil {
		return err
	}
	return decodeStoreResult(body, nil)
}

// Commit triggers OpenViking memory extraction for the specified session.
// It runs in asynchronous mode (`wait=false`) to avoid blocking backend worker loops.
func (d *OpenVikingSessionDAO) Commit(ctx context.Context, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	path := "/api/v1/sessions/" + url.PathEscape(sessionID) + "/commit?wait=false"
	body, err := database.Context.Post(ctx, path, map[string]any{})
	if err != nil {
		return err
	}
	return decodeStoreResult(body, nil)
}

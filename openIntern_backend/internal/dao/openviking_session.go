package dao

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"openIntern/internal/database"
)

type openVikingSessionCreateResult struct {
	SessionID string `json:"session_id"`
}

// OpenVikingSessionDAO provides persistence methods for OpenViking sessions.
type OpenVikingSessionDAO struct{}

// OpenVikingSession is the shared DAO singleton.
var OpenVikingSession = new(OpenVikingSessionDAO)

// Configured reports whether the OpenViking context store is available for session APIs.
func (d *OpenVikingSessionDAO) Configured() bool {
	return contextStoreReady()
}

// Create creates a deterministic OpenViking session for the provided session id.
func (d *OpenVikingSessionDAO) Create(ctx context.Context, sessionID string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", fmt.Errorf("session_id is required")
	}
	body, err := database.Context.Post(ctx, "/api/v1/sessions", map[string]any{
		"session_id": sessionID,
	})
	if err != nil {
		return "", err
	}
	var result openVikingSessionCreateResult
	if err := decodeStoreResult(body, &result); err != nil {
		return "", err
	}
	if strings.TrimSpace(result.SessionID) != "" {
		return strings.TrimSpace(result.SessionID), nil
	}
	return sessionID, nil
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

// UsedContexts records the OpenViking context URIs that were actually used to answer the current turn.
func (d *OpenVikingSessionDAO) UsedContexts(ctx context.Context, sessionID string, contexts []string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	filtered := make([]string, 0, len(contexts))
	for _, item := range contexts {
		uri := strings.TrimSpace(item)
		if uri == "" {
			continue
		}
		filtered = append(filtered, uri)
	}
	if len(filtered) == 0 {
		return nil
	}
	path := "/api/v1/sessions/" + url.PathEscape(sessionID) + "/used"
	body, err := database.Context.Post(ctx, path, map[string]any{
		"contexts": filtered,
	})
	if err != nil {
		return err
	}
	return decodeStoreResult(body, nil)
}

// Commit triggers OpenViking memory extraction for the specified session.
func (d *OpenVikingSessionDAO) Commit(ctx context.Context, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	path := "/api/v1/sessions/" + url.PathEscape(sessionID) + "/commit"
	body, err := database.Context.Post(ctx, path, map[string]any{})
	if err != nil {
		return err
	}
	return decodeStoreResult(body, nil)
}

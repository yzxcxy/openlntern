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

// OpenVikingCommitSubmitResult captures the async commit acceptance payload.
type OpenVikingCommitSubmitResult struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
	TaskID    string `json:"task_id"`
	Message   string `json:"message"`
}

// OpenVikingTaskResult captures a background task status snapshot from /api/v1/tasks/{task_id}.
type OpenVikingTaskResult struct {
	TaskID     string         `json:"task_id"`
	TaskType   string         `json:"task_type"`
	Status     string         `json:"status"`
	ResourceID string         `json:"resource_id"`
	Result     map[string]any `json:"result"`
	Error      string         `json:"error"`
}

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
// It always runs in asynchronous mode (`wait=false`) and returns a task_id for status polling.
func (d *OpenVikingSessionDAO) Commit(ctx context.Context, sessionID string) (*OpenVikingCommitSubmitResult, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	path := "/api/v1/sessions/" + url.PathEscape(sessionID) + "/commit?wait=false"
	body, err := database.Context.Post(ctx, path, map[string]any{})
	if err != nil {
		return nil, err
	}
	var result OpenVikingCommitSubmitResult
	if err := decodeStoreResult(body, &result); err != nil {
		return nil, err
	}
	if strings.TrimSpace(result.TaskID) == "" {
		return nil, fmt.Errorf("openviking commit response missing task_id")
	}
	if strings.TrimSpace(result.SessionID) == "" {
		result.SessionID = sessionID
	}
	return &result, nil
}

// GetTask reads one background task state produced by async session commit.
func (d *OpenVikingSessionDAO) GetTask(ctx context.Context, taskID string) (*OpenVikingTaskResult, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	path := "/api/v1/tasks/" + url.PathEscape(taskID)
	body, err := database.Context.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}
	var result OpenVikingTaskResult
	if err := decodeStoreResult(body, &result); err != nil {
		return nil, err
	}
	if strings.TrimSpace(result.TaskID) == "" {
		result.TaskID = taskID
	}
	return &result, nil
}

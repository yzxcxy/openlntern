package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"openIntern/internal/config"
	"openIntern/internal/dao"
	"openIntern/internal/models"

	agtypes "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
)

const (
	defaultMemorySyncPollInterval = 3 * time.Second
	defaultMemorySyncBatchSize    = 10
	defaultMemorySyncDelay        = 5 * time.Minute
	defaultMemorySyncTimeout      = 10 * time.Minute
	defaultMemorySyncRetryDelay   = 30 * time.Second
)

const (
	memorySyncErrCommitSubmitFailedPrefix = "memory_sync_commit_submit_failed: "
	memorySyncErrCommitTaskFailedPrefix   = "memory_sync_commit_task_failed: "
	memorySyncErrCommitTaskStatusPrefix   = "memory_sync_commit_task_status_invalid: "
)

type sessionSyncMessage struct {
	MsgID   string
	Role    string
	Content string
}

var memorySyncWorkerOnce sync.Once
var (
	memorySyncPollInterval = defaultMemorySyncPollInterval
	memorySyncDelay        = defaultMemorySyncDelay
	memorySyncTimeout      = defaultMemorySyncTimeout
	memorySyncRetryDelay   = defaultMemorySyncRetryDelay
)

// InitMemorySync configures and starts the in-process worker that forwards completed thread deltas to OpenViking.
func InitMemorySync(cfg config.OpenVikingConfig) {
	if !dao.OpenVikingSession.Configured() {
		return
	}
	applyMemorySyncConfig(cfg)
	memorySyncWorkerOnce.Do(func() {
		go runMemorySyncWorker()
	})
}

// runMemorySyncWorker polls the database for threads that need long-term memory synchronization.
func runMemorySyncWorker() {
	if memorySyncPollInterval <= 0 {
		return
	}
	ticker := time.NewTicker(memorySyncPollInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := ProcessPendingMemorySyncStates(context.Background(), defaultMemorySyncBatchSize); err != nil {
			log.Printf("memory sync worker failed err=%v", err)
		}
	}
}

// ProcessPendingMemorySyncStates executes one bounded batch of memory synchronization work.
func ProcessPendingMemorySyncStates(ctx context.Context, limit int) error {
	items, err := MemorySyncState.ListRunnable(limit)
	if err != nil {
		return err
	}
	for _, item := range items {
		claimed, err := MemorySyncState.MarkSyncing(item.ThreadID)
		if err != nil {
			log.Printf("memory sync mark syncing failed thread_id=%s err=%v", item.ThreadID, err)
			continue
		}
		if !claimed {
			continue
		}
		if err := syncThreadMemoryState(ctx, item); err != nil {
			log.Printf("memory sync process failed thread_id=%s err=%v", item.ThreadID, err)
			nextAttemptAt := nextMemorySyncAttemptAt(time.Now())
			commitStatus := strings.TrimSpace(item.CommitStatus)
			if commitStatus == "" {
				commitStatus = models.MemoryCommitStatusPending
			}
			if isMemorySyncCommitFatalError(err) {
				commitStatus = models.MemoryCommitStatusFailed
			}
			if markErr := MemorySyncState.MarkFailed(item.ThreadID, err.Error(), nextAttemptAt, commitStatus); markErr != nil {
				log.Printf("memory sync mark failed failed thread_id=%s err=%v", item.ThreadID, markErr)
			}
		}
	}
	return nil
}

// syncThreadMemoryState synchronizes the unsent thread message delta into an OpenViking session.
func syncThreadMemoryState(ctx context.Context, state models.MemorySyncState) error {
	runCtx := ctx
	if runCtx == nil {
		runCtx = context.Background()
	}
	if _, hasDeadline := runCtx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(runCtx, memorySyncTimeout)
		defer cancel()
	}
	if strings.TrimSpace(state.CommitTaskID) != "" {
		return pollSubmittedCommitTask(runCtx, state)
	}

	threadMessages, err := Message.ListThreadMessages(state.ThreadID)
	if err != nil {
		return err
	}
	pendingUsageLogs, err := MemoryUsageLog.ListPendingByThreadID(state.ThreadID)
	if err != nil {
		return err
	}
	usedContextURIs, usageLogIDs := collectPendingMemoryUsageContextURIs(pendingUsageLogs)
	addCursor := strings.TrimSpace(state.LastAddedMsgID)
	if addCursor == "" {
		addCursor = strings.TrimSpace(state.LastSyncedMsgID)
	}
	sessionMessages, lastMsgID, err := buildSessionSyncMessages(threadMessages, addCursor)
	if err != nil {
		return err
	}
	hasPendingCommit := strings.TrimSpace(state.LastAddedMsgID) != "" && strings.TrimSpace(state.LastAddedMsgID) != strings.TrimSpace(state.LastSyncedMsgID)
	if lastMsgID == "" && len(usedContextURIs) == 0 {
		if !hasPendingCommit {
			return MemorySyncState.MarkReady(state.ThreadID, state.LastSyncedMsgID, state.LastCommittedRunID)
		}
	}
	if len(sessionMessages) == 0 && len(usedContextURIs) == 0 && !hasPendingCommit {
		return MemorySyncState.MarkReady(state.ThreadID, lastMsgID, state.LastCommittedRunID)
	}
	if len(sessionMessages) == 0 && !hasPendingCommit {
		if err := MemoryUsageLog.MarkReportedByIDs(usageLogIDs); err != nil {
			return err
		}
		return MemorySyncState.MarkReady(state.ThreadID, state.LastSyncedMsgID, state.LastCommittedRunID)
	}

	sessionID, err := ensureOpenVikingSession(state)
	if err != nil {
		return err
	}
	for _, item := range sessionMessages {
		if err := dao.OpenVikingSession.AddMessage(runCtx, sessionID, item.Role, item.Content); err != nil {
			return err
		}
		if msgID := strings.TrimSpace(item.MsgID); msgID != "" {
			addCursor = msgID
			if err := MemorySyncState.MarkMessagesAdded(state.ThreadID, addCursor); err != nil {
				return err
			}
		}
	}
	commitResult, err := dao.OpenVikingSession.Commit(runCtx, sessionID)
	if err != nil {
		return fmt.Errorf("%s%w", memorySyncErrCommitSubmitFailedPrefix, err)
	}
	taskID := ""
	if commitResult != nil {
		taskID = strings.TrimSpace(commitResult.TaskID)
	}
	if taskID == "" {
		return fmt.Errorf("%sopenviking returned empty task_id", memorySyncErrCommitSubmitFailedPrefix)
	}
	if err := MemoryUsageLog.MarkReportedByIDs(usageLogIDs); err != nil {
		return err
	}
	if err := MemorySyncState.MarkCommitSubmitted(state.ThreadID, taskID, nextMemorySyncPollAt(time.Now())); err != nil {
		return err
	}
	return nil
}

// pollSubmittedCommitTask checks one async commit task state and advances thread sync cursor only after completion.
func pollSubmittedCommitTask(ctx context.Context, state models.MemorySyncState) error {
	taskID := strings.TrimSpace(state.CommitTaskID)
	if taskID == "" {
		return nil
	}
	task, err := dao.OpenVikingSession.GetTask(ctx, taskID)
	if err != nil {
		return err
	}
	taskStatus := strings.ToLower(strings.TrimSpace(task.Status))
	switch taskStatus {
	case models.MemorySyncStatusPending, "running":
		return MemorySyncState.MarkCommitPolling(state.ThreadID, taskStatus, nextMemorySyncPollAt(time.Now()))
	case "completed":
		finalCursor := strings.TrimSpace(state.LastAddedMsgID)
		if finalCursor == "" {
			finalCursor = strings.TrimSpace(state.LastSyncedMsgID)
		}
		return MemorySyncState.MarkReady(state.ThreadID, finalCursor, state.LastCommittedRunID)
	case models.MemorySyncStatusFailed:
		if err := MemorySyncState.ClearCommitTask(state.ThreadID); err != nil {
			return err
		}
		taskErr := strings.TrimSpace(task.Error)
		if taskErr == "" {
			taskErr = "openviking async commit task failed"
		}
		return fmt.Errorf("%s%s", memorySyncErrCommitTaskFailedPrefix, taskErr)
	default:
		if err := MemorySyncState.ClearCommitTask(state.ThreadID); err != nil {
			return err
		}
		return fmt.Errorf("%s%s", memorySyncErrCommitTaskStatusPrefix, taskStatus)
	}
}

// ensureOpenVikingSession returns the deterministic session id used for OpenViking session APIs.
func ensureOpenVikingSession(state models.MemorySyncState) (string, error) {
	threadID := strings.TrimSpace(state.ThreadID)
	if threadID == "" {
		return "", fmt.Errorf("thread_id is required")
	}
	// Follow thread-scoped deterministic session id: use thread_id as session_id in all /sessions/{id}/... calls.
	return threadID, nil
}

// memorySyncDurationFromSeconds converts a positive seconds value into a duration or returns the fallback.
func memorySyncDurationFromSeconds(seconds int, fallback time.Duration) time.Duration {
	if seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

// applyMemorySyncConfig loads the runtime worker settings from OpenViking configuration.
func applyMemorySyncConfig(cfg config.OpenVikingConfig) {
	memorySyncDelay = memorySyncDurationFromSeconds(cfg.MemorySyncDelaySeconds, defaultMemorySyncDelay)
	memorySyncPollInterval = memorySyncDurationFromSeconds(cfg.MemorySyncPollSeconds, defaultMemorySyncPollInterval)
	memorySyncTimeout = memorySyncDurationFromSeconds(cfg.MemorySyncTimeoutSeconds, defaultMemorySyncTimeout)
	if memorySyncTimeout < 10*time.Minute {
		memorySyncTimeout = 10 * time.Minute
	}
	memorySyncRetryDelay = memorySyncDurationFromSeconds(cfg.MemorySyncRetrySeconds, defaultMemorySyncRetryDelay)
}

// nextMemorySyncAttemptAt computes the next retry time for a failed memory sync attempt.
func nextMemorySyncAttemptAt(now time.Time) *time.Time {
	if memorySyncRetryDelay <= 0 {
		return nil
	}
	next := now.Add(memorySyncRetryDelay)
	return &next
}

// nextMemorySyncPollAt computes the next task-status polling time after async commit submission.
func nextMemorySyncPollAt(now time.Time) *time.Time {
	if memorySyncPollInterval <= 0 {
		return nil
	}
	next := now.Add(memorySyncPollInterval)
	return &next
}

// nextMemorySyncScheduledAt computes the first sync attempt time after a chat run finishes.
func nextMemorySyncScheduledAt(now time.Time) *time.Time {
	if memorySyncDelay <= 0 {
		return nil
	}
	next := now.Add(memorySyncDelay)
	return &next
}

func isMemorySyncCommitFatalError(err error) bool {
	if err == nil {
		return false
	}
	normalized := strings.TrimSpace(err.Error())
	return strings.HasPrefix(normalized, memorySyncErrCommitSubmitFailedPrefix) ||
		strings.HasPrefix(normalized, memorySyncErrCommitTaskFailedPrefix) ||
		strings.HasPrefix(normalized, memorySyncErrCommitTaskStatusPrefix)
}

// collectPendingMemoryUsageContextURIs converts pending usage rows into unique URIs and the row ids to acknowledge later.
func collectPendingMemoryUsageContextURIs(items []models.MemoryUsageLog) ([]string, []uint) {
	uris := make([]string, 0, len(items))
	ids := make([]uint, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
		uri := normalizeMemoryMatchURI(item.MemoryURI)
		if uri == "" {
			continue
		}
		if _, exists := seen[uri]; exists {
			continue
		}
		seen[uri] = struct{}{}
		uris = append(uris, uri)
	}
	return uris, ids
}

// buildSessionSyncMessages converts the unsynced tail of a thread into plain-text session messages.
func buildSessionSyncMessages(messages []models.Message, lastSyncedMsgID string) ([]sessionSyncMessage, string, error) {
	startIndex, err := resolveSyncStartIndex(messages, lastSyncedMsgID)
	if err != nil {
		return nil, "", err
	}
	if startIndex >= len(messages) {
		return []sessionSyncMessage{}, "", nil
	}

	synced := make([]sessionSyncMessage, 0, len(messages)-startIndex)
	lastMsgID := ""
	for i := startIndex; i < len(messages); i++ {
		modelMessage := messages[i]
		lastMsgID = strings.TrimSpace(modelMessage.MsgID)
		sessionMessage, ok, err := buildSessionSyncMessage(modelMessage)
		if err != nil {
			return nil, "", err
		}
		if ok {
			synced = append(synced, sessionMessage)
		}
	}
	return synced, lastMsgID, nil
}

// resolveSyncStartIndex finds the next unsynchronized message index from the saved cursor.
func resolveSyncStartIndex(messages []models.Message, lastSyncedMsgID string) (int, error) {
	cursor := strings.TrimSpace(lastSyncedMsgID)
	if cursor == "" {
		return 0, nil
	}
	for i, item := range messages {
		if strings.TrimSpace(item.MsgID) == cursor {
			return i + 1, nil
		}
	}
	return 0, fmt.Errorf("last_synced_msg_id not found: %s", cursor)
}

// buildSessionSyncMessage converts one stored AG-UI message row into an OpenViking plain-text message.
func buildSessionSyncMessage(modelMessage models.Message) (sessionSyncMessage, bool, error) {
	decoded, err := decodeStoredAGUIMessage(modelMessage)
	if err != nil {
		return sessionSyncMessage{}, false, err
	}

	switch decoded.Role {
	case agtypes.RoleUser:
		content := extractMessageText(*decoded)
		if content == "" {
			return sessionSyncMessage{}, false, nil
		}
		return sessionSyncMessage{MsgID: strings.TrimSpace(modelMessage.MsgID), Role: "user", Content: content}, true, nil
	case agtypes.RoleAssistant:
		content := extractMessageText(*decoded)
		if content == "" {
			return sessionSyncMessage{}, false, nil
		}
		return sessionSyncMessage{MsgID: strings.TrimSpace(modelMessage.MsgID), Role: "assistant", Content: content}, true, nil
	default:
		return sessionSyncMessage{}, false, nil
	}
}

// decodeStoredAGUIMessage decodes the persisted AG-UI message JSON from MySQL.
func decodeStoredAGUIMessage(modelMessage models.Message) (*agtypes.Message, error) {
	var decoded agtypes.Message
	if err := json.Unmarshal([]byte(modelMessage.Content), &decoded); err != nil {
		return nil, fmt.Errorf("decode message %s: %w", strings.TrimSpace(modelMessage.MsgID), err)
	}
	return &decoded, nil
}

// extractMessageText normalizes an AG-UI message into the plain-text content accepted by OpenViking sessions.
func extractMessageText(message agtypes.Message) string {
	if content, ok := message.ContentString(); ok {
		return strings.TrimSpace(content)
	}
	parts, ok := message.ContentInputContents()
	if !ok || len(parts) == 0 {
		return ""
	}
	textParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part.Type) != agtypes.InputContentTypeText {
			continue
		}
		if strings.TrimSpace(part.Text) == "" {
			continue
		}
		textParts = append(textParts, strings.TrimSpace(part.Text))
	}
	return strings.TrimSpace(strings.Join(textParts, "\n"))
}

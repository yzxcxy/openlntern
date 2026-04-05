package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"openIntern/internal/database"
	"openIntern/internal/models"
	chatsvc "openIntern/internal/services/chat"
	"openIntern/internal/services/memory/contracts"

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

var memorySyncWorkerOnce sync.Once
var (
	memorySyncPollInterval = defaultMemorySyncPollInterval
	memorySyncDelay        = defaultMemorySyncDelay
	memorySyncTimeout      = defaultMemorySyncTimeout
	memorySyncRetryDelay   = defaultMemorySyncRetryDelay
)

// InitMemorySync starts the in-process worker that forwards completed thread deltas to the active memory backend.
func InitMemorySync() {
	syncBackend := currentSyncBackend()
	if !syncBackend.Configured() {
		return
	}
	if database.GetRedis() == nil {
		log.Printf("memory sync disabled: redis is not configured")
		return
	}
	applyMemorySyncBackendConfig(syncBackend)
	if err := MemorySyncState.ResetLegacySyncing(); err != nil {
		log.Printf("memory sync reset legacy syncing states failed err=%v", err)
	}
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
		lock, claimed, err := tryAcquireMemorySyncLock(item.ThreadID)
		if err != nil {
			log.Printf("memory sync acquire lock failed thread_id=%s err=%v", item.ThreadID, err)
			continue
		}
		if !claimed {
			continue
		}
		func() {
			defer func() {
				if releaseErr := lock.Release(); releaseErr != nil {
					log.Printf("memory sync release lock failed thread_id=%s err=%v", item.ThreadID, releaseErr)
				}
			}()

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
				if markErr := MemorySyncState.MarkFailed(item.UserID, item.ThreadID, err.Error(), nextAttemptAt, commitStatus); markErr != nil {
					log.Printf("memory sync mark failed failed thread_id=%s err=%v", item.ThreadID, markErr)
				}
			}
		}()
	}
	return nil
}

// syncThreadMemoryState synchronizes the unsent thread message delta into the active memory backend.
func syncThreadMemoryState(ctx context.Context, state models.MemorySyncState) error {
	syncBackend := currentSyncBackend()
	if !syncBackend.Configured() {
		return nil
	}

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

	threadMessages, err := chatsvc.Message.ListThreadMessages(state.UserID, state.ThreadID)
	if err != nil {
		return err
	}
	addCursor := strings.TrimSpace(state.LastAddedMsgID)
	if addCursor == "" {
		addCursor = strings.TrimSpace(state.LastSyncedMsgID)
	}
	sessionMessages, lastMsgID, err := buildSessionSyncMessages(threadMessages, addCursor)
	if err != nil {
		return err
	}
	hasPendingCommit := strings.TrimSpace(state.LastAddedMsgID) != "" && strings.TrimSpace(state.LastAddedMsgID) != strings.TrimSpace(state.LastSyncedMsgID)
	if lastMsgID == "" {
		if !hasPendingCommit {
			return MemorySyncState.MarkReady(state.UserID, state.ThreadID, state.LastSyncedMsgID, state.LastCommittedRunID)
		}
	}
	if len(sessionMessages) == 0 && !hasPendingCommit {
		return MemorySyncState.MarkReady(state.UserID, state.ThreadID, lastMsgID, state.LastCommittedRunID)
	}
	for _, item := range sessionMessages {
		if msgID := strings.TrimSpace(item.MsgID); msgID != "" {
			addCursor = msgID
			if err := MemorySyncState.MarkMessagesAdded(state.UserID, state.ThreadID, addCursor); err != nil {
				return err
			}
		}
	}
	taskID, err := syncBackend.SubmitMessages(runCtx, state, sessionMessages)
	if err != nil {
		return fmt.Errorf("%s%w", memorySyncErrCommitSubmitFailedPrefix, err)
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return fmt.Errorf("%smemory backend returned empty task_id", memorySyncErrCommitSubmitFailedPrefix)
	}
	if err := MemorySyncState.MarkCommitSubmitted(state.UserID, state.ThreadID, taskID, nextMemorySyncPollAt(time.Now())); err != nil {
		return err
	}
	return nil
}

// pollSubmittedCommitTask checks one async commit task state and advances thread sync cursor only after completion.
func pollSubmittedCommitTask(ctx context.Context, state models.MemorySyncState) error {
	syncBackend := currentSyncBackend()
	if !syncBackend.Configured() {
		return nil
	}

	taskID := strings.TrimSpace(state.CommitTaskID)
	if taskID == "" {
		return nil
	}
	taskStatus, taskErr, err := syncBackend.PollOperation(ctx, taskID)
	if err != nil {
		return err
	}
	taskStatus = strings.ToLower(strings.TrimSpace(taskStatus))
	switch taskStatus {
	case models.MemorySyncStatusPending, "running":
		return MemorySyncState.MarkCommitPolling(state.UserID, state.ThreadID, taskStatus, nextMemorySyncPollAt(time.Now()))
	case "completed":
		finalCursor := strings.TrimSpace(state.LastAddedMsgID)
		if finalCursor == "" {
			finalCursor = strings.TrimSpace(state.LastSyncedMsgID)
		}
		return MemorySyncState.MarkReady(state.UserID, state.ThreadID, finalCursor, state.LastCommittedRunID)
	case models.MemorySyncStatusFailed:
		if err := MemorySyncState.ClearCommitTask(state.UserID, state.ThreadID); err != nil {
			return err
		}
		taskErr = strings.TrimSpace(taskErr)
		if taskErr == "" {
			taskErr = "memory backend async commit task failed"
		}
		return fmt.Errorf("%s%s", memorySyncErrCommitTaskFailedPrefix, taskErr)
	default:
		if err := MemorySyncState.ClearCommitTask(state.UserID, state.ThreadID); err != nil {
			return err
		}
		return fmt.Errorf("%s%s", memorySyncErrCommitTaskStatusPrefix, taskStatus)
	}
}

// applyMemorySyncBackendConfig loads the runtime worker settings from the active memory backend.
func applyMemorySyncBackendConfig(syncBackend SyncBackend) {
	if syncBackend == nil {
		syncBackend = noopSyncBackend{}
	}
	memorySyncDelay = syncBackend.SyncDelay()
	if memorySyncDelay <= 0 {
		memorySyncDelay = defaultMemorySyncDelay
	}
	memorySyncPollInterval = syncBackend.SyncPollInterval()
	if memorySyncPollInterval <= 0 {
		memorySyncPollInterval = defaultMemorySyncPollInterval
	}
	memorySyncTimeout = syncBackend.SyncTimeout()
	if memorySyncTimeout <= 0 {
		memorySyncTimeout = defaultMemorySyncTimeout
	}
	if memorySyncTimeout < 10*time.Minute {
		memorySyncTimeout = 10 * time.Minute
	}
	memorySyncRetryDelay = syncBackend.SyncRetryDelay()
	if memorySyncRetryDelay <= 0 {
		memorySyncRetryDelay = defaultMemorySyncRetryDelay
	}
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

// buildSessionSyncMessages converts the unsynced tail of a thread into plain-text session messages.
func buildSessionSyncMessages(messages []models.Message, lastSyncedMsgID string) ([]contracts.SyncMessage, string, error) {
	startIndex, err := resolveSyncStartIndex(messages, lastSyncedMsgID)
	if err != nil {
		return nil, "", err
	}
	if startIndex >= len(messages) {
		return []contracts.SyncMessage{}, "", nil
	}

	synced := make([]contracts.SyncMessage, 0, len(messages)-startIndex)
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

// buildSessionSyncMessage converts one stored AG-UI message row into one provider-agnostic plain-text message.
func buildSessionSyncMessage(modelMessage models.Message) (contracts.SyncMessage, bool, error) {
	decoded, err := decodeStoredAGUIMessage(modelMessage)
	if err != nil {
		return contracts.SyncMessage{}, false, err
	}

	switch decoded.Role {
	case agtypes.RoleUser:
		content := extractMessageText(*decoded)
		if content == "" {
			return contracts.SyncMessage{}, false, nil
		}
		return contracts.SyncMessage{MsgID: strings.TrimSpace(modelMessage.MsgID), Role: "user", Content: content}, true, nil
	case agtypes.RoleAssistant:
		content := extractMessageText(*decoded)
		if content == "" {
			return contracts.SyncMessage{}, false, nil
		}
		return contracts.SyncMessage{MsgID: strings.TrimSpace(modelMessage.MsgID), Role: "assistant", Content: content}, true, nil
	default:
		return contracts.SyncMessage{}, false, nil
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

// extractMessageText normalizes an AG-UI message into the plain-text content accepted by session-based backends.
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

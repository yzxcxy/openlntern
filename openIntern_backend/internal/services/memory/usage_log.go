package memory

import (
	"errors"
	"strings"
	"time"

	"openIntern/internal/dao"
	"openIntern/internal/models"
)

// MemoryUsageLogService manages per-run long-term memory usage records.
type MemoryUsageLogService struct{}

// MemoryUsageLog is the shared service singleton for memory usage tracking.
var MemoryUsageLog = new(MemoryUsageLogService)

// RecordRunMemoryUsage persists the memory URIs that were actually injected during one completed run.
func (s *MemoryUsageLogService) RecordRunMemoryUsage(threadID, runID string, uris []string) error {
	if !memorySyncConfigured() {
		return nil
	}
	threadID = strings.TrimSpace(threadID)
	runID = strings.TrimSpace(runID)
	if threadID == "" {
		return errors.New("thread_id is required")
	}
	if runID == "" {
		return errors.New("run_id is required")
	}

	normalized := normalizeMemoryUsageURIs(uris)
	if len(normalized) == 0 {
		return nil
	}
	if _, err := dao.Thread.GetByThreadID(threadID); err != nil {
		return err
	}

	items := make([]models.MemoryUsageLog, 0, len(normalized))
	for _, uri := range normalized {
		items = append(items, models.MemoryUsageLog{
			ThreadID:  threadID,
			RunID:     runID,
			MemoryURI: uri,
		})
	}
	return dao.MemoryUsageLog.CreateBatch(items)
}

// ListPendingByThreadID loads unreported memory usage rows for one thread.
func (s *MemoryUsageLogService) ListPendingByThreadID(threadID string) ([]models.MemoryUsageLog, error) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return nil, errors.New("thread_id is required")
	}
	return dao.MemoryUsageLog.ListPendingByThreadID(threadID)
}

// MarkReportedByIDs marks the specified usage rows as reported at the current time.
func (s *MemoryUsageLogService) MarkReportedByIDs(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return dao.MemoryUsageLog.MarkReported(ids, time.Now().UTC())
}

// normalizeMemoryUsageURIs trims, de-duplicates, and stabilizes injected memory URIs before persistence.
func normalizeMemoryUsageURIs(uris []string) []string {
	normalized := make([]string, 0, len(uris))
	seen := make(map[string]struct{}, len(uris))
	for _, uri := range uris {
		candidate := normalizeMemoryUsageURI(uri)
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		normalized = append(normalized, candidate)
	}
	return normalized
}

// normalizeMemoryUsageURI keeps legacy usage-log writes stable until that table is fully retired.
func normalizeMemoryUsageURI(uri string) string {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return ""
	}
	if idx := strings.Index(uri, "?"); idx >= 0 {
		uri = uri[:idx]
	}
	if idx := strings.Index(uri, "#"); idx >= 0 {
		uri = uri[:idx]
	}
	return strings.TrimRight(strings.TrimSpace(uri), "/")
}

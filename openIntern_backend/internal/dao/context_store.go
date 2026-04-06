package dao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"openIntern/internal/database"
)

type storeResponse struct {
	Status string          `json:"status"`
	Result json.RawMessage `json:"result"`
	Error  any             `json:"error"`
}

type storeAddResult struct {
	Status  string `json:"status"`
	RootURI string `json:"root_uri"`
	Errors  []any  `json:"errors"`
}

type ResourceEntry struct {
	Path       string
	Name       string
	Type       string
	Size       int64
	ModifiedAt time.Time
	IsDir      bool
}

func contextStoreReady() bool {
	return database.Context != nil && database.Context.Configured()
}

func skillStoreReady() bool {
	return contextStoreReady() && strings.TrimSpace(database.Context.SkillsRoot()) != ""
}

func listEntries(ctx context.Context, uri string, recursive bool) ([]ResourceEntry, error) {
	params := url.Values{}
	params.Set("uri", uri)
	if recursive {
		params.Set("recursive", "true")
	}
	params.Set("output", "agent")
	body, err := database.Context.Get(ctx, "/api/v1/fs/ls", params)
	if err != nil {
		return nil, err
	}
	var raw []map[string]any
	if err := decodeStoreResult(body, &raw); err != nil {
		return nil, err
	}
	return decodeEntries(raw), nil
}

func treeEntries(ctx context.Context, uri string) ([]ResourceEntry, error) {
	params := url.Values{}
	params.Set("uri", uri)
	params.Set("output", "agent")
	body, err := database.Context.Get(ctx, "/api/v1/fs/tree", params)
	if err != nil {
		return nil, err
	}
	var raw []map[string]any
	if err := decodeStoreResult(body, &raw); err != nil {
		return nil, err
	}
	return decodeEntries(raw), nil
}

func readContent(ctx context.Context, uri string, endpoint string) (string, error) {
	params := url.Values{}
	params.Set("uri", uri)
	body, err := database.Context.Get(ctx, endpoint, params)
	if err != nil {
		return "", err
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "", nil
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err == nil {
		if result, ok := payload["result"]; ok {
			switch value := result.(type) {
			case string:
				return value, nil
			case map[string]any:
				if content, ok := value["content"]; ok {
					return fmt.Sprint(content), nil
				}
				if text, ok := value["text"]; ok {
					return fmt.Sprint(text), nil
				}
			}
		}
		if content, ok := payload["content"]; ok {
			return fmt.Sprint(content), nil
		}
	}
	return string(body), nil
}

func addResource(ctx context.Context, resourcePath string, targetURI string, wait bool, timeoutSeconds float64) error {
	_, err := addResourceWithRootURI(ctx, resourcePath, targetURI, wait, timeoutSeconds)
	return err
}

func addResourceWithRootURI(ctx context.Context, resourcePath string, targetURI string, wait bool, timeoutSeconds float64) (string, error) {
	info, err := os.Stat(resourcePath)
	if err != nil {
		return "", err
	}

	var upload *database.TempUploadResult
	if info.IsDir() {
		// OpenViking raw HTTP mode requires local directories to be zipped before upload.
		upload, err = database.Context.UploadTempArchive(ctx, resourcePath, info.Name())
	} else {
		upload, err = database.Context.UploadTempFile(ctx, resourcePath)
	}
	if err != nil {
		return "", err
	}

	body, err := database.Context.Post(ctx, "/api/v1/resources", buildAddResourcePayload(upload.TempFileID, targetURI, wait, timeoutSeconds))
	if err != nil {
		return "", err
	}
	return decodeStoreAddResult(body)
}

func movePath(ctx context.Context, fromURI string, toURI string) error {
	body, err := database.Context.Post(ctx, "/api/v1/fs/mv", map[string]any{
		"from_uri": fromURI,
		"to_uri":   toURI,
	})
	if err != nil {
		return err
	}
	return decodeStoreResult(body, nil)
}

func mkdirPath(ctx context.Context, uri string) error {
	body, err := database.Context.Post(ctx, "/api/v1/fs/mkdir", map[string]any{
		"uri": uri,
	})
	if err != nil {
		return err
	}
	return decodeStoreResult(body, nil)
}

func deletePath(ctx context.Context, uri string, recursive bool) error {
	params := url.Values{}
	params.Set("uri", uri)
	if recursive {
		params.Set("recursive", "true")
	}
	body, err := database.Context.Delete(ctx, "/api/v1/fs", params)
	if err != nil {
		return err
	}
	return decodeStoreResult(body, nil)
}

func importSkill(ctx context.Context, rootDir string) error {
	upload, err := database.Context.UploadTempArchive(ctx, rootDir, "skill")
	if err != nil {
		return err
	}
	body, err := database.Context.Post(ctx, "/api/v1/skills", buildAddSkillPayload(upload.TempFileID, false, 0))
	if err != nil {
		return err
	}
	return decodeStoreResult(body, nil)
}

func buildAddResourcePayload(tempFileID string, targetURI string, wait bool, timeoutSeconds float64) map[string]any {
	payload := map[string]any{
		"temp_file_id": strings.TrimSpace(tempFileID),
		"target":       strings.TrimSpace(targetURI),
		"wait":         wait,
	}
	if timeoutSeconds > 0 {
		payload["timeout"] = timeoutSeconds
	}
	return payload
}

func buildAddSkillPayload(tempFileID string, wait bool, timeoutSeconds float64) map[string]any {
	payload := map[string]any{
		"temp_file_id": strings.TrimSpace(tempFileID),
		"wait":         wait,
	}
	if timeoutSeconds > 0 {
		payload["timeout"] = timeoutSeconds
	}
	return payload
}

func decodeStoreAddResult(body []byte) (string, error) {
	var resp storeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	if err := resp.err(); err != nil {
		return "", err
	}
	if len(resp.Result) == 0 {
		return "", nil
	}
	var result storeAddResult
	if err := json.Unmarshal(resp.Result, &result); err == nil {
		if len(result.Errors) > 0 {
			return "", errors.New(strings.TrimSpace(fmt.Sprint(result.Errors)))
		}
		return strings.TrimSpace(result.RootURI), nil
	}
	return "", nil
}

func decodeStoreResult(body []byte, target any) error {
	if len(body) == 0 {
		return nil
	}
	var resp storeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if err := resp.err(); err != nil {
		return err
	}
	if target == nil || len(resp.Result) == 0 {
		return nil
	}
	return json.Unmarshal(resp.Result, target)
}

func (r storeResponse) err() error {
	if status := strings.ToLower(strings.TrimSpace(r.Status)); status != "" && status != "ok" {
		return errors.New("context store error")
	}
	if r.Error == nil {
		return nil
	}
	msg := strings.TrimSpace(fmt.Sprint(r.Error))
	if msg == "" || msg == "<nil>" {
		return nil
	}
	return errors.New(msg)
}

func decodeEntries(raw []map[string]any) []ResourceEntry {
	if len(raw) == 0 {
		return []ResourceEntry{}
	}
	entries := make([]ResourceEntry, 0, len(raw))
	for _, item := range raw {
		entries = append(entries, ResourceEntry{
			Path:       entryString(item, "path", "uri"),
			Name:       entryString(item, "name"),
			Type:       entryString(item, "type", "kind"),
			Size:       entryInt64(item, "size"),
			ModifiedAt: entryTime(item, "mtime", "modified_at", "date", "updated_at"),
			IsDir:      entryIsDir(item),
		})
	}
	return entries
}

func relativePath(rootURI string, entryPath string) string {
	if entryPath == "" {
		return ""
	}
	root := strings.TrimRight(strings.TrimSpace(rootURI), "/")
	entryPath = strings.TrimSpace(entryPath)
	if root == "" || entryPath == "" {
		return ""
	}
	entryPath = strings.TrimPrefix(entryPath, root)
	return strings.TrimPrefix(entryPath, "/")
}

func entryString(entry map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := entry[key]; ok {
			switch v := value.(type) {
			case string:
				if strings.TrimSpace(v) != "" {
					return strings.TrimSpace(v)
				}
			case fmt.Stringer:
				if str := strings.TrimSpace(v.String()); str != "" {
					return str
				}
			default:
				if str := strings.TrimSpace(fmt.Sprint(value)); str != "" {
					return str
				}
			}
		}
	}
	return ""
}

func entryBool(entry map[string]any, keys ...string) bool {
	for _, key := range keys {
		value, ok := entry[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case bool:
			return v
		case string:
			if strings.EqualFold(strings.TrimSpace(v), "true") {
				return true
			}
		}
	}
	return false
}

func entryIsDir(entry map[string]any) bool {
	if entryBool(entry, "is_dir", "isDir", "dir", "directory", "isDirectory") {
		return true
	}
	switch strings.ToLower(entryString(entry, "type", "kind")) {
	case "dir", "directory", "folder":
		return true
	default:
		return false
	}
}

func entryInt64(entry map[string]any, keys ...string) int64 {
	for _, key := range keys {
		value, ok := entry[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
		case json.Number:
			if n, err := v.Int64(); err == nil {
				return n
			}
		case string:
			if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
				return n
			}
		}
	}
	return 0
}

func entryTime(entry map[string]any, keys ...string) time.Time {
	for _, key := range keys {
		value, ok := entry[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case time.Time:
			return v
		case string:
			if parsed, err := parseTime(v); err == nil {
				return parsed
			}
		case float64:
			return time.Unix(int64(v), 0)
		case json.Number:
			if n, err := v.Int64(); err == nil {
				return time.Unix(n, 0)
			}
		case int64:
			return time.Unix(v, 0)
		case int:
			return time.Unix(int64(v), 0)
		}
	}
	return time.Time{}
}

func parseTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("empty time")
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(n, 0), nil
	}
	return time.Time{}, errors.New("invalid time")
}

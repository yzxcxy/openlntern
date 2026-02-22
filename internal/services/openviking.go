package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"openIntern/internal/config"
)

type OpenVikingService struct {
	baseURL    string
	apiKey     string
	skillsRoot string
	client     *http.Client
}

var OpenViking = new(OpenVikingService)

func InitOpenViking(cfg config.OpenVikingConfig) {
	OpenViking.baseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	OpenViking.apiKey = strings.TrimSpace(cfg.APIKey)
	OpenViking.skillsRoot = strings.TrimRight(strings.TrimSpace(cfg.SkillsRoot), "/")
	if OpenViking.client == nil {
		timeoutSeconds := cfg.TimeoutSeconds
		if timeoutSeconds <= 0 {
			timeoutSeconds = 600
		}
		OpenViking.client = &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second}
	}
}

func (s *OpenVikingService) Configured() bool {
	return s != nil && s.baseURL != "" && s.skillsRoot != ""
}

func (s *OpenVikingService) BaseURL() string {
	if s == nil {
		return ""
	}
	return s.baseURL
}

func (s *OpenVikingService) APIKey() string {
	if s == nil {
		return ""
	}
	return s.apiKey
}

func (s *OpenVikingService) SkillsRoot() string {
	if s == nil {
		return ""
	}
	return s.skillsRoot
}

func (s *OpenVikingService) Client() *http.Client {
	if s == nil {
		return nil
	}
	if s.client == nil {
		s.client = &http.Client{Timeout: 600 * time.Second}
	}
	return s.client
}

type openVikingResponse struct {
	Status string          `json:"status"`
	Result json.RawMessage `json:"result"`
	Error  any             `json:"error"`
}

func KnowledgeBaseRootURI() string {
	return "viking://resources/"
}

func KnowledgeBaseURI(name string) string {
	name = strings.Trim(name, "/")
	return strings.TrimRight(KnowledgeBaseRootURI(), "/") + "/" + name + "/"
}

func CleanKBName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("empty name")
	}
	if strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		return "", errors.New("invalid name")
	}
	return name, nil
}

func NormalizeUploadPath(rel string, fallback string) string {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return path.Clean(fallback)
	}
	rel = strings.TrimPrefix(rel, "/")
	rel = path.Clean(rel)
	if rel == "." {
		return path.Clean(fallback)
	}
	parts := strings.Split(rel, "/")
	if len(parts) > 1 {
		rel = path.Join(parts[1:]...)
	}
	if rel == "" || rel == "." {
		return path.Clean(fallback)
	}
	return rel
}

func ResolveUploadPath(root string, rel string) (string, error) {
	if strings.Contains(rel, "..") {
		return "", errors.New("invalid path")
	}
	rel = strings.TrimPrefix(rel, "/")
	cleaned := path.Clean(rel)
	if cleaned == "." || cleaned == "" {
		return "", errors.New("invalid path")
	}
	targetPath := filepath.Join(root, filepath.FromSlash(cleaned))
	if !strings.HasPrefix(targetPath, root) {
		return "", errors.New("invalid path")
	}
	return targetPath, nil
}

func OpenVikingList(ctx context.Context, uri string, recursive bool) ([]map[string]any, error) {
	params := url.Values{}
	params.Set("uri", uri)
	if recursive {
		params.Set("recursive", "true")
	}
	params.Set("output", "agent")
	body, err := openVikingGet(ctx, "/api/v1/fs/ls", params)
	if err != nil {
		return nil, err
	}
	var resp openVikingResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if strings.TrimSpace(resp.Status) != "" && strings.ToLower(resp.Status) != "ok" {
		return nil, errors.New("openviking error")
	}
	if resp.Error != nil {
		if msg := strings.TrimSpace(fmt.Sprint(resp.Error)); msg != "" && msg != "<nil>" {
			return nil, errors.New(msg)
		}
	}
	if len(resp.Result) == 0 {
		return []map[string]any{}, nil
	}
	var result []map[string]any
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func OpenVikingTree(ctx context.Context, uri string) ([]map[string]any, error) {
	params := url.Values{}
	params.Set("uri", uri)
	params.Set("output", "agent")
	body, err := openVikingGet(ctx, "/api/v1/fs/tree", params)
	if err != nil {
		return nil, err
	}
	var resp openVikingResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if strings.TrimSpace(resp.Status) != "" && strings.ToLower(resp.Status) != "ok" {
		return nil, errors.New("openviking error")
	}
	if resp.Error != nil {
		if msg := strings.TrimSpace(fmt.Sprint(resp.Error)); msg != "" && msg != "<nil>" {
			return nil, errors.New(msg)
		}
	}
	if len(resp.Result) == 0 {
		return []map[string]any{}, nil
	}
	var entries []map[string]any
	if err := json.Unmarshal(resp.Result, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func OpenVikingReadContent(ctx context.Context, uri string) (string, error) {
	params := url.Values{}
	params.Set("uri", uri)
	body, err := openVikingGet(ctx, "/api/v1/content/read", params)
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

func OpenVikingAddResource(ctx context.Context, resourcePath string, targetURI string) error {
	return OpenVikingAddResourceWithOptions(ctx, resourcePath, targetURI, false, 0)
}

type openVikingAddResult struct {
	Status  string `json:"status"`
	RootURI string `json:"root_uri"`
	Errors  []any  `json:"errors"`
}

func OpenVikingAddResourceWithOptions(ctx context.Context, resourcePath string, targetURI string, wait bool, timeoutSeconds float64) error {
	payload := map[string]any{
		"path":   resourcePath,
		"target": targetURI,
	}
	if wait {
		payload["wait"] = true
	}
	if timeoutSeconds > 0 {
		payload["timeout"] = timeoutSeconds
	}
	body, err := openVikingPost(ctx, "/api/v1/resources", payload)
	if err != nil {
		return err
	}
	var resp openVikingResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if strings.TrimSpace(resp.Status) != "" && strings.ToLower(resp.Status) != "ok" {
		return errors.New("openviking error")
	}
	if resp.Error != nil {
		if msg := strings.TrimSpace(fmt.Sprint(resp.Error)); msg != "" && msg != "<nil>" {
			return errors.New(msg)
		}
	}
	if len(resp.Result) > 0 {
		var result openVikingAddResult
		if err := json.Unmarshal(resp.Result, &result); err == nil {
			if len(result.Errors) > 0 {
				return errors.New(strings.TrimSpace(fmt.Sprint(result.Errors)))
			}
		}
	}
	return nil
}

func OpenVikingMove(ctx context.Context, fromURI string, toURI string) error {
	body, err := openVikingPost(ctx, "/api/v1/fs/mv", map[string]any{
		"from_uri": fromURI,
		"to_uri":   toURI,
	})
	if err != nil {
		return err
	}
	var resp openVikingResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if strings.TrimSpace(resp.Status) != "" && strings.ToLower(resp.Status) != "ok" {
		return errors.New("openviking error")
	}
	if resp.Error != nil {
		if msg := strings.TrimSpace(fmt.Sprint(resp.Error)); msg != "" && msg != "<nil>" {
			return errors.New(msg)
		}
	}
	return nil
}

func OpenVikingDeleteResource(ctx context.Context, uri string, recursive bool) error {
	params := url.Values{}
	params.Set("uri", uri)
	if recursive {
		params.Set("recursive", "true")
	}
	body, err := openVikingDelete(ctx, "/api/v1/fs", params)
	if err != nil {
		return err
	}
	var resp openVikingResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if strings.TrimSpace(resp.Status) != "" && strings.ToLower(resp.Status) != "ok" {
		return errors.New("openviking error")
	}
	if resp.Error != nil {
		if msg := strings.TrimSpace(fmt.Sprint(resp.Error)); msg != "" && msg != "<nil>" {
			return errors.New(msg)
		}
	}
	return nil
}

func OpenVikingDeleteSkill(ctx context.Context, uri string) error {
	params := url.Values{}
	params.Set("uri", uri)
	params.Set("recursive", "true")
	body, err := openVikingDelete(ctx, "/api/v1/fs", params)
	if err != nil {
		return err
	}
	var resp openVikingResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if strings.TrimSpace(resp.Status) != "" && strings.ToLower(resp.Status) != "ok" {
		return errors.New("openviking error")
	}
	if resp.Error != nil {
		if msg := strings.TrimSpace(fmt.Sprint(resp.Error)); msg != "" && msg != "<nil>" {
			return errors.New(msg)
		}
	}
	return nil
}

func OpenVikingAddSkill(ctx context.Context, skillDir string) error {
	body, err := openVikingPost(ctx, "/api/v1/skills", map[string]any{
		"data": skillDir,
	})
	if err != nil {
		return err
	}
	var resp openVikingResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if strings.TrimSpace(resp.Status) != "" && strings.ToLower(resp.Status) != "ok" {
		return errors.New("openviking error")
	}
	if resp.Error != nil {
		if msg := strings.TrimSpace(fmt.Sprint(resp.Error)); msg != "" && msg != "<nil>" {
			return errors.New(msg)
		}
	}
	return nil
}

func OpenVikingRelativePath(rootURI string, entryPath string) string {
	if entryPath == "" {
		return ""
	}
	root := strings.TrimRight(rootURI, "/")
	entryPath = strings.TrimPrefix(entryPath, root)
	return strings.TrimPrefix(entryPath, "/")
}

func OpenVikingEntryString(entry map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := entry[key]; ok {
			switch v := value.(type) {
			case string:
				if v != "" {
					return v
				}
			case fmt.Stringer:
				str := v.String()
				if str != "" {
					return str
				}
			default:
				str := strings.TrimSpace(fmt.Sprint(value))
				if str != "" {
					return str
				}
			}
		}
	}
	return ""
}

func OpenVikingEntryBool(entry map[string]any, keys ...string) bool {
	for _, key := range keys {
		if value, ok := entry[key]; ok {
			switch v := value.(type) {
			case bool:
				return v
			case string:
				if strings.EqualFold(strings.TrimSpace(v), "true") {
					return true
				}
			}
		}
	}
	return false
}

func OpenVikingEntryIsDir(entry map[string]any) bool {
	if OpenVikingEntryBool(entry, "is_dir", "isDir", "dir", "directory", "isDirectory") {
		return true
	}
	t := strings.ToLower(OpenVikingEntryString(entry, "type", "kind"))
	return t == "dir" || t == "directory" || t == "folder"
}

func OpenVikingEntryInt64(entry map[string]any, keys ...string) int64 {
	for _, key := range keys {
		if value, ok := entry[key]; ok {
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
	}
	return 0
}

func OpenVikingEntryTime(entry map[string]any, keys ...string) time.Time {
	for _, key := range keys {
		if value, ok := entry[key]; ok {
			switch v := value.(type) {
			case time.Time:
				return v
			case string:
				if parsed, err := parseOpenVikingTime(v); err == nil {
					return parsed
				}
			case float64:
				return time.Unix(int64(v), 0)
			case json.Number:
				if n, err := v.Int64(); err == nil {
					return time.Unix(n, 0)
				}
			}
		}
	}
	return time.Time{}
}

func parseOpenVikingTime(value string) (time.Time, error) {
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

func openVikingGet(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	if !OpenViking.Configured() {
		return nil, errors.New("openviking not configured")
	}
	baseURL := strings.TrimRight(OpenViking.BaseURL(), "/")
	if baseURL == "" {
		return nil, errors.New("openviking base_url not configured")
	}
	requestURL := baseURL + endpoint
	if len(params) > 0 {
		requestURL += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	if apiKey := OpenViking.APIKey(); apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}
	client := OpenViking.Client()
	if client == nil {
		return nil, errors.New("openviking client not configured")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return nil, errors.New(msg)
	}
	return body, nil
}

func openVikingPost(ctx context.Context, endpoint string, payload any) ([]byte, error) {
	if !OpenViking.Configured() {
		return nil, errors.New("openviking not configured")
	}
	baseURL := strings.TrimRight(OpenViking.BaseURL(), "/")
	if baseURL == "" {
		return nil, errors.New("openviking base_url not configured")
	}
	targetURI, pathValue := extractOpenVikingPayload(payload)
	if pathValue != "" {
		pathValue = sanitizeLogPath(pathValue)
	}
	log.Printf("OpenViking POST request endpoint=%s target=%s path=%s", endpoint, targetURI, pathValue)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	requestURL := baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey := OpenViking.APIKey(); apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}
	client := OpenViking.Client()
	if client == nil {
		return nil, errors.New("openviking client not configured")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	respText := truncateLogText(string(respBody), 2000)
	log.Printf("OpenViking POST response endpoint=%s status=%d body=%s", endpoint, resp.StatusCode, respText)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = resp.Status
		}
		return nil, errors.New(msg)
	}
	return respBody, nil
}

func openVikingDelete(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	if !OpenViking.Configured() {
		return nil, errors.New("openviking not configured")
	}
	baseURL := strings.TrimRight(OpenViking.BaseURL(), "/")
	if baseURL == "" {
		return nil, errors.New("openviking base_url not configured")
	}
	requestURL := baseURL + endpoint
	if len(params) > 0 {
		requestURL += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, requestURL, nil)
	if err != nil {
		return nil, err
	}
	if apiKey := OpenViking.APIKey(); apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}
	client := OpenViking.Client()
	if client == nil {
		return nil, errors.New("openviking client not configured")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return nil, errors.New(msg)
	}
	return body, nil
}

func extractOpenVikingPayload(payload any) (string, string) {
	p, ok := payload.(map[string]any)
	if !ok {
		return "", ""
	}
	target := ""
	pathValue := ""
	if value, ok := p["target"]; ok {
		target = strings.TrimSpace(fmt.Sprint(value))
	}
	if value, ok := p["path"]; ok {
		pathValue = strings.TrimSpace(fmt.Sprint(value))
	}
	return target, pathValue
}

func sanitizeLogPath(value string) string {
	if value == "" {
		return ""
	}
	if idx := strings.Index(value, "?"); idx >= 0 {
		return value[:idx]
	}
	return value
}

func truncateLogText(value string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}

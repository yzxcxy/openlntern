package database

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
	"strings"
	"time"

	"openIntern/internal/config"
)

type ContextStore struct {
	baseURL    string
	apiKey     string
	skillsRoot string
	toolsRoot  string
	client     *http.Client
}

var Context = new(ContextStore)

func InitContextStore(cfg config.OpenVikingConfig) {
	Context.baseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	Context.apiKey = strings.TrimSpace(cfg.APIKey)
	Context.skillsRoot = strings.TrimRight(strings.TrimSpace(cfg.SkillsRoot), "/")
	Context.toolsRoot = strings.TrimRight(strings.TrimSpace(cfg.ToolsRoot), "/")
	timeoutSeconds := cfg.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 600
	}
	Context.client = &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second}
}

func (s *ContextStore) Configured() bool {
	return s != nil && s.baseURL != "" && s.client != nil
}

func (s *ContextStore) SkillsRoot() string {
	if s == nil {
		return ""
	}
	return s.skillsRoot
}

// ToolsRoot 返回 OpenViking 中工具索引的根 URI。
func (s *ContextStore) ToolsRoot() string {
	if s == nil {
		return ""
	}
	return s.toolsRoot
}

func (s *ContextStore) Get(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	return s.do(ctx, http.MethodGet, endpoint, params, nil)
}

func (s *ContextStore) Post(ctx context.Context, endpoint string, payload any) ([]byte, error) {
	return s.do(ctx, http.MethodPost, endpoint, nil, payload)
}

func (s *ContextStore) Delete(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	return s.do(ctx, http.MethodDelete, endpoint, params, nil)
}

func (s *ContextStore) do(ctx context.Context, method string, endpoint string, params url.Values, payload any) ([]byte, error) {
	if !s.Configured() {
		return nil, errors.New("context store not configured")
	}
	requestURL := s.baseURL + endpoint
	if len(params) > 0 {
		requestURL += "?" + params.Encode()
	}

	var bodyReader io.Reader
	if payload != nil {
		targetURI, pathValue := extractPayloadSummary(payload)
		if pathValue != "" {
			pathValue = sanitizeLogPath(pathValue)
		}
		log.Printf("ContextStore request method=%s endpoint=%s target=%s path=%s", method, endpoint, targetURI, pathValue)

		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, bodyReader)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if s.apiKey != "" {
		req.Header.Set("x-api-key", s.apiKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		log.Printf(
			"ContextStore response method=%s endpoint=%s status=%d body=%s",
			method,
			endpoint,
			resp.StatusCode,
			truncateLogText(string(respBody), 2000),
		)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = resp.Status
		}
		return nil, errors.New(msg)
	}
	return respBody, nil
}

func extractPayloadSummary(payload any) (string, string) {
	data, ok := payload.(map[string]any)
	if !ok {
		return "", ""
	}
	target := ""
	pathValue := ""
	if value, ok := data["target_uri"]; ok {
		target = strings.TrimSpace(fmt.Sprint(value))
	}
	if target == "" {
		if value, ok := data["targetUri"]; ok {
			target = strings.TrimSpace(fmt.Sprint(value))
		}
	}
	if target == "" {
		if value, ok := data["target"]; ok {
			target = strings.TrimSpace(fmt.Sprint(value))
		}
	}
	if value, ok := data["path"]; ok {
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

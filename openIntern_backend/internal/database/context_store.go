package database

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"openIntern/internal/config"
)

type ContextStore struct {
	baseURL    string
	apiKey     string
	skillsRoot string
	client     *http.Client
}

var Context = new(ContextStore)

// TempUploadResult captures the temporary upload id returned by OpenViking.
type TempUploadResult struct {
	TempFileID string `json:"temp_file_id"`
}

func InitContextStore(cfg config.OpenVikingConfig) {
	Context.baseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	Context.apiKey = strings.TrimSpace(cfg.APIKey)
	Context.skillsRoot = strings.TrimRight(strings.TrimSpace(cfg.SkillsRoot), "/")
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

func (s *ContextStore) Get(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	return s.do(ctx, http.MethodGet, endpoint, params, nil)
}

func (s *ContextStore) Post(ctx context.Context, endpoint string, payload any) ([]byte, error) {
	return s.do(ctx, http.MethodPost, endpoint, nil, payload)
}

func (s *ContextStore) Delete(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	return s.do(ctx, http.MethodDelete, endpoint, params, nil)
}

// UploadTempFile uploads one local file to OpenViking and returns the issued temp_file_id.
func (s *ContextStore) UploadTempFile(ctx context.Context, localPath string) (*TempUploadResult, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(localPath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	return s.uploadMultipart(ctx, "/api/v1/resources/temp_upload", body, writer.FormDataContentType(), filepath.Base(localPath))
}

// UploadTempArchive packages one local directory as a zip archive before uploading it to OpenViking.
func (s *ContextStore) UploadTempArchive(ctx context.Context, rootDir string, archiveBaseName string) (*TempUploadResult, error) {
	archiveBaseName = sanitizeArchiveBaseName(archiveBaseName)
	tempArchive, err := os.CreateTemp("", archiveBaseName+"-*.zip")
	if err != nil {
		return nil, err
	}
	archivePath := tempArchive.Name()
	if err := tempArchive.Close(); err != nil {
		os.Remove(archivePath)
		return nil, err
	}
	defer os.Remove(archivePath)

	if err := createZipArchiveFromDir(rootDir, archivePath); err != nil {
		return nil, err
	}
	return s.UploadTempFile(ctx, archivePath)
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

func (s *ContextStore) uploadMultipart(ctx context.Context, endpoint string, body *bytes.Buffer, contentType string, fileName string) (*TempUploadResult, error) {
	if !s.Configured() {
		return nil, errors.New("context store not configured")
	}
	requestURL := s.baseURL + endpoint
	log.Printf("ContextStore multipart request method=%s endpoint=%s file=%s", http.MethodPost, endpoint, fileName)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
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
	log.Printf(
		"ContextStore multipart response method=%s endpoint=%s status=%d body=%s",
		http.MethodPost,
		endpoint,
		resp.StatusCode,
		truncateLogText(string(respBody), 2000),
	)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = resp.Status
		}
		return nil, errors.New(msg)
	}

	var payload struct {
		Status string            `json:"status"`
		Result *TempUploadResult `json:"result"`
		Error  any               `json:"error"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return nil, err
	}
	if status := strings.ToLower(strings.TrimSpace(payload.Status)); status != "" && status != "ok" {
		msg := strings.TrimSpace(fmt.Sprint(payload.Error))
		if msg == "" {
			msg = "context store multipart upload failed"
		}
		return nil, errors.New(msg)
	}
	if payload.Error != nil {
		msg := strings.TrimSpace(fmt.Sprint(payload.Error))
		if msg != "" && msg != "<nil>" {
			return nil, errors.New(msg)
		}
	}
	if payload.Result == nil || strings.TrimSpace(payload.Result.TempFileID) == "" {
		return nil, errors.New("openviking temp upload response missing temp_file_id")
	}
	return payload.Result, nil
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
	} else if value, ok := data["temp_file_id"]; ok {
		pathValue = "temp_file_id:" + strings.TrimSpace(fmt.Sprint(value))
	}
	return target, pathValue
}

func sanitizeArchiveBaseName(value string) string {
	value = strings.TrimSpace(filepath.Base(value))
	if value == "" || value == "." || value == string(filepath.Separator) {
		return "openviking-upload"
	}
	value = strings.TrimSuffix(value, filepath.Ext(value))
	if value == "" {
		return "openviking-upload"
	}
	return value
}

func createZipArchiveFromDir(rootDir string, archivePath string) error {
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer archiveFile.Close()

	zipWriter := zip.NewWriter(archiveFile)
	defer zipWriter.Close()

	return filepath.WalkDir(rootDir, func(currentPath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(rootDir, currentPath)
		if err != nil {
			return err
		}
		entry, err := zipWriter.Create(filepath.ToSlash(relPath))
		if err != nil {
			return err
		}
		src, err := os.Open(currentPath)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(entry, src)
		closeErr := src.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
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

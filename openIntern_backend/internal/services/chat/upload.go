package chat

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	storagesvc "openIntern/internal/services/storage"
)

const (
	// ChatUploadMaxSizeBytes defines the max upload size for a single chat attachment.
	ChatUploadMaxSizeBytes int64 = 20 * 1024 * 1024
)

var (
	// ErrChatUploadValidation indicates upload validation failure.
	ErrChatUploadValidation = errors.New("chat upload validation failed")
)

// ChatUploadAsset stores normalized metadata for a chat attachment after upload.
type ChatUploadAsset struct {
	Key       string `json:"key"`
	URL       string `json:"url"`
	MIMEType  string `json:"mime_type"`
	FileName  string `json:"file_name"`
	Size      int64  `json:"size"`
	MediaKind string `json:"media_kind"`
}

// ChatUploadService handles chat attachment upload logic.
type ChatUploadService struct{}

// ChatUpload provides chat attachment upload capabilities.
var ChatUpload = new(ChatUploadService)

// Upload uploads a chat attachment to COS and returns the uploaded asset metadata.
func (s *ChatUploadService) Upload(ctx context.Context, ownerID string, threadID string, fileHeader *multipart.FileHeader) (*ChatUploadAsset, error) {
	if fileHeader == nil {
		return nil, newChatUploadValidationError("file is required")
	}
	if fileHeader.Size <= 0 {
		return nil, newChatUploadValidationError("empty file is not allowed")
	}
	if fileHeader.Size > ChatUploadMaxSizeBytes {
		return nil, newChatUploadValidationError(fmt.Sprintf("file too large, max size is %d bytes", ChatUploadMaxSizeBytes))
	}

	mimeType, err := detectUploadedFileMIME(fileHeader)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(mimeType) == "" {
		return nil, newChatUploadValidationError("unable to detect file mime type")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if fileHeader.Header != nil {
		fileHeader.Header.Set("Content-Type", mimeType)
	}

	objectKey := buildChatUploadObjectKey(ownerID, threadID, fileHeader.Filename, mimeType)
	url, err := storagesvc.File.UploadWithKey(ctx, objectKey, file, fileHeader)
	if err != nil {
		return nil, err
	}

	return &ChatUploadAsset{
		Key:       objectKey,
		URL:       url,
		MIMEType:  mimeType,
		FileName:  normalizeFileName(fileHeader.Filename, mimeType),
		Size:      fileHeader.Size,
		MediaKind: classifyMediaKind(mimeType),
	}, nil
}

// detectUploadedFileMIME detects MIME type from file content and falls back to header value.
func detectUploadedFileMIME(fileHeader *multipart.FileHeader) (string, error) {
	if fileHeader == nil {
		return "", newChatUploadValidationError("file is required")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	readSize, readErr := file.Read(buffer)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return "", readErr
	}
	detected := strings.TrimSpace(strings.ToLower(http.DetectContentType(buffer[:readSize])))
	if detected != "" && detected != "application/octet-stream" {
		return detected, nil
	}
	contentType := ""
	if fileHeader.Header != nil {
		contentType = strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	}
	if contentType != "" {
		return strings.ToLower(contentType), nil
	}
	return strings.TrimSpace(strings.ToLower(detected)), nil
}

// buildChatUploadObjectKey builds object key using owner/thread/date namespaces.
func buildChatUploadObjectKey(ownerID string, threadID string, fileName string, mimeType string) string {
	ownerSeg := sanitizePathSegment(ownerID, "anonymous")
	threadSeg := sanitizePathSegment(threadID, "default")
	ext := chooseFileExtension(fileName, mimeType)
	return path.Join(
		"chat",
		ownerSeg,
		threadSeg,
		time.Now().Format("20060102"),
		uuid.NewString()+ext,
	)
}

// chooseFileExtension chooses file extension from original name first, then MIME type.
func chooseFileExtension(fileName string, mimeType string) string {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(fileName)))
	if ext != "" {
		return ext
	}
	extensions, err := mime.ExtensionsByType(strings.TrimSpace(mimeType))
	if err == nil && len(extensions) > 0 {
		return strings.ToLower(strings.TrimSpace(extensions[0]))
	}
	return ".bin"
}

// normalizeFileName normalizes uploaded filename and ensures extension when missing.
func normalizeFileName(fileName string, mimeType string) string {
	base := strings.TrimSpace(filepath.Base(fileName))
	if base == "" {
		base = "upload"
	}
	ext := strings.TrimSpace(filepath.Ext(base))
	if ext != "" {
		return base
	}
	return base + chooseFileExtension("", mimeType)
}

// classifyMediaKind maps MIME types to image/audio/video/file kinds.
func classifyMediaKind(mimeType string) string {
	switch {
	case strings.HasPrefix(strings.TrimSpace(mimeType), "image/"):
		return "image"
	case strings.HasPrefix(strings.TrimSpace(mimeType), "audio/"):
		return "audio"
	case strings.HasPrefix(strings.TrimSpace(mimeType), "video/"):
		return "video"
	default:
		return "file"
	}
}

// sanitizePathSegment sanitizes a path segment and applies a fallback when empty.
func sanitizePathSegment(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	var builder strings.Builder
	for _, r := range trimmed {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_' {
			builder.WriteRune(r)
			continue
		}
		builder.WriteRune('_')
	}
	result := strings.Trim(builder.String(), "_")
	if result == "" {
		return fallback
	}
	return result
}

// newChatUploadValidationError wraps validation errors with a shared sentinel.
func newChatUploadValidationError(message string) error {
	return fmt.Errorf("%w: %s", ErrChatUploadValidation, strings.TrimSpace(message))
}

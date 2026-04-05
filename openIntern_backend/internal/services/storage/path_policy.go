package storage

import (
	"errors"
	"mime"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ObjectPurpose string

const (
	ObjectPurposeChat   ObjectPurpose = "chat"
	ObjectPurposeAvatar ObjectPurpose = "avatar"
	ObjectPurposePlugin ObjectPurpose = "plugin"
)

func sanitizePathSegment(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = strings.TrimSpace(fallback)
	}
	value = strings.ReplaceAll(value, "\\", "/")
	value = strings.ReplaceAll(value, "/", "_")
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
			continue
		}
		b.WriteRune('_')
	}
	sanitized := strings.Trim(b.String(), "._- ")
	if sanitized != "" {
		return sanitized
	}
	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		return "unknown"
	}
	return sanitizePathSegment(fallback, "unknown")
}

func validateRelativeSegments(segments []string) ([]string, error) {
	if len(segments) == 0 {
		return nil, nil
	}
	result := make([]string, 0, len(segments))
	for _, segment := range segments {
		normalized := strings.TrimSpace(strings.ReplaceAll(segment, "\\", "/"))
		if normalized == "" {
			continue
		}
		if strings.Contains(normalized, "/") {
			return nil, errors.New("relative segment must not contain slash")
		}
		if normalized == "." || normalized == ".." {
			return nil, errors.New("relative segment is invalid")
		}
		sanitized := sanitizePathSegment(normalized, "")
		if sanitized == "" || sanitized == "unknown" {
			return nil, errors.New("relative segment is invalid")
		}
		result = append(result, sanitized)
	}
	return result, nil
}

func buildUserObjectKey(userID string, purpose ObjectPurpose, scopeSegments []string, fileName string, contentType string, now time.Time) (string, error) {
	userSegment := sanitizePathSegment(userID, "")
	if userSegment == "" || userSegment == "unknown" {
		return "", errors.New("user id is invalid")
	}
	purposeSegment := sanitizePathSegment(string(purpose), "")
	if purposeSegment == "" || purposeSegment == "unknown" {
		return "", errors.New("purpose is invalid")
	}
	normalizedScope, err := validateRelativeSegments(scopeSegments)
	if err != nil {
		return "", err
	}
	if now.IsZero() {
		now = time.Now()
	}
	ext := inferObjectExt(fileName, contentType)
	file := uuid.NewString() + ext
	parts := []string{"users", userSegment, purposeSegment}
	parts = append(parts, normalizedScope...)
	parts = append(parts, now.Format("20060102"), file)
	return path.Join(parts...), nil
}

func buildPublicObjectKey(domain string, resourceSegments []string, fileName string) (string, error) {
	domainSegment := sanitizePathSegment(domain, "")
	if domainSegment == "" || domainSegment == "unknown" {
		return "", errors.New("domain is invalid")
	}
	normalizedResources, err := validateRelativeSegments(resourceSegments)
	if err != nil {
		return "", err
	}
	file := strings.TrimSpace(filepath.Base(fileName))
	if file == "" || file == "." || file == ".." {
		return "", errors.New("file name is invalid")
	}
	fileSegment := sanitizePathSegment(file, "")
	if fileSegment == "" || fileSegment == "unknown" {
		return "", errors.New("file name is invalid")
	}
	parts := []string{"public", domainSegment}
	parts = append(parts, normalizedResources...)
	parts = append(parts, fileSegment)
	return path.Join(parts...), nil
}

func inferObjectExt(fileName string, contentType string) string {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(strings.TrimSpace(fileName))))
	if ext != "" {
		return ext
	}
	if contentType != "" {
		exts, err := mime.ExtensionsByType(strings.TrimSpace(contentType))
		if err == nil && len(exts) > 0 {
			return strings.ToLower(strings.TrimSpace(exts[0]))
		}
	}
	return ".bin"
}

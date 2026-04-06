package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"openIntern/internal/config"
)

type UploadUserObjectSpec struct {
	Purpose          ObjectPurpose
	ScopeSegments    []string
	OriginalFileName string
	ContentType      string
}

type UploadPublicObjectSpec struct {
	Domain           string
	ResourceSegments []string
	FileName         string
	ContentType      string
}

type ObjectDescriptor struct {
	Bucket      string `json:"bucket"`
	Key         string `json:"key"`
	URL         string `json:"url"`
	Visibility  string `json:"visibility"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

const (
	ObjectVisibilityUser   = "user"
	ObjectVisibilityPublic = "public"
)

type ObjectStorageService struct {
	store *MinIOStore
}

var ObjectStorage = &ObjectStorageService{}

const (
	backendAssetRoutePrefix = "/v1/assets/"
	userObjectURLTTL        = 7 * 24 * time.Hour
)

type ObjectReadResult struct {
	Reader      io.ReadCloser
	ContentType string
	Size        int64
}

func (s *ObjectStorageService) resolveStore() (*MinIOStore, error) {
	liveStoreMu.RLock()
	defer liveStoreMu.RUnlock()
	if s != nil && s.store != nil && objectStore != nil && s.store != objectStore {
		return nil, errors.New("storage service is refreshing, please retry")
	}
	if s != nil && s.store != nil {
		return s.store, nil
	}
	if objectStore != nil {
		return objectStore, nil
	}
	return nil, errors.New("file service not configured")
}

func (s *ObjectStorageService) UploadUserObject(ctx context.Context, userID string, spec UploadUserObjectSpec, reader io.Reader, size int64) (*ObjectDescriptor, error) {
	if reader == nil {
		return nil, errors.New("reader is required")
	}
	key, err := buildUserObjectKey(userID, spec.Purpose, spec.ScopeSegments, spec.OriginalFileName, spec.ContentType, time.Now())
	if err != nil {
		return nil, err
	}
	store, err := s.resolveStore()
	if err != nil {
		return nil, err
	}
	stored, err := store.PutObject(ctx, key, reader, size, PutObjectOptions{
		ContentType: spec.ContentType,
	})
	if err != nil {
		return nil, err
	}
	return &ObjectDescriptor{
		Bucket:      stored.Bucket,
		Key:         stored.Key,
		URL:         stored.URL,
		Visibility:  ObjectVisibilityUser,
		ContentType: stored.ContentType,
		Size:        stored.Size,
	}, nil
}

// uploadSystemPublicObject 仅用于包内系统能力写入 public/... 对象。
func (s *ObjectStorageService) uploadSystemPublicObject(ctx context.Context, spec UploadPublicObjectSpec, reader io.Reader, size int64) (*ObjectDescriptor, error) {
	if reader == nil {
		return nil, errors.New("reader is required")
	}
	key, err := buildPublicObjectKey(spec.Domain, spec.ResourceSegments, spec.FileName)
	if err != nil {
		return nil, err
	}
	store, err := s.resolveStore()
	if err != nil {
		return nil, err
	}
	stored, err := store.PutObject(ctx, key, reader, size, PutObjectOptions{
		ContentType: spec.ContentType,
	})
	if err != nil {
		return nil, err
	}
	return &ObjectDescriptor{
		Bucket:      stored.Bucket,
		Key:         stored.Key,
		URL:         stored.URL,
		Visibility:  ObjectVisibilityPublic,
		ContentType: stored.ContentType,
		Size:        stored.Size,
	}, nil
}

func (s *ObjectStorageService) DeleteUserObject(ctx context.Context, userID string, objectKey string) error {
	userSegment := sanitizePathSegment(userID, "")
	if userSegment == "" || userSegment == "unknown" {
		return errors.New("user id is invalid")
	}
	objectKey = normalizeObjectKey(objectKey)
	if objectKey == "" {
		return errors.New("object key is empty")
	}
	userPrefix := path.Join("users", userSegment) + "/"
	// 仅允许删除用户命名空间内的对象，防止越权删 public 或他人对象。
	if !strings.HasPrefix(objectKey, userPrefix) {
		return errors.New("object key is outside user namespace")
	}
	store, err := s.resolveStore()
	if err != nil {
		return err
	}
	return store.RemoveObject(ctx, objectKey)
}

// ExtractStoredObjectKey normalizes stored object references into stable object keys.
func (s *ObjectStorageService) ExtractStoredObjectKey(storedValue string) (string, error) {
	raw := strings.TrimSpace(storedValue)
	if raw == "" {
		return "", nil
	}
	key := normalizeObjectKey(raw)
	if hasSupportedObjectPrefix(key) {
		return key, nil
	}
	store, err := s.resolveStore()
	if err != nil {
		return "", err
	}
	key, ok := store.ExtractObjectKeyFromURL(raw)
	if !ok || !hasSupportedObjectPrefix(key) {
		return "", errors.New("stored object key is invalid")
	}
	return key, nil
}

// ResolveObjectAccessURL returns a browser-consumable asset URL for a stored object reference.
func (s *ObjectStorageService) ResolveObjectAccessURL(storedValue string) (string, error) {
	key, err := s.ExtractStoredObjectKey(storedValue)
	if err != nil {
		return "", err
	}
	if key == "" {
		return "", nil
	}
	if strings.HasPrefix(key, "public/") {
		return backendAssetRoutePrefix + key, nil
	}
	if !strings.HasPrefix(key, "users/") {
		return "", errors.New("object key root prefix is invalid")
	}
	expiresAt := time.Now().Add(userObjectURLTTL).Unix()
	values := url.Values{}
	values.Set("expires", strconv.FormatInt(expiresAt, 10))
	values.Set("signature", signObjectAccess(key, expiresAt))
	return backendAssetRoutePrefix + key + "?" + values.Encode(), nil
}

// VerifyObjectAccessSignature validates a user object access signature and expiry.
func (s *ObjectStorageService) VerifyObjectAccessSignature(objectKey string, expiresAt int64, signature string) bool {
	key := normalizeObjectKey(objectKey)
	if !strings.HasPrefix(key, "users/") || expiresAt <= 0 {
		return false
	}
	if time.Now().Unix() > expiresAt {
		return false
	}
	expected := signObjectAccess(key, expiresAt)
	return hmac.Equal([]byte(expected), []byte(strings.TrimSpace(signature)))
}

// ReadObject loads an existing object for streaming to the browser.
func (s *ObjectStorageService) ReadObject(ctx context.Context, storedValue string) (*ObjectReadResult, error) {
	key, err := s.ExtractStoredObjectKey(storedValue)
	if err != nil {
		return nil, err
	}
	if !hasSupportedObjectPrefix(key) {
		return nil, errors.New("object key root prefix is invalid")
	}
	store, err := s.resolveStore()
	if err != nil {
		return nil, err
	}
	result, err := store.GetObject(ctx, key)
	if err != nil {
		return nil, err
	}
	return &ObjectReadResult{
		Reader:      result.Reader,
		ContentType: result.ContentType,
		Size:        result.Size,
	}, nil
}

func hasSupportedObjectPrefix(key string) bool {
	return strings.HasPrefix(key, "users/") || strings.HasPrefix(key, "public/")
}

func signObjectAccess(objectKey string, expiresAt int64) string {
	mac := hmac.New(sha256.New, []byte(config.GetConfig().JWT.Secret))
	_, _ = mac.Write([]byte(normalizeObjectKey(objectKey)))
	_, _ = mac.Write([]byte{'\n'})
	_, _ = mac.Write([]byte(strconv.FormatInt(expiresAt, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

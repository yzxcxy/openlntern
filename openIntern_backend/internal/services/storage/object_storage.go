package storage

import (
	"context"
	"errors"
	"io"
	"path"
	"strings"
	"time"
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

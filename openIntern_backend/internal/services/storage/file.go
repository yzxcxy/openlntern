package storage

import (
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"openIntern/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStore struct {
	client        *minio.Client
	bucket        string
	publicBaseURL string
}

type PutObjectOptions struct {
	ContentType  string
	UserMetadata map[string]string
}

type StoredObject struct {
	Bucket      string
	Key         string
	URL         string
	ContentType string
	Size        int64
	ETag        string
}

type ReadObjectResult struct {
	Reader      io.ReadCloser
	ContentType string
	Size        int64
}

var objectStore *MinIOStore
var liveStoreMu sync.RWMutex

func InitObjectStorage(cfg config.MinIOConfig) error {
	if cfg.Endpoint == "" && cfg.AccessKey == "" && cfg.SecretKey == "" && cfg.Bucket == "" {
		liveStoreMu.Lock()
		objectStore = nil
		ObjectStorage.store = nil
		liveStoreMu.Unlock()
		return nil
	}
	if cfg.Endpoint == "" || cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Bucket == "" {
		return errors.New("minio config incomplete")
	}
	endpoint, err := normalizeMinIOEndpoint(cfg.Endpoint)
	if err != nil {
		return err
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return err
	}
	exists, err := client.BucketExists(context.Background(), strings.TrimSpace(cfg.Bucket))
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("minio bucket does not exist")
	}
	store := &MinIOStore{
		client:        client,
		bucket:        strings.TrimSpace(cfg.Bucket),
		publicBaseURL: strings.TrimRight(strings.TrimSpace(cfg.PublicBaseURL), "/"),
	}
	liveStoreMu.Lock()
	objectStore = store
	ObjectStorage.store = store
	liveStoreMu.Unlock()
	return nil
}

func (s *MinIOStore) PutObject(ctx context.Context, key string, reader io.Reader, size int64, options PutObjectOptions) (*StoredObject, error) {
	if s == nil || s.client == nil {
		return nil, errors.New("file service not configured")
	}
	key = normalizeObjectKey(key)
	if key == "" {
		return nil, errors.New("empty key")
	}
	if reader == nil {
		return nil, errors.New("empty reader")
	}
	result, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType:  strings.TrimSpace(options.ContentType),
		UserMetadata: options.UserMetadata,
	})
	if err != nil {
		return nil, err
	}
	url, err := s.BuildObjectURL(key)
	if err != nil {
		return nil, err
	}
	return &StoredObject{
		Bucket:      s.bucket,
		Key:         key,
		URL:         url,
		ContentType: strings.TrimSpace(options.ContentType),
		Size:        result.Size,
		ETag:        strings.Trim(result.ETag, "\""),
	}, nil
}

func (s *MinIOStore) RemoveObject(ctx context.Context, key string) error {
	if s == nil || s.client == nil {
		return errors.New("file service not configured")
	}
	key = normalizeObjectKey(key)
	if key == "" {
		return errors.New("empty key")
	}
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

// GetObject opens an existing object and loads its metadata for streaming responses.
func (s *MinIOStore) GetObject(ctx context.Context, key string) (*ReadObjectResult, error) {
	if s == nil || s.client == nil {
		return nil, errors.New("file service not configured")
	}
	key = normalizeObjectKey(key)
	if key == "" {
		return nil, errors.New("empty key")
	}
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	info, err := object.Stat()
	if err != nil {
		_ = object.Close()
		return nil, err
	}
	contentType := strings.TrimSpace(info.ContentType)
	if contentType == "" {
		if extType := mime.TypeByExtension(filepath.Ext(key)); extType != "" {
			contentType = extType
		}
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return &ReadObjectResult{
		Reader:      object,
		ContentType: contentType,
		Size:        info.Size,
	}, nil
}

func (s *MinIOStore) BuildObjectURL(key string) (string, error) {
	if s == nil {
		return "", errors.New("file service not configured")
	}
	key = normalizeObjectKey(key)
	if key == "" {
		return "", errors.New("empty key")
	}
	if s.publicBaseURL != "" {
		return s.publicBaseURL + "/" + key, nil
	}
	if s.client == nil || s.client.EndpointURL() == nil {
		return "", errors.New("minio endpoint unavailable")
	}
	endpoint := s.client.EndpointURL()
	scheme := strings.TrimSpace(endpoint.Scheme)
	if scheme == "" {
		scheme = "http"
	}
	base := scheme + "://" + endpoint.Host
	return base + "/" + s.bucket + "/" + key, nil
}

// ExtractObjectKeyFromURL converts a known legacy object URL into a stable object key.
func (s *MinIOStore) ExtractObjectKeyFromURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	for _, prefix := range s.objectURLPrefixes() {
		if strings.HasPrefix(raw, prefix+"/") {
			key := normalizeObjectKey(strings.TrimPrefix(raw, prefix+"/"))
			if key != "" {
				return key, true
			}
		}
	}
	return "", false
}

func (s *MinIOStore) objectURLPrefixes() []string {
	if s == nil {
		return nil
	}
	prefixes := make([]string, 0, 2)
	if s.publicBaseURL != "" {
		prefixes = append(prefixes, s.publicBaseURL)
	}
	if s.client != nil && s.client.EndpointURL() != nil && strings.TrimSpace(s.bucket) != "" {
		endpoint := s.client.EndpointURL()
		scheme := strings.TrimSpace(endpoint.Scheme)
		if scheme == "" {
			scheme = "http"
		}
		prefixes = append(prefixes, scheme+"://"+endpoint.Host+"/"+s.bucket)
	}
	return prefixes
}

func getObjectStore() (*MinIOStore, error) {
	liveStoreMu.RLock()
	defer liveStoreMu.RUnlock()
	if objectStore == nil {
		return nil, errors.New("file service not configured")
	}
	return objectStore, nil
}

func normalizeMinIOEndpoint(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("empty endpoint")
	}
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return "", err
		}
		return parsed.Host, nil
	}
	return raw, nil
}

func normalizeObjectKey(key string) string {
	return strings.TrimPrefix(strings.TrimSpace(strings.ReplaceAll(key, "\\", "/")), "/")
}

// BuildPublicObjectURL resolves a stable public object key (public/...) into an application asset URL.
func BuildPublicObjectURL(publicObjectKey string) (string, error) {
	key := normalizeObjectKey(publicObjectKey)
	if !strings.HasPrefix(key, "public/") {
		return "", errors.New("public object key must start with public/")
	}
	return backendAssetRoutePrefix + key, nil
}

type FileService struct{}

var File = new(FileService)

// InitFile 保留旧入口，避免现有调用点立即迁移。
func InitFile(cfg config.MinIOConfig) error {
	return InitObjectStorage(cfg)
}

func (s *FileService) UploadWithKey(ctx context.Context, key string, file multipart.File, header *multipart.FileHeader) (string, error) {
	store, err := getObjectStore()
	if err != nil {
		return "", err
	}
	contentType := ""
	if header != nil && header.Header != nil {
		contentType = header.Header.Get("Content-Type")
	}
	size := int64(-1)
	if header != nil && header.Size > 0 {
		size = header.Size
	}
	stored, err := store.PutObject(ctx, key, file, size, PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return stored.URL, nil
}

func (s *FileService) Delete(ctx context.Context, key string) error {
	store, err := getObjectStore()
	if err != nil {
		return err
	}
	return store.RemoveObject(ctx, key)
}

func (s *FileService) UploadPath(ctx context.Context, key string, filePath string) (string, error) {
	store, err := getObjectStore()
	if err != nil {
		return "", err
	}
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return "", errors.New("empty file path")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	contentType := ""
	if extType := mime.TypeByExtension(filepath.Ext(filePath)); extType != "" {
		contentType = extType
	}
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	stored, err := store.PutObject(ctx, key, file, info.Size(), PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return stored.URL, nil
}

func (s *FileService) UploadReader(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	store, err := getObjectStore()
	if err != nil {
		return "", err
	}
	stored, err := store.PutObject(ctx, key, reader, -1, PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return stored.URL, nil
}

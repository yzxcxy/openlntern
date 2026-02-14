package services

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"openIntern/internal/config"

	"github.com/tencentyun/cos-go-sdk-v5"
)

type FileService struct {
	client *cos.Client
}

var File = new(FileService)

func InitFile(cfg config.COSConfig) error {
	if cfg.SecretID == "" && cfg.SecretKey == "" && cfg.Bucket == "" && cfg.Region == "" {
		return nil
	}
	if cfg.SecretID == "" || cfg.SecretKey == "" || cfg.Bucket == "" || cfg.Region == "" {
		return errors.New("cos config incomplete")
	}
	bucketURL := fmt.Sprintf("https://%s.cos.%s.myqcloud.com", cfg.Bucket, cfg.Region)
	parsedURL, err := url.Parse(bucketURL)
	if err != nil {
		return err
	}
	File.client = cos.NewClient(&cos.BaseURL{BucketURL: parsedURL}, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  cfg.SecretID,
			SecretKey: cfg.SecretKey,
		},
	})
	return nil
}

func (s *FileService) UploadWithKey(ctx context.Context, key string, file multipart.File, header *multipart.FileHeader) (string, error) {
	if s.client == nil {
		return "", errors.New("file service not configured")
	}
	key = strings.TrimPrefix(strings.TrimSpace(key), "/")
	if key == "" {
		return "", errors.New("empty key")
	}
	options := &cos.ObjectPutOptions{}
	contentType := header.Header.Get("Content-Type")
	if contentType != "" {
		options.ObjectPutHeaderOptions = &cos.ObjectPutHeaderOptions{ContentType: contentType}
	}
	_, err := s.client.Object.Put(ctx, key, file, options)
	if err != nil {
		return "", err
	}
	return s.client.Object.GetObjectURL(key).String(), nil
}

func (s *FileService) Delete(ctx context.Context, key string) error {
	if s.client == nil {
		return errors.New("file service not configured")
	}
	key = strings.TrimPrefix(strings.TrimSpace(key), "/")
	if key == "" {
		return errors.New("empty key")
	}
	_, err := s.client.Object.Delete(ctx, key)
	return err
}

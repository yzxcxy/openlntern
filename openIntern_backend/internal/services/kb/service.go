package kb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"strings"

	"openIntern/internal/dao"
)

var (
	ErrNotConfigured  = errors.New("knowledge base storage not configured")
	ErrInvalidInput   = errors.New("invalid knowledge base input")
	ErrInvalidZipPath = errors.New("invalid zip entry path")
)

type Item struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
}

type MoveResult struct {
	FromURI string `json:"from"`
	ToURI   string `json:"to"`
}

type AsyncResult struct {
	Name   string `json:"name,omitempty"`
	Path   string `json:"path,omitempty"`
	Status string `json:"status"`
	Async  bool   `json:"async"`
}

type Service struct{}

var KnowledgeBase = new(Service)

func (s *Service) Configured() bool {
	return dao.KnowledgeBase.Configured()
}

func (s *Service) ensureConfigured() error {
	if !s.Configured() {
		return ErrNotConfigured
	}
	return nil
}

func (s *Service) List(ctx context.Context) ([]Item, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	items, err := dao.KnowledgeBase.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Item, 0, len(items))
	for _, item := range items {
		result = append(result, Item{Name: item.Name, URI: item.URI})
	}
	return result, nil
}

func (s *Service) Tree(ctx context.Context, rawName string) ([]dao.ResourceEntry, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	kbName, err := normalizeKnowledgeBaseName(rawName)
	if err != nil {
		return nil, err
	}
	return dao.KnowledgeBase.Tree(ctx, kbName)
}

func (s *Service) Import(ctx context.Context, rawName string, fileHeader *multipart.FileHeader) (*AsyncResult, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	kbName, err := normalizeKnowledgeBaseName(rawName)
	if err != nil {
		return nil, err
	}
	tempDir, err := os.MkdirTemp("", "kb-import-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	rootDir := filepath.Join(tempDir, kbName)
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return nil, err
	}
	if fileHeader != nil {
		if err := extractZipToDir(fileHeader, rootDir); err != nil {
			return nil, err
		}
	}
	if err := dao.KnowledgeBase.Ingest(ctx, rootDir, dao.KnowledgeBase.URI(kbName), false, 0); err != nil {
		return nil, err
	}
	return &AsyncResult{Name: kbName, Status: "accepted", Async: true}, nil
}

func (s *Service) UploadFile(ctx context.Context, rawName, targetDir string, fileHeader *multipart.FileHeader) (*AsyncResult, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	kbName, err := normalizeKnowledgeBaseName(rawName)
	if err != nil {
		return nil, err
	}
	if fileHeader == nil {
		return nil, fmt.Errorf("%w: file is required", ErrInvalidInput)
	}
	fileName := sanitizeUploadedFileName(fileHeader.Filename)
	tempDir, err := os.MkdirTemp("", "kb-upload-file-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	localPath := filepath.Join(tempDir, fileName)
	src, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()
	dst, err := os.Create(localPath)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return nil, err
	}
	if err := dst.Close(); err != nil {
		return nil, err
	}
	targetURI := resolveUploadTargetURI(kbName, targetDir)
	if err := dao.KnowledgeBase.Ingest(ctx, localPath, targetURI, false, 0); err != nil {
		return nil, err
	}
	return &AsyncResult{Path: targetURI, Status: "accepted", Async: true}, nil
}

func (s *Service) MoveEntry(ctx context.Context, fromURI, toURI string) (*MoveResult, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	result, err := normalizeMoveURIs(fromURI, toURI, false)
	if err != nil {
		return nil, err
	}
	if err := dao.KnowledgeBase.MoveEntry(ctx, result.FromURI, result.ToURI); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) DragEntry(ctx context.Context, fromURI, toURI string) (*MoveResult, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	result, err := normalizeMoveURIs(fromURI, toURI, true)
	if err != nil {
		return nil, err
	}
	if err := dao.KnowledgeBase.MoveEntry(ctx, result.FromURI, result.ToURI); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) Delete(ctx context.Context, rawName string) (string, error) {
	if err := s.ensureConfigured(); err != nil {
		return "", err
	}
	kbName, err := normalizeKnowledgeBaseName(rawName)
	if err != nil {
		return "", err
	}
	if err := dao.KnowledgeBase.Delete(ctx, kbName); err != nil {
		return "", err
	}
	return kbName, nil
}

func (s *Service) DeleteEntry(ctx context.Context, rawURI string, recursive bool) (string, error) {
	if err := s.ensureConfigured(); err != nil {
		return "", err
	}
	uri, err := normalizeEntryURI(rawURI)
	if err != nil {
		return "", err
	}
	if err := dao.KnowledgeBase.DeleteEntry(ctx, uri, recursive); err != nil {
		return "", err
	}
	return uri, nil
}

func (s *Service) ReadContent(ctx context.Context, rawURI string) (string, error) {
	if err := s.ensureConfigured(); err != nil {
		return "", err
	}
	uri, err := normalizeEntryURI(rawURI)
	if err != nil {
		return "", err
	}
	return dao.KnowledgeBase.ReadContent(ctx, uri)
}

func normalizeKnowledgeBaseName(rawName string) (string, error) {
	kbName, err := dao.KnowledgeBase.CleanName(rawName)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrInvalidInput, strings.TrimSpace(err.Error()))
	}
	return kbName, nil
}

func normalizeEntryURI(rawURI string) (string, error) {
	uri := strings.TrimSpace(rawURI)
	if uri == "" {
		return "", fmt.Errorf("%w: uri is required", ErrInvalidInput)
	}
	return uri, nil
}

func normalizeMoveURIs(fromURI, toURI string, forceDirectoryTarget bool) (*MoveResult, error) {
	fromURI = strings.TrimSpace(fromURI)
	toURI = strings.TrimSpace(toURI)
	if fromURI == "" || toURI == "" {
		return nil, fmt.Errorf("%w: from_uri and to_uri are required", ErrInvalidInput)
	}
	if forceDirectoryTarget && !strings.HasSuffix(toURI, "/") {
		toURI += "/"
	}
	return &MoveResult{FromURI: fromURI, ToURI: toURI}, nil
}

func resolveUploadTargetURI(kbName string, targetDir string) string {
	base := strings.TrimRight(dao.KnowledgeBase.InnerURI(kbName), "/")
	dir := strings.TrimSpace(targetDir)
	dir = strings.TrimLeft(dir, "/")
	dir = path.Clean(dir)
	if dir == ".." || strings.HasPrefix(dir, "../") {
		return base + "/"
	}
	if dir == "." || dir == "" {
		return base + "/"
	}
	prefix := kbName + "/"
	for i := 0; i < 2; i++ {
		if dir == kbName {
			dir = ""
			continue
		}
		if strings.HasPrefix(dir, prefix) {
			dir = strings.TrimPrefix(dir, prefix)
		}
	}
	if dir == "." || dir == "" {
		return base + "/"
	}
	return base + "/" + dir + "/"
}

func sanitizeUploadedFileName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(name)
	if name == "" || name == "." || name == ".." {
		return "upload.bin"
	}
	return name
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "404")
}

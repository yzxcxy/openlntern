package dao

import (
	"context"
	"errors"
	"path"
	"path/filepath"
	"strings"
)

type KnowledgeBaseItem struct {
	Name string
	URI  string
}

type KnowledgeBaseDAO struct{}

var KnowledgeBase = new(KnowledgeBaseDAO)

func (d *KnowledgeBaseDAO) Configured() bool {
	return contextStoreReady()
}

func (d *KnowledgeBaseDAO) RootURI(ctx context.Context) (string, error) {
	userID, err := OpenVikingUserIDFromContext(ctx)
	if err != nil {
		return "", err
	}
	return UserKnowledgeBaseRootURI(userID), nil
}

func (d *KnowledgeBaseDAO) CleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("empty name")
	}
	if strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		return "", errors.New("invalid name")
	}
	return name, nil
}

func (d *KnowledgeBaseDAO) URI(ctx context.Context, name string) (string, error) {
	userID, err := OpenVikingUserIDFromContext(ctx)
	if err != nil {
		return "", err
	}
	return UserKnowledgeBaseURI(userID, name), nil
}

func (d *KnowledgeBaseDAO) InnerURI(ctx context.Context, name string) (string, error) {
	userID, err := OpenVikingUserIDFromContext(ctx)
	if err != nil {
		return "", err
	}
	return UserKnowledgeBaseInnerURI(userID, name), nil
}

func (d *KnowledgeBaseDAO) ResolveLocalPath(root string, rel string) (string, error) {
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

func (d *KnowledgeBaseDAO) List(ctx context.Context) ([]KnowledgeBaseItem, error) {
	root, err := d.RootURI(ctx)
	if err != nil {
		return nil, err
	}
	entries, err := listEntries(ctx, root, false)
	if err != nil {
		if isContextStoreNotFound(err) {
			return []KnowledgeBaseItem{}, nil
		}
		return nil, err
	}
	items := make([]KnowledgeBaseItem, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir {
			continue
		}
		rel := relativePath(root, entry.Path)
		if rel == "" {
			rel = entry.Name
		}
		rel = strings.Trim(rel, "/")
		if rel == "" {
			continue
		}
		if d.isReservedTopLevelDir(rel) {
			continue
		}
		items = append(items, KnowledgeBaseItem{
			Name: rel,
			URI:  strings.TrimRight(root, "/") + "/" + rel + "/",
		})
	}
	return items, nil
}

// isReservedTopLevelDir 判断是否是系统保留目录。
func (d *KnowledgeBaseDAO) isReservedTopLevelDir(name string) bool {
	name = strings.Trim(strings.TrimSpace(name), "/")
	if name == "" {
		return false
	}
	topLevel := name
	if idx := strings.Index(topLevel, "/"); idx >= 0 {
		topLevel = topLevel[:idx]
	}
	_, exists := d.reservedTopLevelDirs()[topLevel]
	return exists
}

// reservedTopLevelDirs 返回资源根目录下需要在知识库列表中忽略的目录名。
func (d *KnowledgeBaseDAO) reservedTopLevelDirs() map[string]struct{} {
	return map[string]struct{}{}
}

func (d *KnowledgeBaseDAO) Tree(ctx context.Context, name string) ([]ResourceEntry, error) {
	kbName, err := d.CleanName(name)
	if err != nil {
		return nil, err
	}
	targetURI, err := d.InnerURI(ctx, kbName)
	if err != nil {
		return nil, err
	}
	return treeEntries(ctx, targetURI)
}

func (d *KnowledgeBaseDAO) Ingest(ctx context.Context, sourcePath string, targetURI string, wait bool, timeoutSeconds float64) error {
	return addResource(ctx, sourcePath, targetURI, wait, timeoutSeconds)
}

func (d *KnowledgeBaseDAO) MoveEntry(ctx context.Context, fromURI string, toURI string) error {
	return movePath(ctx, fromURI, toURI)
}

func (d *KnowledgeBaseDAO) Delete(ctx context.Context, name string) error {
	kbName, err := d.CleanName(name)
	if err != nil {
		return err
	}
	targetURI, err := d.URI(ctx, kbName)
	if err != nil {
		return err
	}
	return deletePath(ctx, targetURI, true)
}

func (d *KnowledgeBaseDAO) DeleteEntry(ctx context.Context, uri string, recursive bool) error {
	return deletePath(ctx, uri, recursive)
}

func (d *KnowledgeBaseDAO) ReadContent(ctx context.Context, uri string) (string, error) {
	// Knowledge base entries are OpenViking resources, so content must be read via the content API.
	return readContent(ctx, uri, "/api/v1/content/read")
}

// NormalizeScopedURI ensures externally provided knowledge-base URIs stay within the current user's root.
func (d *KnowledgeBaseDAO) NormalizeScopedURI(ctx context.Context, rawURI string) (string, error) {
	uri := strings.TrimSpace(rawURI)
	if uri == "" {
		return "", errors.New("uri is required")
	}
	root, err := d.RootURI(ctx)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(uri, root) {
		return "", errors.New("uri is outside current user scope")
	}
	return uri, nil
}

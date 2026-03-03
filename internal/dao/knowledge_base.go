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

func (d *KnowledgeBaseDAO) RootURI() string {
	return "viking://resources/"
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

func (d *KnowledgeBaseDAO) URI(name string) string {
	name = strings.Trim(name, "/")
	return strings.TrimRight(d.RootURI(), "/") + "/" + name + "/"
}

func (d *KnowledgeBaseDAO) NormalizeUploadPath(rel string, fallback string) string {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return path.Clean(fallback)
	}
	rel = strings.TrimPrefix(rel, "/")
	rel = path.Clean(rel)
	if rel == "." {
		return path.Clean(fallback)
	}
	parts := strings.Split(rel, "/")
	if len(parts) > 1 {
		rel = path.Join(parts[1:]...)
	}
	if rel == "" || rel == "." {
		return path.Clean(fallback)
	}
	return rel
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
	root := d.RootURI()
	entries, err := listEntries(ctx, root, false)
	if err != nil {
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
		items = append(items, KnowledgeBaseItem{
			Name: rel,
			URI:  strings.TrimRight(root, "/") + "/" + rel + "/",
		})
	}
	return items, nil
}

func (d *KnowledgeBaseDAO) Tree(ctx context.Context, name string) ([]ResourceEntry, error) {
	kbName, err := d.CleanName(name)
	if err != nil {
		return nil, err
	}
	return treeEntries(ctx, d.URI(kbName))
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
	return deletePath(ctx, d.URI(kbName), true)
}

func (d *KnowledgeBaseDAO) DeleteEntry(ctx context.Context, uri string, recursive bool) error {
	return deletePath(ctx, uri, recursive)
}

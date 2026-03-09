package dao

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"openIntern/internal/database"
)

const (
	defaultPluginToolStoreRootURI = "viking://resources/tools/"
	toolStoreTempDirPrefix        = "openintern-tool-store-"
)

// ToolStoreConfigured 返回工具索引存储是否可用。
func (d *PluginDAO) ToolStoreConfigured() bool {
	return contextStoreReady() && strings.TrimSpace(d.ToolStoreRootURI()) != ""
}

// ToolStoreRootURI 返回 OpenViking 工具索引根 URI。
func (d *PluginDAO) ToolStoreRootURI() string {
	if database.Context != nil {
		if root := normalizeToolStoreURI(database.Context.ToolsRoot()); root != "" {
			return root
		}
	}
	return defaultPluginToolStoreRootURI
}

// ToolStorePluginURI 返回插件级工具索引目录 URI。
func (d *PluginDAO) ToolStorePluginURI(pluginID string) string {
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return d.ToolStoreRootURI()
	}
	return strings.TrimRight(d.ToolStoreRootURI(), "/") + "/" + pluginID + "/"
}

// ToolStoreResourceURI 返回工具文档资源 URI。
func (d *PluginDAO) ToolStoreResourceURI(pluginID string, toolID string) string {
	pluginID = strings.TrimSpace(pluginID)
	toolID = strings.TrimSpace(toolID)
	if pluginID == "" || toolID == "" {
		return ""
	}
	return strings.TrimRight(d.ToolStorePluginURI(pluginID), "/") + "/" + toolID + ".md"
}

// UpsertToolStoreResource 将工具文档写入 OpenViking resources/tools 目录。
func (d *PluginDAO) UpsertToolStoreResource(ctx context.Context, pluginID string, toolID string, markdown string) error {
	if !d.ToolStoreConfigured() {
		return errors.New("tool store not configured")
	}
	pluginID = strings.TrimSpace(pluginID)
	toolID = strings.TrimSpace(toolID)
	if pluginID == "" {
		return errors.New("plugin_id is required")
	}
	if toolID == "" {
		return errors.New("tool_id is required")
	}
	if err := validateToolStoreToolID(toolID); err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", toolStoreTempDirPrefix)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, toolID+".md")
	if err := os.WriteFile(filePath, []byte(markdown), 0o600); err != nil {
		return err
	}
	targetURI := d.ToolStoreResourceURI(pluginID, toolID)
	if targetURI == "" {
		return errors.New("tool store target uri is empty")
	}
	if err := deletePath(ctx, targetURI, false); err != nil && !isToolStoreNotFoundError(err) {
		return err
	}

	if err := addResource(ctx, filePath, d.ToolStorePluginURI(pluginID), false, 0); err != nil {
		if !isToolStoreDuplicateError(err) && !isToolStoreAlreadyExistsError(err) {
			return err
		}
		if removeErr := deletePath(ctx, targetURI, false); removeErr != nil && !isToolStoreNotFoundError(removeErr) {
			return removeErr
		}
		return addResource(ctx, filePath, d.ToolStorePluginURI(pluginID), false, 0)
	}
	return nil
}

// ReplaceToolStoreResourcesByPlugin 按插件批量替换工具文档，使用单次导入请求写入 OpenViking。
func (d *PluginDAO) ReplaceToolStoreResourcesByPlugin(ctx context.Context, pluginID string, documents map[string]string) error {
	if !d.ToolStoreConfigured() {
		return errors.New("tool store not configured")
	}
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return errors.New("plugin_id is required")
	}
	// 先通过 ls 探测并按资源级删除，避免目录不存在时触发后端 rm 异常导致连接被重置。
	if err := d.DeleteToolStorePluginURI(ctx, pluginID); err != nil {
		return err
	}
	if len(documents) == 0 {
		return nil
	}

	tempDir, err := os.MkdirTemp("", toolStoreTempDirPrefix)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)
	pluginDir := filepath.Join(tempDir, pluginID)
	if err := os.MkdirAll(pluginDir, 0o700); err != nil {
		return err
	}

	if err := writeToolStoreDocumentFiles(pluginDir, documents); err != nil {
		return err
	}
	return addResource(ctx, pluginDir, d.ToolStoreRootURI(), false, 0)
}

// ListToolStoreResourceURIsByPlugin 返回插件目录下的所有工具资源 URI。
func (d *PluginDAO) ListToolStoreResourceURIsByPlugin(ctx context.Context, pluginID string) ([]string, error) {
	if !d.ToolStoreConfigured() {
		return []string{}, nil
	}
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return []string{}, nil
	}
	rootURI := d.ToolStoreRootURI()
	entries, err := listEntries(ctx, rootURI, true)
	if err != nil {
		if isToolStoreNotFoundError(err) {
			return []string{}, nil
		}
		return nil, err
	}

	seen := make(map[string]struct{}, len(entries))
	uris := make([]string, 0, len(entries))
	pluginPrefix := pluginID + "/"
	for _, entry := range entries {
		if entry.IsDir {
			continue
		}
		relativePath := toolStoreRelativePathFromEntry(rootURI, entry.Path, entry.Name)
		if relativePath == "" || !strings.HasPrefix(relativePath, pluginPrefix) {
			continue
		}
		uri := strings.TrimRight(rootURI, "/") + "/" + relativePath
		if _, ok := seen[uri]; ok {
			continue
		}
		seen[uri] = struct{}{}
		uris = append(uris, uri)
	}
	sort.Strings(uris)
	return uris, nil
}

// DeleteToolStoreResourceURI 删除指定工具资源 URI。
func (d *PluginDAO) DeleteToolStoreResourceURI(ctx context.Context, resourceURI string) error {
	if !d.ToolStoreConfigured() {
		return nil
	}
	resourceURI = strings.TrimSpace(resourceURI)
	if resourceURI == "" {
		return nil
	}
	if err := deletePath(ctx, resourceURI, false); err != nil {
		if isToolStoreNotFoundError(err) {
			return nil
		}
		return err
	}
	return nil
}

// DeleteToolStorePluginURI 删除插件目录下全部工具资源。
func (d *PluginDAO) DeleteToolStorePluginURI(ctx context.Context, pluginID string) error {
	if !d.ToolStoreConfigured() {
		return nil
	}
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return nil
	}
	uris, err := d.ListToolStoreResourceURIsByPlugin(ctx, pluginID)
	if err != nil {
		return err
	}
	for _, uri := range uris {
		if deleteErr := d.DeleteToolStoreResourceURI(ctx, uri); deleteErr != nil {
			if isToolStoreNotFoundError(deleteErr) {
				continue
			}
			return deleteErr
		}
	}
	return nil
}

// normalizeToolStoreURI 将 URI 规范化为尾斜杠形式。
func normalizeToolStoreURI(uri string) string {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return ""
	}
	return strings.TrimRight(uri, "/") + "/"
}

// normalizeToolStoreEntryURI 规范化 fs/ls 返回的资源路径。
func normalizeToolStoreEntryURI(pluginURI string, entryPath string, entryName string) string {
	relativePath := toolStoreRelativePathFromEntry(pluginURI, entryPath, entryName)
	if relativePath == "" {
		return ""
	}
	return strings.TrimRight(pluginURI, "/") + "/" + relativePath
}

// toolStoreRelativePathFromEntry 从 fs/ls 返回条目提取相对根目录路径。
func toolStoreRelativePathFromEntry(rootURI string, entryPath string, entryName string) string {
	candidate := strings.TrimSpace(entryPath)
	if candidate == "" {
		candidate = strings.TrimSpace(entryName)
	}
	if candidate == "" {
		return ""
	}

	rootURI = strings.TrimRight(strings.TrimSpace(rootURI), "/")
	if strings.HasPrefix(candidate, "viking://") {
		if rootURI != "" && strings.HasPrefix(candidate, rootURI) {
			candidate = strings.TrimPrefix(candidate, rootURI)
		} else {
			candidate = pathFromToolsMarker(candidate)
		}
	} else if rootURI != "" && strings.HasPrefix(candidate, rootURI) {
		candidate = strings.TrimPrefix(candidate, rootURI)
	} else {
		candidate = pathFromToolsMarker(candidate)
	}

	candidate = strings.TrimPrefix(strings.TrimSpace(candidate), "/")
	if candidate == "" {
		return ""
	}
	cleaned := strings.TrimPrefix(path.Clean("/"+candidate), "/")
	if cleaned == "." || cleaned == "" {
		return ""
	}
	return cleaned
}

// pathFromToolsMarker 尝试从返回路径中提取 resources/tools 之后的路径。
func pathFromToolsMarker(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "\\", "/")
	const marker = "resources/tools/"
	if idx := strings.Index(value, marker); idx >= 0 {
		return value[idx+len(marker):]
	}
	return value
}

// validateToolStoreToolID 校验 tool_id 是否可用作资源文件名。
func validateToolStoreToolID(toolID string) error {
	toolID = strings.TrimSpace(toolID)
	if toolID == "" {
		return errors.New("tool_id is required")
	}
	if strings.ContainsAny(toolID, `/\\`) {
		return fmt.Errorf("invalid tool_id: %s", toolID)
	}
	return nil
}

// isToolStoreNotFoundError 判断是否为 OpenViking 路径不存在错误。
func isToolStoreNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "not found") ||
		strings.Contains(message, "does not exist") ||
		strings.Contains(message, "no such directory") ||
		strings.Contains(message, "no such file")
}

// isToolStoreAlreadyExistsError 判断目录已存在错误。
func isToolStoreAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "already exists") || strings.Contains(message, "exist")
}

// isToolStoreDuplicateError 判断重复写入相关错误。
func isToolStoreDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	if strings.Contains(message, "duplicate") {
		return true
	}
	if strings.Contains(message, "cannot resolve unique name") {
		return true
	}
	return false
}

// writeToolStoreDocumentFiles 将工具文档集合写入本地临时目录，供批量导入使用。
func writeToolStoreDocumentFiles(targetDir string, documents map[string]string) error {
	targetDir = strings.TrimSpace(targetDir)
	if targetDir == "" {
		return errors.New("target dir is required")
	}
	if len(documents) == 0 {
		return nil
	}
	toolIDs := make([]string, 0, len(documents))
	for toolID := range documents {
		toolIDs = append(toolIDs, toolID)
	}
	sort.Strings(toolIDs)

	for _, toolID := range toolIDs {
		if err := validateToolStoreToolID(toolID); err != nil {
			return err
		}
		filePath := filepath.Join(targetDir, toolID+".md")
		if err := os.WriteFile(filePath, []byte(documents[toolID]), 0o600); err != nil {
			return err
		}
	}
	return nil
}

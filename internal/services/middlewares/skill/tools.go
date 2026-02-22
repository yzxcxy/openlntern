package skillmiddleware

import (
	"context"
	"encoding/json"
	"errors"
	"path"
	"strconv"
	"strings"
	"time"

	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type listSkillFilesInput struct {
	Skill string `json:"skill" jsonschema_description:"技能名称"`
	Path  string `json:"path,omitempty" jsonschema_description:"可选，技能目录内的相对路径"`
}

type readSkillFileInput struct {
	Skill string `json:"skill" jsonschema_description:"技能名称"`
	Path  string `json:"path" jsonschema_description:"技能目录内的相对路径"`
}

type skillFileItem struct {
	Path  string `json:"path"`
	Type  string `json:"type"`
	Size  int64  `json:"size"`
	Mtime string `json:"mtime"`
}

func GetSkillFileTools(client OpenVikingClient) ([]einoTool.BaseTool, error) {
	if client == nil {
		return nil, errors.New("openviking client is required")
	}
	listTool, err := utils.InferTool[listSkillFilesInput, string](
		"list_skill_files",
		"列出技能目录内的文件与子目录，返回相对路径、类型、大小与修改时间。",
		listSkillFilesImpl(client),
	)
	if err != nil {
		return nil, err
	}
	readTool, err := utils.InferTool[readSkillFileInput, string](
		"read_skill_file",
		"专门用于读取技能目录内的指定文件，返回纯文本内容。",
		readSkillFileImpl(client),
	)
	if err != nil {
		return nil, err
	}
	return []einoTool.BaseTool{listTool, readTool}, nil
}

func listSkillFilesImpl(client OpenVikingClient) func(context.Context, listSkillFilesInput) (string, error) {
	return func(ctx context.Context, input listSkillFilesInput) (string, error) {
		skillName, err := validateSkillName(input.Skill)
		if err != nil {
			return "", err
		}
		relPath, err := normalizeRelativePath(input.Path, true)
		if err != nil {
			return "", err
		}
		rootURI := buildSkillURI(strings.TrimRight(client.SkillsRoot(), "/"), skillName, "")
		listURI := rootURI
		if relPath != "" {
			listURI = buildSkillURI(rootURI, relPath, "")
		}
		entries, err := client.List(ctx, listURI, false)
		if err != nil {
			return "", err
		}
		result := make([]skillFileItem, 0, len(entries))
		for _, entry := range entries {
			entryPath := entryString(entry, "path", "uri")
			entryName := entryString(entry, "name")
			rel := relativePath(rootURI, entryPath)
			if rel == "" {
				rel = entryName
			}
			itemType := "file"
			if entryIsDir(entry) {
				itemType = "dir"
			}
			entryMtime := entryTime(entry)
			mtime := ""
			if !entryMtime.IsZero() {
				mtime = entryMtime.Format(time.RFC3339)
			}
			result = append(result, skillFileItem{
				Path:  rel,
				Type:  itemType,
				Size:  entryInt64(entry, "size"),
				Mtime: mtime,
			})
		}
		payload, err := json.Marshal(result)
		if err != nil {
			return "", err
		}
		return string(payload), nil
	}
}

func readSkillFileImpl(client OpenVikingClient) func(context.Context, readSkillFileInput) (string, error) {
	return func(ctx context.Context, input readSkillFileInput) (string, error) {
		skillName, err := validateSkillName(input.Skill)
		if err != nil {
			return "", err
		}
		relPath, err := normalizeRelativePath(input.Path, false)
		if err != nil {
			return "", err
		}
		rootURI := buildSkillURI(strings.TrimRight(client.SkillsRoot(), "/"), skillName, "")
		targetURI := buildSkillURI(rootURI, relPath, "")
		return client.ReadContent(ctx, targetURI)
	}
}

func validateSkillName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", errors.New("skill name is required")
	}
	if strings.Contains(trimmed, "..") || strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") {
		return "", errors.New("invalid skill name")
	}
	return trimmed, nil
}

func normalizeRelativePath(input string, allowEmpty bool) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		if allowEmpty {
			return "", nil
		}
		return "", errors.New("path is required")
	}
	if strings.HasPrefix(trimmed, "/") || strings.Contains(trimmed, "\\") {
		return "", errors.New("invalid path")
	}
	segments := strings.Split(trimmed, "/")
	for _, seg := range segments {
		if seg == ".." {
			return "", errors.New("invalid path")
		}
	}
	cleaned := path.Clean(trimmed)
	if cleaned == "." {
		if allowEmpty {
			return "", nil
		}
		return "", errors.New("invalid path")
	}
	if strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", errors.New("invalid path")
	}
	return cleaned, nil
}

func entryTime(entry map[string]any) time.Time {
	for _, key := range []string{"mtime", "modified", "updated_at"} {
		if value, ok := entry[key]; ok {
			switch v := value.(type) {
			case time.Time:
				return v
			case string:
				parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(v))
				if err == nil {
					return parsed
				}
			case float64:
				return time.Unix(int64(v), 0)
			case int64:
				return time.Unix(v, 0)
			case int:
				return time.Unix(int64(v), 0)
			}
		}
	}
	return time.Time{}
}

func entryInt64(entry map[string]any, key string) int64 {
	value, ok := entry[key]
	if !ok {
		return 0
	}
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0
		}
		parsed, err := strconv.ParseInt(trimmed, 10, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

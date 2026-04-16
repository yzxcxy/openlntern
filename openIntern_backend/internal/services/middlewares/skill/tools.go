package skillmiddleware

import (
	"context"
	"encoding/json"
	"errors"
	"path"
	"strings"

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

func GetSkillFileTools(repo SkillRepository) ([]einoTool.BaseTool, error) {
	if repo == nil {
		return nil, errors.New("skill repository is required")
	}
	listTool, err := utils.InferTool[listSkillFilesInput, string](
		"list_skill_files",
		"列出技能目录内的文件与子目录，返回相对路径、类型、大小与修改时间。Skill 文件不在 sandbox 内，不能通过 bash/cat 直接读取。",
		listSkillFilesImpl(repo),
	)
	if err != nil {
		return nil, err
	}
	readTool, err := utils.InferTool[readSkillFileInput, string](
		"read_skill_file",
		"专门用于读取技能目录内的指定文件，返回纯文本内容。Skill 文件不在 sandbox 内，不能通过 bash/cat 直接读取。",
		readSkillFileImpl(repo),
	)
	if err != nil {
		return nil, err
	}
	return []einoTool.BaseTool{listTool, readTool}, nil
}

func listSkillFilesImpl(repo SkillRepository) func(context.Context, listSkillFilesInput) (string, error) {
	return func(ctx context.Context, input listSkillFilesInput) (string, error) {
		skillName, err := validateSkillName(input.Skill)
		if err != nil {
			return "", err
		}
		relPath, err := normalizeRelativePath(input.Path, true)
		if err != nil {
			return "", err
		}
		entries, err := repo.ListFilesInDirectory(ctx, skillName, relPath)
		if err != nil {
			return "", err
		}
		result := make([]skillFileItem, 0, len(entries))
		for _, entry := range entries {
			result = append(result, skillFileItem{
				Path:  entry.Path,
				Type:  entry.Type,
				Size:  entry.Size,
				Mtime: entry.ModifiedAt,
			})
		}
		payload, err := json.Marshal(result)
		if err != nil {
			return "", err
		}
		return string(payload), nil
	}
}

func readSkillFileImpl(repo SkillRepository) func(context.Context, readSkillFileInput) (string, error) {
	return func(ctx context.Context, input readSkillFileInput) (string, error) {
		skillName, err := validateSkillName(input.Skill)
		if err != nil {
			return "", err
		}
		relPath, err := normalizeRelativePath(input.Path, false)
		if err != nil {
			return "", err
		}
		return repo.ReadFile(ctx, skillName+"/"+relPath)
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

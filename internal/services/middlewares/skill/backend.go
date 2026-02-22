package skillmiddleware

import (
	"context"
	"errors"
	"fmt"
	"strings"

	einoSkill "github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/goccy/go-yaml"
)

type OpenVikingClient interface {
	SkillsRoot() string
	List(ctx context.Context, uri string, recursive bool) ([]map[string]any, error)
	ReadContent(ctx context.Context, uri string) (string, error)
}

type SkillFrontmatterRecord struct {
	SkillName string
	Raw       string
}

type SkillFrontmatterStore interface {
	ListByNames(names []string) ([]SkillFrontmatterRecord, error)
	GetByName(name string) (*SkillFrontmatterRecord, error)
}

type OpenVikingBackend struct {
	client     OpenVikingClient
	store      SkillFrontmatterStore
	skillsRoot string
}

func NewOpenVikingBackend(client OpenVikingClient, store SkillFrontmatterStore) (*OpenVikingBackend, error) {
	if client == nil {
		return nil, errors.New("openviking client is required")
	}
	if store == nil {
		return nil, errors.New("skill frontmatter store is required")
	}
	root := strings.TrimSpace(client.SkillsRoot())
	if root == "" {
		return nil, errors.New("openviking skills_root not configured")
	}
	return &OpenVikingBackend{
		client:     client,
		store:      store,
		skillsRoot: strings.TrimRight(root, "/"),
	}, nil
}

func (b *OpenVikingBackend) List(ctx context.Context) ([]einoSkill.FrontMatter, error) {
	entries, err := b.client.List(ctx, b.skillsRoot, false)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	seen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if !entryIsDir(entry) {
			continue
		}
		entryPath := entryString(entry, "path", "uri")
		entryName := entryString(entry, "name")
		skillPath := relativePath(b.skillsRoot, entryPath)
		if skillPath == "" {
			skillPath = entryName
		}
		if skillPath == "" {
			continue
		}
		if strings.Contains(skillPath, "/") {
			skillPath = strings.Split(skillPath, "/")[0]
		}
		if skillPath == "" || seen[skillPath] {
			continue
		}
		seen[skillPath] = true
		names = append(names, skillPath)
	}
	frontmatters, err := b.store.ListByNames(names)
	if err != nil {
		return nil, err
	}
	descIndex := make(map[string]string, len(frontmatters))
	for _, item := range frontmatters {
		parsed, err := parseFrontmatterRaw(item.Raw)
		if err != nil {
			return nil, err
		}
		if parsed.Name == "" {
			return nil, errors.New("frontmatter missing name")
		}
		descIndex[item.SkillName] = parsed.Description
	}
	result := make([]einoSkill.FrontMatter, 0, len(names))
	for _, name := range names {
		result = append(result, einoSkill.FrontMatter{
			Name:        name,
			Description: descIndex[name],
		})
	}
	return result, nil
}

func (b *OpenVikingBackend) Get(ctx context.Context, name string) (einoSkill.Skill, error) {
	if strings.TrimSpace(name) == "" {
		return einoSkill.Skill{}, errors.New("skill name is required")
	}
	record, err := b.store.GetByName(name)
	if err != nil {
		return einoSkill.Skill{}, err
	}
	if record == nil || strings.TrimSpace(record.SkillName) == "" {
		return einoSkill.Skill{}, errors.New("skill frontmatter not found")
	}
	parsed, err := parseFrontmatterRaw(record.Raw)
	if err != nil {
		return einoSkill.Skill{}, err
	}
	if parsed.Name == "" {
		return einoSkill.Skill{}, errors.New("frontmatter missing name")
	}
	contentURI := buildSkillURI(b.skillsRoot, record.SkillName, "SKILL.md")
	content, err := b.client.ReadContent(ctx, contentURI)
	if err != nil {
		return einoSkill.Skill{}, err
	}
	baseDir := buildSkillURI(b.skillsRoot, record.SkillName, "")
	return einoSkill.Skill{
		FrontMatter: einoSkill.FrontMatter{
			Name:        record.SkillName,
			Description: parsed.Description,
		},
		Content:       strings.TrimSpace(content),
		BaseDirectory: baseDir,
	}, nil
}

type frontmatterPayload struct {
	Name        string
	Description string
}

func parseFrontmatterRaw(raw string) (frontmatterPayload, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return frontmatterPayload{}, errors.New("empty frontmatter")
	}
	var data map[string]any
	if err := yaml.Unmarshal([]byte(trimmed), &data); err != nil {
		return frontmatterPayload{}, err
	}
	return frontmatterPayload{
		Name:        stringFromAny(data["name"]),
		Description: stringFromAny(data["description"]),
	}, nil
}

func stringFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func entryString(entry map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := entry[key]; ok {
			if str := stringFromAny(value); str != "" {
				return str
			}
		}
	}
	return ""
}

func entryBool(entry map[string]any, keys ...string) bool {
	for _, key := range keys {
		value, ok := entry[key]
		if !ok {
			continue
		}
		if v, ok := value.(bool); ok {
			return v
		}
		if str := stringFromAny(value); str != "" {
			return str == "true" || str == "1"
		}
	}
	return false
}

func entryIsDir(entry map[string]any) bool {
	if entryBool(entry, "is_dir") {
		return true
	}
	if entryBool(entry, "dir") {
		return true
	}
	if entryString(entry, "type") == "dir" {
		return true
	}
	return false
}

func relativePath(rootURI string, fullPath string) string {
	trimmedRoot := strings.TrimRight(rootURI, "/")
	trimmedFull := strings.TrimSpace(fullPath)
	if trimmedRoot == "" || trimmedFull == "" {
		return ""
	}
	if strings.HasPrefix(trimmedFull, trimmedRoot) {
		rel := strings.TrimPrefix(trimmedFull, trimmedRoot)
		rel = strings.TrimPrefix(rel, "/")
		return rel
	}
	return ""
}

func buildSkillURI(root string, skill string, suffix string) string {
	base := strings.TrimRight(root, "/") + "/" + strings.TrimLeft(skill, "/")
	if suffix == "" {
		return base
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(suffix, "/")
}

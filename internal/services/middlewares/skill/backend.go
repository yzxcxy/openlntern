package skillmiddleware

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"openIntern/internal/dao"

	einoSkill "github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/goccy/go-yaml"
)

type SkillRepository interface {
	ListSkillNames(ctx context.Context) ([]string, error)
	ListFilesInDirectory(ctx context.Context, skillName string, relPath string) ([]dao.SkillFileEntry, error)
	ReadSummary(ctx context.Context, skillName string) (string, error)
	ReadFile(ctx context.Context, skillPath string) (string, error)
	BuildURI(skillPath string) (string, error)
}

type SkillFrontmatterRecord struct {
	SkillName string
	Raw       string
}

type SkillFrontmatterStore interface {
	ListByNames(names []string) ([]SkillFrontmatterRecord, error)
	GetByName(name string) (*SkillFrontmatterRecord, error)
}

type RemoteBackend struct {
	repo  SkillRepository
	store SkillFrontmatterStore
}

func NewRemoteBackend(repo SkillRepository, store SkillFrontmatterStore) (*RemoteBackend, error) {
	if repo == nil {
		return nil, errors.New("skill repository is required")
	}
	if store == nil {
		return nil, errors.New("skill frontmatter store is required")
	}
	if _, err := repo.BuildURI(""); err != nil {
		return nil, err
	}
	return &RemoteBackend{
		repo:  repo,
		store: store,
	}, nil
}

func (b *RemoteBackend) List(ctx context.Context) ([]einoSkill.FrontMatter, error) {
	names, err := b.repo.ListSkillNames(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]einoSkill.FrontMatter, 0, len(names))
	for _, name := range names {
		abstract, err := b.repo.ReadSummary(ctx, name)
		if err != nil {
			return nil, err
		}
		result = append(result, einoSkill.FrontMatter{
			Name:        name,
			Description: strings.TrimSpace(abstract),
		})
	}
	return result, nil
}

func (b *RemoteBackend) Get(ctx context.Context, name string) (einoSkill.Skill, error) {
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
	content, err := b.repo.ReadFile(ctx, record.SkillName+"/SKILL.md")
	if err != nil {
		return einoSkill.Skill{}, err
	}
	baseDir, err := b.repo.BuildURI(record.SkillName)
	if err != nil {
		return einoSkill.Skill{}, err
	}
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

var _ SkillRepository = (*dao.SkillStoreDAO)(nil)

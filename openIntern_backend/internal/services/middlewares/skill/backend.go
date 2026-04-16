package skillmiddleware

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	neturl "net/url"
	"strings"
	"syscall"

	"openIntern/internal/dao"

	einoSkill "github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/goccy/go-yaml"
)

type SkillRepository interface {
	ListSkillNames(ctx context.Context) ([]string, error)
	ListFilesInDirectory(ctx context.Context, skillName string, relPath string) ([]dao.SkillFileEntry, error)
	ReadSummary(ctx context.Context, skillName string) (string, error)
	ReadFile(ctx context.Context, skillPath string) (string, error)
	BuildURI(ctx context.Context, skillPath string) (string, error)
}

type SkillFrontmatterRecord struct {
	SkillName string
	Raw       string
}

type SkillFrontmatterStore interface {
	ListByUserIDAndNames(userID string, names []string) ([]SkillFrontmatterRecord, error)
	GetByUserIDAndName(userID, name string) (*SkillFrontmatterRecord, error)
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
	return &RemoteBackend{
		repo:  repo,
		store: store,
	}, nil
}

func (b *RemoteBackend) List(ctx context.Context) ([]einoSkill.FrontMatter, error) {
	names, err := b.repo.ListSkillNames(ctx)
	if err != nil {
		if isTransientSkillStoreError(err) {
			// OpenViking 短暂不可用时退化为空技能集，避免整次对话直接失败。
			log.Printf("Skill backend list degraded because repository is unavailable err=%v", err)
			return []einoSkill.FrontMatter{}, nil
		}
		return nil, err
	}
	result := make([]einoSkill.FrontMatter, 0, len(names))
	for _, name := range names {
		abstract, err := b.repo.ReadSummary(ctx, name)
		if err != nil {
			if isTransientSkillStoreError(err) {
				// 单个技能摘要加载失败时跳过该技能，保留其余技能能力。
				log.Printf("Skill backend skip summary because repository is unavailable skill=%s err=%v", name, err)
				continue
			}
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
	userID, err := dao.OpenVikingUserIDFromContext(ctx)
	if err != nil {
		return einoSkill.Skill{}, err
	}
	record, err := b.store.GetByUserIDAndName(userID, name)
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
	baseDir, err := b.repo.BuildURI(ctx, record.SkillName)
	if err != nil {
		return einoSkill.Skill{}, err
	}
	return einoSkill.Skill{
		FrontMatter: einoSkill.FrontMatter{
			Name:        record.SkillName,
			Description: parsed.Description,
		},
		Content: strings.TrimSpace(content),
		// 这里返回给上游 skill middleware 的 BaseDirectory 只用于向模型展示技能来源，
		// 不是 sandbox 内可直接访问的真实目录，否则模型容易误判为可用 bash/cat 直接读取。
		BaseDirectory: buildLogicalSkillBaseDirectory(baseDir),
	}, nil
}

func buildLogicalSkillBaseDirectory(baseDir string) string {
	trimmed := strings.TrimSpace(baseDir)
	if trimmed == "" {
		return "skill repository (logical only; not mounted in sandbox. Use list_skill_files/read_skill_file to access files.)"
	}
	return trimmed + " [logical skill repository only; not mounted in sandbox. Use list_skill_files/read_skill_file to access files.]"
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

// isTransientSkillStoreError identifies short-lived OpenViking/network failures that should not abort the whole run.
func isTransientSkillStoreError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) || errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) {
		return true
	}
	var urlErr *neturl.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return true
		}
		err = urlErr.Err
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "connection reset by peer") ||
		strings.Contains(lower, "broken pipe") ||
		strings.Contains(lower, "unexpected eof") ||
		strings.Contains(lower, "context deadline exceeded") ||
		strings.Contains(lower, "timeout")
}

var _ SkillRepository = (*dao.SkillStoreDAO)(nil)

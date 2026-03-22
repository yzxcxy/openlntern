package skillmiddleware

import (
	"context"
	"errors"
	"strings"

	"openIntern/internal/dao"
	"openIntern/internal/util"
)

// ScopedRepository limits visible skills to the configured allow-list.
type ScopedRepository struct {
	repo    SkillRepository
	allowed map[string]struct{}
}

// ScopedFrontmatterStore limits frontmatter access to the configured allow-list.
type ScopedFrontmatterStore struct {
	store   SkillFrontmatterStore
	allowed map[string]struct{}
}

// NewScopedRepository creates a skill repository wrapper constrained to agent-bound skills.
func NewScopedRepository(repo SkillRepository, allowedNames []string) (*ScopedRepository, error) {
	if repo == nil {
		return nil, errors.New("skill repository is required")
	}
	return &ScopedRepository{
		repo:    repo,
		allowed: buildAllowedSkillSet(allowedNames),
	}, nil
}

// NewScopedFrontmatterStore creates a frontmatter wrapper constrained to agent-bound skills.
func NewScopedFrontmatterStore(store SkillFrontmatterStore, allowedNames []string) (*ScopedFrontmatterStore, error) {
	if store == nil {
		return nil, errors.New("skill frontmatter store is required")
	}
	return &ScopedFrontmatterStore{
		store:   store,
		allowed: buildAllowedSkillSet(allowedNames),
	}, nil
}

func (r *ScopedRepository) ListSkillNames(ctx context.Context) ([]string, error) {
	names, err := r.repo.ListSkillNames(ctx)
	if err != nil {
		return nil, err
	}
	return r.filterNames(names), nil
}

func (r *ScopedRepository) ListFilesInDirectory(ctx context.Context, skillName string, relPath string) ([]dao.SkillFileEntry, error) {
	if err := r.ensureAllowed(skillName); err != nil {
		return nil, err
	}
	return r.repo.ListFilesInDirectory(ctx, skillName, relPath)
}

func (r *ScopedRepository) ReadSummary(ctx context.Context, skillName string) (string, error) {
	if err := r.ensureAllowed(skillName); err != nil {
		return "", err
	}
	return r.repo.ReadSummary(ctx, skillName)
}

func (r *ScopedRepository) ReadFile(ctx context.Context, skillPath string) (string, error) {
	skillName := firstSkillSegment(skillPath)
	if err := r.ensureAllowed(skillName); err != nil {
		return "", err
	}
	return r.repo.ReadFile(ctx, skillPath)
}

func (r *ScopedRepository) BuildURI(skillPath string) (string, error) {
	skillName := firstSkillSegment(skillPath)
	if skillName != "" {
		if err := r.ensureAllowed(skillName); err != nil {
			return "", err
		}
	}
	return r.repo.BuildURI(skillPath)
}

func (s *ScopedFrontmatterStore) ListByNames(names []string) ([]SkillFrontmatterRecord, error) {
	allowedNames := s.filterNames(names)
	if len(allowedNames) == 0 {
		return []SkillFrontmatterRecord{}, nil
	}
	return s.store.ListByNames(allowedNames)
}

func (s *ScopedFrontmatterStore) GetByName(name string) (*SkillFrontmatterRecord, error) {
	if err := s.ensureAllowed(name); err != nil {
		return nil, err
	}
	return s.store.GetByName(name)
}

func (r *ScopedRepository) ensureAllowed(skillName string) error {
	if _, ok := r.allowed[strings.TrimSpace(skillName)]; !ok {
		return errors.New("skill is not bound to current agent")
	}
	return nil
}

func (s *ScopedFrontmatterStore) ensureAllowed(skillName string) error {
	if _, ok := s.allowed[strings.TrimSpace(skillName)]; !ok {
		return errors.New("skill is not bound to current agent")
	}
	return nil
}

func (r *ScopedRepository) filterNames(names []string) []string {
	filtered := make([]string, 0, len(names))
	for _, name := range util.NormalizeUniqueStringList(names) {
		if _, ok := r.allowed[name]; ok {
			filtered = append(filtered, name)
		}
	}
	return filtered
}

func (s *ScopedFrontmatterStore) filterNames(names []string) []string {
	filtered := make([]string, 0, len(names))
	for _, name := range util.NormalizeUniqueStringList(names) {
		if _, ok := s.allowed[name]; ok {
			filtered = append(filtered, name)
		}
	}
	return filtered
}

func buildAllowedSkillSet(names []string) map[string]struct{} {
	normalized := util.NormalizeUniqueStringList(names)
	result := make(map[string]struct{}, len(normalized))
	for _, name := range normalized {
		result[name] = struct{}{}
	}
	return result
}

func firstSkillSegment(skillPath string) string {
	trimmed := strings.Trim(strings.TrimSpace(skillPath), "/")
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	return strings.TrimSpace(parts[0])
}

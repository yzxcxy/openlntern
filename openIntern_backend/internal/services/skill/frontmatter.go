package skill

import (
	"errors"
	"openIntern/internal/dao"
	"openIntern/internal/models"
)

type SkillFrontmatterService struct{}

var SkillFrontmatter = new(SkillFrontmatterService)

func (s *SkillFrontmatterService) CreateOrReplaceByName(frontmatter *models.SkillFrontmatter) error {
	if frontmatter == nil || frontmatter.SkillName == "" {
		return errors.New("invalid frontmatter")
	}
	return dao.SkillFrontmatter.ReplaceBySkillName(frontmatter)
}

func (s *SkillFrontmatterService) GetByName(name string) (*models.SkillFrontmatter, error) {
	if name == "" {
		return nil, errors.New("empty skill name")
	}
	return dao.SkillFrontmatter.GetLatestByName(name)
}

func (s *SkillFrontmatterService) ListByNames(names []string) ([]models.SkillFrontmatter, error) {
	if len(names) == 0 {
		return []models.SkillFrontmatter{}, nil
	}
	results, err := dao.SkillFrontmatter.ListLatestByNames(names)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(names))
	filtered := make([]models.SkillFrontmatter, 0, len(names))
	for _, item := range results {
		if item.SkillName == "" || seen[item.SkillName] {
			continue
		}
		seen[item.SkillName] = true
		filtered = append(filtered, item)
	}
	return filtered, nil
}

func (s *SkillFrontmatterService) DeleteByName(name string) error {
	if name == "" {
		return errors.New("empty skill name")
	}
	return dao.SkillFrontmatter.DeleteByName(name)
}

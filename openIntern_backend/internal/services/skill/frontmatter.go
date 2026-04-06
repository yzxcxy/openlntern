package skill

import (
	"errors"
	"openIntern/internal/dao"
	"openIntern/internal/models"
	"strings"
)

type SkillFrontmatterService struct{}

var SkillFrontmatter = new(SkillFrontmatterService)

func (s *SkillFrontmatterService) CreateOrReplaceByUserIDAndName(frontmatter *models.SkillFrontmatter) error {
	if frontmatter == nil || strings.TrimSpace(frontmatter.UserID) == "" || frontmatter.SkillName == "" {
		return errors.New("invalid frontmatter")
	}
	return dao.SkillFrontmatter.ReplaceByUserIDAndSkillName(frontmatter)
}

func (s *SkillFrontmatterService) GetByUserIDAndName(userID, name string) (*models.SkillFrontmatter, error) {
	if strings.TrimSpace(userID) == "" || name == "" {
		return nil, errors.New("empty user_id or skill name")
	}
	return dao.SkillFrontmatter.GetLatestByUserIDAndName(strings.TrimSpace(userID), name)
}

func (s *SkillFrontmatterService) ListByUserIDAndNames(userID string, names []string) ([]models.SkillFrontmatter, error) {
	if strings.TrimSpace(userID) == "" || len(names) == 0 {
		return []models.SkillFrontmatter{}, nil
	}
	results, err := dao.SkillFrontmatter.ListLatestByUserIDAndNames(strings.TrimSpace(userID), names)
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

func (s *SkillFrontmatterService) DeleteByUserIDAndName(userID, name string) error {
	if strings.TrimSpace(userID) == "" || name == "" {
		return errors.New("empty user_id or skill name")
	}
	return dao.SkillFrontmatter.DeleteByUserIDAndName(strings.TrimSpace(userID), name)
}

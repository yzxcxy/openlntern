package services

import (
	"errors"
	"openIntern/internal/database"
	"openIntern/internal/models"

	"gorm.io/gorm"
)

type SkillFrontmatterService struct{}

var SkillFrontmatter = new(SkillFrontmatterService)

func (s *SkillFrontmatterService) CreateOrReplaceByName(frontmatter *models.SkillFrontmatter) error {
	if frontmatter == nil || frontmatter.SkillName == "" {
		return errors.New("invalid frontmatter")
	}
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("skill_name = ?", frontmatter.SkillName).Delete(&models.SkillFrontmatter{}).Error; err != nil {
			return err
		}
		return tx.Create(frontmatter).Error
	})
}

func (s *SkillFrontmatterService) GetByName(name string) (*models.SkillFrontmatter, error) {
	if name == "" {
		return nil, errors.New("empty skill name")
	}
	var result models.SkillFrontmatter
	if err := database.DB.Where("skill_name = ?", name).Order("created_at desc").First(&result).Error; err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *SkillFrontmatterService) ListByNames(names []string) ([]models.SkillFrontmatter, error) {
	if len(names) == 0 {
		return []models.SkillFrontmatter{}, nil
	}
	var results []models.SkillFrontmatter
	if err := database.DB.Where("skill_name IN ?", names).Order("created_at desc").Find(&results).Error; err != nil {
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
	return database.DB.Where("skill_name = ?", name).Delete(&models.SkillFrontmatter{}).Error
}

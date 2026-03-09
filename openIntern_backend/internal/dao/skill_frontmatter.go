package dao

import (
	"openIntern/internal/database"
	"openIntern/internal/models"

	"gorm.io/gorm"
)

type SkillFrontmatterDAO struct{}

var SkillFrontmatter = new(SkillFrontmatterDAO)

func (d *SkillFrontmatterDAO) ReplaceBySkillName(frontmatter *models.SkillFrontmatter) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("skill_name = ?", frontmatter.SkillName).Delete(&models.SkillFrontmatter{}).Error; err != nil {
			return err
		}
		return tx.Create(frontmatter).Error
	})
}

func (d *SkillFrontmatterDAO) GetLatestByName(name string) (*models.SkillFrontmatter, error) {
	var item models.SkillFrontmatter
	if err := database.DB.Where("skill_name = ?", name).Order("created_at desc").First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *SkillFrontmatterDAO) ListLatestByNames(names []string) ([]models.SkillFrontmatter, error) {
	var items []models.SkillFrontmatter
	if err := database.DB.Where("skill_name IN ?", names).Order("created_at desc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (d *SkillFrontmatterDAO) DeleteByName(name string) error {
	return database.DB.Where("skill_name = ?", name).Delete(&models.SkillFrontmatter{}).Error
}

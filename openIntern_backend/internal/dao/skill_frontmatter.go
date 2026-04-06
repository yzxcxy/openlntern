package dao

import (
	"openIntern/internal/database"
	"openIntern/internal/models"

	"gorm.io/gorm"
)

type SkillFrontmatterDAO struct{}

var SkillFrontmatter = new(SkillFrontmatterDAO)

func (d *SkillFrontmatterDAO) ReplaceByUserIDAndSkillName(frontmatter *models.SkillFrontmatter) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND skill_name = ?", frontmatter.UserID, frontmatter.SkillName).Delete(&models.SkillFrontmatter{}).Error; err != nil {
			return err
		}
		return tx.Create(frontmatter).Error
	})
}

func (d *SkillFrontmatterDAO) GetLatestByUserIDAndName(userID, name string) (*models.SkillFrontmatter, error) {
	var item models.SkillFrontmatter
	if err := database.DB.Where("user_id = ? AND skill_name = ?", userID, name).Order("created_at desc").First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *SkillFrontmatterDAO) ListLatestByUserIDAndNames(userID string, names []string) ([]models.SkillFrontmatter, error) {
	var items []models.SkillFrontmatter
	if err := database.DB.Where("user_id = ? AND skill_name IN ?", userID, names).Order("created_at desc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (d *SkillFrontmatterDAO) DeleteByUserIDAndName(userID, name string) error {
	return database.DB.Where("user_id = ? AND skill_name = ?", userID, name).Delete(&models.SkillFrontmatter{}).Error
}

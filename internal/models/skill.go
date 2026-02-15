package models

type SkillType string

const (
	SkillTypeOfficial SkillType = "official"
	SkillTypeCustom   SkillType = "custom"
)

type Skill struct {
	SkillID     string    `json:"skill_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        SkillType `json:"type"`
	Source      string    `json:"source"`
	Icon        string    `json:"icon"`
	Path        string    `json:"path"`
	UserID      string    `json:"user_id"`
}

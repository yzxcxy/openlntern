package models

type Skill struct {
	SkillID     string `json:"skill_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
	Icon        string `json:"icon"`
	Path        string `json:"path"`
}

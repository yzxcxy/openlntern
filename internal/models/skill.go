package models

type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Frontmatter string `json:"frontmatter"`
}

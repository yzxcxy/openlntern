package services

import (
	skillmiddleware "openIntern/internal/services/middlewares/skill"
)

type skillFrontmatterStore struct{}

func (skillFrontmatterStore) ListByNames(names []string) ([]skillmiddleware.SkillFrontmatterRecord, error) {
	items, err := SkillFrontmatter.ListByNames(names)
	if err != nil {
		return nil, err
	}
	records := make([]skillmiddleware.SkillFrontmatterRecord, 0, len(items))
	for _, item := range items {
		records = append(records, skillmiddleware.SkillFrontmatterRecord{
			SkillName: item.SkillName,
			Raw:       item.Raw,
		})
	}
	return records, nil
}

func (skillFrontmatterStore) GetByName(name string) (*skillmiddleware.SkillFrontmatterRecord, error) {
	item, err := SkillFrontmatter.GetByName(name)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	return &skillmiddleware.SkillFrontmatterRecord{
		SkillName: item.SkillName,
		Raw:       item.Raw,
	}, nil
}

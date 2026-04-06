package skill

import (
	skillmiddleware "openIntern/internal/services/middlewares/skill"
)

// FrontmatterReader describes the subset of frontmatter operations required by the skill middleware.
type FrontmatterReader interface {
	ListByUserIDAndNames(userID string, names []string) ([]skillmiddleware.SkillFrontmatterRecord, error)
	GetByUserIDAndName(userID, name string) (*skillmiddleware.SkillFrontmatterRecord, error)
}

// FrontmatterStoreAdapter adapts the skill service to the skill middleware storage interface.
type FrontmatterStoreAdapter struct {
	Store *SkillFrontmatterService
}

func (a FrontmatterStoreAdapter) ListByUserIDAndNames(userID string, names []string) ([]skillmiddleware.SkillFrontmatterRecord, error) {
	items, err := a.Store.ListByUserIDAndNames(userID, names)
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

func (a FrontmatterStoreAdapter) GetByUserIDAndName(userID, name string) (*skillmiddleware.SkillFrontmatterRecord, error) {
	item, err := a.Store.GetByUserIDAndName(userID, name)
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

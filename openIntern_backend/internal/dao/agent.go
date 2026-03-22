package dao

import (
	"strings"

	"openIntern/internal/database"
	"openIntern/internal/models"
)

type AgentListFilter struct {
	OwnerID   string
	Keyword   string
	AgentType string
	Status    string
}

type AgentDAO struct{}

var Agent = new(AgentDAO)

func (d *AgentDAO) Create(item *models.Agent) error {
	return database.DB.Create(item).Error
}

func (d *AgentDAO) GetByAgentID(agentID string) (*models.Agent, error) {
	var item models.Agent
	if err := database.DB.Where("agent_id = ?", agentID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *AgentDAO) GetByAgentIDAndOwner(agentID, ownerID string) (*models.Agent, error) {
	var item models.Agent
	if err := database.DB.Where("agent_id = ? AND owner_id = ?", agentID, ownerID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *AgentDAO) UpdateByAgentIDAndOwner(agentID, ownerID string, updates map[string]any) (int64, error) {
	result := database.DB.Model(&models.Agent{}).Where("agent_id = ? AND owner_id = ?", agentID, ownerID).Updates(updates)
	return result.RowsAffected, result.Error
}

func (d *AgentDAO) DeleteByAgentIDAndOwner(agentID, ownerID string) (int64, error) {
	result := database.DB.Where("agent_id = ? AND owner_id = ?", agentID, ownerID).Delete(&models.Agent{})
	return result.RowsAffected, result.Error
}

func (d *AgentDAO) List(page, pageSize int, filter AgentListFilter) ([]models.Agent, int64, error) {
	_, pageSize, offset := normalizePagination(page, pageSize, 10)

	query := database.DB.Model(&models.Agent{})
	if ownerID := strings.TrimSpace(filter.OwnerID); ownerID != "" {
		query = query.Where("owner_id = ?", ownerID)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Where("(name LIKE ? OR description LIKE ?)", pattern, pattern)
	}
	if agentType := strings.TrimSpace(filter.AgentType); agentType != "" {
		query = query.Where("agent_type = ?", agentType)
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []models.Agent
	if err := query.Order("sort ASC").Order("updated_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (d *AgentDAO) ListByAgentIDs(ownerID string, agentIDs []string) ([]models.Agent, error) {
	agentIDs = normalizeStringIDs(agentIDs)
	if len(agentIDs) == 0 {
		return []models.Agent{}, nil
	}
	query := database.DB.Model(&models.Agent{}).Where("agent_id IN ?", agentIDs)
	if ownerID = strings.TrimSpace(ownerID); ownerID != "" {
		query = query.Where("owner_id = ?", ownerID)
	}
	var items []models.Agent
	if err := query.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (d *AgentDAO) ListAllByOwner(ownerID string) ([]models.Agent, error) {
	var items []models.Agent
	if err := database.DB.Where("owner_id = ?", ownerID).Order("updated_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (d *AgentDAO) ListEnabledByOwner(ownerID string) ([]models.Agent, error) {
	var items []models.Agent
	if err := database.DB.Where("owner_id = ? AND status = ?", ownerID, "enabled").Order("sort ASC").Order("updated_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func normalizeStringIDs(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

package dao

import (
	"openIntern/internal/database"
	"openIntern/internal/models"

	"gorm.io/gorm"
)

type AgentBindingDAO struct{}

var AgentBinding = new(AgentBindingDAO)

func (d *AgentBindingDAO) ReplaceByAgentID(agentID string, bindings []models.AgentBinding) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("agent_id = ?", agentID).Delete(&models.AgentBinding{}).Error; err != nil {
			return err
		}
		if len(bindings) == 0 {
			return nil
		}
		return tx.Create(&bindings).Error
	})
}

func (d *AgentBindingDAO) DeleteByAgentID(agentID string) error {
	return database.DB.Where("agent_id = ?", agentID).Delete(&models.AgentBinding{}).Error
}

func (d *AgentBindingDAO) ListByAgentID(agentID string) ([]models.AgentBinding, error) {
	var items []models.AgentBinding
	if err := database.DB.Where("agent_id = ?", agentID).Order("sort ASC").Order("created_at ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (d *AgentBindingDAO) ListByAgentIDs(agentIDs []string) ([]models.AgentBinding, error) {
	agentIDs = normalizeStringIDs(agentIDs)
	if len(agentIDs) == 0 {
		return []models.AgentBinding{}, nil
	}
	var items []models.AgentBinding
	if err := database.DB.Where("agent_id IN ?", agentIDs).Order("sort ASC").Order("created_at ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (d *AgentBindingDAO) ListReferencingSubAgents(ownerID string, targetAgentID string, enabledOnly bool) ([]models.Agent, error) {
	query := database.DB.Table("agent").
		Select("agent.*").
		Joins("JOIN agent_binding ON agent.agent_id = agent_binding.agent_id").
		Where("agent.owner_id = ? AND agent_binding.binding_type = ? AND agent_binding.binding_target_id = ?", ownerID, "sub_agent", targetAgentID)
	if enabledOnly {
		query = query.Where("agent.status = ?", "enabled")
	}
	var items []models.Agent
	if err := query.Order("agent.updated_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

package dao

import (
	"openIntern/internal/database"
	"openIntern/internal/models"

	"gorm.io/gorm"
)

type AgentBindingDAO struct{}

var AgentBinding = new(AgentBindingDAO)

func (d *AgentBindingDAO) ReplaceByAgentID(userID, agentID string, bindings []models.AgentBinding) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Bindings are fully replaced on each save, so stale rows must be hard-deleted to release the unique key.
		if err := tx.Unscoped().Where("user_id = ? AND agent_id = ?", userID, agentID).Delete(&models.AgentBinding{}).Error; err != nil {
			return err
		}
		if len(bindings) == 0 {
			return nil
		}
		return tx.Create(&bindings).Error
	})
}

func (d *AgentBindingDAO) DeleteByAgentID(userID, agentID string) error {
	// Agent bindings are disposable association rows and should not be retained as soft-deleted history.
	return database.DB.Unscoped().Where("user_id = ? AND agent_id = ?", userID, agentID).Delete(&models.AgentBinding{}).Error
}

func (d *AgentBindingDAO) ListByAgentID(userID, agentID string) ([]models.AgentBinding, error) {
	var items []models.AgentBinding
	if err := database.DB.Where("user_id = ? AND agent_id = ?", userID, agentID).Order("sort ASC").Order("created_at ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (d *AgentBindingDAO) ListByAgentIDs(userID string, agentIDs []string) ([]models.AgentBinding, error) {
	agentIDs = normalizeStringIDs(agentIDs)
	if len(agentIDs) == 0 {
		return []models.AgentBinding{}, nil
	}
	var items []models.AgentBinding
	if err := database.DB.Where("user_id = ? AND agent_id IN ?", userID, agentIDs).Order("sort ASC").Order("created_at ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (d *AgentBindingDAO) ListReferencingSubAgents(ownerID string, targetAgentID string, enabledOnly bool) ([]models.Agent, error) {
	query := database.DB.Table("agent").
		Select("DISTINCT agent.*").
		Joins("JOIN agent_binding ON agent.agent_id = agent_binding.agent_id AND agent.user_id = agent_binding.user_id").
		Where("agent.user_id = ? AND agent_binding.binding_type = ? AND agent_binding.binding_target_id = ? AND agent_binding.deleted_at IS NULL", ownerID, "sub_agent", targetAgentID)
	if enabledOnly {
		query = query.Where("agent.status = ?", "enabled")
	}
	var items []models.Agent
	if err := query.Order("agent.updated_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

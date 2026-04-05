package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"openIntern/internal/dao"
	"openIntern/internal/models"
	kbsvc "openIntern/internal/services/kb"
	modelsvc "openIntern/internal/services/model"
	"openIntern/internal/util"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	AgentTypeSingle     = "single"
	AgentTypeSupervisor = "supervisor"

	AgentStatusDraft    = "draft"
	AgentStatusEnabled  = "enabled"
	AgentStatusDisabled = "disabled"

	AgentBindingTool     = "tool"
	AgentBindingSkill    = "skill"
	AgentBindingKB       = "kb"
	AgentBindingSubAgent = "sub_agent"

	maxAgentCompileDepth = 3
	// Keep backend-created agent avatars aligned with the frontend default avatar.
	defaultAgentAvatarURL = "https://open-intern-1307855818.cos.ap-guangzhou.myqcloud.com/avatar/openIntern%40200%C3%97200.jpg"
)

type AgentDefinitionService struct{}

type AgentListFilter struct {
	Keyword   string
	AgentType string
	Status    string
}

type UpsertAgentInput struct {
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	AgentType          string   `json:"agent_type"`
	SystemPrompt       string   `json:"system_prompt"`
	AvatarURL          string   `json:"avatar_url"`
	ChatBackgroundJSON string   `json:"chat_background_json"`
	ExampleQuestions   []string `json:"example_questions"`
	DefaultModelID     string   `json:"default_model_id"`
	AgentMemoryEnabled bool     `json:"agent_memory_enabled"`
	ToolIDs            []string `json:"tool_ids"`
	SkillNames         []string `json:"skill_names"`
	KnowledgeBaseNames []string `json:"knowledge_base_names"`
	SubAgentIDs        []string `json:"sub_agent_ids"`
}

type AgentListItem struct {
	AgentID             string    `json:"agent_id"`
	OwnerID             string    `json:"owner_id"`
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	AgentType           string    `json:"agent_type"`
	Status              string    `json:"status"`
	AvatarURL           string    `json:"avatar_url"`
	DefaultModelID      string    `json:"default_model_id"`
	DefaultModelName    string    `json:"default_model_name"`
	AgentMemoryEnabled  bool      `json:"agent_memory_enabled"`
	ToolCount           int       `json:"tool_count"`
	SkillCount          int       `json:"skill_count"`
	KnowledgeBaseCount  int       `json:"knowledge_base_count"`
	SubAgentCount       int       `json:"sub_agent_count"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type AgentDetailView struct {
	AgentID             string    `json:"agent_id"`
	OwnerID             string    `json:"owner_id"`
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	AgentType           string    `json:"agent_type"`
	Status              string    `json:"status"`
	SystemPrompt        string    `json:"system_prompt"`
	AvatarURL           string    `json:"avatar_url"`
	ChatBackgroundJSON  string    `json:"chat_background_json"`
	ExampleQuestions    []string  `json:"example_questions"`
	DefaultModelID      string    `json:"default_model_id"`
	DefaultModelName    string    `json:"default_model_name"`
	AgentMemoryEnabled  bool      `json:"agent_memory_enabled"`
	ToolIDs             []string  `json:"tool_ids"`
	SkillNames          []string  `json:"skill_names"`
	KnowledgeBaseNames  []string  `json:"knowledge_base_names"`
	SubAgentIDs         []string  `json:"sub_agent_ids"`
	ToolCount           int       `json:"tool_count"`
	SkillCount          int       `json:"skill_count"`
	KnowledgeBaseCount  int       `json:"knowledge_base_count"`
	SubAgentCount       int       `json:"sub_agent_count"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type EnabledAgentOption struct {
	AgentID            string   `json:"agent_id"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	AgentType          string   `json:"agent_type"`
	Status             string   `json:"status"`
	AvatarURL          string   `json:"avatar_url"`
	ChatBackgroundJSON string   `json:"chat_background_json"`
	DefaultModelID     string   `json:"default_model_id"`
	DefaultModelName   string   `json:"default_model_name"`
	ExampleQuestions   []string `json:"example_questions"`
}

var AgentDefinition = new(AgentDefinitionService)

func (s *AgentDefinitionService) Create(ctx context.Context, ownerID string, input UpsertAgentInput) (*AgentDetailView, error) {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return nil, errors.New("owner_id is required")
	}
	item := &models.Agent{
		AgentID:               uuid.NewString(),
		UserID:                ownerID,
		Status:                AgentStatusDraft,
		Name:                  strings.TrimSpace(input.Name),
		Description:           strings.TrimSpace(input.Description),
		AgentType:             normalizeAgentType(input.AgentType),
		SystemPrompt:          strings.TrimSpace(input.SystemPrompt),
		AvatarURL:             normalizeAgentAvatarURL(input.AvatarURL),
		ChatBackgroundJSON:    strings.TrimSpace(input.ChatBackgroundJSON),
		DefaultModelID:        strings.TrimSpace(input.DefaultModelID),
		AgentMemoryEnabled:    input.AgentMemoryEnabled,
		ExampleQuestionsJSON:  mustMarshalStringList(normalizeQuestionList(input.ExampleQuestions)),
	}
	normalized, err := s.validateAndNormalizeInput(ctx, ownerID, item.AgentID, input)
	if err != nil {
		return nil, err
	}
	if err := dao.Agent.Create(item); err != nil {
		return nil, err
	}
	if err := dao.AgentBinding.ReplaceByAgentID(ownerID, item.AgentID, buildBindings(ownerID, item.AgentID, normalized)); err != nil {
		return nil, err
	}
	return s.Get(ownerID, item.AgentID)
}

func (s *AgentDefinitionService) Update(ctx context.Context, ownerID, agentID string, input UpsertAgentInput) (*AgentDetailView, error) {
	item, err := dao.Agent.GetByAgentIDAndOwner(strings.TrimSpace(agentID), strings.TrimSpace(ownerID))
	if err != nil {
		return nil, err
	}
	normalized, err := s.validateAndNormalizeInput(ctx, ownerID, item.AgentID, input)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{
		"name":                   strings.TrimSpace(input.Name),
		"description":            strings.TrimSpace(input.Description),
		"agent_type":             normalizeAgentType(input.AgentType),
		"system_prompt":          strings.TrimSpace(input.SystemPrompt),
		"avatar_url":             normalizeAgentAvatarURL(input.AvatarURL),
		"chat_background_json":   strings.TrimSpace(input.ChatBackgroundJSON),
		"example_questions_json": mustMarshalStringList(normalizeQuestionList(input.ExampleQuestions)),
		"default_model_id":       strings.TrimSpace(input.DefaultModelID),
		"agent_memory_enabled":   input.AgentMemoryEnabled,
	}
	if _, err := dao.Agent.UpdateByAgentIDAndOwner(item.AgentID, ownerID, updates); err != nil {
		return nil, err
	}
	if err := dao.AgentBinding.ReplaceByAgentID(ownerID, item.AgentID, buildBindings(ownerID, item.AgentID, normalized)); err != nil {
		return nil, err
	}
	return s.Get(ownerID, item.AgentID)
}

// BuildDebugDetail validates a transient definition and materializes it into a runtime-ready view without persistence.
func (s *AgentDefinitionService) BuildDebugDetail(ctx context.Context, ownerID string, input UpsertAgentInput) (*AgentDetailView, error) {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return nil, errors.New("owner_id is required")
	}
	agentID := "debug-" + uuid.NewString()
	normalized, err := s.validateAndNormalizeInput(ctx, ownerID, agentID, input)
	if err != nil {
		return nil, err
	}
	item := models.Agent{
		AgentID:              agentID,
		UserID:               ownerID,
		Status:               AgentStatusDraft,
		Name:                 strings.TrimSpace(input.Name),
		Description:          strings.TrimSpace(input.Description),
		AgentType:            normalizeAgentType(input.AgentType),
		SystemPrompt:         strings.TrimSpace(input.SystemPrompt),
		AvatarURL:            normalizeAgentAvatarURL(input.AvatarURL),
		ChatBackgroundJSON:   strings.TrimSpace(input.ChatBackgroundJSON),
		ExampleQuestionsJSON: mustMarshalStringList(normalizeQuestionList(input.ExampleQuestions)),
		DefaultModelID:       strings.TrimSpace(input.DefaultModelID),
		AgentMemoryEnabled:   input.AgentMemoryEnabled,
	}
	return buildAgentDetailView(item, buildBindings(ownerID, agentID, normalized), loadModelName(item.DefaultModelID))
}

func (s *AgentDefinitionService) Get(ownerID, agentID string) (*AgentDetailView, error) {
	item, err := dao.Agent.GetByAgentIDAndOwner(strings.TrimSpace(agentID), strings.TrimSpace(ownerID))
	if err != nil {
		return nil, err
	}
	bindings, err := dao.AgentBinding.ListByAgentID(ownerID, item.AgentID)
	if err != nil {
		return nil, err
	}
	modelName := loadModelName(item.DefaultModelID)
	return buildAgentDetailView(*item, bindings, modelName)
}

func (s *AgentDefinitionService) List(ownerID string, page, pageSize int, filter AgentListFilter) ([]AgentListItem, int64, error) {
	items, total, err := dao.Agent.List(page, pageSize, dao.AgentListFilter{
		UserID:    strings.TrimSpace(ownerID),
		Keyword:   strings.TrimSpace(filter.Keyword),
		AgentType: normalizeOptionalAgentType(filter.AgentType),
		Status:    normalizeOptionalAgentStatus(filter.Status),
	})
	if err != nil {
		return nil, 0, err
	}
	agentIDs := make([]string, 0, len(items))
	for _, item := range items {
		agentIDs = append(agentIDs, item.AgentID)
	}
	bindings, err := dao.AgentBinding.ListByAgentIDs(ownerID, agentIDs)
	if err != nil {
		return nil, 0, err
	}
	bindingMap := groupBindingsByAgentID(bindings)
	modelNames := loadModelNames(items)
	result := make([]AgentListItem, 0, len(items))
	for _, item := range items {
		result = append(result, buildAgentListItem(item, bindingMap[item.AgentID], modelNames[item.DefaultModelID]))
	}
	return result, total, nil
}

func (s *AgentDefinitionService) SetEnabled(ctx context.Context, ownerID, agentID string, enabled bool) (*AgentDetailView, error) {
	item, err := dao.Agent.GetByAgentIDAndOwner(strings.TrimSpace(agentID), strings.TrimSpace(ownerID))
	if err != nil {
		return nil, err
	}
	if enabled {
		detail, err := s.Get(ownerID, agentID)
		if err != nil {
			return nil, err
		}
		if err := s.validateForEnable(ctx, ownerID, *detail); err != nil {
			return nil, err
		}
		if _, err := dao.Agent.UpdateByAgentIDAndOwner(agentID, ownerID, map[string]any{"status": AgentStatusEnabled}); err != nil {
			return nil, err
		}
		return s.Get(ownerID, agentID)
	}
	referencing, err := dao.AgentBinding.ListReferencingSubAgents(ownerID, item.AgentID, true)
	if err != nil {
		return nil, err
	}
	if len(referencing) > 0 {
		names := make([]string, 0, len(referencing))
		for _, ref := range referencing {
			names = append(names, ref.Name)
		}
		return nil, fmt.Errorf("agent is still referenced by enabled supervisors: %s", strings.Join(names, ", "))
	}
	if _, err := dao.Agent.UpdateByAgentIDAndOwner(agentID, ownerID, map[string]any{"status": AgentStatusDisabled}); err != nil {
		return nil, err
	}
	return s.Get(ownerID, agentID)
}

func (s *AgentDefinitionService) Delete(ownerID, agentID string) error {
	item, err := dao.Agent.GetByAgentIDAndOwner(strings.TrimSpace(agentID), strings.TrimSpace(ownerID))
	if err != nil {
		return err
	}
	referencing, err := dao.AgentBinding.ListReferencingSubAgents(ownerID, item.AgentID, true)
	if err != nil {
		return err
	}
	if len(referencing) > 0 {
		names := make([]string, 0, len(referencing))
		for _, ref := range referencing {
			names = append(names, ref.Name)
		}
		return fmt.Errorf("agent is still referenced by enabled supervisors: %s", strings.Join(names, ", "))
	}
	if err := dao.AgentBinding.DeleteByAgentID(ownerID, item.AgentID); err != nil {
		return err
	}
	_, err = dao.Agent.DeleteByAgentIDAndOwner(item.AgentID, ownerID)
	return err
}

func (s *AgentDefinitionService) ListEnabledOptions(ownerID string) ([]EnabledAgentOption, error) {
	items, err := dao.Agent.ListEnabledByOwner(strings.TrimSpace(ownerID))
	if err != nil {
		return nil, err
	}
	modelNames := loadModelNames(items)
	result := make([]EnabledAgentOption, 0, len(items))
	for _, item := range items {
		result = append(result, EnabledAgentOption{
			AgentID:            item.AgentID,
			Name:               item.Name,
			Description:        item.Description,
			AgentType:          item.AgentType,
			Status:             item.Status,
			AvatarURL:          normalizeAgentAvatarURL(item.AvatarURL),
			ChatBackgroundJSON: item.ChatBackgroundJSON,
			DefaultModelID:     item.DefaultModelID,
			DefaultModelName:   modelNames[item.DefaultModelID],
			ExampleQuestions:   parseExampleQuestions(item.ExampleQuestionsJSON),
		})
	}
	return result, nil
}

type normalizedBindings struct {
	toolIDs            []string
	skillNames         []string
	knowledgeBaseNames []string
	subAgentIDs        []string
}

func (s *AgentDefinitionService) validateAndNormalizeInput(ctx context.Context, ownerID, currentAgentID string, input UpsertAgentInput) (*normalizedBindings, error) {
	agentType := normalizeAgentType(input.AgentType)
	if agentType == "" {
		return nil, errors.New("agent_type must be single or supervisor")
	}
	if strings.TrimSpace(input.Name) == "" {
		return nil, errors.New("name is required")
	}
	if background := strings.TrimSpace(input.ChatBackgroundJSON); background != "" && !json.Valid([]byte(background)) {
		return nil, errors.New("chat_background_json must be valid json")
	}
	if defaultModelID := strings.TrimSpace(input.DefaultModelID); defaultModelID != "" {
		if _, err := modelsvc.ModelCatalog.GetByModelID(defaultModelID); err != nil {
			return nil, fmt.Errorf("default model not found")
		}
	}
	normalized := &normalizedBindings{
		toolIDs:            util.NormalizeUniqueStringList(input.ToolIDs),
		skillNames:         util.NormalizeUniqueStringList(input.SkillNames),
		knowledgeBaseNames: util.NormalizeUniqueStringList(input.KnowledgeBaseNames),
		subAgentIDs:        util.NormalizeUniqueStringList(input.SubAgentIDs),
	}
	if agentType == AgentTypeSingle && len(normalized.subAgentIDs) > 0 {
		return nil, errors.New("single agent cannot bind sub agents")
	}
	if err := validateEnabledTools(normalized.toolIDs); err != nil {
		return nil, err
	}
	if err := validateExistingSkills(ctx, normalized.skillNames); err != nil {
		return nil, err
	}
	if err := validateExistingKnowledgeBases(ctx, normalized.knowledgeBaseNames); err != nil {
		return nil, err
	}
	if err := s.validateSubAgents(ownerID, currentAgentID, agentType, normalized.subAgentIDs); err != nil {
		return nil, err
	}
	return normalized, nil
}

func (s *AgentDefinitionService) validateSubAgents(ownerID, currentAgentID, agentType string, subAgentIDs []string) error {
	if len(subAgentIDs) == 0 {
		return nil
	}
	if agentType != AgentTypeSupervisor {
		return errors.New("only supervisor can bind sub agents")
	}
	for _, subAgentID := range subAgentIDs {
		if subAgentID == currentAgentID {
			return errors.New("agent cannot reference itself as sub agent")
		}
	}
	items, err := dao.Agent.ListByAgentIDs(ownerID, subAgentIDs)
	if err != nil {
		return err
	}
	if len(items) != len(subAgentIDs) {
		return errors.New("some sub agents do not exist")
	}
	for _, item := range items {
		if item.Status != AgentStatusEnabled {
			return fmt.Errorf("sub agent %s is not enabled", item.Name)
		}
	}
	return validateAgentGraph(ownerID, currentAgentID, subAgentIDs)
}

func validateAgentGraph(ownerID, currentAgentID string, currentSubAgentIDs []string) error {
	allAgents, err := dao.Agent.ListAllByOwner(ownerID)
	if err != nil {
		return err
	}
	agentIDs := make([]string, 0, len(allAgents))
	for _, item := range allAgents {
		agentIDs = append(agentIDs, item.AgentID)
	}
	bindings, err := dao.AgentBinding.ListByAgentIDs(ownerID, agentIDs)
	if err != nil {
		return err
	}
	graph := make(map[string][]string, len(allAgents)+1)
	for _, binding := range bindings {
		if binding.BindingType != AgentBindingSubAgent {
			continue
		}
		graph[binding.AgentID] = append(graph[binding.AgentID], binding.BindingTargetID)
	}
	graph[currentAgentID] = append([]string{}, currentSubAgentIDs...)
	seen := make(map[string]bool)
	inPath := make(map[string]bool)
	var dfs func(agentID string, depth int) error
	dfs = func(agentID string, depth int) error {
		if depth > maxAgentCompileDepth {
			return fmt.Errorf("agent graph depth exceeds %d", maxAgentCompileDepth)
		}
		if inPath[agentID] {
			return errors.New("agent graph cycle detected")
		}
		if seen[agentID] {
			return nil
		}
		inPath[agentID] = true
		for _, next := range util.NormalizeUniqueStringList(graph[agentID]) {
			if err := dfs(next, depth+1); err != nil {
				return err
			}
		}
		inPath[agentID] = false
		seen[agentID] = true
		return nil
	}
	return dfs(currentAgentID, 1)
}

func (s *AgentDefinitionService) validateForEnable(ctx context.Context, ownerID string, detail AgentDetailView) error {
	if strings.TrimSpace(detail.SystemPrompt) == "" {
		return errors.New("system_prompt is required before enabling")
	}
	if strings.TrimSpace(detail.Description) == "" {
		return errors.New("description is required before enabling")
	}
	if detail.AgentType == AgentTypeSupervisor && len(detail.SubAgentIDs) == 0 {
		return errors.New("supervisor must bind at least one sub agent before enabling")
	}
	input := UpsertAgentInput{
		Name:               detail.Name,
		Description:        detail.Description,
		AgentType:          detail.AgentType,
		SystemPrompt:       detail.SystemPrompt,
		AvatarURL:          detail.AvatarURL,
		ChatBackgroundJSON: detail.ChatBackgroundJSON,
		ExampleQuestions:   detail.ExampleQuestions,
		DefaultModelID:     detail.DefaultModelID,
		AgentMemoryEnabled: detail.AgentMemoryEnabled,
		ToolIDs:            detail.ToolIDs,
		SkillNames:         detail.SkillNames,
		KnowledgeBaseNames: detail.KnowledgeBaseNames,
		SubAgentIDs:        detail.SubAgentIDs,
	}
	_, err := s.validateAndNormalizeInput(ctx, ownerID, detail.AgentID, input)
	return err
}

func buildBindings(userID, agentID string, normalized *normalizedBindings) []models.AgentBinding {
	result := make([]models.AgentBinding, 0, len(normalized.toolIDs)+len(normalized.skillNames)+len(normalized.knowledgeBaseNames)+len(normalized.subAgentIDs))
	appendBindings := func(bindingType string, ids []string) {
		for idx, id := range ids {
			result = append(result, models.AgentBinding{
				UserID:          userID,
				AgentID:         agentID,
				BindingType:     bindingType,
				BindingTargetID: id,
				Sort:            idx,
			})
		}
	}
	appendBindings(AgentBindingTool, normalized.toolIDs)
	appendBindings(AgentBindingSkill, normalized.skillNames)
	appendBindings(AgentBindingKB, normalized.knowledgeBaseNames)
	appendBindings(AgentBindingSubAgent, normalized.subAgentIDs)
	return result
}

func buildAgentDetailView(item models.Agent, bindings []models.AgentBinding, modelName string) (*AgentDetailView, error) {
	grouped := splitBindings(bindings)
	return &AgentDetailView{
		AgentID:            item.AgentID,
		OwnerID:            item.UserID,
		Name:               item.Name,
		Description:        item.Description,
		AgentType:          item.AgentType,
		Status:             item.Status,
		SystemPrompt:       item.SystemPrompt,
		AvatarURL:          normalizeAgentAvatarURL(item.AvatarURL),
		ChatBackgroundJSON: item.ChatBackgroundJSON,
		ExampleQuestions:   parseExampleQuestions(item.ExampleQuestionsJSON),
		DefaultModelID:     item.DefaultModelID,
		DefaultModelName:   modelName,
		AgentMemoryEnabled: item.AgentMemoryEnabled,
		ToolIDs:            grouped[AgentBindingTool],
		SkillNames:         grouped[AgentBindingSkill],
		KnowledgeBaseNames: grouped[AgentBindingKB],
		SubAgentIDs:        grouped[AgentBindingSubAgent],
		ToolCount:          len(grouped[AgentBindingTool]),
		SkillCount:         len(grouped[AgentBindingSkill]),
		KnowledgeBaseCount: len(grouped[AgentBindingKB]),
		SubAgentCount:      len(grouped[AgentBindingSubAgent]),
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	}, nil
}

func buildAgentListItem(item models.Agent, bindings []models.AgentBinding, modelName string) AgentListItem {
	grouped := splitBindings(bindings)
	return AgentListItem{
		AgentID:            item.AgentID,
		OwnerID:            item.UserID,
		Name:               item.Name,
		Description:        item.Description,
		AgentType:          item.AgentType,
		Status:             item.Status,
		AvatarURL:          normalizeAgentAvatarURL(item.AvatarURL),
		DefaultModelID:     item.DefaultModelID,
		DefaultModelName:   modelName,
		AgentMemoryEnabled: item.AgentMemoryEnabled,
		ToolCount:          len(grouped[AgentBindingTool]),
		SkillCount:         len(grouped[AgentBindingSkill]),
		KnowledgeBaseCount: len(grouped[AgentBindingKB]),
		SubAgentCount:      len(grouped[AgentBindingSubAgent]),
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	}
}

func groupBindingsByAgentID(bindings []models.AgentBinding) map[string][]models.AgentBinding {
	result := make(map[string][]models.AgentBinding, len(bindings))
	for _, binding := range bindings {
		result[binding.AgentID] = append(result[binding.AgentID], binding)
	}
	return result
}

func normalizeAgentAvatarURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed != "" {
		return trimmed
	}
	return defaultAgentAvatarURL
}

func splitBindings(bindings []models.AgentBinding) map[string][]string {
	result := map[string][]string{
		AgentBindingTool:     {},
		AgentBindingSkill:    {},
		AgentBindingKB:       {},
		AgentBindingSubAgent: {},
	}
	for _, binding := range bindings {
		result[binding.BindingType] = append(result[binding.BindingType], binding.BindingTargetID)
	}
	for key, ids := range result {
		result[key] = util.NormalizeUniqueStringList(ids)
	}
	return result
}

func validateEnabledTools(toolIDs []string) error {
	if len(toolIDs) == 0 {
		return nil
	}
	items, err := dao.Plugin.ListEnabledToolsByIDs(toolIDs)
	if err != nil {
		return err
	}
	if len(items) != len(toolIDs) {
		return errors.New("some tools are missing or disabled")
	}
	return nil
}

func validateExistingSkills(ctx context.Context, skillNames []string) error {
	if len(skillNames) == 0 {
		return nil
	}
	if !dao.SkillStore.Configured() {
		return errors.New("skill storage not configured")
	}
	names, err := dao.SkillStore.ListSkillNames(ctx)
	if err != nil {
		return err
	}
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		set[name] = struct{}{}
	}
	for _, skillName := range skillNames {
		if _, ok := set[skillName]; !ok {
			return fmt.Errorf("skill not found: %s", skillName)
		}
	}
	return nil
}

func validateExistingKnowledgeBases(ctx context.Context, kbNames []string) error {
	if len(kbNames) == 0 {
		return nil
	}
	items, err := kbsvc.KnowledgeBase.List(ctx)
	if err != nil {
		return err
	}
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		set[item.Name] = struct{}{}
	}
	for _, kbName := range kbNames {
		if _, ok := set[kbName]; !ok {
			return fmt.Errorf("knowledge base not found: %s", kbName)
		}
	}
	return nil
}

func normalizeAgentType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case AgentTypeSingle:
		return AgentTypeSingle
	case AgentTypeSupervisor:
		return AgentTypeSupervisor
	default:
		return ""
	}
}

func normalizeOptionalAgentType(value string) string {
	return normalizeAgentType(value)
}

func normalizeOptionalAgentStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case AgentStatusDraft:
		return AgentStatusDraft
	case AgentStatusEnabled:
		return AgentStatusEnabled
	case AgentStatusDisabled:
		return AgentStatusDisabled
	default:
		return ""
	}
}

func normalizeQuestionList(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return util.NormalizeUniqueStringList(result)
}

func mustMarshalStringList(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	payload, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(payload)
}

func parseExampleQuestions(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []string{}
	}
	var items []string
	if err := json.Unmarshal([]byte(trimmed), &items); err != nil {
		return []string{}
	}
	return normalizeQuestionList(items)
}

func loadModelName(modelID string) string {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return ""
	}
	item, err := modelsvc.ModelCatalog.GetByModelID(modelID)
	if err != nil || item == nil {
		return ""
	}
	return item.Name
}

func loadModelNames(items []models.Agent) map[string]string {
	result := make(map[string]string, len(items))
	for _, item := range items {
		modelID := strings.TrimSpace(item.DefaultModelID)
		if modelID == "" {
			continue
		}
		if _, ok := result[modelID]; ok {
			continue
		}
		result[modelID] = loadModelName(modelID)
	}
	return result
}

var _ = gorm.ErrRecordNotFound

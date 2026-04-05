package model

import (
	"errors"
	"openIntern/internal/dao"
	"openIntern/internal/models"
	"sort"
	"strings"
)

const SystemDefaultChatModelConfigKey = "system_default_chat_model"

type CreateModelCatalogInput struct {
	ProviderID       string `json:"provider_id"`
	ModelKey         string `json:"model_key"`
	Name             string `json:"name"`
	Avatar           string `json:"avatar"`
	CapabilitiesJSON string `json:"capabilities_json"`
	Enabled          *bool  `json:"enabled"`
	Sort             *int   `json:"sort"`
}

type UpdateModelCatalogInput struct {
	ProviderID       *string `json:"provider_id"`
	ModelKey         *string `json:"model_key"`
	Name             *string `json:"name"`
	Avatar           *string `json:"avatar"`
	CapabilitiesJSON *string `json:"capabilities_json"`
	Enabled          *bool   `json:"enabled"`
	Sort             *int    `json:"sort"`
}

type ModelCatalogView struct {
	ModelID          string `json:"model_id"`
	ProviderID       string `json:"provider_id"`
	ProviderName     string `json:"provider_name"`
	ProviderAvatar   string `json:"provider_avatar"`
	APIType          string `json:"api_type"`
	ModelKey         string `json:"model_key"`
	Name             string `json:"name"`
	Avatar           string `json:"avatar"`
	CapabilitiesJSON string `json:"capabilities_json"`
	Enabled          bool   `json:"enabled"`
	Sort             int    `json:"sort"`
	IsSystemDefault  bool   `json:"is_system_default"`
	CreatedAt        any    `json:"created_at"`
	UpdatedAt        any    `json:"updated_at"`
}

type ModelCatalogOption struct {
	ProviderID      string `json:"provider_id"`
	ProviderName    string `json:"provider_name"`
	ProviderAvatar  string `json:"provider_avatar"`
	APIType         string `json:"api_type"`
	ModelID         string `json:"model_id"`
	ModelKey        string `json:"model_key"`
	ModelName       string `json:"model_name"`
	ModelAvatar     string `json:"model_avatar"`
	IsSystemDefault bool   `json:"is_system_default"`
}

type RuntimeModelSelection struct {
	Provider *models.ModelProvider
	Model    *models.ModelCatalog
}

type ModelCatalogService struct{}

type DefaultModelConfigService struct{}

var ModelCatalog = new(ModelCatalogService)
var DefaultModel = new(DefaultModelConfigService)

func (s *ModelCatalogService) Create(userID string, input CreateModelCatalogInput) (*models.ModelCatalog, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("user_id is required")
	}
	providerID := strings.TrimSpace(input.ProviderID)
	modelKey := strings.TrimSpace(input.ModelKey)
	name := strings.TrimSpace(input.Name)
	if providerID == "" || modelKey == "" || name == "" {
		return nil, errors.New("provider_id, model_key and name are required")
	}
	if _, err := ModelProvider.GetByProviderID(userID, providerID); err != nil {
		return nil, err
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	sortValue := 0
	if input.Sort != nil {
		sortValue = *input.Sort
	}
	item := &models.ModelCatalog{
		UserID:           userID,
		ProviderID:       providerID,
		ModelKey:         modelKey,
		Name:             name,
		Avatar:           strings.TrimSpace(input.Avatar),
		CapabilitiesJSON: strings.TrimSpace(input.CapabilitiesJSON),
		Enabled:          enabled,
		Sort:             sortValue,
	}
	if err := dao.ModelCatalog.Create(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ModelCatalogService) GetByModelID(userID string, modelID string) (*models.ModelCatalog, error) {
	return dao.ModelCatalog.GetByUserIDAndModelID(userID, modelID)
}

func (s *ModelCatalogService) Update(userID string, modelID string, input UpdateModelCatalogInput) error {
	updates := make(map[string]any)
	if input.ProviderID != nil {
		providerID := strings.TrimSpace(*input.ProviderID)
		if providerID == "" {
			return errors.New("provider_id cannot be empty")
		}
		if _, err := ModelProvider.GetByProviderID(userID, providerID); err != nil {
			return err
		}
		updates["provider_id"] = providerID
	}
	if input.ModelKey != nil {
		modelKey := strings.TrimSpace(*input.ModelKey)
		if modelKey == "" {
			return errors.New("model_key cannot be empty")
		}
		updates["model_key"] = modelKey
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return errors.New("name cannot be empty")
		}
		updates["name"] = name
	}
	if input.Avatar != nil {
		updates["avatar"] = strings.TrimSpace(*input.Avatar)
	}
	if input.CapabilitiesJSON != nil {
		updates["capabilities_json"] = strings.TrimSpace(*input.CapabilitiesJSON)
	}
	if input.Enabled != nil {
		updates["enabled"] = *input.Enabled
	}
	if input.Sort != nil {
		updates["sort"] = *input.Sort
	}
	if len(updates) == 0 {
		return nil
	}
	rowsAffected, err := dao.ModelCatalog.UpdateByUserIDAndModelID(userID, modelID, updates)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("model not found")
	}
	return nil
}

func (s *ModelCatalogService) Delete(userID string, modelID string) error {
	cfg, err := DefaultModel.Get(userID)
	if err == nil && cfg != nil && cfg.ModelID == modelID {
		return errors.New("cannot delete system default model")
	}
	rowsAffected, err := dao.ModelCatalog.DeleteByUserIDAndModelID(userID, modelID)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("model not found")
	}
	return nil
}

func (s *ModelCatalogService) List(userID string, page, pageSize int, keyword, providerID string) ([]ModelCatalogView, int64, error) {
	items, total, err := dao.ModelCatalog.List(page, pageSize, dao.ModelCatalogListFilter{
		UserID:     strings.TrimSpace(userID),
		Keyword:    keyword,
		ProviderID: providerID,
	})
	if err != nil {
		return nil, 0, err
	}
	defaultID, _ := DefaultModel.GetModelID(userID)
	providers, err := loadProvidersByID(userID, items)
	if err != nil {
		return nil, 0, err
	}
	views := make([]ModelCatalogView, 0, len(items))
	for _, item := range items {
		views = append(views, buildModelCatalogView(item, providers[item.ProviderID], item.ModelID == defaultID))
	}
	return views, total, nil
}

func (s *ModelCatalogService) GetView(userID string, modelID string) (*ModelCatalogView, error) {
	item, err := s.GetByModelID(userID, modelID)
	if err != nil {
		return nil, err
	}
	provider, err := ModelProvider.GetByProviderID(userID, item.ProviderID)
	if err != nil {
		return nil, err
	}
	defaultID, _ := DefaultModel.GetModelID(userID)
	view := buildModelCatalogView(*item, provider, item.ModelID == defaultID)
	return &view, nil
}

func (s *ModelCatalogService) ListCatalogOptions(userID string) ([]ModelCatalogOption, error) {
	items, err := dao.ModelCatalog.ListEnabled(userID)
	if err != nil {
		return nil, err
	}
	defaultID, _ := DefaultModel.GetModelID(userID)
	providers, err := loadProvidersByID(userID, items)
	if err != nil {
		return nil, err
	}
	options := make([]ModelCatalogOption, 0, len(items))
	for _, item := range items {
		provider := providers[item.ProviderID]
		if provider == nil || !provider.Enabled {
			continue
		}
		options = append(options, ModelCatalogOption{
			ProviderID:      provider.ProviderID,
			ProviderName:    provider.Name,
			ProviderAvatar:  provider.Avatar,
			APIType:         provider.APIType,
			ModelID:         item.ModelID,
			ModelKey:        item.ModelKey,
			ModelName:       item.Name,
			ModelAvatar:     item.Avatar,
			IsSystemDefault: item.ModelID == defaultID,
		})
	}
	sort.SliceStable(options, func(i, j int) bool {
		if options[i].IsSystemDefault != options[j].IsSystemDefault {
			return options[i].IsSystemDefault
		}
		if options[i].ProviderName != options[j].ProviderName {
			return options[i].ProviderName < options[j].ProviderName
		}
		return options[i].ModelName < options[j].ModelName
	})
	return options, nil
}

func (s *ModelCatalogService) ResolveRuntimeSelection(userID, modelID, providerID string) (*RuntimeModelSelection, error) {
	selectedModelID := strings.TrimSpace(modelID)
	if selectedModelID == "" {
		var err error
		selectedModelID, err = DefaultModel.GetModelID(userID)
		if err != nil {
			return nil, err
		}
	}
	if selectedModelID == "" {
		return nil, nil
	}
	modelItem, err := s.GetByModelID(userID, selectedModelID)
	if err != nil {
		return nil, err
	}
	if !modelItem.Enabled {
		return nil, errors.New("model is disabled")
	}
	if providerID = strings.TrimSpace(providerID); providerID != "" && providerID != modelItem.ProviderID {
		return nil, errors.New("model does not belong to provider")
	}
	providerItem, err := ModelProvider.GetByProviderID(userID, modelItem.ProviderID)
	if err != nil {
		return nil, err
	}
	if !providerItem.Enabled {
		return nil, errors.New("provider is disabled")
	}
	return &RuntimeModelSelection{
		Provider: providerItem,
		Model:    modelItem,
	}, nil
}

func (s *DefaultModelConfigService) Get(userID string) (*models.DefaultModelConfig, error) {
	return dao.DefaultModelConfig.GetByUserIDAndConfigKey(userID, SystemDefaultChatModelConfigKey)
}

func (s *DefaultModelConfigService) GetModelID(userID string) (string, error) {
	item, err := s.Get(userID)
	if err != nil {
		return "", err
	}
	if item == nil {
		return "", nil
	}
	return strings.TrimSpace(item.ModelID), nil
}

func (s *DefaultModelConfigService) Set(userID, modelID string) (*models.DefaultModelConfig, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, errors.New("model_id is required")
	}
	if _, err := ModelCatalog.GetByModelID(userID, modelID); err != nil {
		return nil, err
	}
	return dao.DefaultModelConfig.UpsertByUserIDAndConfigKey(userID, SystemDefaultChatModelConfigKey, modelID)
}

func buildModelCatalogView(item models.ModelCatalog, provider *models.ModelProvider, isDefault bool) ModelCatalogView {
	view := ModelCatalogView{
		ModelID:          item.ModelID,
		ProviderID:       item.ProviderID,
		ModelKey:         item.ModelKey,
		Name:             item.Name,
		Avatar:           item.Avatar,
		CapabilitiesJSON: item.CapabilitiesJSON,
		Enabled:          item.Enabled,
		Sort:             item.Sort,
		IsSystemDefault:  isDefault,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
	if provider != nil {
		view.ProviderName = provider.Name
		view.ProviderAvatar = provider.Avatar
		view.APIType = provider.APIType
	}
	return view
}

func loadProvidersByID(userID string, items []models.ModelCatalog) (map[string]*models.ModelProvider, error) {
	providerIDs := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.ProviderID == "" {
			continue
		}
		if _, ok := seen[item.ProviderID]; ok {
			continue
		}
		seen[item.ProviderID] = struct{}{}
		providerIDs = append(providerIDs, item.ProviderID)
	}
	return dao.ModelProvider.LoadByUserIDAndProviderIDs(userID, providerIDs)
}

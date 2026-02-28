package services

import (
	"errors"
	"openIntern/internal/database"
	"openIntern/internal/models"
	"sort"
	"strings"

	"gorm.io/gorm"
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

func (s *ModelCatalogService) Create(input CreateModelCatalogInput) (*models.ModelCatalog, error) {
	providerID := strings.TrimSpace(input.ProviderID)
	modelKey := strings.TrimSpace(input.ModelKey)
	name := strings.TrimSpace(input.Name)
	if providerID == "" || modelKey == "" || name == "" {
		return nil, errors.New("provider_id, model_key and name are required")
	}
	if _, err := ModelProvider.GetByProviderID(providerID); err != nil {
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
		ProviderID:       providerID,
		ModelKey:         modelKey,
		Name:             name,
		Avatar:           strings.TrimSpace(input.Avatar),
		CapabilitiesJSON: strings.TrimSpace(input.CapabilitiesJSON),
		Enabled:          enabled,
		Sort:             sortValue,
	}
	if err := database.DB.Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ModelCatalogService) GetByModelID(modelID string) (*models.ModelCatalog, error) {
	var item models.ModelCatalog
	err := database.DB.Where("model_id = ?", modelID).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *ModelCatalogService) Update(modelID string, input UpdateModelCatalogInput) error {
	updates := make(map[string]any)
	if input.ProviderID != nil {
		providerID := strings.TrimSpace(*input.ProviderID)
		if providerID == "" {
			return errors.New("provider_id cannot be empty")
		}
		if _, err := ModelProvider.GetByProviderID(providerID); err != nil {
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
	result := database.DB.Model(&models.ModelCatalog{}).Where("model_id = ?", modelID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("model not found")
	}
	return nil
}

func (s *ModelCatalogService) Delete(modelID string) error {
	cfg, err := DefaultModel.Get()
	if err == nil && cfg != nil && cfg.ModelID == modelID {
		return errors.New("cannot delete system default model")
	}
	result := database.DB.Where("model_id = ?", modelID).Delete(&models.ModelCatalog{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("model not found")
	}
	return nil
}

func (s *ModelCatalogService) List(page, pageSize int, keyword, providerID string) ([]ModelCatalogView, int64, error) {
	var items []models.ModelCatalog
	var total int64
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	db := database.DB.Model(&models.ModelCatalog{})
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		pattern := "%" + keyword + "%"
		db = db.Where("(name LIKE ? OR model_key LIKE ?)", pattern, pattern)
	}
	if providerID = strings.TrimSpace(providerID); providerID != "" {
		db = db.Where("provider_id = ?", providerID)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("sort ASC").Order("updated_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	defaultID, _ := DefaultModel.GetModelID()
	providers, err := loadProvidersByID(items)
	if err != nil {
		return nil, 0, err
	}
	views := make([]ModelCatalogView, 0, len(items))
	for _, item := range items {
		views = append(views, buildModelCatalogView(item, providers[item.ProviderID], item.ModelID == defaultID))
	}
	return views, total, nil
}

func (s *ModelCatalogService) GetView(modelID string) (*ModelCatalogView, error) {
	item, err := s.GetByModelID(modelID)
	if err != nil {
		return nil, err
	}
	provider, err := ModelProvider.GetByProviderID(item.ProviderID)
	if err != nil {
		return nil, err
	}
	defaultID, _ := DefaultModel.GetModelID()
	view := buildModelCatalogView(*item, provider, item.ModelID == defaultID)
	return &view, nil
}

func (s *ModelCatalogService) ListCatalogOptions() ([]ModelCatalogOption, error) {
	var items []models.ModelCatalog
	if err := database.DB.Where("enabled = ?", true).Order("sort ASC").Order("updated_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	defaultID, _ := DefaultModel.GetModelID()
	providers, err := loadProvidersByID(items)
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

func (s *ModelCatalogService) ResolveRuntimeSelection(modelID, providerID string) (*RuntimeModelSelection, error) {
	selectedModelID := strings.TrimSpace(modelID)
	if selectedModelID == "" {
		var err error
		selectedModelID, err = DefaultModel.GetModelID()
		if err != nil {
			return nil, err
		}
	}
	if selectedModelID == "" {
		return nil, nil
	}
	modelItem, err := s.GetByModelID(selectedModelID)
	if err != nil {
		return nil, err
	}
	if !modelItem.Enabled {
		return nil, errors.New("model is disabled")
	}
	if providerID = strings.TrimSpace(providerID); providerID != "" && providerID != modelItem.ProviderID {
		return nil, errors.New("model does not belong to provider")
	}
	providerItem, err := ModelProvider.GetByProviderID(modelItem.ProviderID)
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

func (s *DefaultModelConfigService) Get() (*models.DefaultModelConfig, error) {
	var item models.DefaultModelConfig
	err := database.DB.Where("config_key = ?", SystemDefaultChatModelConfigKey).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *DefaultModelConfigService) GetModelID() (string, error) {
	item, err := s.Get()
	if err != nil {
		return "", err
	}
	if item == nil {
		return "", nil
	}
	return strings.TrimSpace(item.ModelID), nil
}

func (s *DefaultModelConfigService) Set(modelID string) (*models.DefaultModelConfig, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, errors.New("model_id is required")
	}
	if _, err := ModelCatalog.GetByModelID(modelID); err != nil {
		return nil, err
	}
	item, err := s.Get()
	if err != nil {
		return nil, err
	}
	if item == nil {
		created := models.DefaultModelConfig{
			ConfigKey: SystemDefaultChatModelConfigKey,
			ModelID:   modelID,
		}
		if createErr := database.DB.Create(&created).Error; createErr != nil {
			return nil, createErr
		}
		return &created, nil
	}
	item.ModelID = modelID
	if saveErr := database.DB.Save(item).Error; saveErr != nil {
		return nil, saveErr
	}
	return item, nil
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

func loadProvidersByID(items []models.ModelCatalog) (map[string]*models.ModelProvider, error) {
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
	result := make(map[string]*models.ModelProvider, len(providerIDs))
	if len(providerIDs) == 0 {
		return result, nil
	}
	var providers []models.ModelProvider
	if err := database.DB.Where("provider_id IN ?", providerIDs).Find(&providers).Error; err != nil {
		return nil, err
	}
	for i := range providers {
		item := providers[i]
		result[item.ProviderID] = &item
	}
	return result, nil
}

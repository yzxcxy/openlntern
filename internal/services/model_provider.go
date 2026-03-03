package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"openIntern/internal/dao"
	"openIntern/internal/models"
	"strings"
)

const fallbackModelCryptoSecret = "openintern-model-provider-secret"

type CreateModelProviderInput struct {
	Name            string `json:"name"`
	APIType         string `json:"api_type"`
	BaseURL         string `json:"base_url"`
	APIKey          string `json:"api_key"`
	Avatar          string `json:"avatar"`
	ExtraConfigJSON string `json:"extra_config_json"`
	Enabled         *bool  `json:"enabled"`
}

type UpdateModelProviderInput struct {
	Name            *string `json:"name"`
	APIType         *string `json:"api_type"`
	BaseURL         *string `json:"base_url"`
	APIKey          *string `json:"api_key"`
	Avatar          *string `json:"avatar"`
	ExtraConfigJSON *string `json:"extra_config_json"`
	Enabled         *bool   `json:"enabled"`
}

type ModelProviderView struct {
	ProviderID      string `json:"provider_id"`
	Name            string `json:"name"`
	APIType         string `json:"api_type"`
	BaseURL         string `json:"base_url"`
	APIKeyMasked    string `json:"api_key_masked"`
	Avatar          string `json:"avatar"`
	ExtraConfigJSON string `json:"extra_config_json"`
	Enabled         bool   `json:"enabled"`
	CreatedAt       any    `json:"created_at"`
	UpdatedAt       any    `json:"updated_at"`
}

type ModelProviderService struct{}

var ModelProvider = new(ModelProviderService)

func (s *ModelProviderService) Create(input CreateModelProviderInput) (*models.ModelProvider, error) {
	name := strings.TrimSpace(input.Name)
	apiType := strings.ToLower(strings.TrimSpace(input.APIType))
	if name == "" || apiType == "" {
		return nil, errors.New("name and api_type are required")
	}
	if apiType != "openai" && apiType != "ark" {
		return nil, errors.New("api_type must be one of: openai, ark")
	}
	var err error
	ciphertext, err := encryptModelSecret(strings.TrimSpace(input.APIKey))
	if err != nil {
		return nil, err
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	item := &models.ModelProvider{
		Name:             name,
		APIType:          apiType,
		BaseURL:          strings.TrimSpace(input.BaseURL),
		APIKeyCiphertext: ciphertext,
		Avatar:           strings.TrimSpace(input.Avatar),
		ExtraConfigJSON:  strings.TrimSpace(input.ExtraConfigJSON),
		Enabled:          enabled,
	}
	if err := dao.ModelProvider.Create(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ModelProviderService) GetByProviderID(providerID string) (*models.ModelProvider, error) {
	return dao.ModelProvider.GetByProviderID(providerID)
}

func (s *ModelProviderService) Update(providerID string, input UpdateModelProviderInput) error {
	updates := make(map[string]any)
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return errors.New("name cannot be empty")
		}
		updates["name"] = name
	}
	if input.APIType != nil {
		apiType := strings.ToLower(strings.TrimSpace(*input.APIType))
		if apiType == "" {
			return errors.New("api_type cannot be empty")
		}
		if apiType != "openai" && apiType != "ark" {
			return errors.New("api_type must be one of: openai, ark")
		}
		updates["api_type"] = apiType
	}
	if input.BaseURL != nil {
		updates["base_url"] = strings.TrimSpace(*input.BaseURL)
	}
	if input.APIKey != nil {
		apiKey := strings.TrimSpace(*input.APIKey)
		if apiKey != "" {
			ciphertext, err := encryptModelSecret(apiKey)
			if err != nil {
				return err
			}
			updates["api_key_ciphertext"] = ciphertext
		}
	}
	if input.Avatar != nil {
		updates["avatar"] = strings.TrimSpace(*input.Avatar)
	}
	if input.ExtraConfigJSON != nil {
		updates["extra_config_json"] = strings.TrimSpace(*input.ExtraConfigJSON)
	}
	if input.Enabled != nil {
		updates["enabled"] = *input.Enabled
	}
	if len(updates) == 0 {
		return nil
	}
	rowsAffected, err := dao.ModelProvider.UpdateByProviderID(providerID, updates)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("model provider not found")
	}
	return nil
}

func (s *ModelProviderService) Delete(providerID string) error {
	count, err := dao.ModelProvider.CountModelsByProviderID(providerID)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("provider still has models")
	}
	rowsAffected, err := dao.ModelProvider.DeleteByProviderID(providerID)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("model provider not found")
	}
	return nil
}

func (s *ModelProviderService) List(page, pageSize int, keyword string) ([]ModelProviderView, int64, error) {
	items, total, err := dao.ModelProvider.List(page, pageSize, dao.ModelProviderListFilter{
		Keyword: keyword,
	})
	if err != nil {
		return nil, 0, err
	}
	views := make([]ModelProviderView, 0, len(items))
	for _, item := range items {
		views = append(views, buildModelProviderView(item))
	}
	return views, total, nil
}

func (s *ModelProviderService) ToView(item *models.ModelProvider) ModelProviderView {
	if item == nil {
		return ModelProviderView{}
	}
	return buildModelProviderView(*item)
}

func (s *ModelProviderService) ResolveAPIKey(item *models.ModelProvider) (string, error) {
	if item == nil {
		return "", errors.New("model provider is required")
	}
	return decryptModelSecret(item.APIKeyCiphertext)
}

func buildModelProviderView(item models.ModelProvider) ModelProviderView {
	apiKey, _ := decryptModelSecret(item.APIKeyCiphertext)
	return ModelProviderView{
		ProviderID:      item.ProviderID,
		Name:            item.Name,
		APIType:         item.APIType,
		BaseURL:         item.BaseURL,
		APIKeyMasked:    maskModelSecret(apiKey),
		Avatar:          item.Avatar,
		ExtraConfigJSON: item.ExtraConfigJSON,
		Enabled:         item.Enabled,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}
}

func encryptModelSecret(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	key := sha256.Sum256([]byte(modelCryptoSecret()))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(value), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func decryptModelSecret(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return value, nil
	}
	key := sha256.Sum256([]byte(modelCryptoSecret()))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return value, nil
	}
	nonce := raw[:gcm.NonceSize()]
	ciphertext := raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return value, nil
	}
	return string(plain), nil
}

func modelCryptoSecret() string {
	if strings.TrimSpace(authSecret) != "" {
		return authSecret
	}
	return fallbackModelCryptoSecret
}

func maskModelSecret(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= 8 {
		return strings.Repeat("*", len(runes))
	}
	return string(runes[:4]) + strings.Repeat("*", len(runes)-8) + string(runes[len(runes)-4:])
}

package configsvc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	runtimeconfig "openIntern/internal/config"
	"openIntern/internal/dao"
)

const (
	UserRuntimeConfigKeyAgent              = "agent"
	UserRuntimeConfigKeyContextCompression = "context_compression"
)

var allowedUserRuntimeConfigKeys = map[string]struct{}{
	UserRuntimeConfigKeyAgent:              {},
	UserRuntimeConfigKeyContextCompression: {},
}

// ResolvedUserRuntimeConfig is the effective runtime config for one user.
type ResolvedUserRuntimeConfig struct {
	Agent              runtimeconfig.AgentConfig
	ContextCompression runtimeconfig.ContextCompressionConfig
}

// AgentConfigPatch is the persisted user override payload for the agent block.
type AgentConfigPatch struct {
	MaxIterations int `json:"max_iterations"`
}

// ContextCompressionPatch is the persisted user override payload for the context_compression block.
type ContextCompressionPatch struct {
	Enabled                *bool `json:"enabled"`
	SoftLimitTokens        int   `json:"soft_limit_tokens"`
	HardLimitTokens        int   `json:"hard_limit_tokens"`
	OutputReserveTokens    int   `json:"output_reserve_tokens"`
	MaxRecentMessages      int   `json:"max_recent_messages"`
	EstimatedCharsPerToken int   `json:"estimated_chars_per_token"`
}

// UserRuntimeConfigService validates, persists, and resolves user runtime config blocks.
type UserRuntimeConfigService struct{}

// UserRuntimeConfig is the shared user runtime config service singleton.
var UserRuntimeConfig = new(UserRuntimeConfigService)

// IsAllowedUserRuntimeConfigKey reports whether a config block can be persisted per user.
func IsAllowedUserRuntimeConfigKey(configKey string) bool {
	_, ok := allowedUserRuntimeConfigKeys[strings.TrimSpace(configKey)]
	return ok
}

// ValidateAgentPatch validates a raw agent config payload with strict field checking.
func ValidateAgentPatch(input map[string]any) (*AgentConfigPatch, error) {
	if len(input) == 0 {
		return nil, errors.New("agent config is required")
	}

	normalized, err := decodeStrictMap[AgentConfigPatch](input)
	if err != nil {
		return nil, fmt.Errorf("invalid agent config: %w", err)
	}
	if err := validateAgentPatchStruct(normalized); err != nil {
		return nil, err
	}
	return normalized, nil
}

// ValidateContextCompressionPatch validates a raw context compression payload with strict field checking.
func ValidateContextCompressionPatch(input map[string]any) (*ContextCompressionPatch, error) {
	if len(input) == 0 {
		return nil, errors.New("context_compression config is required")
	}

	normalized, err := decodeStrictMap[ContextCompressionPatch](input)
	if err != nil {
		return nil, fmt.Errorf("invalid context_compression config: %w", err)
	}
	if err := validateContextCompressionPatchStruct(normalized, true); err != nil {
		return nil, err
	}
	return normalized, nil
}

// Resolve merges global runtime defaults with persisted per-user overrides.
func (s *UserRuntimeConfigService) Resolve(userID string) (*ResolvedUserRuntimeConfig, error) {
	defaults := runtimeconfig.GetRuntimeConfigSnapshot()
	effectiveDefaultEnabled := runtimeconfig.EffectiveContextCompressionEnabled(defaults.ContextCompression)
	resolved := &ResolvedUserRuntimeConfig{
		Agent: defaults.Agent,
		ContextCompression: runtimeconfig.ContextCompressionConfig{
			Enabled:                boolPtr(effectiveDefaultEnabled),
			SoftLimitTokens:        defaults.ContextCompression.SoftLimitTokens,
			HardLimitTokens:        defaults.ContextCompression.HardLimitTokens,
			OutputReserveTokens:    defaults.ContextCompression.OutputReserveTokens,
			MaxRecentMessages:      defaults.ContextCompression.MaxRecentMessages,
			EstimatedCharsPerToken: defaults.ContextCompression.EstimatedCharsPerToken,
		},
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		return resolved, nil
	}

	items, err := dao.UserRuntimeConfig.ListByUserID(userID, []string{
		UserRuntimeConfigKeyAgent,
		UserRuntimeConfigKeyContextCompression,
	})
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		switch item.ConfigKey {
		case UserRuntimeConfigKeyAgent:
			patch, decodeErr := decodeStrictBytes[AgentConfigPatch](item.ConfigValue)
			if decodeErr != nil {
				return nil, fmt.Errorf("decode user runtime config %q: %w", item.ConfigKey, decodeErr)
			}
			if err := validateAgentPatchStruct(patch); err != nil {
				return nil, err
			}
			resolved.Agent.MaxIterations = patch.MaxIterations
		case UserRuntimeConfigKeyContextCompression:
			patch, decodeErr := decodeStrictBytes[ContextCompressionPatch](item.ConfigValue)
			if decodeErr != nil {
				return nil, fmt.Errorf("decode user runtime config %q: %w", item.ConfigKey, decodeErr)
			}
			if err := validateContextCompressionPatchStruct(patch, false); err != nil {
				return nil, err
			}
			effectiveEnabled := effectiveDefaultEnabled
			if patch.Enabled != nil {
				effectiveEnabled = *patch.Enabled
			}
			resolved.ContextCompression = runtimeconfig.ContextCompressionConfig{
				Enabled:                boolPtr(effectiveEnabled),
				SoftLimitTokens:        patch.SoftLimitTokens,
				HardLimitTokens:        patch.HardLimitTokens,
				OutputReserveTokens:    patch.OutputReserveTokens,
				MaxRecentMessages:      patch.MaxRecentMessages,
				EstimatedCharsPerToken: patch.EstimatedCharsPerToken,
			}
		default:
			return nil, fmt.Errorf("unsupported user runtime config key %q", item.ConfigKey)
		}
	}

	return resolved, nil
}

// SaveAgent validates and persists the user-scoped agent config block only.
func (s *UserRuntimeConfigService) SaveAgent(userID string, input map[string]any) error {
	patch, err := ValidateAgentPatch(input)
	if err != nil {
		return err
	}
	return s.savePatch(userID, UserRuntimeConfigKeyAgent, patch)
}

// SaveContextCompression validates and persists the user-scoped context_compression block only.
func (s *UserRuntimeConfigService) SaveContextCompression(userID string, input map[string]any) error {
	patch, err := ValidateContextCompressionPatch(input)
	if err != nil {
		return err
	}
	return s.savePatch(userID, UserRuntimeConfigKeyContextCompression, patch)
}

func (s *UserRuntimeConfigService) savePatch(userID, configKey string, payload any) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("user_id is required")
	}
	if !IsAllowedUserRuntimeConfigKey(configKey) {
		return fmt.Errorf("config block %q is not allowed for user overrides", configKey)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = dao.UserRuntimeConfig.Upsert(userID, configKey, data)
	return err
}

func validateAgentPatchStruct(input *AgentConfigPatch) error {
	if input == nil {
		return errors.New("agent config is required")
	}
	if input.MaxIterations < 1 {
		return errors.New("agent.max_iterations must be greater than 0")
	}
	return nil
}

func validateContextCompressionPatchStruct(input *ContextCompressionPatch, requireEnabled bool) error {
	if input == nil {
		return errors.New("context_compression config is required")
	}
	if requireEnabled && input.Enabled == nil {
		return errors.New("context_compression.enabled must be explicitly set")
	}
	if input.HardLimitTokens <= 0 {
		return errors.New("context_compression.hard_limit_tokens must be greater than 0")
	}
	if input.SoftLimitTokens <= 0 || input.SoftLimitTokens >= input.HardLimitTokens {
		return errors.New("context_compression.soft_limit_tokens must be greater than 0 and less than hard_limit_tokens")
	}
	if input.OutputReserveTokens <= 0 {
		return errors.New("context_compression.output_reserve_tokens must be greater than 0")
	}
	if input.MaxRecentMessages <= 0 {
		return errors.New("context_compression.max_recent_messages must be greater than 0")
	}
	if input.EstimatedCharsPerToken <= 0 {
		return errors.New("context_compression.estimated_chars_per_token must be greater than 0")
	}
	return nil
}

func decodeStrictMap[T any](input map[string]any) (*T, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	return decodeStrictBytes[T](data)
}

func decodeStrictBytes[T any](data []byte) (*T, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var value T
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(struct{})); err != nil && !errors.Is(err, io.EOF) {
		return nil, errors.New("unexpected trailing JSON content")
	}
	return &value, nil
}

func boolPtr(v bool) *bool {
	return &v
}

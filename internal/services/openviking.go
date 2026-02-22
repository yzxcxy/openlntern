package services

import (
	"net/http"
	"strings"
	"time"

	"openIntern/internal/config"
)

type OpenVikingService struct {
	baseURL    string
	apiKey     string
	skillsRoot string
	client     *http.Client
}

var OpenViking = new(OpenVikingService)

func InitOpenViking(cfg config.OpenVikingConfig) {
	OpenViking.baseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	OpenViking.apiKey = strings.TrimSpace(cfg.APIKey)
	OpenViking.skillsRoot = strings.TrimRight(strings.TrimSpace(cfg.SkillsRoot), "/")
	if OpenViking.client == nil {
		OpenViking.client = &http.Client{Timeout: 15 * time.Second}
	}
}

func (s *OpenVikingService) Configured() bool {
	return s != nil && s.baseURL != "" && s.skillsRoot != ""
}

func (s *OpenVikingService) BaseURL() string {
	if s == nil {
		return ""
	}
	return s.baseURL
}

func (s *OpenVikingService) APIKey() string {
	if s == nil {
		return ""
	}
	return s.apiKey
}

func (s *OpenVikingService) SkillsRoot() string {
	if s == nil {
		return ""
	}
	return s.skillsRoot
}

func (s *OpenVikingService) Client() *http.Client {
	if s == nil {
		return nil
	}
	if s.client == nil {
		s.client = &http.Client{Timeout: 15 * time.Second}
	}
	return s.client
}

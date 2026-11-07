package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

// SettingsService persists user preferences to ~/.clrsig/settings.json.
type SettingsService struct {
	ctx             context.Context
	settings        domain.Settings
	onSave          func() // called after each successful save; wired to RequestService.RebuildExecutor
}

func NewSettingsService() *SettingsService {
	s := &SettingsService{settings: domain.DefaultSettings()}
	s.settings = s.load()
	return s
}

func (s *SettingsService) setContext(ctx context.Context) {
	s.ctx = ctx
}

// GetSettings returns the current settings.
func (s *SettingsService) GetSettings() domain.Settings {
	return s.settings
}

// SaveSettings persists updated settings and returns them.
func (s *SettingsService) SaveSettings(updated domain.Settings) domain.Settings {
	// Clamp font size to a sane range.
	if updated.FontSize < 10 {
		updated.FontSize = 10
	}
	if updated.FontSize > 24 {
		updated.FontSize = 24
	}
	if updated.Theme != "light" {
		updated.Theme = "dark"
	}
	s.settings = updated
	s.persist()
	if s.onSave != nil {
		s.onSave()
	}
	return s.settings
}

// ResetSettings restores defaults and persists them.
func (s *SettingsService) ResetSettings() domain.Settings {
	s.settings = domain.DefaultSettings()
	s.persist()
	return s.settings
}

func settingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".clrsig")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

func (s *SettingsService) load() domain.Settings {
	defaults := domain.DefaultSettings()
	p, err := settingsPath()
	if err != nil {
		return defaults
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return defaults
	}
	var loaded domain.Settings
	if err := json.Unmarshal(data, &loaded); err != nil {
		return defaults
	}
	// Apply defaults for zero values so new fields are always set.
	if loaded.Theme == "" {
		loaded.Theme = defaults.Theme
	}
	if loaded.FontSize == 0 {
		loaded.FontSize = defaults.FontSize
	}
	if loaded.DefaultTimeoutMs == 0 {
		loaded.DefaultTimeoutMs = defaults.DefaultTimeoutMs
	}
	return loaded
}

func (s *SettingsService) persist() {
	p, err := settingsPath()
	if err != nil {
		return
	}
	data, err := json.MarshalIndent(s.settings, "", "  ")
	if err != nil {
		return
	}
	_ = atomicWrite(p, data)
}

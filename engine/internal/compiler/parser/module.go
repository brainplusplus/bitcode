package parser

import (
	"encoding/json"
	"fmt"
	"os"
)

type GroupDefinition struct {
	Label   string   `json:"label"`
	Implies []string `json:"implies,omitempty"`
}

type MenuItemDefinition struct {
	Label    string               `json:"label"`
	Icon     string               `json:"icon,omitempty"`
	View     string               `json:"view,omitempty"`
	Page     string               `json:"page,omitempty"`
	Groups   []string             `json:"groups,omitempty"`
	Children []MenuItemDefinition `json:"children,omitempty"`
}

type SettingDefinition struct {
	Type    string `json:"type"`
	Default any    `json:"default,omitempty"`
}

type IncludeMenuDefinition struct {
	Module string   `json:"module"`
	Views  []string `json:"views,omitempty"`
}

type TableConfig struct {
	Prefix string `json:"prefix"`
}

// OfflineConfig holds advanced offline mode settings.
// All fields are optional — sensible defaults are applied by the engine.
type OfflineConfig struct {
	MaxOfflineHours     int    `json:"max_offline_hours,omitempty"`      // Force re-auth after N hours offline. Default: 72
	SyncBatchSize       int    `json:"sync_batch_size,omitempty"`        // Operations per sync batch. Default: 100
	InventoryOversell   string `json:"inventory_oversell,omitempty"`     // "allow" (default) | "block" | "warn"
	ConflictOnSameField string `json:"conflict_on_same_field,omitempty"` // "latest" (default) | "ask_user" | "server_wins"
}

// ModuleAppConfig holds the app-level configuration for a module.
// This controls whether the module runs in online or offline mode.
//
// Example module.json:
//
//	{
//	  "app": {
//	    "mode": "offline",
//	    "offline": { "max_offline_hours": 48 }
//	  }
//	}
type ModuleAppConfig struct {
	Mode    string         `json:"mode,omitempty"`    // "online" (default) | "offline"
	Offline *OfflineConfig `json:"offline,omitempty"` // Advanced offline settings (optional)
}

type ModuleDefinition struct {
	Name        string                       `json:"name"`
	Version     string                       `json:"version"`
	Label       string                       `json:"label,omitempty"`
	Depends     []string                     `json:"depends,omitempty"`
	Category    string                       `json:"category,omitempty"`
	Auth        *bool                        `json:"auth,omitempty"`
	Table       *TableConfig                 `json:"table,omitempty"`
	Models      []string                     `json:"models,omitempty"`
	APIs        []string                     `json:"apis,omitempty"`
	Processes   []string                     `json:"processes,omitempty"`
	Agents      []string                     `json:"agents,omitempty"`
	Views       []string                     `json:"views,omitempty"`
	Templates   []string                     `json:"templates,omitempty"`
	Scripts     []string                     `json:"scripts,omitempty"`
	Migrations  []string                     `json:"migrations,omitempty"`
	I18n        []string                     `json:"i18n,omitempty"`
	Permissions map[string]string            `json:"permissions,omitempty"`
	Groups      map[string]GroupDefinition   `json:"groups,omitempty"`
	Menu           []MenuItemDefinition         `json:"menu,omitempty"`
	MenuVisibility string                       `json:"menu_visibility,omitempty"`
	IncludeMenus   []IncludeMenuDefinition      `json:"include_menus,omitempty"`
	Settings       map[string]SettingDefinition  `json:"settings,omitempty"`
	Securities []string `json:"securities,omitempty"`
	Pages      []string `json:"pages,omitempty"`

	App *ModuleAppConfig `json:"app,omitempty"`

	EnvAllow   []string `json:"env_allow,omitempty"`
	EnvDeny    []string `json:"env_deny,omitempty"`
	ExecAllow  []string `json:"exec_allow,omitempty"`
	ExecDeny   []string `json:"exec_deny,omitempty"`
	FSAllow    []string `json:"fs_allow,omitempty"`
	FSDeny     []string `json:"fs_deny,omitempty"`
	SudoAllow  bool     `json:"sudo_allow,omitempty"`
}

func (m *ModuleDefinition) RequiresAuth() bool {
	if m.Auth == nil {
		return true
	}
	return *m.Auth
}

func (m *ModuleDefinition) IsOffline() bool {
	if m.App == nil {
		return false
	}
	return m.App.Mode == "offline"
}

func (m *ModuleDefinition) GetOfflineConfig() OfflineConfig {
	defaults := OfflineConfig{
		MaxOfflineHours:     72,
		SyncBatchSize:       100,
		InventoryOversell:   "allow",
		ConflictOnSameField: "latest",
	}
	if m.App == nil || m.App.Offline == nil {
		return defaults
	}
	cfg := *m.App.Offline
	if cfg.MaxOfflineHours == 0 {
		cfg.MaxOfflineHours = defaults.MaxOfflineHours
	}
	if cfg.SyncBatchSize == 0 {
		cfg.SyncBatchSize = defaults.SyncBatchSize
	}
	if cfg.InventoryOversell == "" {
		cfg.InventoryOversell = defaults.InventoryOversell
	}
	if cfg.ConflictOnSameField == "" {
		cfg.ConflictOnSameField = defaults.ConflictOnSameField
	}
	return cfg
}

func ParseModule(data []byte) (*ModuleDefinition, error) {
	var mod ModuleDefinition
	if err := json.Unmarshal(data, &mod); err != nil {
		return nil, fmt.Errorf("invalid module JSON: %w", err)
	}
	if mod.Name == "" {
		return nil, fmt.Errorf("module name is required")
	}
	if mod.Version == "" {
		mod.Version = "0.1.0"
	}
	return &mod, nil
}

func ParseModuleFile(path string) (*ModuleDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read module file %s: %w", path, err)
	}
	return ParseModule(data)
}

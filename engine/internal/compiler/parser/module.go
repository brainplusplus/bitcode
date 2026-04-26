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
}

func (m *ModuleDefinition) RequiresAuth() bool {
	if m.Auth == nil {
		return true
	}
	return *m.Auth
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

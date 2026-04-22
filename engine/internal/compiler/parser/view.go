package parser

import (
	"encoding/json"
	"fmt"
	"os"
)

type ViewType string

const (
	ViewList     ViewType = "list"
	ViewForm     ViewType = "form"
	ViewKanban   ViewType = "kanban"
	ViewCalendar ViewType = "calendar"
	ViewChart    ViewType = "chart"
	ViewCustom   ViewType = "custom"
	ViewGantt    ViewType = "gantt"
	ViewMap      ViewType = "map"
	ViewTree     ViewType = "tree"
	ViewActivity ViewType = "activity"
	ViewReport   ViewType = "report"
)

type ActionDefinition struct {
	Label      string `json:"label"`
	Process    string `json:"process,omitempty"`
	Permission string `json:"permission,omitempty"`
	Variant    string `json:"variant,omitempty"`
	Visible    string `json:"visible,omitempty"`
	Confirm    string `json:"confirm,omitempty"`
}

type SortDefinition struct {
	Field string `json:"field"`
	Order string `json:"order,omitempty"`
}

type LayoutRow struct {
	Field    string `json:"field,omitempty"`
	Width    int    `json:"width,omitempty"`
	Readonly bool   `json:"readonly,omitempty"`
	Widget   string `json:"widget,omitempty"`
	Formula  string `json:"formula,omitempty"`
}

type TabDefinition struct {
	Label   string   `json:"label"`
	View    string   `json:"view,omitempty"`
	Fields  []string `json:"fields,omitempty"`
	Visible string   `json:"visible,omitempty"`
}

type HeaderDefinition struct {
	StatusField string             `json:"status_field,omitempty"`
	Widget      string             `json:"widget,omitempty"`
	Buttons     []ActionDefinition `json:"buttons,omitempty"`
}

type SmartButtonDefinition struct {
	Label       string  `json:"label"`
	Icon        string  `json:"icon,omitempty"`
	CountModel  string  `json:"count_model,omitempty"`
	CountDomain [][]any `json:"count_domain,omitempty"`
	View        string  `json:"view,omitempty"`
}

type SectionDefinition struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Collapsible bool   `json:"collapsible,omitempty"`
	CollapsedBy string `json:"collapsed_by,omitempty"`
}

type SeparatorDefinition struct {
	Label string `json:"label,omitempty"`
}

type ChildTableColumn struct {
	Field    string `json:"field"`
	Width    int    `json:"width,omitempty"`
	Readonly bool   `json:"readonly,omitempty"`
	Widget   string `json:"widget,omitempty"`
	Formula  string `json:"formula,omitempty"`
}

type ChildTableDefinition struct {
	Field   string             `json:"field"`
	Columns []ChildTableColumn `json:"columns"`
	Summary map[string]string  `json:"summary,omitempty"`
}

type LayoutItem struct {
	Row  []LayoutRow     `json:"row,omitempty"`
	Tabs []TabDefinition `json:"tabs,omitempty"`

	Header     *HeaderDefinition       `json:"header,omitempty"`
	ButtonBox  []SmartButtonDefinition `json:"button_box,omitempty"`
	Section    *SectionDefinition      `json:"section,omitempty"`
	Rows       []LayoutItem            `json:"rows,omitempty"`
	ChildTable *ChildTableDefinition   `json:"child_table,omitempty"`
	Chatter    bool                    `json:"chatter,omitempty"`
	Separator  *SeparatorDefinition    `json:"separator,omitempty"`
}

type DataSourceDefinition struct {
	Model   string  `json:"model,omitempty"`
	Domain  [][]any `json:"domain,omitempty"`
	Process string  `json:"process,omitempty"`
}

type ViewDefinition struct {
	Name        string                          `json:"name"`
	Type        ViewType                        `json:"type"`
	Model       string                          `json:"model,omitempty"`
	Title       string                          `json:"title,omitempty"`
	Fields      []string                        `json:"fields,omitempty"`
	Filters     []string                        `json:"filters,omitempty"`
	Sort        *SortDefinition                 `json:"sort,omitempty"`
	Layout      []LayoutItem                    `json:"layout,omitempty"`
	Actions     []ActionDefinition              `json:"actions,omitempty"`
	Template    string                          `json:"template,omitempty"`
	DataSources map[string]DataSourceDefinition `json:"data_sources,omitempty"`
	GroupBy     string                          `json:"group_by,omitempty"`
	RegisterTo  []string                        `json:"register_to,omitempty"`
	DateField   string                          `json:"date_field,omitempty"`
	StartField  string                          `json:"start_field,omitempty"`
	EndField    string                          `json:"end_field,omitempty"`
	ParentField string                          `json:"parent_field,omitempty"`
}

func ParseView(data []byte) (*ViewDefinition, error) {
	var view ViewDefinition
	if err := json.Unmarshal(data, &view); err != nil {
		return nil, fmt.Errorf("invalid view JSON: %w", err)
	}
	if view.Name == "" {
		return nil, fmt.Errorf("view name is required")
	}
	if view.Type == "" {
		return nil, fmt.Errorf("view type is required")
	}
	if view.Type == ViewCustom && view.Template == "" {
		return nil, fmt.Errorf("custom view requires a template")
	}
	return &view, nil
}

func ParseViewFile(path string) (*ViewDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read view file %s: %w", path, err)
	}
	return ParseView(data)
}

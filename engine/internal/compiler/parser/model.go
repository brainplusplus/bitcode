package parser

import (
	"encoding/json"
	"fmt"
	"os"
)

type FieldType string

const (
	FieldString    FieldType = "string"
	FieldText      FieldType = "text"
	FieldInteger   FieldType = "integer"
	FieldDecimal   FieldType = "decimal"
	FieldBoolean   FieldType = "boolean"
	FieldDate      FieldType = "date"
	FieldDatetime  FieldType = "datetime"
	FieldSelection FieldType = "selection"
	FieldEmail     FieldType = "email"
	FieldMany2One  FieldType = "many2one"
	FieldOne2Many  FieldType = "one2many"
	FieldMany2Many FieldType = "many2many"
	FieldJSON      FieldType = "json"
	FieldFile      FieldType = "file"
	FieldComputed  FieldType = "computed"

	FieldSmallText   FieldType = "smalltext"
	FieldRichText    FieldType = "richtext"
	FieldMarkdown    FieldType = "markdown"
	FieldHTML        FieldType = "html"
	FieldCode        FieldType = "code"
	FieldPassword    FieldType = "password"
	FieldFloat       FieldType = "float"
	FieldCurrency    FieldType = "currency"
	FieldPercent     FieldType = "percent"
	FieldToggle      FieldType = "toggle"
	FieldRadio       FieldType = "radio"
	FieldDynamicLink FieldType = "dynamic_link"
	FieldTime        FieldType = "time"
	FieldDuration    FieldType = "duration"
	FieldImage       FieldType = "image"
	FieldSignature   FieldType = "signature"
	FieldBarcode     FieldType = "barcode"
	FieldColor       FieldType = "color"
	FieldGeolocation FieldType = "geolocation"
	FieldRating      FieldType = "rating"
)

type FieldDefinition struct {
	Type      FieldType `json:"type"`
	Label     string    `json:"label,omitempty"`
	Required  bool      `json:"required,omitempty"`
	Unique    bool      `json:"unique,omitempty"`
	Default   any       `json:"default,omitempty"`
	Max       int       `json:"max,omitempty"`
	Min       int       `json:"min,omitempty"`
	Precision int       `json:"precision,omitempty"`
	MaxSize   string    `json:"max_size,omitempty"`

	Options []string `json:"options,omitempty"`

	Model   string `json:"model,omitempty"`
	Inverse string `json:"inverse,omitempty"`

	Computed string `json:"computed,omitempty"`

	Auto bool `json:"auto,omitempty"`

	Widget string `json:"widget,omitempty"`

	DependsOn   string `json:"depends_on,omitempty"`
	ReadOnlyIf  string `json:"readonly_if,omitempty"`
	MandatoryIf string `json:"mandatory_if,omitempty"`
	FetchFrom   string `json:"fetch_from,omitempty"`
	Formula     string `json:"formula,omitempty"`

	Language     string `json:"language,omitempty"`
	Toolbar      string `json:"toolbar,omitempty"`
	CurrencyCode string `json:"currency,omitempty"`
	Format       string `json:"format,omitempty"`
	DrawMode     string `json:"draw_mode,omitempty"`
	MaxStars     int    `json:"max_stars,omitempty"`
	HalfStars    bool   `json:"half_stars,omitempty"`
	Rows         int    `json:"rows,omitempty"`
	Accept       string `json:"accept,omitempty"`
}

type RecordRuleDefinition struct {
	Groups []string `json:"groups"`
	Domain [][]any  `json:"domain"`
}

type ModelDefinition struct {
	Name        string                     `json:"name"`
	Module      string                     `json:"module,omitempty"`
	Label       string                     `json:"label,omitempty"`
	Inherit     string                     `json:"inherit,omitempty"`
	Fields      map[string]FieldDefinition `json:"fields"`
	RecordRules []RecordRuleDefinition     `json:"record_rules,omitempty"`
	Indexes     [][]string                 `json:"indexes,omitempty"`
}

func ParseModel(data []byte) (*ModelDefinition, error) {
	var model ModelDefinition
	if err := json.Unmarshal(data, &model); err != nil {
		return nil, fmt.Errorf("invalid model JSON: %w", err)
	}
	if model.Name == "" {
		return nil, fmt.Errorf("model name is required")
	}
	if model.Fields == nil || len(model.Fields) == 0 {
		return nil, fmt.Errorf("model must have at least one field")
	}
	for name, field := range model.Fields {
		if field.Type == "" {
			return nil, fmt.Errorf("field %q must have a type", name)
		}
		if field.Type == FieldMany2One && field.Model == "" {
			return nil, fmt.Errorf("many2one field %q must specify model", name)
		}
		if field.Type == FieldOne2Many && (field.Model == "" || field.Inverse == "") {
			return nil, fmt.Errorf("one2many field %q must specify model and inverse", name)
		}
		if field.Type == FieldMany2Many && field.Model == "" {
			return nil, fmt.Errorf("many2many field %q must specify model", name)
		}
		if field.Type == FieldSelection && len(field.Options) == 0 {
			return nil, fmt.Errorf("selection field %q must have options", name)
		}
		if field.Type == FieldRadio && len(field.Options) == 0 {
			return nil, fmt.Errorf("radio field %q must have options", name)
		}
		if field.Type == FieldDynamicLink && field.Model == "" {
			return nil, fmt.Errorf("dynamic_link field %q must specify model", name)
		}
	}
	return &model, nil
}

func ParseModelFile(path string) (*ModelDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read model file %s: %w", path, err)
	}
	return ParseModel(data)
}

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

type PKStrategy string

const (
	PKAutoIncrement PKStrategy = "auto_increment"
	PKComposite     PKStrategy = "composite"
	PKUUID          PKStrategy = "uuid"
	PKNaturalKey    PKStrategy = "natural_key"
	PKNamingSeries  PKStrategy = "naming_series"
	PKManual        PKStrategy = "manual"
)

type SequenceConfig struct {
	Reset string `json:"reset,omitempty"`
	Step  int    `json:"step,omitempty"`
}

type PrimaryKeyConfig struct {
	Strategy  PKStrategy      `json:"strategy"`
	Field     string          `json:"field,omitempty"`
	Fields    []string        `json:"fields,omitempty"`
	Surrogate *bool           `json:"surrogate,omitempty"`
	Version   string          `json:"version,omitempty"`
	Format    string          `json:"format,omitempty"`
	Namespace string          `json:"namespace,omitempty"`
	Sequence  *SequenceConfig `json:"sequence,omitempty"`
}

func (pk *PrimaryKeyConfig) IsSurrogate() bool {
	if pk.Surrogate == nil {
		return true
	}
	return *pk.Surrogate
}

type AutoFormatConfig struct {
	Format   string          `json:"format"`
	Sequence *SequenceConfig `json:"sequence,omitempty"`
}

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
	Multiple     bool   `json:"multiple,omitempty"`
	PathFormat   string `json:"path_format,omitempty"`
	NameFormat   string `json:"name_format,omitempty"`
	AutoFormat   *AutoFormatConfig `json:"auto_format,omitempty"`
}

type FileConfig struct {
	MaxSize           int64    `json:"max_size,omitempty"`
	AllowedExtensions []string `json:"allowed_extensions,omitempty"`
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
	PrimaryKey  *PrimaryKeyConfig          `json:"primary_key,omitempty"`
	Fields      map[string]FieldDefinition `json:"fields"`
	RecordRules []RecordRuleDefinition     `json:"record_rules,omitempty"`
	Indexes     [][]string                 `json:"indexes,omitempty"`
	FileConfig  *FileConfig                `json:"file_config,omitempty"`
}

func ParseModel(data []byte) (*ModelDefinition, error) {
	var model ModelDefinition
	if err := json.Unmarshal(data, &model); err != nil {
		return nil, fmt.Errorf("invalid model JSON: %w", err)
	}
	if model.Name == "" {
		return nil, fmt.Errorf("model name is required")
	}
	if len(model.Fields) == 0 {
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
	if err := validatePrimaryKey(&model); err != nil {
		return nil, err
	}
	return &model, nil
}

func validatePrimaryKey(model *ModelDefinition) error {
	if model.PrimaryKey == nil {
		return nil
	}

	pk := model.PrimaryKey
	if pk.Sequence != nil {
		switch pk.Sequence.Reset {
		case "", "never", "minutely", "hourly", "daily", "monthly", "yearly", "key":
		default:
			return fmt.Errorf("primary key sequence reset must be one of: never, minutely, hourly, daily, monthly, yearly, key")
		}
	}

	switch pk.Strategy {
	case PKAutoIncrement:
		return nil
	case PKComposite:
		if len(pk.Fields) < 2 {
			return fmt.Errorf("composite primary key must specify at least two fields")
		}
		for _, fieldName := range pk.Fields {
			if _, ok := model.Fields[fieldName]; !ok {
				return fmt.Errorf("composite primary key field %q does not exist", fieldName)
			}
		}
		return nil
	case PKUUID:
		switch pk.Version {
		case "", "v4", "v7":
			return nil
		case "format":
			if pk.Format == "" {
				return fmt.Errorf("uuid primary key with format version must specify format")
			}
			return nil
		default:
			return fmt.Errorf("uuid primary key version must be one of: v4, v7, format")
		}
	case PKNaturalKey:
		field, err := primaryKeyField(model, pk.Field)
		if err != nil {
			return err
		}
		if !field.Required {
			return fmt.Errorf("natural key field %q must be required", pk.Field)
		}
		return nil
	case PKNamingSeries:
		if _, err := primaryKeyField(model, pk.Field); err != nil {
			return err
		}
		if pk.Format == "" {
			return fmt.Errorf("naming_series primary key must specify format")
		}
		return nil
	case PKManual:
		_, err := primaryKeyField(model, pk.Field)
		return err
	default:
		return fmt.Errorf("unsupported primary key strategy %q", pk.Strategy)
	}
}

func primaryKeyField(model *ModelDefinition, fieldName string) (FieldDefinition, error) {
	if fieldName == "" {
		return FieldDefinition{}, fmt.Errorf("primary key must specify field")
	}

	field, ok := model.Fields[fieldName]
	if !ok {
		return FieldDefinition{}, fmt.Errorf("primary key field %q does not exist", fieldName)
	}

	return field, nil
}

func ParseModelFile(path string) (*ModelDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read model file %s: %w", path, err)
	}
	return ParseModel(data)
}

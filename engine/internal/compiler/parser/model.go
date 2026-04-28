package parser

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
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

	// Phase 6A: New types
	FieldUUID   FieldType = "uuid"
	FieldIP     FieldType = "ip"
	FieldIPv6   FieldType = "ipv6"
	FieldYear   FieldType = "year"
	FieldVector FieldType = "vector"
	FieldBinary FieldType = "binary"

	// Phase 6A: JSON variants
	FieldJSONObject FieldType = "json:object"
	FieldJSONArray  FieldType = "json:array"
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

type ScriptRef struct {
	Lang string `json:"lang"`
	File string `json:"file"`
}

type EventHandler struct {
	Process    string       `json:"process,omitempty"`
	Script     *ScriptRef   `json:"script,omitempty"`
	Condition  string       `json:"condition,omitempty"`
	Sync       *bool        `json:"sync,omitempty"`
	OnError    string       `json:"on_error,omitempty"`
	Retry      *RetryConfig `json:"retry,omitempty"`
	Timeout    string       `json:"timeout,omitempty"`
	ServerOnly bool         `json:"server_only,omitempty"`
	Priority   int          `json:"priority,omitempty"`
	BulkMode   string       `json:"bulk_mode,omitempty"`
}

func (h *EventHandler) IsSync(eventName string) bool {
	if h.Sync != nil {
		return *h.Sync
	}
	return IsBeforeEvent(eventName)
}

func (h *EventHandler) GetOnError(eventName string) string {
	if h.OnError != "" {
		return h.OnError
	}
	if IsBeforeEvent(eventName) {
		return "fail"
	}
	return "log"
}

func (h *EventHandler) GetTimeout(eventName string) string {
	if h.Timeout != "" {
		return h.Timeout
	}
	if IsBeforeEvent(eventName) {
		return "30s"
	}
	return "60s"
}

func (h *EventHandler) GetBulkMode() string {
	if h.BulkMode != "" {
		return h.BulkMode
	}
	return "each"
}

func IsBeforeEvent(name string) bool {
	switch name {
	case "before_validate", "after_validate",
		"before_create", "before_update", "before_delete",
		"before_save", "before_soft_delete", "before_hard_delete",
		"before_restore", "on_change":
		return true
	}
	return false
}

type EventsDefinition struct {
	BeforeValidate   []EventHandler              `json:"before_validate,omitempty"`
	AfterValidate    []EventHandler              `json:"after_validate,omitempty"`
	BeforeCreate     []EventHandler              `json:"before_create,omitempty"`
	AfterCreate      []EventHandler              `json:"after_create,omitempty"`
	BeforeUpdate     []EventHandler              `json:"before_update,omitempty"`
	AfterUpdate      []EventHandler              `json:"after_update,omitempty"`
	BeforeDelete     []EventHandler              `json:"before_delete,omitempty"`
	AfterDelete      []EventHandler              `json:"after_delete,omitempty"`
	BeforeSave       []EventHandler              `json:"before_save,omitempty"`
	AfterSave        []EventHandler              `json:"after_save,omitempty"`
	BeforeSoftDelete []EventHandler              `json:"before_soft_delete,omitempty"`
	AfterSoftDelete  []EventHandler              `json:"after_soft_delete,omitempty"`
	BeforeHardDelete []EventHandler              `json:"before_hard_delete,omitempty"`
	AfterHardDelete  []EventHandler              `json:"after_hard_delete,omitempty"`
	BeforeRestore    []EventHandler              `json:"before_restore,omitempty"`
	AfterRestore     []EventHandler              `json:"after_restore,omitempty"`
	OnChange         map[string][]EventHandler   `json:"on_change,omitempty"`
}

func (e *EventsDefinition) GetHandlers(eventName string) []EventHandler {
	if e == nil {
		return nil
	}
	switch eventName {
	case "before_validate":
		return e.BeforeValidate
	case "after_validate":
		return e.AfterValidate
	case "before_create":
		return e.BeforeCreate
	case "after_create":
		return e.AfterCreate
	case "before_update":
		return e.BeforeUpdate
	case "after_update":
		return e.AfterUpdate
	case "before_delete":
		return e.BeforeDelete
	case "after_delete":
		return e.AfterDelete
	case "before_save":
		return e.BeforeSave
	case "after_save":
		return e.AfterSave
	case "before_soft_delete":
		return e.BeforeSoftDelete
	case "after_soft_delete":
		return e.AfterSoftDelete
	case "before_hard_delete":
		return e.BeforeHardDelete
	case "after_hard_delete":
		return e.AfterHardDelete
	case "before_restore":
		return e.BeforeRestore
	case "after_restore":
		return e.AfterRestore
	}
	return nil
}

type UniqueConfig struct {
	Scope           []string `json:"scope,omitempty"`
	CaseInsensitive bool     `json:"case_insensitive,omitempty"`
	IncludeTrashed  bool     `json:"include_trashed,omitempty"`
}

type ImmutableAfterConfig struct {
	Field  string   `json:"-"`
	Values []string `json:"-"`
}

type CustomValidator struct {
	Process string     `json:"process,omitempty"`
	Script  *ScriptRef `json:"script,omitempty"`
	Message string     `json:"message,omitempty"`
}

type ValidationRule struct {
	Regex        string `json:"regex,omitempty"`
	RegexMessage string `json:"regex_message,omitempty"`
	Min          *int   `json:"min,omitempty"`
	Max          *int   `json:"max,omitempty"`
	MinLength    *int   `json:"min_length,omitempty"`
	MaxLength    *int   `json:"max_length,omitempty"`
	When         any    `json:"when,omitempty"`
}

type FieldValidation struct {
	Required        any  `json:"required,omitempty"`
	Email           bool `json:"email,omitempty"`
	URL             bool `json:"url,omitempty"`
	Phone           bool `json:"phone,omitempty"`
	IP              bool `json:"ip,omitempty"`
	IPv4            bool `json:"ipv4,omitempty"`
	IPv6            bool `json:"ipv6,omitempty"`
	UUID            bool `json:"uuid,omitempty"`
	JSON            bool `json:"json,omitempty"`

	Alpha        bool   `json:"alpha,omitempty"`
	AlphaNum     bool   `json:"alpha_num,omitempty"`
	AlphaDash    bool   `json:"alpha_dash,omitempty"`
	Numeric      bool   `json:"numeric,omitempty"`
	Regex        string `json:"regex,omitempty"`
	RegexMessage string `json:"regex_message,omitempty"`
	StartsWith   any    `json:"starts_with,omitempty"`
	EndsWith     any    `json:"ends_with,omitempty"`
	Contains     string `json:"contains,omitempty"`
	NotContains  string `json:"not_contains,omitempty"`
	Lowercase    bool   `json:"lowercase,omitempty"`
	Uppercase    bool   `json:"uppercase,omitempty"`

	Min           *float64   `json:"min,omitempty"`
	Max           *float64   `json:"max,omitempty"`
	MinLength     *int       `json:"min_length,omitempty"`
	MaxLength     *int       `json:"max_length,omitempty"`
	Between       []float64  `json:"between,omitempty"`
	LengthBetween []int      `json:"length_between,omitempty"`
	Size          *int       `json:"size,omitempty"`

	In        []any  `json:"in,omitempty"`
	NotIn     []any  `json:"not_in,omitempty"`
	Confirmed string `json:"confirmed,omitempty"`
	Different string `json:"different,omitempty"`
	Gt        string `json:"gt,omitempty"`
	Gte       string `json:"gte,omitempty"`
	Lt        string `json:"lt,omitempty"`
	Lte       string `json:"lte,omitempty"`

	DateBefore        string `json:"date_before,omitempty"`
	DateAfter         string `json:"date_after,omitempty"`
	DateBeforeOrEqual string `json:"date_before_or_equal,omitempty"`
	DateAfterOrEqual  string `json:"date_after_or_equal,omitempty"`

	UniqueSimple bool          `json:"-"`
	UniqueConfig *UniqueConfig `json:"-"`
	UniqueRaw    any           `json:"unique,omitempty"`

	Exists      bool           `json:"exists,omitempty"`
	ExistsWhere map[string]any `json:"exists_where,omitempty"`
	MinItems    any            `json:"min_items,omitempty"`
	MaxItems    any            `json:"max_items,omitempty"`

	FileSize string   `json:"file_size,omitempty"`
	FileType []string `json:"file_type,omitempty"`

	Immutable      bool `json:"immutable,omitempty"`
	ImmutableAfter any  `json:"immutable_after,omitempty"`

	RequiredIf         map[string]any `json:"required_if,omitempty"`
	RequiredUnless     map[string]any `json:"required_unless,omitempty"`
	RequiredWith       []string       `json:"required_with,omitempty"`
	RequiredWithAll    []string       `json:"required_with_all,omitempty"`
	RequiredWithout    []string       `json:"required_without,omitempty"`
	RequiredWithoutAll []string       `json:"required_without_all,omitempty"`
	ExcludeIf          map[string]any `json:"exclude_if,omitempty"`
	ExcludeUnless      map[string]any `json:"exclude_unless,omitempty"`

	When any `json:"when,omitempty"`

	Rules []ValidationRule `json:"rules,omitempty"`

	Custom []CustomValidator `json:"custom,omitempty"`

	Messages map[string]string `json:"messages,omitempty"`
}

type ModelValidator struct {
	Name       string     `json:"name"`
	Expression string     `json:"expression,omitempty"`
	Process    string     `json:"process,omitempty"`
	Script     *ScriptRef `json:"script,omitempty"`
	Message    string     `json:"message,omitempty"`
	Condition  string     `json:"condition,omitempty"`
	On         string     `json:"on,omitempty"`
}

func (v *ModelValidator) GetOn() string {
	if v.On != "" {
		return v.On
	}
	return "always"
}

type SanitizeConfig struct {
	AllStrings []string `json:"_all_strings,omitempty"`
}

type NumberFormatConfig struct {
	ThousandSeparator string `json:"thousand_separator,omitempty"`
	DecimalSeparator  string `json:"decimal_separator,omitempty"`
	Precision         int    `json:"precision,omitempty"`
	Prefix            string `json:"prefix,omitempty"`
	Suffix            string `json:"suffix,omitempty"`
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
	Encrypted    bool              `json:"encrypted,omitempty"`

	Validation *FieldValidation `json:"validation,omitempty"`
	Sanitize   []string         `json:"sanitize,omitempty"`

	Mask       bool     `json:"mask,omitempty"`
	MaskLength int      `json:"mask_length,omitempty"`
	Groups     []string `json:"groups,omitempty"`

	Hidden        bool                `json:"hidden,omitempty"`
	Storage       string              `json:"storage,omitempty"`
	Scale         int                 `json:"scale,omitempty"`
	DisplayField  string              `json:"display_field,omitempty"`
	CurrencyField string              `json:"currency_field,omitempty"`
	Dimensions    int                 `json:"dimensions,omitempty"`
	FieldIndex    any                 `json:"index,omitempty"`
	NumberFormat  *NumberFormatConfig  `json:"number_format,omitempty"`
	Models        []string            `json:"models,omitempty"`
	Morph         string              `json:"morph,omitempty"`
}

type FileConfig struct {
	MaxSize           int64    `json:"max_size,omitempty"`
	AllowedExtensions []string `json:"allowed_extensions,omitempty"`
}

type RecordRuleDefinition struct {
	Groups []string `json:"groups"`
	Domain [][]any  `json:"domain"`
}

type ModelTableConfig struct {
	Prefix string `json:"prefix"`
	Plural *bool  `json:"plural,omitempty"`
	Name   string `json:"name,omitempty"`
}

type APIConfig struct {
	AutoCRUD   bool            `json:"auto_crud"`
	Auth       bool            `json:"auth"`
	AutoPages  json.RawMessage `json:"auto_pages,omitempty"`
	Modal      bool            `json:"modal,omitempty"`
	Protocols  ProtocolConfig  `json:"protocols,omitempty"`
	Search     []string        `json:"search,omitempty"`
	SoftDelete *bool           `json:"soft_delete,omitempty"`
}

type ProtocolConfig struct {
	REST      bool `json:"rest"`
	GraphQL   bool `json:"graphql"`
	WebSocket bool `json:"websocket"`
}

func (a *APIConfig) IsAutoPages() bool {
	if a.AutoPages == nil {
		return true
	}
	var b bool
	if err := json.Unmarshal(a.AutoPages, &b); err == nil {
		return b
	}
	return true
}

func (a *APIConfig) IsSoftDelete() bool {
	if a.SoftDelete == nil {
		return true
	}
	return *a.SoftDelete
}

type ModelDefinition struct {
	Name         string                     `json:"name"`
	Module       string                     `json:"module,omitempty"`
	Label        string                     `json:"label,omitempty"`
	Inherit      string                     `json:"inherit,omitempty"`
	PrimaryKey   *PrimaryKeyConfig          `json:"primary_key,omitempty"`
	Fields       map[string]FieldDefinition `json:"fields"`
	RecordRules  []RecordRuleDefinition     `json:"record_rules,omitempty"`
	Indexes      [][]string                 `json:"indexes,omitempty"`
	FileConfig   *FileConfig                `json:"file_config,omitempty"`
	TitleField   string                     `json:"title_field,omitempty"`
	SearchField  []string                   `json:"search_field,omitempty"`
	TableRaw     json.RawMessage            `json:"table,omitempty"`
	TableName    string                     `json:"-"`
	TablePrefix  *string                    `json:"-"`
	TablePlural  *bool                      `json:"-"`
	Version       *bool                      `json:"version,omitempty"`
	Timestamps    *bool                      `json:"timestamps,omitempty"`
	TimestampsBy  *bool                      `json:"timestamps_by,omitempty"`
	SoftDeletes   *bool                      `json:"soft_deletes,omitempty"`
	SoftDeletesBy *bool                      `json:"soft_deletes_by,omitempty"`
	TenantScoped  *bool                      `json:"tenant_scoped,omitempty"`

	Events     *EventsDefinition `json:"events,omitempty"`
	Validators []ModelValidator  `json:"validators,omitempty"`
	Sanitize   *SanitizeConfig   `json:"sanitize,omitempty"`

	APIRaw json.RawMessage `json:"api,omitempty"`
	API    *APIConfig      `json:"-"`

	App *ModelAppConfig `json:"app,omitempty"`

	ModulePath    string `json:"-"`
	OfflineModule bool   `json:"-"`
}

type ModelAppConfig struct {
	Mode string `json:"mode,omitempty"` // "online" (default) | "offline"
}

func (m *ModelDefinition) IsOffline() bool {
	if m.App == nil {
		return false
	}
	return m.App.Mode == "offline"
}

func (m *ModelDefinition) IsVersion() bool {
	if m.Version == nil {
		return false
	}
	return *m.Version
}

func (m *ModelDefinition) IsTimestamps() bool {
	if m.Timestamps == nil {
		return true
	}
	return *m.Timestamps
}

func (m *ModelDefinition) IsTimestampsBy() bool {
	if m.TimestampsBy == nil {
		return true
	}
	return *m.TimestampsBy
}

func (m *ModelDefinition) IsSoftDeletes() bool {
	if m.SoftDeletes == nil {
		return false
	}
	return *m.SoftDeletes
}

func (m *ModelDefinition) IsSoftDeletesBy() bool {
	if m.SoftDeletesBy == nil {
		return false
	}
	return *m.SoftDeletesBy
}

func (m *ModelDefinition) IsTenantScoped() bool {
	if m.TenantScoped == nil {
		return true
	}
	return *m.TenantScoped
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

		rawType := string(field.Type)
		switch rawType {
		case "json:object":
			field.Type = FieldJSONObject
		case "json:array":
			field.Type = FieldJSONArray
		case "ip:v4":
			field.Type = FieldIP
		case "ip:v6":
			field.Type = FieldIPv6
		default:
			if strings.Contains(rawType, ":") {
				base := strings.SplitN(rawType, ":", 2)[0]
				if base != "json" && base != "ip" {
					return nil, fmt.Errorf("field %q: type %q does not support variants (only json and ip do)", name, rawType)
				}
			}
		}
		model.Fields[name] = field

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
		if field.Type == FieldVector && field.Dimensions == 0 {
			return nil, fmt.Errorf("vector field %q must specify dimensions", name)
		}
		if field.CurrencyCode != "" && field.CurrencyField != "" {
			return nil, fmt.Errorf("field %q cannot have both currency and currency_field", name)
		}
		if field.DisplayField != "" && field.Type != FieldMany2One {
			log.Printf("WARN: display_field on non-many2one field %q will be ignored", name)
		}
		if field.Storage != "" && !isValidStorageHint(field.Type, field.Storage) {
			return nil, fmt.Errorf("field %q: invalid storage hint %q for type %q", name, field.Storage, field.Type)
		}
	}
	if err := validatePrimaryKey(&model); err != nil {
		return nil, err
	}
	if model.TitleField == "" {
		model.TitleField = resolveTitleField(&model)
	}
	if len(model.SearchField) == 0 {
		if strings.Contains(model.TitleField, "{") {
			model.SearchField = extractSearchableFields(&model, model.TitleField)
		} else {
			model.SearchField = []string{model.TitleField}
		}
	}
	if len(model.TableRaw) > 0 {
		var tableName string
		if err := json.Unmarshal(model.TableRaw, &tableName); err == nil {
			model.TableName = tableName
		} else {
			var tableConfig ModelTableConfig
			if err := json.Unmarshal(model.TableRaw, &tableConfig); err == nil {
				prefix := tableConfig.Prefix
				model.TablePrefix = &prefix
				model.TablePlural = tableConfig.Plural
				if tableConfig.Name != "" {
					model.TableName = tableConfig.Name
				}
			}
		}
	}
	resolveFieldValidation(&model)

	if model.APIRaw != nil {
		var apiBool bool
		if err := json.Unmarshal(model.APIRaw, &apiBool); err == nil {
			if apiBool {
				model.API = &APIConfig{
					AutoCRUD: true,
					Auth:     true,
					Protocols: ProtocolConfig{REST: true},
				}
			}
		} else {
			var apiConfig APIConfig
			if err := json.Unmarshal(model.APIRaw, &apiConfig); err == nil {
				model.API = &apiConfig
			}
		}
	}

	return &model, nil
}

func resolveFieldValidation(model *ModelDefinition) {
	for name, field := range model.Fields {
		if field.Validation == nil {
			continue
		}
		v := field.Validation
		switch raw := v.UniqueRaw.(type) {
		case bool:
			v.UniqueSimple = raw
		case map[string]any:
			cfg := &UniqueConfig{}
			if scope, ok := raw["scope"].([]any); ok {
				for _, s := range scope {
					if str, ok := s.(string); ok {
						cfg.Scope = append(cfg.Scope, str)
					}
				}
			}
			if ci, ok := raw["case_insensitive"].(bool); ok {
				cfg.CaseInsensitive = ci
			}
			if it, ok := raw["include_trashed"].(bool); ok {
				cfg.IncludeTrashed = it
			}
			v.UniqueConfig = cfg
		}
		field.Validation = v
		model.Fields[name] = field
	}
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

func resolveTitleField(model *ModelDefinition) string {
	candidates := []string{"name", "label", "title", "code", "username", "description"}
	for _, c := range candidates {
		if _, ok := model.Fields[c]; ok {
			return c
		}
	}
	for fieldName, field := range model.Fields {
		if fieldName == "id" {
			continue
		}
		switch field.Type {
		case FieldString, FieldText, FieldSmallText, FieldEmail:
			return fieldName
		}
	}
	return "id"
}

func ParseModelFile(path string) (*ModelDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read model file %s: %w", path, err)
	}
	return ParseModel(data)
}

func isValidStorageHint(fieldType FieldType, storage string) bool {
	valid := map[FieldType][]string{
		FieldInteger:  {"smallint", "bigint"},
		FieldDecimal:  {"numeric", "double"},
		FieldFloat:    {"double", "real"},
		FieldText:     {"mediumtext", "longtext"},
		FieldString:   {"char"},
		FieldBinary:   {"mediumblob", "longblob"},
		FieldDatetime: {"naive"},
		FieldCurrency: {"numeric"},
	}
	hints, ok := valid[fieldType]
	if !ok {
		return false
	}
	for _, h := range hints {
		if h == storage {
			return true
		}
	}
	return false
}

var dataFieldRe = regexp.MustCompile(`\{(?:[a-z_]+\()?data\.([a-z_]+)`)

func extractSearchableFields(model *ModelDefinition, format string) []string {
	matches := dataFieldRe.FindAllStringSubmatch(format, -1)

	var fields []string
	seen := make(map[string]bool)
	for _, m := range matches {
		fieldName := m[1]
		if seen[fieldName] {
			continue
		}
		seen[fieldName] = true
		if fd, ok := model.Fields[fieldName]; ok {
			if isTextSearchable(fd.Type) {
				fields = append(fields, fieldName)
			}
		}
	}

	if len(fields) == 0 {
		return []string{resolveTitleField(model)}
	}
	return fields
}

func isTextSearchable(ft FieldType) bool {
	switch ft {
	case FieldString, FieldText, FieldEmail, FieldPassword, FieldBarcode,
		FieldColor, FieldCode, FieldSmallText, FieldRichText, FieldMarkdown,
		FieldHTML, FieldIP, FieldIPv6, FieldUUID:
		return true
	}
	return false
}

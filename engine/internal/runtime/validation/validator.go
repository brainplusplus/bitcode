package validation

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

type UniqueChecker func(ctx context.Context, tableName string, fieldName string, value any, excludeID string, cfg *parser.UniqueConfig, isSoftDelete bool, data map[string]any) (bool, error)

type ExistsChecker func(ctx context.Context, tableName string, id any, conditions map[string]any) (bool, error)

type RelationCounter func(ctx context.Context, tableName string, foreignKey string, parentID any) (int64, error)

type CustomValidatorRunner func(ctx context.Context, cv parser.CustomValidator, fieldName string, fieldValue any, data map[string]any, modulePath string) error

type Validator struct {
	translator        func(locale, key string) string
	uniqueChecker     UniqueChecker
	existsChecker     ExistsChecker
	relationCounter   RelationCounter
	customRunner      CustomValidatorRunner
	tableNameResolver func(modelName string) string
}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) SetTranslator(fn func(locale, key string) string) {
	v.translator = fn
}

func (v *Validator) SetUniqueChecker(fn UniqueChecker) {
	v.uniqueChecker = fn
}

func (v *Validator) SetExistsChecker(fn ExistsChecker) {
	v.existsChecker = fn
}

func (v *Validator) SetRelationCounter(fn RelationCounter) {
	v.relationCounter = fn
}

func (v *Validator) SetCustomRunner(fn CustomValidatorRunner) {
	v.customRunner = fn
}

func (v *Validator) SetTableNameResolver(fn func(string) string) {
	v.tableNameResolver = fn
}

func (v *Validator) ValidateCreate(modelDef *parser.ModelDefinition, data map[string]any, locale string) *ValidationErrors {
	return v.validate(context.Background(), modelDef, data, nil, "create", "", locale)
}

func (v *Validator) ValidateCreateWithContext(ctx context.Context, modelDef *parser.ModelDefinition, data map[string]any, locale string) *ValidationErrors {
	return v.validate(ctx, modelDef, data, nil, "create", "", locale)
}

func (v *Validator) ValidateUpdate(modelDef *parser.ModelDefinition, mergedData map[string]any, changes map[string]any, locale string) *ValidationErrors {
	recordID := ""
	if old, ok := mergedData["__old"].(map[string]any); ok {
		if id, ok := old["id"].(string); ok {
			recordID = id
		}
	}
	return v.validate(context.Background(), modelDef, mergedData, changes, "update", recordID, locale)
}

func (v *Validator) ValidateUpdateWithContext(ctx context.Context, modelDef *parser.ModelDefinition, mergedData map[string]any, changes map[string]any, locale string) *ValidationErrors {
	recordID := ""
	if old, ok := mergedData["__old"].(map[string]any); ok {
		if id, ok := old["id"].(string); ok {
			recordID = id
		}
	}
	return v.validate(ctx, modelDef, mergedData, changes, "update", recordID, locale)
}

func (v *Validator) validate(ctx context.Context, modelDef *parser.ModelDefinition, data map[string]any, changes map[string]any, operation string, currentRecordID string, locale string) *ValidationErrors {
	errs := NewValidationErrors()

	for fieldName, fieldDef := range modelDef.Fields {
		if fieldDef.Computed != "" {
			continue
		}

		if operation == "update" && changes != nil {
			if !v.shouldValidateOnUpdate(fieldName, &fieldDef, changes, data) {
				continue
			}
		}

		v.validateField(ctx, fieldName, &fieldDef, modelDef, data, operation, currentRecordID, locale, errs)
	}

	v.validateModelLevel(ctx, modelDef, data, operation, locale, errs)

	return errs
}

func (v *Validator) shouldValidateOnUpdate(fieldName string, fieldDef *parser.FieldDefinition, changes map[string]any, data map[string]any) bool {
	if _, changed := changes[fieldName]; changed {
		return true
	}

	if fieldDef.Validation == nil {
		return false
	}
	val := fieldDef.Validation

	if len(val.RequiredIf) > 0 {
		for depField := range val.RequiredIf {
			if _, changed := changes[depField]; changed {
				return true
			}
		}
	}
	if len(val.RequiredUnless) > 0 {
		for depField := range val.RequiredUnless {
			if _, changed := changes[depField]; changed {
				return true
			}
		}
	}
	if len(val.RequiredWith) > 0 {
		for _, depField := range val.RequiredWith {
			if _, changed := changes[depField]; changed {
				return true
			}
		}
	}
	if len(val.RequiredWithAll) > 0 {
		for _, depField := range val.RequiredWithAll {
			if _, changed := changes[depField]; changed {
				return true
			}
		}
	}
	if len(val.RequiredWithout) > 0 {
		for _, depField := range val.RequiredWithout {
			if _, changed := changes[depField]; changed {
				return true
			}
		}
	}
	if len(val.RequiredWithoutAll) > 0 {
		for _, depField := range val.RequiredWithoutAll {
			if _, changed := changes[depField]; changed {
				return true
			}
		}
	}

	if val.When != nil {
		if whenMap, ok := val.When.(map[string]any); ok {
			for depField := range whenMap {
				if _, changed := changes[depField]; changed {
					return true
				}
			}
		}
	}

	if len(val.ExcludeIf) > 0 {
		for depField := range val.ExcludeIf {
			if _, changed := changes[depField]; changed {
				return true
			}
		}
	}
	if len(val.ExcludeUnless) > 0 {
		for depField := range val.ExcludeUnless {
			if _, changed := changes[depField]; changed {
				return true
			}
		}
	}

	if val.ImmutableAfter != nil {
		if _, changed := changes[fieldName]; changed {
			return true
		}
	}

	return false
}

func (v *Validator) validateField(ctx context.Context, fieldName string, fieldDef *parser.FieldDefinition, modelDef *parser.ModelDefinition, data map[string]any, operation string, currentRecordID string, locale string, errs *ValidationErrors) {
	val := data[fieldName]
	validation := fieldDef.Validation

	hasExplicitValidation := validation != nil
	if !hasExplicitValidation {
		validation = v.autoMapValidation(fieldDef)
	}
	if validation == nil {
		return
	}

	if len(validation.ExcludeIf) > 0 && checkExcludeIf(validation.ExcludeIf, data) {
		return
	}
	if len(validation.ExcludeUnless) > 0 && checkExcludeUnless(validation.ExcludeUnless, data) {
		return
	}

	if !evaluateWhen(validation.When, data) {
		return
	}

	label := fieldDef.Label
	if label == "" {
		label = fieldName
	}

	requiredFailed := false

	if v.isRequiredActive(validation.Required, operation) {
		if isEmpty(val) {
			requiredFailed = true
			errs.Add(fieldName, "required", v.msg(validation, "required", locale, fmt.Sprintf("%s is required", label)), map[string]any{"field": fieldName, "label": label})
		}
	}

	if !requiredFailed {
		if len(validation.RequiredIf) > 0 && checkRequiredIf(validation.RequiredIf, data) && isEmpty(val) {
			requiredFailed = true
			errs.Add(fieldName, "required_if", v.msg(validation, "required_if", locale, fmt.Sprintf("%s is required", label)), nil)
		}
		if len(validation.RequiredUnless) > 0 && checkRequiredUnless(validation.RequiredUnless, data) && isEmpty(val) {
			requiredFailed = true
			errs.Add(fieldName, "required_unless", v.msg(validation, "required_unless", locale, fmt.Sprintf("%s is required", label)), nil)
		}
		if len(validation.RequiredWith) > 0 && checkRequiredWith(validation.RequiredWith, data) && isEmpty(val) {
			requiredFailed = true
			errs.Add(fieldName, "required_with", v.msg(validation, "required_with", locale, fmt.Sprintf("%s is required", label)), nil)
		}
		if len(validation.RequiredWithAll) > 0 && checkRequiredWithAll(validation.RequiredWithAll, data) && isEmpty(val) {
			requiredFailed = true
			errs.Add(fieldName, "required_with_all", v.msg(validation, "required_with_all", locale, fmt.Sprintf("%s is required", label)), nil)
		}
		if len(validation.RequiredWithout) > 0 && checkRequiredWithout(validation.RequiredWithout, data) && isEmpty(val) {
			requiredFailed = true
			errs.Add(fieldName, "required_without", v.msg(validation, "required_without", locale, fmt.Sprintf("%s is required", label)), nil)
		}
		if len(validation.RequiredWithoutAll) > 0 && checkRequiredWithoutAll(validation.RequiredWithoutAll, data) && isEmpty(val) {
			requiredFailed = true
			errs.Add(fieldName, "required_without_all", v.msg(validation, "required_without_all", locale, fmt.Sprintf("%s is required", label)), nil)
		}
	}

	if requiredFailed && isEmpty(val) {
		return
	}

	if isEmpty(val) {
		return
	}

	strVal := toString(val)

	if validation.Immutable && operation == "update" {
		if oldData, ok := data["__old"].(map[string]any); ok {
			if oldVal, exists := oldData[fieldName]; exists && toString(oldVal) != strVal {
				errs.Add(fieldName, "immutable", v.msg(validation, "immutable", locale, fmt.Sprintf("%s cannot be changed", label)), nil)
				return
			}
		}
	}

	if validation.ImmutableAfter != nil && operation == "update" {
		if v.checkImmutableAfter(validation.ImmutableAfter, data, fieldName, strVal, label, locale, validation, errs) {
			return
		}
	}

	if validation.Email && !validateEmail(strVal) {
		errs.Add(fieldName, "email", v.msg(validation, "email", locale, fmt.Sprintf("%s must be a valid email address", label)), nil)
	}
	if validation.URL && !validateURL(strVal) {
		errs.Add(fieldName, "url", v.msg(validation, "url", locale, fmt.Sprintf("%s must be a valid URL", label)), nil)
	}
	if validation.Phone && !validatePhone(strVal) {
		errs.Add(fieldName, "phone", v.msg(validation, "phone", locale, fmt.Sprintf("%s must be a valid phone number", label)), nil)
	}
	if validation.IP && !validateIP(strVal) {
		errs.Add(fieldName, "ip", v.msg(validation, "ip", locale, fmt.Sprintf("%s must be a valid IP address", label)), nil)
	}
	if validation.IPv4 && !validateIPv4(strVal) {
		errs.Add(fieldName, "ipv4", v.msg(validation, "ipv4", locale, fmt.Sprintf("%s must be a valid IPv4 address", label)), nil)
	}
	if validation.IPv6 && !validateIPv6(strVal) {
		errs.Add(fieldName, "ipv6", v.msg(validation, "ipv6", locale, fmt.Sprintf("%s must be a valid IPv6 address", label)), nil)
	}
	if validation.UUID && !validateUUID(strVal) {
		errs.Add(fieldName, "uuid", v.msg(validation, "uuid", locale, fmt.Sprintf("%s must be a valid UUID", label)), nil)
	}
	if validation.JSON && !validateJSON(strVal) {
		errs.Add(fieldName, "json", v.msg(validation, "json", locale, fmt.Sprintf("%s must be valid JSON", label)), nil)
	}

	if validation.Alpha && !validateAlpha(strVal) {
		errs.Add(fieldName, "alpha", v.msg(validation, "alpha", locale, fmt.Sprintf("%s must contain only letters", label)), nil)
	}
	if validation.AlphaNum && !validateAlphaNum(strVal) {
		errs.Add(fieldName, "alpha_num", v.msg(validation, "alpha_num", locale, fmt.Sprintf("%s must contain only letters and numbers", label)), nil)
	}
	if validation.AlphaDash && !validateAlphaDash(strVal) {
		errs.Add(fieldName, "alpha_dash", v.msg(validation, "alpha_dash", locale, fmt.Sprintf("%s must contain only letters, numbers, dashes, and underscores", label)), nil)
	}
	if validation.Numeric && !validateNumeric(strVal) {
		errs.Add(fieldName, "numeric", v.msg(validation, "numeric", locale, fmt.Sprintf("%s must be numeric", label)), nil)
	}
	if validation.Lowercase && strVal != strings.ToLower(strVal) {
		errs.Add(fieldName, "lowercase", v.msg(validation, "lowercase", locale, fmt.Sprintf("%s must be lowercase", label)), nil)
	}
	if validation.Uppercase && strVal != strings.ToUpper(strVal) {
		errs.Add(fieldName, "uppercase", v.msg(validation, "uppercase", locale, fmt.Sprintf("%s must be uppercase", label)), nil)
	}

	if validation.Regex != "" && !validateRegex(strVal, validation.Regex) {
		msg := fmt.Sprintf("%s format is invalid", label)
		if validation.RegexMessage != "" {
			msg = validation.RegexMessage
		}
		errs.Add(fieldName, "regex", v.msg(validation, "regex", locale, msg), map[string]any{"pattern": validation.Regex})
	}

	if validation.StartsWith != nil && !validateStartsWith(strVal, validation.StartsWith) {
		errs.Add(fieldName, "starts_with", v.msg(validation, "starts_with", locale, fmt.Sprintf("%s must start with the correct prefix", label)), nil)
	}
	if validation.EndsWith != nil && !validateEndsWith(strVal, validation.EndsWith) {
		errs.Add(fieldName, "ends_with", v.msg(validation, "ends_with", locale, fmt.Sprintf("%s must end with the correct suffix", label)), nil)
	}
	if validation.Contains != "" && !strings.Contains(strVal, validation.Contains) {
		errs.Add(fieldName, "contains", v.msg(validation, "contains", locale, fmt.Sprintf("%s must contain '%s'", label, validation.Contains)), nil)
	}
	if validation.NotContains != "" && strings.Contains(strVal, validation.NotContains) {
		errs.Add(fieldName, "not_contains", v.msg(validation, "not_contains", locale, fmt.Sprintf("%s must not contain '%s'", label, validation.NotContains)), nil)
	}

	if validation.MinLength != nil {
		if len(strVal) < *validation.MinLength {
			errs.Add(fieldName, "min_length", v.msg(validation, "min_length", locale, fmt.Sprintf("%s must be at least %d characters", label, *validation.MinLength)), map[string]any{"min_length": *validation.MinLength})
		}
	}
	if validation.MaxLength != nil {
		if len(strVal) > *validation.MaxLength {
			errs.Add(fieldName, "max_length", v.msg(validation, "max_length", locale, fmt.Sprintf("%s must not exceed %d characters", label, *validation.MaxLength)), map[string]any{"max_length": *validation.MaxLength})
		}
	}
	if validation.Size != nil {
		if len(strVal) != *validation.Size {
			errs.Add(fieldName, "size", v.msg(validation, "size", locale, fmt.Sprintf("%s must be exactly %d characters", label, *validation.Size)), map[string]any{"size": *validation.Size})
		}
	}
	if len(validation.LengthBetween) == 2 {
		l := len(strVal)
		if l < validation.LengthBetween[0] || l > validation.LengthBetween[1] {
			errs.Add(fieldName, "length_between", v.msg(validation, "length_between", locale, fmt.Sprintf("%s must be between %d and %d characters", label, validation.LengthBetween[0], validation.LengthBetween[1])), nil)
		}
	}

	if numVal, ok := toFloat(val); ok {
		if validation.Min != nil && numVal < *validation.Min {
			errs.Add(fieldName, "min", v.msg(validation, "min", locale, fmt.Sprintf("%s must be at least %v", label, *validation.Min)), map[string]any{"min": *validation.Min})
		}
		if validation.Max != nil && numVal > *validation.Max {
			errs.Add(fieldName, "max", v.msg(validation, "max", locale, fmt.Sprintf("%s must not exceed %v", label, *validation.Max)), map[string]any{"max": *validation.Max})
		}
		if len(validation.Between) == 2 && (numVal < validation.Between[0] || numVal > validation.Between[1]) {
			errs.Add(fieldName, "between", v.msg(validation, "between", locale, fmt.Sprintf("%s must be between %v and %v", label, validation.Between[0], validation.Between[1])), nil)
		}
	}

	if len(validation.In) > 0 && !anyInList(val, validation.In) {
		errs.Add(fieldName, "in", v.msg(validation, "in", locale, fmt.Sprintf("%s must be one of the allowed values", label)), nil)
	}
	if len(validation.NotIn) > 0 && anyInList(val, validation.NotIn) {
		errs.Add(fieldName, "not_in", v.msg(validation, "not_in", locale, fmt.Sprintf("%s contains a forbidden value", label)), nil)
	}

	if validation.Confirmed != "" {
		otherVal := toString(data[validation.Confirmed])
		if strVal != otherVal {
			errs.Add(fieldName, "confirmed", v.msg(validation, "confirmed", locale, fmt.Sprintf("%s confirmation does not match", label)), nil)
		}
	}
	if validation.Different != "" {
		otherVal := toString(data[validation.Different])
		if strVal == otherVal {
			errs.Add(fieldName, "different", v.msg(validation, "different", locale, fmt.Sprintf("%s must be different from %s", label, validation.Different)), nil)
		}
	}

	if validation.Gt != "" {
		if numVal, ok := toFloat(val); ok {
			if otherNum, ok2 := toFloat(data[validation.Gt]); ok2 && numVal <= otherNum {
				errs.Add(fieldName, "gt", v.msg(validation, "gt", locale, fmt.Sprintf("%s must be greater than %s", label, validation.Gt)), nil)
			}
		}
	}
	if validation.Gte != "" {
		if numVal, ok := toFloat(val); ok {
			if otherNum, ok2 := toFloat(data[validation.Gte]); ok2 && numVal < otherNum {
				errs.Add(fieldName, "gte", v.msg(validation, "gte", locale, fmt.Sprintf("%s must be greater than or equal to %s", label, validation.Gte)), nil)
			}
		}
	}
	if validation.Lt != "" {
		if numVal, ok := toFloat(val); ok {
			if otherNum, ok2 := toFloat(data[validation.Lt]); ok2 && numVal >= otherNum {
				errs.Add(fieldName, "lt", v.msg(validation, "lt", locale, fmt.Sprintf("%s must be less than %s", label, validation.Lt)), nil)
			}
		}
	}
	if validation.Lte != "" {
		if numVal, ok := toFloat(val); ok {
			if otherNum, ok2 := toFloat(data[validation.Lte]); ok2 && numVal > otherNum {
				errs.Add(fieldName, "lte", v.msg(validation, "lte", locale, fmt.Sprintf("%s must be less than or equal to %s", label, validation.Lte)), nil)
			}
		}
	}

	if validation.DateBefore != "" {
		if t, ok := parseDate(strVal); ok {
			if ref, ok2 := resolveDateValue(validation.DateBefore, data); ok2 && !t.Before(ref) {
				errs.Add(fieldName, "date_before", v.msg(validation, "date_before", locale, fmt.Sprintf("%s must be before %s", label, validation.DateBefore)), nil)
			}
		}
	}
	if validation.DateAfter != "" {
		if t, ok := parseDate(strVal); ok {
			if ref, ok2 := resolveDateValue(validation.DateAfter, data); ok2 && !t.After(ref) {
				errs.Add(fieldName, "date_after", v.msg(validation, "date_after", locale, fmt.Sprintf("%s must be after %s", label, validation.DateAfter)), nil)
			}
		}
	}
	if validation.DateBeforeOrEqual != "" {
		if t, ok := parseDate(strVal); ok {
			if ref, ok2 := resolveDateValue(validation.DateBeforeOrEqual, data); ok2 && t.After(ref) {
				errs.Add(fieldName, "date_before_or_equal", v.msg(validation, "date_before_or_equal", locale, fmt.Sprintf("%s must be on or before %s", label, validation.DateBeforeOrEqual)), nil)
			}
		}
	}
	if validation.DateAfterOrEqual != "" {
		if t, ok := parseDate(strVal); ok {
			if ref, ok2 := resolveDateValue(validation.DateAfterOrEqual, data); ok2 && t.Before(ref) {
				errs.Add(fieldName, "date_after_or_equal", v.msg(validation, "date_after_or_equal", locale, fmt.Sprintf("%s must be on or after %s", label, validation.DateAfterOrEqual)), nil)
			}
		}
	}

	for _, rule := range validation.Rules {
		if !evaluateWhen(rule.When, data) {
			continue
		}
		if rule.Regex != "" && !validateRegex(strVal, rule.Regex) {
			msg := fmt.Sprintf("%s format is invalid", label)
			if rule.RegexMessage != "" {
				msg = rule.RegexMessage
			}
			errs.Add(fieldName, "regex", msg, map[string]any{"pattern": rule.Regex})
		}
		if rule.MinLength != nil && len(strVal) < *rule.MinLength {
			errs.Add(fieldName, "min_length", fmt.Sprintf("%s must be at least %d characters", label, *rule.MinLength), nil)
		}
		if rule.MaxLength != nil && len(strVal) > *rule.MaxLength {
			errs.Add(fieldName, "max_length", fmt.Sprintf("%s must not exceed %d characters", label, *rule.MaxLength), nil)
		}
		if rule.Min != nil {
			if numVal, ok := toFloat(val); ok && numVal < float64(*rule.Min) {
				errs.Add(fieldName, "min", fmt.Sprintf("%s must be at least %d", label, *rule.Min), nil)
			}
		}
		if rule.Max != nil {
			if numVal, ok := toFloat(val); ok && numVal > float64(*rule.Max) {
				errs.Add(fieldName, "max", fmt.Sprintf("%s must not exceed %d", label, *rule.Max), nil)
			}
		}
	}

	if errs.HasFieldErrors(fieldName) {
		return
	}

	if validation.FileSize != "" || len(validation.FileType) > 0 {
		v.validateFile(fieldName, data, label, locale, validation, errs)
	}

	v.validateUnique(ctx, fieldName, fieldDef, modelDef, val, data, operation, currentRecordID, label, locale, validation, errs)
	v.validateExists(ctx, fieldName, fieldDef, val, label, locale, validation, errs)
	v.validateRelationItems(ctx, fieldName, fieldDef, modelDef, data, operation, label, locale, validation, errs)
	v.validateCustom(ctx, fieldName, val, data, modelDef.ModulePath, label, locale, validation, errs)
}

func (v *Validator) validateFile(fieldName string, data map[string]any, label string, locale string, validation *parser.FieldValidation, errs *ValidationErrors) {
	if validation.FileSize != "" {
		sizeKey := fieldName + "_size"
		if fileSize, ok := data[sizeKey]; ok {
			maxBytes := parseFileSize(validation.FileSize)
			if maxBytes > 0 {
				if actualSize, ok := toFloat(fileSize); ok && int64(actualSize) > maxBytes {
					errs.Add(fieldName, "file_size", v.msg(validation, "file_size", locale, fmt.Sprintf("%s must not exceed %s", label, validation.FileSize)), map[string]any{"max_size": validation.FileSize})
				}
			}
		}
	}

	if len(validation.FileType) > 0 {
		typeKey := fieldName + "_type"
		if fileType, ok := data[typeKey].(string); ok && fileType != "" {
			allowed := false
			for _, t := range validation.FileType {
				if strings.EqualFold(fileType, t) {
					allowed = true
					break
				}
			}
			if !allowed {
				errs.Add(fieldName, "file_type", v.msg(validation, "file_type", locale, fmt.Sprintf("%s must be one of the allowed file types", label)), map[string]any{"allowed_types": validation.FileType})
			}
		}
	}
}

func (v *Validator) validateUnique(ctx context.Context, fieldName string, fieldDef *parser.FieldDefinition, modelDef *parser.ModelDefinition, val any, data map[string]any, operation string, currentRecordID string, label string, locale string, validation *parser.FieldValidation, errs *ValidationErrors) {
	if !validation.UniqueSimple && validation.UniqueConfig == nil {
		return
	}
	if v.uniqueChecker == nil {
		return
	}

	tableName := modelDef.Name
	if v.tableNameResolver != nil {
		tableName = v.tableNameResolver(modelDef.Name)
	}

	excludeID := ""
	if operation == "update" {
		excludeID = currentRecordID
	}

	isSoftDelete := modelDef.IsSoftDeletes()

	exists, err := v.uniqueChecker(ctx, tableName, fieldName, val, excludeID, validation.UniqueConfig, isSoftDelete, data)
	if err != nil {
		log.Printf("[VALIDATION] unique check error for %s.%s: %v", modelDef.Name, fieldName, err)
		return
	}
	if exists {
		errs.Add(fieldName, "unique", v.msg(validation, "unique", locale, fmt.Sprintf("%s has already been taken", label)), nil)
	}
}

func (v *Validator) validateExists(ctx context.Context, fieldName string, fieldDef *parser.FieldDefinition, val any, label string, locale string, validation *parser.FieldValidation, errs *ValidationErrors) {
	if !validation.Exists && len(validation.ExistsWhere) == 0 {
		return
	}
	if v.existsChecker == nil {
		return
	}
	if fieldDef.Model == "" {
		return
	}

	tableName := fieldDef.Model
	if v.tableNameResolver != nil {
		tableName = v.tableNameResolver(fieldDef.Model)
	}

	exists, err := v.existsChecker(ctx, tableName, val, validation.ExistsWhere)
	if err != nil {
		log.Printf("[VALIDATION] exists check error for %s: %v", fieldName, err)
		return
	}
	if !exists {
		errs.Add(fieldName, "exists", v.msg(validation, "exists", locale, fmt.Sprintf("%s references a record that does not exist", label)), nil)
	}
}

func (v *Validator) validateRelationItems(ctx context.Context, fieldName string, fieldDef *parser.FieldDefinition, modelDef *parser.ModelDefinition, data map[string]any, operation string, label string, locale string, validation *parser.FieldValidation, errs *ValidationErrors) {
	if validation.MinItems == nil && validation.MaxItems == nil {
		return
	}
	if v.relationCounter == nil {
		return
	}
	if fieldDef.Type != parser.FieldOne2Many && fieldDef.Type != parser.FieldMany2Many {
		return
	}

	relTable := fieldDef.Model
	if v.tableNameResolver != nil {
		relTable = v.tableNameResolver(fieldDef.Model)
	}

	pkCol := "id"
	if modelDef.PrimaryKey != nil && modelDef.PrimaryKey.Field != "" {
		pkCol = modelDef.PrimaryKey.Field
	}
	parentID := data[pkCol]
	if parentID == nil {
		return
	}

	foreignKey := fieldDef.Inverse
	if foreignKey == "" {
		foreignKey = modelDef.Name + "_id"
	}

	count, err := v.relationCounter(ctx, relTable, foreignKey, parentID)
	if err != nil {
		log.Printf("[VALIDATION] relation count error for %s: %v", fieldName, err)
		return
	}

	if validation.MinItems != nil {
		minItems := resolveIntValue(validation.MinItems)
		if minItems > 0 && count < int64(minItems) {
			errs.Add(fieldName, "min_items", v.msg(validation, "min_items", locale, fmt.Sprintf("%s must have at least %d items", label, minItems)), map[string]any{"min_items": minItems})
		}
	}
	if validation.MaxItems != nil {
		maxItems := resolveIntValue(validation.MaxItems)
		if maxItems > 0 && count > int64(maxItems) {
			errs.Add(fieldName, "max_items", v.msg(validation, "max_items", locale, fmt.Sprintf("%s must not exceed %d items", label, maxItems)), map[string]any{"max_items": maxItems})
		}
	}
}

func resolveIntValue(val any) int {
	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case map[string]any:
		if raw, ok := v["value"]; ok {
			return resolveIntValue(raw)
		}
	}
	return 0
}

func (v *Validator) validateCustom(ctx context.Context, fieldName string, val any, data map[string]any, modulePath string, label string, locale string, validation *parser.FieldValidation, errs *ValidationErrors) {
	if len(validation.Custom) == 0 {
		return
	}
	if v.customRunner == nil {
		return
	}

	for _, cv := range validation.Custom {
		if err := v.customRunner(ctx, cv, fieldName, val, data, modulePath); err != nil {
			msg := cv.Message
			if msg == "" {
				msg = err.Error()
			}
			errs.Add(fieldName, "custom", v.msg(validation, "custom", locale, msg), nil)
		}
	}
}

func (v *Validator) checkImmutableAfter(immutableAfter any, data map[string]any, fieldName string, strVal string, label string, locale string, validation *parser.FieldValidation, errs *ValidationErrors) bool {
	oldData, ok := data["__old"].(map[string]any)
	if !ok {
		return false
	}
	oldVal, exists := oldData[fieldName]
	if !exists || toString(oldVal) == strVal {
		return false
	}

	switch ia := immutableAfter.(type) {
	case map[string]any:
		for condField, condValues := range ia {
			oldFieldVal := oldData[condField]
			switch cv := condValues.(type) {
			case []any:
				if anyInList(oldFieldVal, cv) {
					errs.Add(fieldName, "immutable_after", v.msg(validation, "immutable_after", locale, fmt.Sprintf("%s cannot be changed in current state", label)), nil)
					return true
				}
			default:
				if anyEquals(oldFieldVal, cv) {
					errs.Add(fieldName, "immutable_after", v.msg(validation, "immutable_after", locale, fmt.Sprintf("%s cannot be changed in current state", label)), nil)
					return true
				}
			}
		}
	}
	return false
}

func (v *Validator) autoMapValidation(fieldDef *parser.FieldDefinition) *parser.FieldValidation {
	var val parser.FieldValidation
	hasRule := false

	if fieldDef.Required {
		val.Required = true
		hasRule = true
	}

	if fieldDef.Unique {
		val.UniqueSimple = true
		hasRule = true
	}

	if fieldDef.Type == parser.FieldEmail {
		val.Email = true
		hasRule = true
	}

	switch fieldDef.Type {
	case parser.FieldUUID:
		val.UUID = true
		hasRule = true
	case parser.FieldIP:
		val.IP = true
		hasRule = true
	case parser.FieldIPv6:
		val.IPv6 = true
		hasRule = true
	case parser.FieldColor:
		val.Regex = `^#[0-9a-fA-F]{6}$`
		val.RegexMessage = "invalid color format (expected #RRGGBB)"
		hasRule = true
	case parser.FieldYear:
		minY := float64(1900)
		maxY := float64(2300)
		val.Min = &minY
		val.Max = &maxY
		hasRule = true
	}

	isStringType := false
	switch fieldDef.Type {
	case parser.FieldString, parser.FieldText, parser.FieldSmallText, parser.FieldEmail,
		parser.FieldPassword, parser.FieldCode, parser.FieldMarkdown, parser.FieldHTML,
		parser.FieldRichText:
		isStringType = true
	}

	if fieldDef.Max > 0 {
		if isStringType {
			maxLen := fieldDef.Max
			val.MaxLength = &maxLen
		} else {
			maxF := float64(fieldDef.Max)
			val.Max = &maxF
		}
		hasRule = true
	}
	if fieldDef.Min > 0 {
		if isStringType {
			minLen := fieldDef.Min
			val.MinLength = &minLen
		} else {
			minF := float64(fieldDef.Min)
			val.Min = &minF
		}
		hasRule = true
	}

	if !hasRule {
		return nil
	}
	return &val
}

func (v *Validator) isRequiredActive(required any, operation string) bool {
	if required == nil {
		return false
	}
	switch r := required.(type) {
	case bool:
		return r
	case map[string]any:
		on, _ := r["on"].(string)
		if on == "" || on == "always" {
			return true
		}
		return on == operation
	}
	return false
}

func (v *Validator) validateModelLevel(ctx context.Context, modelDef *parser.ModelDefinition, data map[string]any, operation string, locale string, errs *ValidationErrors) {
	for _, mv := range modelDef.Validators {
		on := mv.GetOn()
		if on != "always" && on != operation {
			continue
		}

		if mv.Condition != "" && !evaluateSimpleExpression(mv.Condition, data) {
			continue
		}

		if mv.Expression != "" {
			if !evaluateSimpleExpression(mv.Expression, data) {
				msg := mv.Message
				if msg == "" {
					msg = fmt.Sprintf("Validation failed: %s", mv.Name)
				}
				msg = v.translate(locale, msg)
				errs.AddModel(mv.Name, msg)
			}
			continue
		}

		if (mv.Process != "" || mv.Script != nil) && v.customRunner != nil {
			cv := parser.CustomValidator{
				Process: mv.Process,
				Script:  mv.Script,
				Message: mv.Message,
			}
			if err := v.customRunner(ctx, cv, "_model", nil, data, modelDef.ModulePath); err != nil {
				msg := mv.Message
				if msg == "" {
					msg = err.Error()
				}
				msg = v.translate(locale, msg)
				errs.AddModel(mv.Name, msg)
			}
		}
	}
}

func (v *Validator) msg(validation *parser.FieldValidation, rule string, locale string, defaultMsg string) string {
	if validation != nil && validation.Messages != nil {
		if custom, ok := validation.Messages[rule]; ok {
			return v.translate(locale, custom)
		}
	}
	return v.translate(locale, defaultMsg)
}

func (v *Validator) translate(locale, msg string) string {
	if v.translator == nil || locale == "" {
		return msg
	}
	if strings.Contains(msg, ".") && !strings.Contains(msg, " ") {
		translated := v.translator(locale, msg)
		if translated != msg {
			return translated
		}
	}
	return msg
}

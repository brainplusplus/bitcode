package validation

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

type Sanitizer struct{}

func NewSanitizer() *Sanitizer {
	return &Sanitizer{}
}

func (s *Sanitizer) SanitizeRecord(modelDef *parser.ModelDefinition, data map[string]any) {
	modelSanitizers := s.getModelSanitizers(modelDef)

	for fieldName, fieldDef := range modelDef.Fields {
		val, exists := data[fieldName]
		if !exists || val == nil {
			continue
		}
		strVal, ok := val.(string)
		if !ok {
			continue
		}

		sanitizers := fieldDef.Sanitize
		if len(sanitizers) == 0 {
			sanitizers = s.resolveFieldSanitizers(fieldDef, modelSanitizers)
		}
		if len(sanitizers) == 0 {
			continue
		}

		result := strVal
		for _, sanitizer := range sanitizers {
			result = s.applySanitizer(sanitizer, result)
		}
		if result != strVal {
			data[fieldName] = result
		}
	}
}

func (s *Sanitizer) SanitizeChangedFields(modelDef *parser.ModelDefinition, data map[string]any, changes map[string]any) {
	modelSanitizers := s.getModelSanitizers(modelDef)

	for fieldName := range changes {
		fieldDef, ok := modelDef.Fields[fieldName]
		if !ok {
			continue
		}
		val, exists := data[fieldName]
		if !exists || val == nil {
			continue
		}
		strVal, ok := val.(string)
		if !ok {
			continue
		}

		sanitizers := fieldDef.Sanitize
		if len(sanitizers) == 0 {
			sanitizers = s.resolveFieldSanitizers(fieldDef, modelSanitizers)
		}
		if len(sanitizers) == 0 {
			continue
		}

		result := strVal
		for _, sanitizer := range sanitizers {
			result = s.applySanitizer(sanitizer, result)
		}
		if result != strVal {
			data[fieldName] = result
		}
	}
}

func (s *Sanitizer) getModelSanitizers(modelDef *parser.ModelDefinition) []string {
	if modelDef.Sanitize != nil && len(modelDef.Sanitize.AllStrings) > 0 {
		return modelDef.Sanitize.AllStrings
	}
	return nil
}

func (s *Sanitizer) resolveFieldSanitizers(fieldDef parser.FieldDefinition, modelSanitizers []string) []string {
	if len(modelSanitizers) == 0 {
		return nil
	}
	if fieldDef.Type == parser.FieldPassword {
		return nil
	}
	switch fieldDef.Type {
	case parser.FieldString, parser.FieldText, parser.FieldSmallText, parser.FieldEmail,
		parser.FieldCode, parser.FieldMarkdown, parser.FieldHTML, parser.FieldRichText:
		return modelSanitizers
	}
	return nil
}

var slugifyRegex = regexp.MustCompile(`[^a-z0-9]+`)
var multiSpaceRegex = regexp.MustCompile(`\s+`)

func (s *Sanitizer) applySanitizer(name string, val string) string {
	if strings.HasPrefix(name, "truncate:") {
		nStr := strings.TrimPrefix(name, "truncate:")
		n, err := strconv.Atoi(nStr)
		if err == nil && len(val) > n {
			return val[:n]
		}
		return val
	}

	switch name {
	case "trim":
		return strings.TrimSpace(val)
	case "trim_left":
		return strings.TrimLeftFunc(val, unicode.IsSpace)
	case "trim_right":
		return strings.TrimRightFunc(val, unicode.IsSpace)
	case "lowercase":
		return strings.ToLower(val)
	case "uppercase":
		return strings.ToUpper(val)
	case "title_case":
		return toTitleCase(val)
	case "strip_tags":
		return stripTags(val)
	case "strip_whitespace":
		return multiSpaceRegex.ReplaceAllString(strings.TrimSpace(val), " ")
	case "slugify":
		slug := strings.ToLower(val)
		slug = slugifyRegex.ReplaceAllString(slug, "-")
		slug = strings.Trim(slug, "-")
		return slug
	case "normalize_email":
		return strings.TrimSpace(strings.ToLower(val))
	case "normalize_phone":
		return normalizePhone(val)
	case "escape_html":
		return escapeHTML(val)
	}
	return val
}

func toTitleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			runes := []rune(w)
			runes[0] = unicode.ToUpper(runes[0])
			for j := 1; j < len(runes); j++ {
				runes[j] = unicode.ToLower(runes[j])
			}
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

var tagRegex = regexp.MustCompile(`<[^>]*>`)

func stripTags(s string) string {
	return tagRegex.ReplaceAllString(s, "")
}

func normalizePhone(val string) string {
	var digits strings.Builder
	hasPlus := strings.HasPrefix(strings.TrimSpace(val), "+")
	for _, r := range val {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	d := digits.String()
	if hasPlus && len(d) > 0 {
		return "+" + d
	}
	return d
}

func escapeHTML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}

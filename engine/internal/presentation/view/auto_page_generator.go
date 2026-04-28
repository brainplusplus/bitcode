package view

import (
	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

var excludeFromListFields = map[parser.FieldType]bool{
	parser.FieldText:       true,
	parser.FieldRichText:   true,
	parser.FieldMarkdown:   true,
	parser.FieldHTML:       true,
	parser.FieldCode:       true,
	parser.FieldJSON:       true,
	parser.FieldJSONObject: true,
	parser.FieldJSONArray:  true,
	parser.FieldOne2Many:   true,
	parser.FieldVector:     true,
	parser.FieldBinary:     true,
}

func GenerateListView(model *parser.ModelDefinition, moduleName string) *parser.ViewDefinition {
	var fields []string
	for name, field := range model.Fields {
		if excludeFromListFields[field.Type] {
			continue
		}
		if field.Computed != "" {
			continue
		}
		if field.Hidden {
			continue
		}
		fields = append(fields, name)
	}

	if len(fields) == 0 {
		for name := range model.Fields {
			fields = append(fields, name)
			if len(fields) >= 5 {
				break
			}
		}
	}

	var filters []string
	for name, field := range model.Fields {
		if field.Type == parser.FieldSelection || field.Type == parser.FieldMany2One || field.Type == parser.FieldBoolean {
			filters = append(filters, name)
		}
	}

	sortField := model.TitleField
	if sortField == "" {
		sortField = "created_at"
	}

	title := model.Label
	if title == "" {
		title = model.Name
	}

	return &parser.ViewDefinition{
		Name:    model.Name + "_list",
		Type:    parser.ViewList,
		Model:   model.Name,
		Title:   title,
		Fields:  fields,
		Filters: filters,
		Sort:    &parser.SortDefinition{Field: sortField, Order: "asc"},
	}
}

func GenerateFormView(model *parser.ModelDefinition, moduleName string) *parser.ViewDefinition {
	var layout []parser.LayoutItem

	var regularFields []parser.LayoutRow
	var tabFields []parser.TabDefinition

	for name, field := range model.Fields {
		if field.Hidden {
			continue
		}
		if field.Type == parser.FieldVector || field.Type == parser.FieldBinary {
			continue
		}
		if field.Type == parser.FieldOne2Many {
			label := field.Label
			if label == "" {
				label = name
			}
			tabFields = append(tabFields, parser.TabDefinition{
				Label:  label,
				Fields: []string{name},
			})
			continue
		}
		width := 6
		if field.Type == parser.FieldText || field.Type == parser.FieldRichText ||
			field.Type == parser.FieldMarkdown || field.Type == parser.FieldHTML ||
			field.Type == parser.FieldCode {
			width = 12
		}
		regularFields = append(regularFields, parser.LayoutRow{
			Field: name,
			Width: width,
		})
	}

	for i := 0; i < len(regularFields); i += 2 {
		var row []parser.LayoutRow
		row = append(row, regularFields[i])
		if i+1 < len(regularFields) {
			row = append(row, regularFields[i+1])
		}
		layout = append(layout, parser.LayoutItem{Row: row})
	}

	if len(tabFields) > 0 {
		layout = append(layout, parser.LayoutItem{Tabs: tabFields})
	}

	title := model.Label
	if title == "" {
		title = model.Name
	}

	return &parser.ViewDefinition{
		Name:   model.Name + "_form",
		Type:   parser.ViewForm,
		Model:  model.Name,
		Title:  title,
		Layout: layout,
	}
}

func ShouldAutoGeneratePages(model *parser.ModelDefinition) bool {
	if model.API == nil {
		return false
	}
	return model.API.IsAutoPages()
}

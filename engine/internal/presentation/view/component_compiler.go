package view

import (
	"fmt"
	"html"
	"strings"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

type EmbeddedViewRenderer func(viewName string) string

type ComponentCompiler struct {
	modelLookup          ModelLookup
	embeddedViewRenderer EmbeddedViewRenderer
}

func NewComponentCompiler() *ComponentCompiler {
	return &ComponentCompiler{}
}

func fieldTypeToTag(fieldType parser.FieldType, widget string) string {
	if widget != "" {
		return fmt.Sprintf("bc-widget-%s", widget)
	}
	switch fieldType {
	case parser.FieldString:
		return "bc-field-string"
	case parser.FieldSmallText:
		return "bc-field-smalltext"
	case parser.FieldText:
		return "bc-field-text"
	case parser.FieldRichText:
		return "bc-field-richtext"
	case parser.FieldMarkdown:
		return "bc-field-markdown"
	case parser.FieldHTML:
		return "bc-field-html"
	case parser.FieldCode:
		return "bc-field-code"
	case parser.FieldPassword:
		return "bc-field-password"
	case parser.FieldInteger:
		return "bc-field-integer"
	case parser.FieldFloat:
		return "bc-field-float"
	case parser.FieldDecimal:
		return "bc-field-decimal"
	case parser.FieldCurrency:
		return "bc-field-currency"
	case parser.FieldPercent:
		return "bc-field-percent"
	case parser.FieldBoolean:
		return "bc-field-checkbox"
	case parser.FieldToggle:
		return "bc-field-toggle"
	case parser.FieldSelection:
		return "bc-field-select"
	case parser.FieldRadio:
		return "bc-field-radio"
	case parser.FieldMany2One:
		return "bc-field-link"
	case parser.FieldDynamicLink:
		return "bc-field-dynlink"
	case parser.FieldMany2Many:
		return "bc-field-tags"
	case parser.FieldDate:
		return "bc-field-date"
	case parser.FieldTime:
		return "bc-field-time"
	case parser.FieldDatetime:
		return "bc-field-datetime"
	case parser.FieldDuration:
		return "bc-field-duration"
	case parser.FieldFile:
		return "bc-field-file"
	case parser.FieldImage:
		return "bc-field-image"
	case parser.FieldSignature:
		return "bc-field-signature"
	case parser.FieldBarcode:
		return "bc-field-barcode"
	case parser.FieldColor:
		return "bc-field-color"
	case parser.FieldGeolocation:
		return "bc-field-geo"
	case parser.FieldRating:
		return "bc-field-rating"
	case parser.FieldJSON:
		return "bc-field-json"
	case parser.FieldEmail:
		return "bc-field-string"
	default:
		return "bc-field-string"
	}
}

func (c *ComponentCompiler) getFieldDef(modelName string, fieldName string) *parser.FieldDefinition {
	if c.modelLookup == nil {
		return nil
	}
	model, err := c.modelLookup.Get(modelName)
	if err != nil {
		return nil
	}
	if fd, ok := model.Fields[fieldName]; ok {
		return &fd
	}
	return nil
}

func (c *ComponentCompiler) CompileForm(viewDef *parser.ViewDefinition, record map[string]any) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<bc-sheet><bc-view-form model="%s" view-title="%s">`, esc(viewDef.Model), esc(viewDef.Title)))

	for _, item := range viewDef.Layout {
		b.WriteString(c.compileLayoutItem(item, record, viewDef.Model))
	}

	if len(viewDef.Actions) > 0 {
		b.WriteString(c.compileActions(viewDef.Actions))
	}

	b.WriteString(`</bc-view-form></bc-sheet>`)
	return b.String()
}

func (c *ComponentCompiler) CompileFormFull(viewDef *parser.ViewDefinition, record map[string]any, meta map[string]any) string {
	recordID, _ := meta["recordId"].(string)
	listUrl, _ := meta["listUrl"].(string)
	formAction, _ := meta["formAction"].(string)
	model := viewDef.Model

	var b strings.Builder

	b.WriteString(`<div class="bc-card">`)
	b.WriteString(`<div class="bc-card-header"><div style="display:flex;align-items:center;gap:0.75rem;">`)
	if listUrl != "" {
		b.WriteString(fmt.Sprintf(`<a href="%s" class="bc-btn-sm" title="Back to list">&larr;</a>`, esc(listUrl)))
	}
	if recordID != "" {
		b.WriteString(fmt.Sprintf(`<h2>Edit %s</h2>`, esc(viewDef.Title)))
	} else {
		b.WriteString(fmt.Sprintf(`<h2>New %s</h2>`, esc(viewDef.Title)))
	}
	b.WriteString(`</div></div>`)
	b.WriteString(`<div class="bc-card-body">`)
	if formAction == "" {
		formAction = "#"
	}
	b.WriteString(fmt.Sprintf(`<form method="POST" action="%s">`, esc(formAction)))
	if recordID != "" {
		b.WriteString(fmt.Sprintf(`<input type="hidden" name="_id" value="%s">`, esc(recordID)))
		b.WriteString(`<input type="hidden" name="_method" value="PUT">`)
	}
	for _, item := range viewDef.Layout {
		if item.Header != nil {
			b.WriteString(c.compileFormHeader(item.Header, record, model))
			continue
		}
		if len(item.ButtonBox) > 0 {
			b.WriteString(c.compileButtonBox(item.ButtonBox))
			continue
		}
		if item.Section != nil {
			b.WriteString(c.compileFormSection(item, record, model))
			continue
		}
		if len(item.Row) > 0 {
			b.WriteString(c.compileFormRow(item.Row, record, model))
			continue
		}
		if len(item.Tabs) > 0 {
			b.WriteString(c.compileFormTabs(item.Tabs, record, model))
			continue
		}
		if item.Chatter {
			b.WriteString(`<div style="margin-top:1.5rem;border-top:1px solid var(--border,#e5e7eb);padding-top:1rem;"><bc-chatter></bc-chatter></div>`)
			continue
		}
		if item.Separator != nil {
			label := ""
			if item.Separator.Label != "" {
				label = item.Separator.Label
			}
			b.WriteString(fmt.Sprintf(`<hr style="margin:1rem 0;border:none;border-top:1px solid var(--border,#e5e7eb);"><span style="font-size:0.8rem;color:var(--text-muted,#999);">%s</span>`, esc(label)))
			continue
		}
	}
	b.WriteString(`<div style="margin-top:1.5rem;display:flex;gap:0.5rem;">`)
	if recordID != "" {
		b.WriteString(`<button type="submit" class="bc-btn bc-btn-primary">Save Changes</button>`)
	} else {
		b.WriteString(`<button type="submit" class="bc-btn bc-btn-primary">Create</button>`)
	}
	if listUrl != "" {
		b.WriteString(fmt.Sprintf(`<a href="%s" class="bc-btn bc-btn-secondary">Cancel</a>`, esc(listUrl)))
	}
	if recordID != "" {
		b.WriteString(fmt.Sprintf(`<button type="button" class="bc-btn bc-btn-danger" style="margin-left:auto;" onclick="if(confirm('Delete this record?')){fetch('/api/%ss/%s',{method:'DELETE'}).then(function(r){if(r.ok)window.location='%s'})}">Delete</button>`,
			esc(model), esc(recordID), esc(listUrl)))
	}
	b.WriteString(`</div>`)

	b.WriteString(`</form></div></div>`)
	if recordID != "" {
		b.WriteString(fmt.Sprintf(`<script>
function executeAction(process){
  fetch('/api/%ss/%s/'+process,{method:'POST',headers:{'Content-Type':'application/json'}})
    .then(function(r){if(r.ok)location.reload();else r.json().then(function(d){alert(d.error||'Action failed')})})
    .catch(function(){alert('Action failed')});
}
</script>`, esc(model), esc(recordID)))
	}

	return b.String()
}

func (c *ComponentCompiler) compileFormHeader(h *parser.HeaderDefinition, record map[string]any, model string) string {
	var b strings.Builder
	b.WriteString(`<div style="display:flex;align-items:center;justify-content:space-between;padding:0.75rem 0;margin-bottom:1rem;border-bottom:1px solid var(--border,#e5e7eb);">`)
	b.WriteString(`<div style="display:flex;gap:0.4rem;">`)
	for _, btn := range h.Buttons {
		variant := "bc-btn-secondary"
		if btn.Variant == "primary" {
			variant = "bc-btn-primary"
		} else if btn.Variant == "danger" {
			variant = "bc-btn-danger"
		}
		b.WriteString(fmt.Sprintf(`<button type="button" class="bc-btn %s" onclick="executeAction('%s')">%s</button>`,
			variant, esc(btn.Process), esc(btn.Label)))
	}
	b.WriteString(`</div>`)
	if h.StatusField != "" {
		statusVal := ""
		if record != nil {
			if v, ok := record[h.StatusField]; ok {
				statusVal = fmt.Sprintf("%v", v)
			}
		}
		fieldDef := c.getFieldDef(model, h.StatusField)
		if fieldDef != nil && len(fieldDef.Options) > 0 {
			b.WriteString(`<div style="display:flex;gap:0;">`)
			for _, opt := range fieldDef.Options {
				cls := "background:var(--body-bg,#f8fafc);color:var(--text-muted,#999);"
				if opt == statusVal {
					cls = "background:var(--primary,#6366f1);color:#fff;font-weight:600;"
				}
				b.WriteString(fmt.Sprintf(`<span style="padding:0.2rem 0.65rem;font-size:0.72rem;border:1px solid var(--border,#e5e7eb);%s text-transform:capitalize;">%s</span>`, cls, esc(opt)))
			}
			b.WriteString(`</div>`)
		}
	}

	b.WriteString(`</div>`)
	return b.String()
}

func (c *ComponentCompiler) compileFormSection(item parser.LayoutItem, record map[string]any, model string) string {
	var b strings.Builder
	title := ""
	if item.Section.Title != "" {
		title = item.Section.Title
	}
	b.WriteString(fmt.Sprintf(`<fieldset style="border:1px solid var(--border,#e5e7eb);border-radius:8px;padding:1rem;margin-bottom:1rem;">`))
	if title != "" {
		b.WriteString(fmt.Sprintf(`<legend style="font-weight:600;font-size:0.85rem;padding:0 0.5rem;color:var(--text-secondary,#475569);">%s</legend>`, esc(title)))
	}
	for _, row := range item.Rows {
		if len(row.Row) > 0 {
			b.WriteString(c.compileFormRow(row.Row, record, model))
		}
	}
	b.WriteString(`</fieldset>`)
	return b.String()
}

func (c *ComponentCompiler) shouldHideField(modelName string, fieldName string, isEdit bool) bool {
	if c.modelLookup == nil {
		return false
	}
	model, err := c.modelLookup.Get(modelName)
	if err != nil || model.PrimaryKey == nil {
		return false
	}

	pk := model.PrimaryKey
	switch pk.Strategy {
	case parser.PKAutoIncrement:
		return fieldName == "id"
	case parser.PKUUID:
		return fieldName == "id"
	case parser.PKNamingSeries:
		return fieldName == pk.Field
	}

	fd, ok := model.Fields[fieldName]
	if ok && fd.AutoFormat != nil && !isEdit {
		return true
	}

	return false
}

func (c *ComponentCompiler) shouldReadonlyField(modelName string, fieldName string, isEdit bool) bool {
	if !isEdit || c.modelLookup == nil {
		return false
	}
	model, err := c.modelLookup.Get(modelName)
	if err != nil || model.PrimaryKey == nil {
		return false
	}

	pk := model.PrimaryKey
	switch pk.Strategy {
	case parser.PKNaturalKey:
		return fieldName == pk.Field
	case parser.PKManual:
		return fieldName == pk.Field
	case parser.PKComposite:
		if !pk.IsSurrogate() {
			for _, f := range pk.Fields {
				if f == fieldName {
					return true
				}
			}
		}
	}

	fd, ok := model.Fields[fieldName]
	if ok && fd.AutoFormat != nil {
		return true
	}

	return false
}

func (c *ComponentCompiler) compileFormRow(fields []parser.LayoutRow, record map[string]any, model string) string {
	var b strings.Builder
	b.WriteString(`<div class="bc-row">`)
	isEdit := record != nil && len(record) > 0
	for _, f := range fields {
		if c.shouldHideField(model, f.Field, isEdit) {
			continue
		}

		width := f.Width
		if width == 0 {
			width = 6
		}
		b.WriteString(fmt.Sprintf(`<div class="bc-col-%d">`, width))
		b.WriteString(`<div class="bc-form-group">`)

		val := ""
		if record != nil {
			if v, ok := record[f.Field]; ok && v != nil {
				val = fmt.Sprintf("%v", v)
			}
		}

		fieldDef := c.getFieldDef(model, f.Field)
		label := f.Field
		if fieldDef != nil && fieldDef.Label != "" {
			label = fieldDef.Label
		}

		b.WriteString(fmt.Sprintf(`<label class="bc-label">%s</label>`, esc(label)))

		readonly := f.Readonly
		if fieldDef != nil && fieldDef.Computed != "" {
			readonly = true
		}
		if c.shouldReadonlyField(model, f.Field, isEdit) {
			readonly = true
		}

		if fieldDef == nil {
			b.WriteString(c.renderInputField(f.Field, val, "text", readonly))
		} else {
			widget := f.Widget
			if widget == "" {
				widget = fieldDef.Widget
			}
			if widget == "email" {
				b.WriteString(fmt.Sprintf(`<div style="display:flex;align-items:center;gap:0.5rem;"><input class="bc-input" type="email" name="%s" value="%s" placeholder="email@example.com" %s><a href="mailto:%s" style="font-size:0.8rem;">✉</a></div>`,
					esc(f.Field), esc(val), readonlyAttr(readonly), esc(val)))
			} else if widget == "phone" {
				b.WriteString(fmt.Sprintf(`<div style="display:flex;align-items:center;gap:0.5rem;"><input class="bc-input" type="tel" name="%s" value="%s" placeholder="+62..." %s><a href="tel:%s" style="font-size:0.8rem;">📞</a></div>`,
					esc(f.Field), esc(val), readonlyAttr(readonly), esc(val)))
			} else {
				switch fieldDef.Type {
				case parser.FieldSelection, parser.FieldRadio:
					b.WriteString(c.renderSelectField(f.Field, val, fieldDef.Options, readonly))
				case parser.FieldBoolean:
					checked := val == "true" || val == "1"
					b.WriteString(fmt.Sprintf(`<input type="checkbox" name="%s" %s %s style="width:auto;">`,
						esc(f.Field), checkedAttr(checked), readonlyAttr(readonly)))
				case parser.FieldToggle:
					checked := val == "true" || val == "1"
					b.WriteString(fmt.Sprintf(`<label style="display:flex;align-items:center;gap:0.5rem;cursor:pointer;"><input type="checkbox" name="%s" %s %s style="width:auto;accent-color:var(--primary);"><span style="font-size:0.82rem;">%s</span></label>`,
						esc(f.Field), checkedAttr(checked), readonlyAttr(readonly), esc(label)))
				case parser.FieldText:
					b.WriteString(fmt.Sprintf(`<textarea class="bc-input" name="%s" rows="3" %s>%s</textarea>`,
						esc(f.Field), readonlyAttr(readonly), esc(val)))
				case parser.FieldRichText:
					b.WriteString(fmt.Sprintf(`<textarea class="bc-input" name="%s" rows="5" placeholder="Rich text..." %s>%s</textarea>`,
						esc(f.Field), readonlyAttr(readonly), esc(val)))
				case parser.FieldDate:
					b.WriteString(c.renderInputField(f.Field, val, "date", readonly))
				case parser.FieldTime:
					b.WriteString(c.renderInputField(f.Field, val, "time", readonly))
				case parser.FieldDatetime:
					b.WriteString(c.renderInputField(f.Field, val, "datetime-local", readonly))
				case parser.FieldInteger:
					b.WriteString(c.renderInputField(f.Field, val, "number", readonly))
				case parser.FieldFloat, parser.FieldDecimal:
					b.WriteString(fmt.Sprintf(`<input class="bc-input" type="number" step="0.01" name="%s" value="%s" %s>`,
						esc(f.Field), esc(val), readonlyAttr(readonly)))
				case parser.FieldCurrency:
					cur := fieldDef.CurrencyCode
					if cur == "" {
						cur = "IDR"
					}
					b.WriteString(fmt.Sprintf(`<div style="display:flex;align-items:center;gap:0.5rem;"><span style="font-size:0.8rem;font-weight:600;color:var(--text-muted);">%s</span><input class="bc-input" type="number" step="1" name="%s" value="%s" %s></div>`,
						esc(cur), esc(f.Field), esc(val), readonlyAttr(readonly)))
				case parser.FieldPercent:
					b.WriteString(fmt.Sprintf(`<div style="display:flex;align-items:center;gap:0.5rem;"><input class="bc-input" type="number" min="0" max="100" name="%s" value="%s" %s><span style="font-size:0.85rem;color:var(--text-muted);">%%</span></div>`,
						esc(f.Field), esc(val), readonlyAttr(readonly)))
				case parser.FieldEmail:
					b.WriteString(c.renderInputField(f.Field, val, "email", readonly))
				case parser.FieldPassword:
					b.WriteString(c.renderInputField(f.Field, val, "password", readonly))
				case parser.FieldMany2One:
					b.WriteString(fmt.Sprintf(`<input class="bc-input" type="text" name="%s" value="%s" placeholder="Search %s..." %s>`,
						esc(f.Field), esc(val), esc(fieldDef.Model), readonlyAttr(readonly)))
				case parser.FieldMany2Many:
					b.WriteString(fmt.Sprintf(`<input class="bc-input" type="text" name="%s" value="%s" placeholder="Add tags..." %s>`,
						esc(f.Field), esc(val), readonlyAttr(readonly)))
				case parser.FieldFile, parser.FieldImage:
					b.WriteString(fmt.Sprintf(`<input type="file" name="%s" class="bc-input" %s %s>`,
						esc(f.Field), readonlyAttr(readonly),
						func() string {
							if fieldDef.Accept != "" {
								return fmt.Sprintf(`accept="%s"`, esc(fieldDef.Accept))
							}
							return ""
						}()))
				case parser.FieldColor:
					if val == "" {
						val = "#000000"
					}
					b.WriteString(fmt.Sprintf(`<div style="display:flex;align-items:center;gap:0.5rem;"><input type="color" name="%s" value="%s" %s style="width:3rem;height:2rem;border:1px solid var(--border);border-radius:4px;cursor:pointer;"><span style="font-family:var(--mono);font-size:0.8rem;color:var(--text-muted);">%s</span></div>`,
						esc(f.Field), esc(val), readonlyAttr(readonly), esc(val)))
				case parser.FieldRating:
					maxStars := fieldDef.MaxStars
					if maxStars == 0 {
						maxStars = 5
					}
					rating := 0
					fmt.Sscanf(val, "%d", &rating)
					b.WriteString(`<div style="display:flex;gap:2px;">`)
					for i := 1; i <= maxStars; i++ {
						star := "☆"
						color := "var(--text-muted,#ccc)"
						if i <= rating {
							star = "★"
							color = "var(--warning,#f59e0b)"
						}
						b.WriteString(fmt.Sprintf(`<span onclick="setRating('%s',%d)" style="cursor:pointer;font-size:1.4rem;color:%s;">%s</span>`,
							esc(f.Field), i, color, star))
					}
					b.WriteString(fmt.Sprintf(`<input type="hidden" name="%s" value="%s" id="rating-%s">`, esc(f.Field), esc(val), esc(f.Field)))
					b.WriteString(`</div>`)
				case parser.FieldJSON:
					b.WriteString(fmt.Sprintf(`<textarea class="bc-input" name="%s" rows="4" style="font-family:var(--mono,monospace);font-size:0.8rem;" %s>%s</textarea>`,
						esc(f.Field), readonlyAttr(readonly), esc(val)))
				default:
					b.WriteString(c.renderInputField(f.Field, val, "text", readonly))
				}
			}
		}

		b.WriteString(`</div></div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func (c *ComponentCompiler) compileFormTabs(tabs []parser.TabDefinition, record map[string]any, model string) string {
	var b strings.Builder
	b.WriteString(`<div class="bc-tabs" style="margin-top:1rem;">`)
	for i, tab := range tabs {
		active := ""
		if i == 0 {
			active = " active"
		}
		b.WriteString(fmt.Sprintf(`<div class="bc-tab%s" onclick="switchTab(this,'bc-tab-%d')">%s</div>`, active, i, esc(tab.Label)))
	}
	b.WriteString(`</div>`)
	for i, tab := range tabs {
		display := ""
		if i > 0 {
			display = `style="display:none"`
		}
		b.WriteString(fmt.Sprintf(`<div class="bc-tab-content" id="bc-tab-%d" %s>`, i, display))
		if tab.View != "" {
			if c.embeddedViewRenderer != nil {
				b.WriteString(c.embeddedViewRenderer(tab.View))
			} else {
				b.WriteString(fmt.Sprintf(`<p style="color:var(--text-muted);font-size:0.85rem;padding:0.75rem 0;">View: %s (loading...)</p>`, esc(tab.View)))
			}
		}
		for _, field := range tab.Fields {
			val := ""
			if record != nil {
				if v, ok := record[field]; ok && v != nil {
					val = fmt.Sprintf("%v", v)
				}
			}
			fieldDef := c.getFieldDef(model, field)
			label := field
			if fieldDef != nil && fieldDef.Label != "" {
				label = fieldDef.Label
			}
			b.WriteString(`<div class="bc-form-group">`)
			b.WriteString(fmt.Sprintf(`<label class="bc-label">%s</label>`, esc(label)))
			if fieldDef != nil && (fieldDef.Type == parser.FieldText || fieldDef.Type == parser.FieldRichText) {
				b.WriteString(fmt.Sprintf(`<textarea class="bc-input" name="%s" rows="4">%s</textarea>`, esc(field), esc(val)))
			} else {
				b.WriteString(fmt.Sprintf(`<input class="bc-input" name="%s" value="%s" placeholder="Enter %s">`, esc(field), esc(val), esc(label)))
			}
			b.WriteString(`</div>`)
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`<script>function switchTab(el,id){el.parentElement.querySelectorAll('.bc-tab').forEach(function(t){t.classList.remove('active')});el.classList.add('active');document.querySelectorAll('.bc-tab-content').forEach(function(c){c.style.display='none'});document.getElementById(id).style.display='block';}</script>`)
	return b.String()
}

func (c *ComponentCompiler) renderInputField(name, value, inputType string, readonly bool) string {
	return fmt.Sprintf(`<input class="bc-input" type="%s" name="%s" value="%s" %s>`,
		inputType, esc(name), esc(value), readonlyAttr(readonly))
}

func (c *ComponentCompiler) renderSelectField(name, value string, options []string, readonly bool) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<select class="bc-input bc-select" name="%s" %s>`, esc(name), readonlyAttr(readonly)))
	b.WriteString(`<option value="">-- Select --</option>`)
	for _, opt := range options {
		selected := ""
		if opt == value {
			selected = " selected"
		}
		b.WriteString(fmt.Sprintf(`<option value="%s"%s>%s</option>`, esc(opt), selected, esc(opt)))
	}
	b.WriteString(`</select>`)
	return b.String()
}

func readonlyAttr(readonly bool) string {
	if readonly {
		return `disabled`
	}
	return ""
}

func checkedAttr(checked bool) string {
	if checked {
		return `checked`
	}
	return ""
}

func (c *ComponentCompiler) CompileList(viewDef *parser.ViewDefinition) string {
	var b strings.Builder
	fieldsJSON := toJSONArray(viewDef.Fields)
	b.WriteString(fmt.Sprintf(`<bc-view-list model="%s" view-title="%s" fields='%s'></bc-view-list>`, esc(viewDef.Model), esc(viewDef.Title), fieldsJSON))
	return b.String()
}

func (c *ComponentCompiler) CompileKanban(viewDef *parser.ViewDefinition) string {
	fieldsJSON := toJSONArray(viewDef.Fields)
	return fmt.Sprintf(`<bc-view-kanban model="%s" view-title="%s" fields='%s' config='{"group_by":"%s"}'></bc-view-kanban>`,
		esc(viewDef.Model), esc(viewDef.Title), fieldsJSON, esc(viewDef.GroupBy))
}

func (c *ComponentCompiler) compileLayoutItem(item parser.LayoutItem, record map[string]any, modelName string) string {
	var b strings.Builder

	if item.Header != nil {
		b.WriteString(c.compileHeader(item.Header, record))
	}

	if len(item.ButtonBox) > 0 {
		b.WriteString(c.compileButtonBox(item.ButtonBox))
	}

	if item.Section != nil {
		b.WriteString(c.compileSection(item, record, modelName))
	} else if len(item.Row) > 0 {
		b.WriteString(c.compileRow(item.Row, record, modelName))
	}

	if len(item.Tabs) > 0 {
		b.WriteString(c.compileTabs(item.Tabs))
	}

	if item.ChildTable != nil {
		b.WriteString(c.compileChildTable(item.ChildTable))
	}

	if item.Chatter {
		b.WriteString(`<bc-chatter></bc-chatter>`)
	}

	if item.Separator != nil {
		label := ""
		if item.Separator.Label != "" {
			label = fmt.Sprintf(` label="%s"`, esc(item.Separator.Label))
		}
		b.WriteString(fmt.Sprintf(`<bc-separator%s></bc-separator>`, label))
	}

	return b.String()
}

func (c *ComponentCompiler) compileHeader(h *parser.HeaderDefinition, record map[string]any) string {
	var b strings.Builder
	statusVal := ""
	if record != nil && h.StatusField != "" {
		if v, ok := record[h.StatusField]; ok {
			statusVal = fmt.Sprintf("%v", v)
		}
	}
	buttonsJSON := actionsToJSON(h.Buttons)
	b.WriteString(fmt.Sprintf(`<bc-header status-field="%s" status-value="%s" buttons='%s'></bc-header>`,
		esc(h.StatusField), esc(statusVal), buttonsJSON))
	return b.String()
}

func (c *ComponentCompiler) compileButtonBox(buttons []parser.SmartButtonDefinition) string {
	var parts []string
	for _, btn := range buttons {
		parts = append(parts, fmt.Sprintf(`{"label":"%s","icon":"%s"}`, esc(btn.Label), esc(btn.Icon)))
	}
	return fmt.Sprintf(`<bc-button-box buttons='[%s]'></bc-button-box>`, strings.Join(parts, ","))
}

func (c *ComponentCompiler) compileSection(item parser.LayoutItem, record map[string]any, modelName string) string {
	var b strings.Builder
	attrs := ""
	if item.Section.Title != "" {
		attrs += fmt.Sprintf(` section-title="%s"`, esc(item.Section.Title))
	}
	if item.Section.Collapsible {
		attrs += ` collapsible`
	}
	b.WriteString(fmt.Sprintf(`<bc-section%s>`, attrs))
	for _, row := range item.Rows {
		b.WriteString(c.compileLayoutItem(row, record, modelName))
	}
	b.WriteString(`</bc-section>`)
	return b.String()
}

func (c *ComponentCompiler) compileRow(fields []parser.LayoutRow, record map[string]any, modelName string) string {
	var b strings.Builder
	b.WriteString(`<bc-row>`)
	for _, f := range fields {
		width := f.Width
		if width == 0 {
			width = 12
		}
		b.WriteString(fmt.Sprintf(`<bc-column width="%d">`, width))

		val := ""
		if record != nil {
			if v, ok := record[f.Field]; ok {
				val = fmt.Sprintf("%v", v)
			}
		}

		tag := "bc-field-string"
		var extraAttrs string

		fieldDef := c.getFieldDef(modelName, f.Field)
		if fieldDef != nil {
			if f.Widget != "" {
				tag = fmt.Sprintf("bc-widget-%s", f.Widget)
			} else {
				tag = fieldTypeToTag(fieldDef.Type, fieldDef.Widget)
			}

			if fieldDef.Label != "" {
				extraAttrs += fmt.Sprintf(` label="%s"`, esc(fieldDef.Label))
			}
			if len(fieldDef.Options) > 0 {
				optJSON := `[`
				for i, opt := range fieldDef.Options {
					if i > 0 {
						optJSON += ","
					}
					optJSON += fmt.Sprintf(`"%s"`, esc(opt))
				}
				optJSON += `]`
				extraAttrs += fmt.Sprintf(` options='%s'`, optJSON)
			}
			if fieldDef.Model != "" {
				extraAttrs += fmt.Sprintf(` model="%s"`, esc(fieldDef.Model))
			}
			if fieldDef.CurrencyCode != "" {
				extraAttrs += fmt.Sprintf(` currency="%s"`, esc(fieldDef.CurrencyCode))
			}
			if fieldDef.Precision > 0 {
				extraAttrs += fmt.Sprintf(` precision="%d"`, fieldDef.Precision)
			}
			if fieldDef.MaxStars > 0 {
				extraAttrs += fmt.Sprintf(` max-stars="%d"`, fieldDef.MaxStars)
			}
			if fieldDef.Language != "" {
				extraAttrs += fmt.Sprintf(` language="%s"`, esc(fieldDef.Language))
			}
			if fieldDef.Toolbar != "" {
				extraAttrs += fmt.Sprintf(` toolbar="%s"`, esc(fieldDef.Toolbar))
			}
			if fieldDef.Required {
				extraAttrs += ` required`
			}
			if fieldDef.DependsOn != "" {
				extraAttrs += fmt.Sprintf(` depends-on="%s"`, esc(fieldDef.DependsOn))
			}
		} else if f.Widget != "" {
			tag = fmt.Sprintf("bc-widget-%s", f.Widget)
		}

		attrs := fmt.Sprintf(`name="%s" value="%s"%s`, esc(f.Field), esc(val), extraAttrs)
		if f.Readonly {
			attrs += ` readonly`
		}

		b.WriteString(fmt.Sprintf(`<%s %s></%s>`, tag, attrs, tag))
		b.WriteString(`</bc-column>`)
	}
	b.WriteString(`</bc-row>`)
	return b.String()
}

func (c *ComponentCompiler) compileTabs(tabs []parser.TabDefinition) string {
	var b strings.Builder
	b.WriteString(`<bc-tabs>`)
	for _, tab := range tabs {
		b.WriteString(fmt.Sprintf(`<bc-tab label="%s">`, esc(tab.Label)))
		if tab.View != "" {
			b.WriteString(fmt.Sprintf(`<bc-view-list model="%s"></bc-view-list>`, esc(tab.View)))
		}
		for _, field := range tab.Fields {
			b.WriteString(fmt.Sprintf(`<bc-field-text name="%s" label="%s"></bc-field-text>`, esc(field), esc(field)))
		}
		b.WriteString(`</bc-tab>`)
	}
	b.WriteString(`</bc-tabs>`)
	return b.String()
}

func (c *ComponentCompiler) compileChildTable(ct *parser.ChildTableDefinition) string {
	var cols []string
	for _, col := range ct.Columns {
		cols = append(cols, fmt.Sprintf(`{"field":"%s","width":%d}`, esc(col.Field), col.Width))
	}
	return fmt.Sprintf(`<bc-child-table field="%s" columns='[%s]'></bc-child-table>`,
		esc(ct.Field), strings.Join(cols, ","))
}

func (c *ComponentCompiler) compileActions(actions []parser.ActionDefinition) string {
	var parts []string
	for _, a := range actions {
		parts = append(parts, fmt.Sprintf(`{"label":"%s","process":"%s","variant":"%s"}`,
			esc(a.Label), esc(a.Process), esc(a.Variant)))
	}
	return fmt.Sprintf(`<bc-header buttons='[%s]'></bc-header>`, strings.Join(parts, ","))
}

func esc(s string) string {
	return html.EscapeString(s)
}

func toJSONArray(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	var parts []string
	for _, item := range items {
		parts = append(parts, fmt.Sprintf(`"%s"`, esc(item)))
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ","))
}

func actionsToJSON(actions []parser.ActionDefinition) string {
	if len(actions) == 0 {
		return "[]"
	}
	var parts []string
	for _, a := range actions {
		parts = append(parts, fmt.Sprintf(`{"label":"%s","process":"%s","variant":"%s"}`,
			esc(a.Label), esc(a.Process), esc(a.Variant)))
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ","))
}

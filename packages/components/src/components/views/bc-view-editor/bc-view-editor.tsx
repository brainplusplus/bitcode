import { Component, Method, Prop, State, Event, EventEmitter, Watch, h } from '@stencil/core';
import { EditorLayout, EditorSection, ModelFieldInfo, genId } from './types';

@Component({
  tag: 'bc-view-editor',
  styleUrl: 'bc-view-editor.css',
  shadow: false,
})
export class BcViewEditor {
  @Prop() viewJson: string = '{}';
  @Prop() modelFields: string = '[]';
  @Prop() readonly: boolean = false;

  @Event() viewChanged!: EventEmitter<{ json: string }>;

  @State() layout: EditorLayout = { sections: [], tabs: [], hasChatter: false };
  @State() fields: ModelFieldInfo[] = [];
  @State() selectedId: string | null = null;
  @State() dragField: string | null = null;
  @State() filterText: string = '';

  @Watch('viewJson')
  onViewJsonChange(val: string) {
    this.parseViewJson(val);
  }

  @Watch('modelFields')
  onModelFieldsChange(val: string) {
    try { this.fields = JSON.parse(val); } catch { this.fields = []; }
  }

  componentWillLoad() {
    this.parseViewJson(this.viewJson);
    try { this.fields = JSON.parse(this.modelFields); } catch { this.fields = []; }
  }

  private parseViewJson(json: string) {
    try {
      const view = JSON.parse(json);
      const layout: EditorLayout = { sections: [], tabs: [], hasChatter: false };

      if (view.layout && Array.isArray(view.layout)) {
        for (const item of view.layout) {
          if (item.section) {
            const section: EditorSection = {
              id: genId(),
              title: item.section.title || '',
              collapsible: item.section.collapsible || false,
              rows: [],
            };
            if (item.rows) {
              for (const r of item.rows) {
                if (r.row) {
                  section.rows.push({
                    id: genId(),
                    fields: r.row.map((f: any) => ({
                      field: f.field || '',
                      width: f.width || 6,
                      readonly: f.readonly || false,
                      widget: f.widget || '',
                      formula: f.formula || '',
                    })),
                  });
                }
              }
            }
            layout.sections.push(section);
          } else if (item.tabs) {
            layout.tabs = item.tabs.map((t: any) => ({
              id: genId(),
              label: t.label || '',
              view: t.view || '',
              fields: t.fields || [],
            }));
          } else if (item.chatter) {
            layout.hasChatter = true;
          } else if (item.row) {
            if (layout.sections.length === 0) {
              layout.sections.push({ id: genId(), title: '', rows: [] });
            }
            const lastSection = layout.sections[layout.sections.length - 1];
            lastSection.rows.push({
              id: genId(),
              fields: item.row.map((f: any) => ({
                field: f.field || '',
                width: f.width || 6,
                readonly: f.readonly || false,
                widget: f.widget || '',
              })),
            });
          }
        }
      }

      this.layout = { ...layout };
    } catch {
      // invalid JSON, keep current layout
    }
  }

  private emitChange() {
    const json = this.serializeLayout();
    this.viewChanged.emit({ json });
  }

  private serializeLayout(): string {
    try {
      const view = JSON.parse(this.viewJson);
      const layoutItems: any[] = [];

      for (const section of this.layout.sections) {
        const item: any = {};
        if (section.title) {
          item.section = { title: section.title };
          if (section.collapsible) item.section.collapsible = true;
        }
        if (section.rows.length > 0) {
          item.rows = section.rows.map(r => ({
            row: r.fields.map(f => {
              const obj: any = { field: f.field, width: f.width };
              if (f.readonly) obj.readonly = true;
              if (f.widget) obj.widget = f.widget;
              if (f.formula) obj.formula = f.formula;
              return obj;
            }),
          }));
        }
        layoutItems.push(item);
      }

      if (this.layout.tabs.length > 0) {
        layoutItems.push({
          tabs: this.layout.tabs.map(t => {
            const obj: any = { label: t.label };
            if (t.view) obj.view = t.view;
            if (t.fields && t.fields.length > 0) obj.fields = t.fields;
            return obj;
          }),
        });
      }

      if (this.layout.hasChatter) {
        layoutItems.push({ chatter: true });
      }

      view.layout = layoutItems;
      return JSON.stringify(view, null, 2);
    } catch {
      return this.viewJson;
    }
  }

  private usedFields(): Set<string> {
    const used = new Set<string>();
    for (const s of this.layout.sections) {
      for (const r of s.rows) {
        for (const f of r.fields) {
          used.add(f.field);
        }
      }
    }
    return used;
  }

  private onDragStart(fieldName: string) {
    this.dragField = fieldName;
  }

  private onDropOnRow(sectionIdx: number, rowIdx: number) {
    if (!this.dragField || this.readonly) return;
    const newLayout = { ...this.layout };
    newLayout.sections = [...newLayout.sections];
    const section = { ...newLayout.sections[sectionIdx] };
    section.rows = [...section.rows];
    const row = { ...section.rows[rowIdx] };
    row.fields = [...row.fields, { field: this.dragField, width: 4 }];
    section.rows[rowIdx] = row;
    newLayout.sections[sectionIdx] = section;
    this.layout = newLayout;
    this.dragField = null;
    this.emitChange();
  }

  private onDropNewRow(sectionIdx: number) {
    if (!this.dragField || this.readonly) return;
    const newLayout = { ...this.layout };
    newLayout.sections = [...newLayout.sections];
    const section = { ...newLayout.sections[sectionIdx] };
    section.rows = [...section.rows, { id: genId(), fields: [{ field: this.dragField, width: 6 }] }];
    newLayout.sections[sectionIdx] = section;
    this.layout = newLayout;
    this.dragField = null;
    this.emitChange();
  }

  private addSection() {
    if (this.readonly) return;
    const newLayout = { ...this.layout };
    newLayout.sections = [...newLayout.sections, { id: genId(), title: 'New Section', rows: [{ id: genId(), fields: [] }] }];
    this.layout = newLayout;
    this.emitChange();
  }

  private addRow(sectionIdx: number) {
    if (this.readonly) return;
    const newLayout = { ...this.layout };
    newLayout.sections = [...newLayout.sections];
    const section = { ...newLayout.sections[sectionIdx] };
    section.rows = [...section.rows, { id: genId(), fields: [] }];
    newLayout.sections[sectionIdx] = section;
    this.layout = newLayout;
    this.emitChange();
  }

  private removeField(sectionIdx: number, rowIdx: number, fieldIdx: number) {
    if (this.readonly) return;
    const newLayout = { ...this.layout };
    newLayout.sections = [...newLayout.sections];
    const section = { ...newLayout.sections[sectionIdx] };
    section.rows = [...section.rows];
    const row = { ...section.rows[rowIdx] };
    row.fields = row.fields.filter((_, i) => i !== fieldIdx);
    section.rows[rowIdx] = row;
    newLayout.sections[sectionIdx] = section;
    this.layout = newLayout;
    this.emitChange();
  }

  private removeRow(sectionIdx: number, rowIdx: number) {
    if (this.readonly) return;
    const newLayout = { ...this.layout };
    newLayout.sections = [...newLayout.sections];
    const section = { ...newLayout.sections[sectionIdx] };
    section.rows = section.rows.filter((_, i) => i !== rowIdx);
    newLayout.sections[sectionIdx] = section;
    this.layout = newLayout;
    this.emitChange();
  }

  private removeSection(sectionIdx: number) {
    if (this.readonly) return;
    const newLayout = { ...this.layout };
    newLayout.sections = newLayout.sections.filter((_, i) => i !== sectionIdx);
    this.layout = newLayout;
    this.emitChange();
  }

  private updateFieldWidth(sectionIdx: number, rowIdx: number, fieldIdx: number, width: number) {
    if (this.readonly) return;
    const newLayout = { ...this.layout };
    newLayout.sections = [...newLayout.sections];
    const section = { ...newLayout.sections[sectionIdx] };
    section.rows = [...section.rows];
    const row = { ...section.rows[rowIdx] };
    row.fields = [...row.fields];
    row.fields[fieldIdx] = { ...row.fields[fieldIdx], width };
    section.rows[rowIdx] = row;
    newLayout.sections[sectionIdx] = section;
    this.layout = newLayout;
    this.emitChange();
  }

  private updateSectionTitle(sectionIdx: number, title: string) {
    if (this.readonly) return;
    const newLayout = { ...this.layout };
    newLayout.sections = [...newLayout.sections];
    newLayout.sections[sectionIdx] = { ...newLayout.sections[sectionIdx], title };
    this.layout = newLayout;
    this.emitChange();
  }  @Method() async refresh(): Promise<void> { }

  render() {
    const used = this.usedFields();
    const available = this.fields.filter(f =>
      !used.has(f.name) &&
      f.type !== 'one2many' && f.type !== 'many2many' && f.type !== 'computed' &&
      (this.filterText === '' || f.name.toLowerCase().includes(this.filterText.toLowerCase()))
    );

    return (
      <div class="ve-container">
        <div class="ve-palette">
          <div class="ve-palette-title">Fields</div>
          <input
            type="text"
            class="ve-search"
            placeholder="Filter fields..."
            value={this.filterText}
            onInput={(e: any) => this.filterText = e.target.value}
          />
          <div class="ve-field-list">
            {available.map(f => (
              <div
                class="ve-field-item"
                draggable={!this.readonly}
                onDragStart={() => this.onDragStart(f.name)}
              >
                <span class="ve-field-name">{f.name}</span>
                <span class="ve-field-type">{f.type}</span>
              </div>
            ))}
            {available.length === 0 && <div class="ve-empty">No available fields</div>}
          </div>
        </div>

        <div class="ve-canvas">
          {this.layout.sections.map((section, si) => (
            <div class="ve-section">
              <div class="ve-section-header">
                <input
                  type="text"
                  class="ve-section-title-input"
                  value={section.title}
                  placeholder="Section title"
                  readOnly={this.readonly}
                  onInput={(e: any) => this.updateSectionTitle(si, e.target.value)}
                />
                {!this.readonly && <button class="ve-btn-remove" onClick={() => this.removeSection(si)}>&times;</button>}
              </div>
              {section.rows.map((row, ri) => (
                <div
                  class="ve-row"
                  onDragOver={(e: DragEvent) => e.preventDefault()}
                  onDrop={() => this.onDropOnRow(si, ri)}
                >
                  {row.fields.map((field, fi) => (
                    <div class="ve-field" style={{ flex: `0 0 ${(field.width / 12) * 100}%` }}>
                      <div class="ve-field-label">{field.field}</div>
                      <div class="ve-field-controls">
                        <select
                          disabled={this.readonly}
                          onChange={(e: any) => this.updateFieldWidth(si, ri, fi, parseInt(e.target.value))}
                        >
                          {[1,2,3,4,5,6,7,8,9,10,11,12].map(w => <option value={String(w)} selected={field.width === w}>{w}/12</option>)}
                        </select>
                        {!this.readonly && <button class="ve-btn-remove-sm" onClick={() => this.removeField(si, ri, fi)}>&times;</button>}
                      </div>
                    </div>
                  ))}
                  {row.fields.length === 0 && (
                    <div
                      class="ve-drop-zone"
                      onDragOver={(e: DragEvent) => e.preventDefault()}
                      onDrop={() => this.onDropOnRow(si, ri)}
                    >
                      Drop fields here
                    </div>
                  )}
                  {!this.readonly && <button class="ve-btn-remove-row" onClick={() => this.removeRow(si, ri)}>&times;</button>}
                </div>
              ))}
              <div
                class="ve-add-row"
                onDragOver={(e: DragEvent) => e.preventDefault()}
                onDrop={() => this.onDropNewRow(si)}
              >
                {!this.readonly && <button class="ve-btn-add" onClick={() => this.addRow(si)}>+ Add Row</button>}
              </div>
            </div>
          ))}
          {!this.readonly && (
            <button class="ve-btn-add-section" onClick={() => this.addSection()}>+ Add Section</button>
          )}
        </div>

        <div class="ve-properties">
          <div class="ve-palette-title">Properties</div>
          {this.selectedId ? (
            <div class="ve-prop-content">Selected: {this.selectedId}</div>
          ) : (
            <div class="ve-empty">Click an element to edit properties</div>
          )}
        </div>
      </div>
    );
  }
}


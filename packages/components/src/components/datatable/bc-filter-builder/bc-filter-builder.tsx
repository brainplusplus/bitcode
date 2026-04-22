import { Component, Prop, State, Event, EventEmitter, Element, h } from '@stencil/core';
import { i18n } from '../../../core/i18n';

interface FilterCondition {
  id: string;
  field: string;
  operator: string;
  value: any;
}

interface FilterGroup {
  id: string;
  logic: 'AND' | 'OR';
  filters: Array<FilterCondition | FilterGroup>;
}

function isGroup(f: FilterCondition | FilterGroup): f is FilterGroup {
  return 'logic' in f;
}

function uid(): string { return Math.random().toString(36).slice(2, 9); }

@Component({ tag: 'bc-filter-builder', styleUrl: 'bc-filter-builder.css', shadow: false })
export class BcFilterBuilder {
  @Element() el!: HTMLElement;
  @Prop() fields: string = '[]';
  @Prop() operators: string = '["=","!=",">","<",">=","<=","contains","starts_with","ends_with","in","is_null","is_not_null"]';
  @Prop({ mutable: true }) value: string = '';
  @Prop() showJsonToggle: boolean = false;

  @State() root: FilterGroup = { id: uid(), logic: 'AND', filters: [] };
  @State() jsonMode: boolean = false;
  @State() jsonText: string = '';

  @Event() lcFilterChange!: EventEmitter<{ filter: FilterGroup }>;

  private getFields(): Array<{ field: string; label: string; type?: string }> {
    try { return JSON.parse(this.fields); } catch { return []; }
  }

  private getOperators(): string[] {
    try { return JSON.parse(this.operators); } catch { return ['=', '!=', 'contains']; }
  }

  componentWillRender() {
    this.el.dir = i18n.dir;
  }

  componentWillLoad() {
    if (this.value) {
      try { this.root = JSON.parse(this.value); } catch { /* keep default */ }
    }
  }

  private emit() {
    this.value = JSON.stringify(this.root);
    this.lcFilterChange.emit({ filter: this.root });
  }

  private addCondition(group: FilterGroup) {
    const fields = this.getFields();
    group.filters.push({
      id: uid(),
      field: fields.length > 0 ? fields[0].field : '',
      operator: '=',
      value: '',
    });
    this.root = { ...this.root };
    this.emit();
  }

  private addGroup(parent: FilterGroup) {
    parent.filters.push({ id: uid(), logic: 'AND', filters: [] });
    this.root = { ...this.root };
    this.emit();
  }

  private removeFilter(parent: FilterGroup, index: number) {
    parent.filters.splice(index, 1);
    this.root = { ...this.root };
    this.emit();
  }

  private toggleLogic(group: FilterGroup) {
    group.logic = group.logic === 'AND' ? 'OR' : 'AND';
    this.root = { ...this.root };
    this.emit();
  }

  private updateCondition(cond: FilterCondition, key: string, val: any) {
    (cond as any)[key] = val;
    this.root = { ...this.root };
    this.emit();
  }

  private applyJson() {
    try {
      this.root = JSON.parse(this.jsonText);
      this.emit();
    } catch { /* invalid json */ }
  }

  private renderGroup(group: FilterGroup, depth: number = 0): any {
    const fields = this.getFields();
    const ops = this.getOperators();
    return (
      <div class={'bc-fb-group depth-' + Math.min(depth, 3)}>
        <div class="bc-fb-group-header">
          <button type="button" class={'bc-fb-logic ' + group.logic.toLowerCase()} onClick={() => this.toggleLogic(group)}>
            {group.logic}
          </button>
          <div class="bc-fb-group-actions">
            <button type="button" class="bc-fb-btn" onClick={() => this.addCondition(group)}>{i18n.t('filter.addCondition')}</button>
            <button type="button" class="bc-fb-btn" onClick={() => this.addGroup(group)}>{i18n.t('filter.addGroup')}</button>
          </div>
        </div>
        <div class="bc-fb-group-body">
          {group.filters.map((f, i) =>
            isGroup(f) ? (
              <div class="bc-fb-nested">
                {this.renderGroup(f, depth + 1)}
                <button type="button" class="bc-fb-remove" onClick={() => this.removeFilter(group, i)} title={i18n.t('filter.removeGroup')}>&times;</button>
              </div>
            ) : (
              <div class="bc-fb-condition">
                <select class="bc-fb-select" onChange={(e) => this.updateCondition(f, 'field', (e.target as HTMLSelectElement).value)}>
                  {fields.map(fd => <option value={fd.field} {...(fd.field === f.field ? {selected: true} : {})}>{fd.label || fd.field}</option>)}
                </select>
                <select class="bc-fb-select bc-fb-op" onChange={(e) => this.updateCondition(f, 'operator', (e.target as HTMLSelectElement).value)}>
                  {ops.map(op => <option value={op} {...(op === f.operator ? {selected: true} : {})}>{op}</option>)}
                </select>
                {f.operator !== 'is_null' && f.operator !== 'is_not_null' && (
                  <input type="text" class="bc-fb-input" value={String(f.value ?? '')} onInput={(e) => this.updateCondition(f, 'value', (e.target as HTMLInputElement).value)} placeholder={i18n.t('filter.valuePlaceholder')} />
                )}
                <button type="button" class="bc-fb-remove" onClick={() => this.removeFilter(group, i)} title={i18n.t('filter.remove')}>&times;</button>
              </div>
            )
          )}
          {group.filters.length === 0 && <div class="bc-fb-empty">{i18n.t('filter.empty')}</div>}
        </div>
      </div>
    );
  }

  render() {
    return (
      <div class="bc-filter-builder">
        {this.showJsonToggle && (
          <div class="bc-fb-mode-toggle">
            <button type="button" class={'bc-fb-mode ' + (!this.jsonMode ? 'active' : '')} onClick={() => { this.jsonMode = false; }}>{i18n.t('filter.visual')}</button>
            <button type="button" class={'bc-fb-mode ' + (this.jsonMode ? 'active' : '')} onClick={() => { this.jsonMode = true; this.jsonText = JSON.stringify(this.root, null, 2); }}>{i18n.t('filter.json')}</button>
          </div>
        )}
        {this.jsonMode ? (
          <div class="bc-fb-json">
            <textarea class="bc-fb-json-editor" value={this.jsonText} onInput={(e) => { this.jsonText = (e.target as HTMLTextAreaElement).value; }}></textarea>
            <button type="button" class="bc-fb-btn bc-fb-apply" onClick={() => this.applyJson()}>{i18n.t('filter.applyJson')}</button>
          </div>
        ) : (
          this.renderGroup(this.root)
        )}
      </div>
    );
  }
}

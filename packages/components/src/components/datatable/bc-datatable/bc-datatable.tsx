import { Component, Prop, State, Event, EventEmitter, Element, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { i18n } from '../../../core/i18n';
import Sortable from 'sortablejs';
import * as XLSX from 'xlsx';

interface ColumnDef {
  field: string;
  label?: string;
  width?: number;
  minWidth?: number;
  sortable?: boolean;
  filterable?: boolean;
  visible?: boolean;
  type?: string;
  format?: string;
  align?: string;
  frozen?: boolean;
}

interface SortDef { field: string; direction: 'asc' | 'desc'; }
interface FilterCondition { field: string; operator: string; value: any; }
interface FilterGroup { logic: 'AND' | 'OR'; filters: Array<FilterCondition | FilterGroup>; }
interface BulkAction { label: string; action: string; variant?: string; confirm?: string; }

@Component({ tag: 'bc-datatable', styleUrl: 'bc-datatable.css', shadow: false })
export class BcDatatable {
  @Element() el!: HTMLElement;

  @Prop() model: string = '';
  @Prop() columns: string = '[]';
  @Prop() apiUrl: string = '';
  @Prop() pageSize: number = 20;
  @Prop() selectable: boolean = true;
  @Prop() draggableColumns: boolean = true;
  @Prop() exportXls: boolean = true;
  @Prop() showFilterBuilder: boolean = true;
  @Prop() showJsonFilter: boolean = false;
  @Prop() actions: string = '[]';
  @Prop() serverSide: boolean = true;
  @Prop() savedPresets: string = '[]';

  @State() data: Array<Record<string, unknown>> = [];
  @State() total: number = 0;
  @State() page: number = 1;
  @State() limit: number = 20;
  @State() sorts: SortDef[] = [];
  @State() filter: FilterGroup = { logic: 'AND', filters: [] };
  @State() colDefs: ColumnDef[] = [];
  @State() visibleCols: Set<string> = new Set();
  @State() selected: Set<string> = new Set();
  @State() loading: boolean = false;
  @State() showColPicker: boolean = false;
  @State() showFilterPanel: boolean = false;
  @State() showPresets: boolean = false;
  @State() presets: Array<{ name: string; filter: FilterGroup }> = [];
  @State() colWidths: Record<string, number> = {};
  @State() columnFilterValues: Record<string, string[]> = {};
  @State() showColumnFilter: string = '';

  @Event() lcRowClick!: EventEmitter<{ record: Record<string, unknown> }>;
  @Event() lcSelectionChange!: EventEmitter<{ ids: string[] }>;
  @Event() lcBulkAction!: EventEmitter<{ action: string; ids: string[] }>;

  private getCols(): ColumnDef[] { try { return JSON.parse(this.columns); } catch { return []; } }
  private getActions(): BulkAction[] { try { return JSON.parse(this.actions); } catch { return []; } }

  componentWillRender() {
    this.el.dir = i18n.dir;
  }

  componentWillLoad() {
    this.limit = this.pageSize;
    this.colDefs = this.getCols();
    this.visibleCols = new Set(this.colDefs.filter(c => c.visible !== false).map(c => c.field));
    try { this.presets = JSON.parse(this.savedPresets); } catch { this.presets = []; }
  }

  async componentDidLoad() {
    await this.fetchData();
    if (this.draggableColumns) this.initColumnDrag();
  }

  private getApiUrl(): string {
    if (this.apiUrl) return this.apiUrl;
    if (this.model) return '/api/' + this.model + 's';
    return '';
  }

  private async fetchData() {
    const url = this.getApiUrl();
    if (!url) return;
    this.loading = true;
    try {
      const body: Record<string, unknown> = { page: this.page, limit: this.limit };
      if (this.sorts.length > 0) body['sort'] = this.sorts;
      if (this.filter.filters.length > 0) body['filter'] = this.filter;

      const res = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });

      if (!res.ok) {
        const api = getApiClient();
        const listRes = await api.list(this.model, {
          page: this.page, pageSize: this.limit,
          sort: this.sorts.length > 0 ? this.sorts[0].field : undefined,
          order: this.sorts.length > 0 ? this.sorts[0].direction : undefined,
        });
        this.data = listRes.data;
        this.total = listRes.total;
      } else {
        const json = await res.json();
        this.data = json.data || [];
        this.total = json.total || 0;
      }
    } catch {
      try {
        const api = getApiClient();
        const listRes = await api.list(this.model, { page: this.page, pageSize: this.limit });
        this.data = listRes.data;
        this.total = listRes.total;
      } catch { this.data = []; this.total = 0; }
    }
    this.loading = false;
  }

  private async handleSort(field: string, e: MouseEvent) {
    if (e.shiftKey) {
      const idx = this.sorts.findIndex(s => s.field === field);
      if (idx >= 0) {
        if (this.sorts[idx].direction === 'asc') this.sorts[idx].direction = 'desc';
        else this.sorts.splice(idx, 1);
      } else {
        this.sorts.push({ field, direction: 'asc' });
      }
      this.sorts = [...this.sorts];
    } else {
      const existing = this.sorts.find(s => s.field === field);
      if (existing && existing.direction === 'asc') this.sorts = [{ field, direction: 'desc' }];
      else if (existing && existing.direction === 'desc') this.sorts = [];
      else this.sorts = [{ field, direction: 'asc' }];
    }
    this.page = 1;
    await this.fetchData();
  }

  private async handlePage(p: number) { this.page = p; await this.fetchData(); }
  private async handlePageSize(size: number) { this.limit = size; this.page = 1; await this.fetchData(); }

  private toggleSelect(id: string) {
    const s = new Set(this.selected);
    if (s.has(id)) s.delete(id); else s.add(id);
    this.selected = s;
    this.lcSelectionChange.emit({ ids: Array.from(s) });
  }

  private toggleSelectAll() {
    if (this.selected.size === this.data.length) this.selected = new Set();
    else this.selected = new Set(this.data.map(r => String(r['id'] || '')));
    this.lcSelectionChange.emit({ ids: Array.from(this.selected) });
  }

  private toggleColumn(field: string) {
    const v = new Set(this.visibleCols);
    if (v.has(field)) v.delete(field); else v.add(field);
    this.visibleCols = v;
  }

  private getSortIcon(field: string): string {
    const s = this.sorts.find(s => s.field === field);
    if (!s) return '';
    return s.direction === 'asc' ? ' \u25B2' : ' \u25BC';
  }

  private getSortIndex(field: string): number {
    if (this.sorts.length <= 1) return -1;
    return this.sorts.findIndex(s => s.field === field);
  }

  private formatCell(value: unknown, col: ColumnDef): string {
    if (value === null || value === undefined) return '';
    const v = String(value);
    switch (col.type) {
      case 'currency': {
        const num = Number(value);
        if (isNaN(num)) return v;
        const fmt = col.format || 'IDR';
        return i18n.tf.currency(num, fmt, { maximumFractionDigits: 0 });
      }
      case 'number': return i18n.tf.number(Number(value));
      case 'date': { try { return i18n.tf.date(v, { day: 'numeric', month: 'short', year: 'numeric' }); } catch { return v; } }
      case 'boolean': return value ? '\u2713' : '\u2717';
      default: return v;
    }
  }

  private async handleFilterChange(e: CustomEvent) {
    this.filter = e.detail.filter;
    this.page = 1;
    await this.fetchData();
  }

  private exportToXls() {
    const visibleFields = this.colDefs.filter(c => this.visibleCols.has(c.field));
    const exportData = (this.selected.size > 0 ? this.data.filter(r => this.selected.has(String(r['id'] || ''))) : this.data)
      .map(row => {
        const obj: Record<string, unknown> = {};
        visibleFields.forEach(c => { obj[c.label || c.field] = row[c.field]; });
        return obj;
      });
    const ws = XLSX.utils.json_to_sheet(exportData);
    const wb = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(wb, ws, this.model || 'Data');
    XLSX.writeFile(wb, (this.model || 'export') + '.xlsx');
  }

  private async handleBulkAction(action: BulkAction) {
    const ids = Array.from(this.selected);
    if (ids.length === 0) return;
    if (action.confirm && !confirm(action.confirm)) return;
    if (action.action === 'export') { this.exportToXls(); return; }
    if (action.action === 'delete') {
      const api = getApiClient();
      for (const id of ids) { try { await api.remove(this.model, id); } catch { /* skip */ } }
      this.selected = new Set();
      await this.fetchData();
      return;
    }
    this.lcBulkAction.emit({ action: action.action, ids });
  }

  private savePreset() {
    const name = prompt(i18n.t('datatable.presetName'));
    if (!name) return;
    this.presets = [...this.presets, { name, filter: JSON.parse(JSON.stringify(this.filter)) }];
  }

  private loadPreset(preset: { name: string; filter: FilterGroup }) {
    this.filter = JSON.parse(JSON.stringify(preset.filter));
    this.page = 1;
    this.showPresets = false;
    this.fetchData();
  }

  private initColumnDrag() {
    setTimeout(() => {
      const headerRow = this.el.querySelector('.bc-dt-header-row');
      if (!headerRow) return;
      Sortable.create(headerRow as HTMLElement, {
        animation: 150,
        ghostClass: 'bc-dt-col-ghost',
        filter: '.bc-dt-check-col',
        onEnd: (evt) => {
          const offset = this.selectable ? 1 : 0;
          const from = evt.oldIndex! - offset;
          const to = evt.newIndex! - offset;
          if (from < 0 || to < 0) return;
          const cols = [...this.colDefs];
          const [moved] = cols.splice(from, 1);
          cols.splice(to, 0, moved);
          this.colDefs = cols;
        },
      });
    }, 200);
  }

  private getColumnFilterValues(field: string): string[] {
    const vals = new Set<string>();
    this.data.forEach(r => { const v = r[field]; if (v !== null && v !== undefined) vals.add(String(v)); });
    return Array.from(vals).sort();
  }

  render() {
    const visibleDefs = this.colDefs.filter(c => this.visibleCols.has(c.field));
    const totalPages = Math.ceil(this.total / this.limit);
    const allActions: BulkAction[] = [
      ...(this.exportXls ? [{ label: i18n.t('datatable.exportXls'), action: 'export' }] : []),
      { label: i18n.t('datatable.deleteSelected'), action: 'delete', variant: 'danger', confirm: i18n.t('confirm.message') },
      ...this.getActions(),
    ];
    const fields = this.colDefs.map(c => ({ field: c.field, label: c.label || c.field, type: c.type }));

    return (
      <div class="bc-datatable">
        <div class="bc-dt-toolbar">
          <div class="bc-dt-toolbar-left">
            {this.showFilterBuilder && (
              <button type="button" class={'bc-dt-btn ' + (this.showFilterPanel ? 'active' : '')} onClick={() => { this.showFilterPanel = !this.showFilterPanel; }}>
                {'\uD83D\uDD0D'} {i18n.t('common.filter')} {this.filter.filters.length > 0 ? '(' + this.filter.filters.length + ')' : ''}
              </button>
            )}
            <button type="button" class={'bc-dt-btn ' + (this.showColPicker ? 'active' : '')} onClick={() => { this.showColPicker = !this.showColPicker; }}>
              {i18n.t('datatable.columns')}
            </button>
            {this.presets.length > 0 && (
              <button type="button" class="bc-dt-btn" onClick={() => { this.showPresets = !this.showPresets; }}>{i18n.t('datatable.presets')}</button>
            )}
            {this.showFilterBuilder && this.filter.filters.length > 0 && (
              <button type="button" class="bc-dt-btn" onClick={() => this.savePreset()}>{i18n.t('datatable.saveFilter')}</button>
            )}
          </div>
          <div class="bc-dt-toolbar-right">
            <span class="bc-dt-count">{i18n.plural('common.records', this.total)}</span>
            {this.exportXls && <button type="button" class="bc-dt-btn" onClick={() => this.exportToXls()}>{i18n.t('datatable.exportXls')}</button>}
          </div>
        </div>

        {this.showColPicker && (
          <div class="bc-dt-col-picker">
            {this.colDefs.map(c => (
              <label class="bc-dt-col-check">
                <input type="checkbox" checked={this.visibleCols.has(c.field)} onChange={() => this.toggleColumn(c.field)} />
                <span>{c.label || c.field}</span>
              </label>
            ))}
          </div>
        )}

        {this.showPresets && (
          <div class="bc-dt-presets">
            {this.presets.map(p => (
              <button type="button" class="bc-dt-preset-btn" onClick={() => this.loadPreset(p)}>{p.name}</button>
            ))}
          </div>
        )}

        {this.showFilterPanel && (
          <div class="bc-dt-filter-panel">
            <bc-filter-builder
              fields={JSON.stringify(fields)}
              value={JSON.stringify(this.filter)}
              show-json-toggle={this.showJsonFilter}
              onLcFilterChange={(e: CustomEvent) => this.handleFilterChange(e)}
            ></bc-filter-builder>
          </div>
        )}

        {this.selected.size > 0 && (
          <div class="bc-dt-bulk-bar">
            <span class="bc-dt-bulk-count">{this.selected.size} {i18n.t('common.records_other', { count: this.selected.size })}</span>
            {allActions.map(a => (
              <button type="button" class={'bc-dt-btn ' + (a.variant === 'danger' ? 'bc-dt-btn-danger' : '')} onClick={() => this.handleBulkAction(a)}>
                {a.label}
              </button>
            ))}
            <button type="button" class="bc-dt-btn" onClick={() => { this.selected = new Set(); }}>{i18n.t('common.reset')}</button>
          </div>
        )}

        <slot name="filters"></slot>

        {this.loading && <div class="bc-dt-loading">{i18n.t('common.loading')}</div>}

        <div class="bc-dt-table-wrap">
          <table class="bc-dt-table">
            <thead>
              <tr class="bc-dt-header-row">
                {this.selectable && (
                  <th class="bc-dt-check-col">
                    <input type="checkbox" checked={this.selected.size === this.data.length && this.data.length > 0} onChange={() => this.toggleSelectAll()} />
                  </th>
                )}
                {visibleDefs.map(col => (
                  <th
                    class={'bc-dt-th ' + (col.sortable !== false ? 'sortable ' : '') + (this.sorts.find(s => s.field === col.field) ? 'sorted' : '')}
                     style={{ width: (this.colWidths[col.field] || col.width) ? ((this.colWidths[col.field] || col.width) + 'px') : 'auto', textAlign: (col.align || (i18n.isRTL ? 'right' : 'left')) as any }}
                    onClick={(e) => col.sortable !== false && this.handleSort(col.field, e)}
                  >
                    <span class="bc-dt-th-label">{col.label || col.field}</span>
                    {this.getSortIcon(col.field) && <span class="bc-dt-sort-icon">{this.getSortIcon(col.field)}</span>}
                    {this.getSortIndex(col.field) >= 0 && <span class="bc-dt-sort-idx">{this.getSortIndex(col.field) + 1}</span>}
                    {col.filterable !== false && (
                      <button type="button" class="bc-dt-col-filter-btn" onClick={(e) => { e.stopPropagation(); this.showColumnFilter = this.showColumnFilter === col.field ? '' : col.field; }} title={i18n.t('datatable.filterColumn')}>{'\u25BC'}</button>
                    )}
                    {this.showColumnFilter === col.field && (
                      <div class="bc-dt-col-filter-dropdown" onClick={(e) => e.stopPropagation()}>
                        {this.getColumnFilterValues(col.field).map(v => (
                          <label class="bc-dt-col-filter-item">
                            <input type="checkbox" onChange={() => {
                              const cond: FilterCondition = { field: col.field, operator: '=', value: v };
                              this.filter = { ...this.filter, filters: [...this.filter.filters, cond] };
                              this.showColumnFilter = '';
                              this.page = 1;
                              this.fetchData();
                            }} />
                            <span>{v}</span>
                          </label>
                        ))}
                      </div>
                    )}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {this.data.map(row => {
                const id = String(row['id'] || '');
                return (
                  <tr class={'bc-dt-row ' + (this.selected.has(id) ? 'selected' : '')} onClick={() => this.lcRowClick.emit({ record: row })}>
                    {this.selectable && (
                      <td class="bc-dt-check-col" onClick={(e) => e.stopPropagation()}>
                        <input type="checkbox" checked={this.selected.has(id)} onChange={() => this.toggleSelect(id)} />
                      </td>
                    )}
                    {visibleDefs.map(col => (
                      <td class="bc-dt-td" style={{ textAlign: (col.align || (i18n.isRTL ? 'right' : 'left')) as any }}>
                        {this.formatCell(row[col.field], col)}
                      </td>
                    ))}
                  </tr>
                );
              })}
              {this.data.length === 0 && !this.loading && (
                <tr><td colSpan={visibleDefs.length + (this.selectable ? 1 : 0)} class="bc-dt-empty">{i18n.t('datatable.noRecords')}</td></tr>
              )}
            </tbody>
          </table>
        </div>

        <div class="bc-dt-footer">
          <div class="bc-dt-page-size">
            <span>{i18n.t('datatable.show')}</span>
            {[10, 20, 50, 100].map(s => (
              <button type="button" class={'bc-dt-ps-btn ' + (this.limit === s ? 'active' : '')} onClick={() => this.handlePageSize(s)}>{s}</button>
            ))}
          </div>
          <div class="bc-dt-pagination">
            <button type="button" class="bc-dt-page-btn" disabled={this.page <= 1} onClick={() => this.handlePage(1)}>{'\u00AB'}</button>
            <button type="button" class="bc-dt-page-btn" disabled={this.page <= 1} onClick={() => this.handlePage(this.page - 1)}>{'\u2039'}</button>
            <span class="bc-dt-page-info">{i18n.t('common.page')} {this.page} {i18n.t('common.of')} {totalPages || 1}</span>
            <button type="button" class="bc-dt-page-btn" disabled={this.page >= totalPages} onClick={() => this.handlePage(this.page + 1)}>{'\u203A'}</button>
            <button type="button" class="bc-dt-page-btn" disabled={this.page >= totalPages} onClick={() => this.handlePage(totalPages)}>{'\u00BB'}</button>
          </div>
          <div class="bc-dt-total">{i18n.t('common.total')}: {i18n.tf.number(this.total)}</div>
        </div>
      </div>
    );
  }
}

import { Component, Prop, State, Event, EventEmitter, Element, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { i18n } from '../../../core/i18n';

@Component({ tag: 'bc-lookup-modal', styleUrl: 'bc-lookup-modal.css', shadow: false })
export class BcLookupModal {
  @Element() el!: HTMLElement;
  @Prop({ mutable: true }) open: boolean = false;
  @Prop() model: string = '';
  @Prop() displayField: string = 'name';
  @Prop() columns: string = '[]';
  @Prop() multiple: boolean = false;
  @Prop() apiUrl: string = '';
  @Prop() modalTitle: string = '';

  @State() data: Array<Record<string, unknown>> = [];
  @State() total: number = 0;
  @State() page: number = 1;
  @State() query: string = '';
  @State() selected: Set<string> = new Set();
  @State() loading: boolean = false;

  @Event() lcLookupSelect!: EventEmitter<{ records: Array<Record<string, unknown>> }>;
  @Event() lcLookupClose!: EventEmitter<void>;

  private getCols(): Array<{ field: string; label?: string }> {
    try { return JSON.parse(this.columns); } catch { return [{ field: this.displayField }]; }
  }

  componentWillRender() { this.el.dir = i18n.dir; }

  async componentDidLoad() { if (this.open) await this.fetchData(); }

  private async fetchData() {
    this.loading = true;
    try {
      const api = getApiClient();
      const res = await api.list(this.model, { page: this.page, pageSize: 10, q: this.query || undefined });
      this.data = res.data;
      this.total = res.total;
    } catch { this.data = []; this.total = 0; }
    this.loading = false;
  }

  private async handleSearch(q: string) {
    this.query = q;
    this.page = 1;
    await this.fetchData();
  }

  private selectRow(row: Record<string, unknown>) {
    if (this.multiple) {
      const id = String(row['id'] || '');
      const s = new Set(this.selected);
      if (s.has(id)) s.delete(id); else s.add(id);
      this.selected = s;
    } else {
      this.lcLookupSelect.emit({ records: [row] });
      this.close();
    }
  }

  private confirmSelection() {
    const selectedRecords = this.data.filter(r => this.selected.has(String(r['id'] || '')));
    this.lcLookupSelect.emit({ records: selectedRecords });
    this.close();
  }

  private close() {
    this.open = false;
    this.query = '';
    this.selected = new Set();
    this.lcLookupClose.emit();
  }

  render() {
    if (!this.open) return null;
    const cols = this.getCols();
    const totalPages = Math.ceil(this.total / 10);
    const title = this.modalTitle || ('Select ' + (this.model || 'Record'));

    return (
      <div class="bc-lookup-overlay" onClick={() => this.close()}>
        <div class="bc-lookup-dialog" onClick={(e) => e.stopPropagation()}>
          <div class="bc-lookup-header">
            <h3>{title}</h3>
            <button type="button" class="bc-lookup-close" onClick={() => this.close()}>{'\u00D7'}</button>
          </div>
          <div class="bc-lookup-search">
            <input type="search" class="bc-lookup-search-input" placeholder={i18n.t('common.search')} value={this.query} onInput={(e) => this.handleSearch((e.target as HTMLInputElement).value)} autoFocus />
            <span class="bc-lookup-total">{i18n.plural('common.records', this.total)}</span>
          </div>
          <div class="bc-lookup-table-wrap">
            {this.loading && <div class="bc-lookup-loading">{i18n.t('common.loading')}</div>}
            <table class="bc-lookup-table">
              <thead><tr>
                {this.multiple && <th class="bc-lookup-check"></th>}
                {cols.map(c => <th>{c.label || c.field}</th>)}
              </tr></thead>
              <tbody>
                {this.data.map(row => {
                  const id = String(row['id'] || '');
                  const isSelected = this.selected.has(id);
                  return (
                    <tr class={'bc-lookup-row ' + (isSelected ? 'selected' : '')} onClick={() => this.selectRow(row)}>
                      {this.multiple && <td class="bc-lookup-check"><input type="checkbox" checked={isSelected} /></td>}
                      {cols.map(c => <td>{String(row[c.field] ?? '')}</td>)}
                    </tr>
                  );
                })}
                {this.data.length === 0 && !this.loading && <tr><td colSpan={cols.length + (this.multiple ? 1 : 0)} class="bc-lookup-empty">{i18n.t('common.noResults')}</td></tr>}
              </tbody>
            </table>
          </div>
          <div class="bc-lookup-footer">
            <div class="bc-lookup-pagination">
              <button type="button" disabled={this.page <= 1} onClick={() => { this.page--; this.fetchData(); }}>{'\u2039'}</button>
              <span>{this.page}/{totalPages || 1}</span>
              <button type="button" disabled={this.page >= totalPages} onClick={() => { this.page++; this.fetchData(); }}>{'\u203A'}</button>
            </div>
            {this.multiple && (
              <div class="bc-lookup-confirm">
                <span>{this.selected.size}</span>
                <button type="button" class="bc-lookup-confirm-btn" onClick={() => this.confirmSelection()} disabled={this.selected.size === 0}>{i18n.t('common.confirm')}</button>
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }
}

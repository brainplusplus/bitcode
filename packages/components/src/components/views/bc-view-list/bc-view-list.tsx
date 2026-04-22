import { Component, Prop, State, Event, EventEmitter, Element, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { i18n } from '../../../core/i18n';

@Component({
  tag: 'bc-view-list',
  styleUrl: 'bc-view-list.css',
  shadow: false,
})
export class BcViewList {
  @Element() el!: HTMLElement;
  @Prop() model: string = '';
  @Prop() viewTitle: string = '';
  @Prop() fields: string = '[]';
  @Prop() config: string = '{}';

  @State() data: Array<Record<string, unknown>> = [];
  @State() total: number = 0;
  @State() page: number = 1;
  @State() pageSize: number = 20;
  @State() sortField: string = '';
  @State() sortOrder: 'asc' | 'desc' = 'asc';
  @State() loading: boolean = false;
  @State() selected: Set<string> = new Set();
  @State() searchQuery: string = '';

  @Event() lcRowSelect!: EventEmitter<{ids: string[]}>;

  private getFields(): string[] { try { return JSON.parse(this.fields); } catch { return []; } }

  componentWillRender() {
    this.el.dir = i18n.dir;
  }

  async componentDidLoad() { await this.fetchData(); }

  private async fetchData() {
    if (!this.model) return;
    this.loading = true;
    try {
      const api = getApiClient();
      const res = await api.list(this.model, {
        page: this.page, pageSize: this.pageSize,
        sort: this.sortField || undefined,
        order: this.sortField ? this.sortOrder : undefined,
        q: this.searchQuery || undefined,
      });
      this.data = res.data;
      this.total = res.total;
    } catch { this.data = []; this.total = 0; }
    this.loading = false;
  }

  private async handleSort(field: string) {
    if (this.sortField === field) { this.sortOrder = this.sortOrder === 'asc' ? 'desc' : 'asc'; }
    else { this.sortField = field; this.sortOrder = 'asc'; }
    await this.fetchData();
  }

  private async handlePage(p: number) { this.page = p; await this.fetchData(); }

  private toggleSelect(id: string) {
    const s = new Set(this.selected);
    if (s.has(id)) s.delete(id); else s.add(id);
    this.selected = s;
    this.lcRowSelect.emit({ ids: Array.from(s) });
  }

  private toggleSelectAll() {
    if (this.selected.size === this.data.length) { this.selected = new Set(); }
    else { this.selected = new Set(this.data.map(r => String(r['id'] || ''))); }
    this.lcRowSelect.emit({ ids: Array.from(this.selected) });
  }

  private async handleSearch(q: string) { this.searchQuery = q; this.page = 1; await this.fetchData(); }

  render() {
    const fields = this.getFields();
    const totalPages = Math.ceil(this.total / this.pageSize);
    return (
      <div class="bc-view bc-view-list">
        <div class="bc-list-header">
          <h2>{this.viewTitle || this.model}</h2>
          <div class="bc-list-actions">
            <input type="search" class="bc-list-search" placeholder={i18n.t('common.search')} value={this.searchQuery} onInput={(e: Event) => this.handleSearch((e.target as HTMLInputElement).value)} />
            <span class="bc-list-count">{i18n.plural('common.records', this.total)}</span>
          </div>
        </div>
        {this.loading && <div class="bc-list-loading">{i18n.t('common.loading')}</div>}
        <div class="bc-list-table-wrap">
          <table class="bc-list-table">
            <thead><tr>
              <th class="bc-list-check"><input type="checkbox" checked={this.selected.size === this.data.length && this.data.length > 0} onChange={() => this.toggleSelectAll()} /></th>
              {fields.map(f => (
                <th class={{'sortable': true, 'sorted': this.sortField === f}} onClick={() => this.handleSort(f)}>
                  {f}{this.sortField === f && <span class="sort-icon">{this.sortOrder === 'asc' ? ' \u25B2' : ' \u25BC'}</span>}
                </th>
              ))}
            </tr></thead>
            <tbody>
              {this.data.map(row => {
                const id = String(row['id'] || '');
                return (<tr class={{'selected': this.selected.has(id)}}>
                  <td class="bc-list-check"><input type="checkbox" checked={this.selected.has(id)} onChange={() => this.toggleSelect(id)} /></td>
                  {fields.map(f => <td>{String(row[f] ?? '')}</td>)}
                </tr>);
              })}
              {this.data.length === 0 && !this.loading && <tr><td colSpan={fields.length + 1} class="bc-list-empty">{i18n.t('datatable.noRecords')}</td></tr>}
            </tbody>
          </table>
        </div>
        {totalPages > 1 && (
          <div class="bc-list-pagination">
            <button type="button" disabled={this.page <= 1} onClick={() => this.handlePage(this.page - 1)}>{'\u2190'} {i18n.t('common.prev')}</button>
            <span>{i18n.t('common.page')} {this.page} {i18n.t('common.of')} {totalPages}</span>
            <button type="button" disabled={this.page >= totalPages} onClick={() => this.handlePage(this.page + 1)}>{i18n.t('common.next')} {'\u2192'}</button>
          </div>
        )}
      </div>
    );
  }
}


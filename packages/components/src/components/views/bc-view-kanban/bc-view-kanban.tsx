import { Component, Method, Prop, State, Event, EventEmitter, Element, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { i18n } from '../../../core/i18n';
import Sortable from 'sortablejs';

@Component({ tag: 'bc-view-kanban', styleUrl: 'bc-view-kanban.css', shadow: false })
export class BcViewKanban {
  @Element() el!: HTMLElement;
  @Prop() model: string = '';
  @Prop() viewTitle: string = '';
  @Prop() fields: string = '[]';
  @Prop() config: string = '{}';
  @State() columns: Map<string, Array<Record<string, unknown>>> = new Map();
  @State() loading: boolean = false;
  @Event() lcKanbanMove!: EventEmitter<{id: string; from: string; to: string}>;

  private getConfig(): Record<string, unknown> { try { return JSON.parse(this.config); } catch { return {}; } }
  private getFields(): string[] { try { return JSON.parse(this.fields); } catch { return []; } }

  componentWillRender() { this.el.dir = i18n.dir; }

  async componentDidLoad() {
    if (!this.model) return;
    this.loading = true;
    try {
      const api = getApiClient();
      const cfg = this.getConfig();
      const groupBy = String(cfg['group_by'] || 'status');
      const res = await api.list(this.model, { pageSize: 100 });
      const cols = new Map<string, Array<Record<string, unknown>>>();
      for (const row of res.data) {
        const col = String(row[groupBy] || 'Other');
        if (!cols.has(col)) cols.set(col, []);
        cols.get(col)!.push(row);
      }
      this.columns = cols;
    } catch { this.columns = new Map(); }
    this.loading = false;
    this.initSortable();
  }

  private initSortable() {
    setTimeout(() => {
      this.el.querySelectorAll('.bc-kanban-cards').forEach(list => {
        Sortable.create(list as HTMLElement, {
          group: 'kanban', animation: 150, ghostClass: 'bc-kanban-ghost',
          onEnd: (evt) => {
            const id = evt.item.getAttribute('data-id') || '';
            const from = evt.from.getAttribute('data-column') || '';
            const to = evt.to.getAttribute('data-column') || '';
            if (from !== to) this.lcKanbanMove.emit({ id, from, to });
          },
        });
      });
    }, 100);
  }  @Method() async refresh(): Promise<void> { }

  render() {
    const fields = this.getFields();
    return (
      <div class="bc-view bc-view-kanban">
        <div class="bc-kanban-header"><h2>{this.viewTitle || this.model}</h2></div>
        {this.loading && <div class="bc-kanban-loading">{i18n.t('common.loading')}</div>}
        <div class="bc-kanban-board">
          {Array.from(this.columns.entries()).map(([colName, cards]) => (
            <div class="bc-kanban-column">
              <div class="bc-kanban-col-header">
                <span>{colName}</span><span class="bc-kanban-col-count">{cards.length}</span>
              </div>
              <div class="bc-kanban-cards" data-column={colName}>
                {cards.map(card => (
                  <div class="bc-kanban-card" data-id={String(card['id'] || '')}>
                    {fields.slice(0, 3).map(f => (
                      <div class="bc-kanban-card-field">
                        <span class="bc-kf-label">{f}</span>
                        <span class="bc-kf-value">{String(card[f] ?? '')}</span>
                      </div>
                    ))}
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }
}


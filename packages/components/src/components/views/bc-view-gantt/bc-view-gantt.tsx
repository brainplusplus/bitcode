import { Component, Prop, State, Element, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import Gantt from 'frappe-gantt';

@Component({ tag: 'bc-view-gantt', styleUrl: 'bc-view-gantt.css', shadow: false })
export class BcViewGantt {
  @Element() el!: HTMLElement;
  @Prop() model: string = '';
  @Prop() viewTitle: string = '';
  @Prop() fields: string = '[]';
  @Prop() config: string = '{}';
  @State() loading: boolean = false;

  async componentDidLoad() {
    const container = this.el.querySelector('.bc-gantt-container') as HTMLElement;
    if (!container) return;
    let tasks: Array<{id: string; name: string; start: string; end: string; progress: number}> = [];
    if (this.model) {
      this.loading = true;
      try {
        const api = getApiClient();
        const res = await api.list(this.model, { pageSize: 100 });
        tasks = res.data.map(r => ({
          id: String(r['id'] || Math.random().toString(36).slice(2)),
          name: String(r['name'] || r['title'] || ''),
          start: String(r['start_date'] || r['start'] || new Date().toISOString().split('T')[0]),
          end: String(r['end_date'] || r['end'] || new Date(Date.now() + 7 * 86400000).toISOString().split('T')[0]),
          progress: Number(r['progress'] || 0),
        }));
      } catch {}
      this.loading = false;
    }
    if (tasks.length === 0) {
      tasks = [{ id: '1', name: 'No tasks found', start: new Date().toISOString().split('T')[0], end: new Date(Date.now() + 7 * 86400000).toISOString().split('T')[0], progress: 0 }];
    }
    const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    container.appendChild(svg);
    new Gantt(svg, tasks, {
      view_mode: 'Week',
      on_click: (task: {id: string}) => { console.log('Gantt task clicked:', task.id); },
      on_date_change: (task: {id: string}, start: Date, end: Date) => { console.log('Date changed:', task.id, start, end); },
      on_progress_change: (task: {id: string}, progress: number) => { console.log('Progress:', task.id, progress); },
    });
  }

  render() {
    return (
      <div class="bc-view bc-view-gantt">
        <div class="bc-gantt-header"><h2>{this.viewTitle || 'Gantt Chart'}</h2></div>
        {this.loading && <div class="bc-gantt-loading">Loading tasks...</div>}
        <div class="bc-gantt-container"></div>
      </div>
    );
  }
}

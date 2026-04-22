import { Component, Prop, State, Event, EventEmitter, h } from '@stencil/core';

interface Activity { id: string; type: string; summary: string; dueDate: string; done: boolean; assignee: string; }

@Component({ tag: 'bc-activity', styleUrl: 'bc-activity.css', shadow: false })
export class BcActivity {
  @Prop() recordId: string = '';
  @Prop() model: string = '';
  @State() activities: Activity[] = [];
  @State() showForm: boolean = false;
  @State() newType: string = 'call';
  @State() newSummary: string = '';
  @State() newDueDate: string = '';
  @Event() lcActivitySchedule!: EventEmitter<{type: string; summary: string; dueDate: string}>;

  private schedule() {
    if (!this.newSummary.trim()) return;
    const act: Activity = {
      id: Date.now().toString(36), type: this.newType, summary: this.newSummary,
      dueDate: this.newDueDate || new Date().toISOString().split('T')[0], done: false, assignee: 'You',
    };
    this.activities = [act, ...this.activities];
    this.lcActivitySchedule.emit({ type: this.newType, summary: this.newSummary, dueDate: this.newDueDate });
    this.newSummary = ''; this.newDueDate = ''; this.showForm = false;
  }

  private toggleDone(id: string) {
    this.activities = this.activities.map(a => a.id === id ? { ...a, done: !a.done } : a);
  }

  private typeIcon(type: string): string {
    switch (type) { case 'call': return '\u260E'; case 'meeting': return '\uD83D\uDCC5'; case 'email': return '\u2709'; case 'todo': return '\u2611'; default: return '\u25CF'; }
  }

  render() {
    const pending = this.activities.filter(a => !a.done);
    const done = this.activities.filter(a => a.done);
    return (
      <div class="bc-activity-widget">
        <div class="bc-aw-header">
          <h4>Activities</h4>
          <button type="button" class="bc-aw-schedule-btn" onClick={() => { this.showForm = !this.showForm; }}>+ Schedule</button>
        </div>
        {this.showForm && (
          <div class="bc-aw-form">
            <select class="bc-aw-select" onChange={(e) => { this.newType = (e.target as HTMLSelectElement).value; }}>
              <option value="call">Call</option><option value="meeting">Meeting</option><option value="email">Email</option><option value="todo">To-Do</option>
            </select>
            <input type="text" class="bc-aw-input" placeholder="Summary..." value={this.newSummary} onInput={(e: Event) => { this.newSummary = (e.target as HTMLInputElement).value; }} />
            <input type="date" class="bc-aw-date" value={this.newDueDate} onInput={(e: Event) => { this.newDueDate = (e.target as HTMLInputElement).value; }} />
            <button type="button" class="bc-aw-add" onClick={() => this.schedule()}>Add</button>
          </div>
        )}
        {pending.length > 0 && <div class="bc-aw-section-label">Planned</div>}
        {pending.map(a => (
          <div class="bc-aw-item">
            <span class="bc-aw-icon">{this.typeIcon(a.type)}</span>
            <div class="bc-aw-item-body">
              <span class="bc-aw-summary">{a.summary}</span>
              <span class="bc-aw-due">{a.dueDate} - {a.assignee}</span>
            </div>
            <button type="button" class="bc-aw-done-btn" onClick={() => this.toggleDone(a.id)}>{'\u2713'}</button>
          </div>
        ))}
        {done.length > 0 && <div class="bc-aw-section-label">Done</div>}
        {done.map(a => (
          <div class="bc-aw-item done">
            <span class="bc-aw-icon">{this.typeIcon(a.type)}</span>
            <div class="bc-aw-item-body"><span class="bc-aw-summary">{a.summary}</span></div>
          </div>
        ))}
      </div>
    );
  }
}

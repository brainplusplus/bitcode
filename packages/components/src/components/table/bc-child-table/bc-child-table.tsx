import { Component, Prop, State, Event, EventEmitter, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';

interface ColDef { field: string; width?: number; readonly?: boolean; type?: string; }

@Component({ tag: 'bc-child-table', styleUrl: 'bc-child-table.css', shadow: false })
export class BcChildTable {
  @Prop() field: string = '';
  @Prop() columns: string = '[]';
  @Prop({ mutable: true }) data: string = '[]';
  @Prop() summary: string = '{}';
  @Prop() readonly: boolean = false;
  @State() rows: Array<Record<string, unknown>> = [];
  @State() editingCell: { row: number; col: string } | null = null;
  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  private getCols(): ColDef[] { try { return JSON.parse(this.columns); } catch { return []; } }
  private getSummary(): Record<string, string> { try { return JSON.parse(this.summary); } catch { return {}; } }

  componentWillLoad() { try { this.rows = JSON.parse(this.data); } catch { this.rows = []; } }

  private emitChange() {
    const old = this.data;
    this.data = JSON.stringify(this.rows);
    this.lcFieldChange.emit({ name: this.field, value: this.rows, oldValue: old });
  }

  private addRow() {
    const cols = this.getCols();
    const newRow: Record<string, unknown> = { _id: Date.now().toString(36) };
    cols.forEach(c => { newRow[c.field] = ''; });
    this.rows = [...this.rows, newRow];
    this.emitChange();
  }

  private deleteRow(idx: number) {
    this.rows = this.rows.filter((_, i) => i !== idx);
    this.emitChange();
  }

  private updateCell(rowIdx: number, field: string, value: string) {
    const updated = [...this.rows];
    updated[rowIdx] = { ...updated[rowIdx], [field]: value };
    this.rows = updated;
    this.emitChange();
  }

  private computeSummary(field: string, fn: string): string {
    const vals = this.rows.map(r => Number(r[field]) || 0);
    if (vals.length === 0) return '';
    switch (fn) {
      case 'sum': return vals.reduce((a, b) => a + b, 0).toLocaleString();
      case 'avg': return (vals.reduce((a, b) => a + b, 0) / vals.length).toLocaleString(undefined, { maximumFractionDigits: 2 });
      case 'count': return String(vals.length);
      case 'min': return Math.min(...vals).toLocaleString();
      case 'max': return Math.max(...vals).toLocaleString();
      default: return '';
    }
  }

  private moveRow(from: number, to: number) {
    if (to < 0 || to >= this.rows.length) return;
    const updated = [...this.rows];
    const [moved] = updated.splice(from, 1);
    updated.splice(to, 0, moved);
    this.rows = updated;
    this.emitChange();
  }

  render() {
    const cols = this.getCols();
    const summaryDef = this.getSummary();
    const hasSummary = Object.keys(summaryDef).length > 0;
    return (
      <div class="bc-child-table">
        <table>
          <thead><tr>
            {!this.readonly && <th class="bc-ct-action">#</th>}
            {cols.map(c => <th style={{ width: c.width ? (c.width / 12 * 100) + '%' : 'auto' }}>{c.field}</th>)}
            {!this.readonly && <th class="bc-ct-action"></th>}
          </tr></thead>
          <tbody>
            {this.rows.map((row, ri) => (
              <tr>
                {!this.readonly && <td class="bc-ct-action">
                  <div class="bc-ct-reorder">
                    <button type="button" class="bc-ct-move" onClick={() => this.moveRow(ri, ri - 1)} disabled={ri === 0}>{'\u25B2'}</button>
                    <button type="button" class="bc-ct-move" onClick={() => this.moveRow(ri, ri + 1)} disabled={ri === this.rows.length - 1}>{'\u25BC'}</button>
                  </div>
                </td>}
                {cols.map(c => (
                  <td class={{'bc-ct-editing': this.editingCell?.row === ri && this.editingCell?.col === c.field}} onClick={() => { if (!this.readonly && !c.readonly) this.editingCell = { row: ri, col: c.field }; }}>
                    {this.editingCell?.row === ri && this.editingCell?.col === c.field ? (
                      <input type="text" class="bc-ct-input" value={String(row[c.field] ?? '')} autoFocus onInput={(e: Event) => this.updateCell(ri, c.field, (e.target as HTMLInputElement).value)} onBlur={() => { this.editingCell = null; }} onKeyDown={(e: KeyboardEvent) => { if (e.key === 'Enter' || e.key === 'Tab') this.editingCell = null; }} />
                    ) : (
                      <span class="bc-ct-cell-value">{String(row[c.field] ?? '')}</span>
                    )}
                  </td>
                ))}
                {!this.readonly && <td class="bc-ct-action">
                  <button type="button" class="bc-ct-delete" onClick={() => this.deleteRow(ri)}>{'\u00D7'}</button>
                </td>}
              </tr>
            ))}
            {!this.readonly && (
              <tr class="bc-ct-add-row">
                <td colSpan={cols.length + (this.readonly ? 0 : 2)}>
                  <button type="button" class="bc-ct-add-btn" onClick={() => this.addRow()}>+ Add Row</button>
                </td>
              </tr>
            )}
          </tbody>
          {hasSummary && (
            <tfoot><tr class="bc-ct-summary">
              {!this.readonly && <td></td>}
              {cols.map(c => (
                <td class="bc-ct-summary-cell">{summaryDef[c.field] ? this.computeSummary(c.field, summaryDef[c.field]) : ''}</td>
              ))}
              {!this.readonly && <td></td>}
            </tr></tfoot>
          )}
        </table>
      </div>
    );
  }
}

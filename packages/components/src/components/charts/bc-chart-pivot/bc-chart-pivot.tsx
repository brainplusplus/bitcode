import { Component, Prop, State, Method, h } from '@stencil/core';

@Component({ tag: 'bc-chart-pivot', styleUrl: 'bc-chart-pivot.css', shadow: false })
export class BcChartPivot {
  @Prop({ mutable: true }) data: string = '[]';

  @Method() async updateData(newData: unknown): Promise<void> { this.data = typeof newData === 'string' ? newData : JSON.stringify(newData); this.computePivot(); }
  @Method() async refresh(): Promise<void> { this.computePivot(); }
  @Prop() rows: string = '';
  @Prop() cols: string = '';
  @Prop() valueField: string = 'value';
  @Prop() aggFunc: string = 'sum';
  @State() pivotData: Map<string, Map<string, number>> = new Map();
  @State() rowKeys: string[] = [];
  @State() colKeys: string[] = [];

  componentWillLoad() { this.computePivot(); }

  private computePivot() {
    let parsed: Array<Record<string, unknown>> = [];
    try { parsed = JSON.parse(this.data); } catch { return; }
    const rowField = this.rows;
    const colField = this.cols;
    if (!rowField || !colField) return;
    const pivot = new Map<string, Map<string, number>>();
    const colSet = new Set<string>();
    for (const row of parsed) {
      const rk = String(row[rowField] || 'Other');
      const ck = String(row[colField] || 'Other');
      const val = Number(row[this.valueField] || 0);
      colSet.add(ck);
      if (!pivot.has(rk)) pivot.set(rk, new Map());
      const existing = pivot.get(rk)!.get(ck) || 0;
      pivot.get(rk)!.set(ck, existing + val);
    }
    this.pivotData = pivot;
    this.rowKeys = Array.from(pivot.keys()).sort();
    this.colKeys = Array.from(colSet).sort();
  }

  render() {
    return (
      <div class="bc-pivot">
        <table class="bc-pivot-table">
          <thead><tr>
            <th>{this.rows}</th>
            {this.colKeys.map(ck => <th>{ck}</th>)}
            <th class="bc-pivot-total">Total</th>
          </tr></thead>
          <tbody>
            {this.rowKeys.map(rk => {
              const rowMap = this.pivotData.get(rk)!;
              const rowTotal = this.colKeys.reduce((s, ck) => s + (rowMap.get(ck) || 0), 0);
              return (
                <tr>
                  <td class="bc-pivot-row-header">{rk}</td>
                  {this.colKeys.map(ck => <td class="bc-pivot-cell">{(rowMap.get(ck) || 0).toLocaleString()}</td>)}
                  <td class="bc-pivot-total">{rowTotal.toLocaleString()}</td>
                </tr>
              );
            })}
          </tbody>
          <tfoot><tr class="bc-pivot-footer">
            <td>Total</td>
            {this.colKeys.map(ck => {
              const colTotal = this.rowKeys.reduce((s, rk) => s + (this.pivotData.get(rk)?.get(ck) || 0), 0);
              return <td class="bc-pivot-cell">{colTotal.toLocaleString()}</td>;
            })}
            <td class="bc-pivot-total">{this.rowKeys.reduce((s, rk) => s + this.colKeys.reduce((s2, ck) => s2 + (this.pivotData.get(rk)?.get(ck) || 0), 0), 0).toLocaleString()}</td>
          </tr></tfoot>
        </table>
      </div>
    );
  }
}

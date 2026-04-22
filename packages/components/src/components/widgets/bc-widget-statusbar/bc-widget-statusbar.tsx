import { Component, Prop, h } from '@stencil/core';

@Component({
  tag: 'bc-widget-statusbar',
  styleUrl: 'bc-widget-statusbar.css',
  shadow: true,
})
export class BcWidgetStatusbar {
  @Prop() states: string = '[]';
  @Prop() value: string = '';

  private getStates(): string[] { try { return JSON.parse(this.states); } catch { return []; } }
  render() {
    const sts = this.getStates();
    return (<div class="bc-statusbar">{sts.map(s => <span class={{'step': true, 'active': s === this.value, 'done': sts.indexOf(s) < sts.indexOf(this.value)}}>{s}</span>)}</div>);
  }
}

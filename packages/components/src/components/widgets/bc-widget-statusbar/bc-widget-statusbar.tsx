import { Component, Prop, Method, h } from '@stencil/core';

@Component({
  tag: 'bc-widget-statusbar',
  styleUrl: 'bc-widget-statusbar.css',
  shadow: false,
})
export class BcWidgetStatusbar {
  @Prop() states: string = '[]';
  @Prop({ mutable: true }) value: string = '';

  private getStates(): string[] { try { return JSON.parse(this.states); } catch { return []; } }  @Method() async setValue(value: unknown): Promise<void> { this.value = String(value ?? ''); }
  @Method() async getValue(): Promise<unknown> { return this.value; }

  render() {
    const sts = this.getStates();
    return (<div class="bc-statusbar">{sts.map(s => <span class={{'step': true, 'active': s === this.value, 'done': sts.indexOf(s) < sts.indexOf(this.value)}}>{s}</span>)}</div>);
  }
}




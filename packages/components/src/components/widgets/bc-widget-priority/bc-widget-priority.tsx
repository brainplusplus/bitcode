import { Component, Prop, h, Event, EventEmitter } from '@stencil/core';

@Component({
  tag: 'bc-widget-priority',
  styleUrl: 'bc-widget-priority.css',
  shadow: true,
})
export class BcWidgetPriority {
  @Prop({ mutable: true }) value: number = 0;
  @Prop() max: number = 3;
  @Prop() disabled: boolean = false;
  @Event() lcFieldChange!: EventEmitter<{name: string; value: unknown; oldValue: unknown}>;

  private handleClick(v: number) { if (this.disabled) return; const old = this.value; this.value = v; this.lcFieldChange.emit({name:'priority',value:v,oldValue:old}); }
  render() {
    return (<div class="bc-priority">{Array.from({length:this.max},(_,i)=>i+1).map(i => <button type="button" class={{'star':true,'filled':i<=this.value}} disabled={this.disabled} onClick={()=>this.handleClick(i)}>{i<=this.value?'\u2605':'\u2606'}</button>)}</div>);
  }
}

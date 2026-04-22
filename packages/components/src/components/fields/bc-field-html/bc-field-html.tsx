import { Component, Prop, Event, EventEmitter, State, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';

@Component({
  tag: 'bc-field-html',
  styleUrl: 'bc-field-html.css',
  shadow: false,
})
export class BcFieldHtml {
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = '';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() language: string = '';
  @Prop() toolbar: string = 'full';

  @State() showSource: boolean = false;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  private handleInput(e: Event) {
    const old = this.value;
    this.value = (e.target as HTMLTextAreaElement).value;
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
  }

  render() {
    return (
      <div class="bc-field bc-html-wrap">
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}</label>}
        <div class="bc-html-tabs">
          <button type="button" class={{'bc-html-tab':true,'active':!this.showSource}} onClick={()=>{this.showSource=false;}}>Preview</button>
          <button type="button" class={{'bc-html-tab':true,'active':this.showSource}} onClick={()=>{this.showSource=true;}}>Source</button>
        </div>
        {this.showSource ? (
          <textarea class="bc-html-source" value={this.value} readOnly={this.readonly} disabled={this.disabled} onInput={(e:Event)=>this.handleInput(e)}></textarea>
        ) : (
          <div class="bc-html-preview" innerHTML={this.value}></div>
        )}
      </div>
    );
  }
}

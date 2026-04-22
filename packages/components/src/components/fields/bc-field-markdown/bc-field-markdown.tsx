import { Component, Prop, Event, EventEmitter, State, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';
import MarkdownIt from 'markdown-it';

@Component({
  tag: 'bc-field-markdown',
  styleUrl: 'bc-field-markdown.css',
  shadow: false,
})
export class BcFieldMarkdown {
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = 'Write markdown...';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() language: string = '';
  @Prop() toolbar: string = 'full';

  @State() showPreview: boolean = false;

  private md = new MarkdownIt({ html: true, linkify: true, typographer: true });

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  private handleInput(e: Event) {
    const target = e.target as HTMLTextAreaElement;
    const old = this.value;
    this.value = target.value;
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
  }

  render() {
    const rendered = this.md.render(this.value || '');
    return (
      <div class="bc-field bc-md-wrap">
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}</label>}
        <div class="bc-md-tabs">
          <button type="button" class={{'bc-md-tab':true,'active':!this.showPreview}} onClick={()=>{this.showPreview=false;}}>Write</button>
          <button type="button" class={{'bc-md-tab':true,'active':this.showPreview}} onClick={()=>{this.showPreview=true;}}>Preview</button>
        </div>
        {!this.showPreview ? (
          <textarea class="bc-md-editor" value={this.value} placeholder={this.placeholder} required={this.required} readOnly={this.readonly} disabled={this.disabled} onInput={(e:Event)=>this.handleInput(e)}></textarea>
        ) : (
          <div class="bc-md-preview" innerHTML={rendered}></div>
        )}
      </div>
    );
  }
}

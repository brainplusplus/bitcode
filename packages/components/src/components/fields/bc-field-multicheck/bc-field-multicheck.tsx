import { Component, Prop, Event, EventEmitter, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';

@Component({
  tag: 'bc-field-multicheck',
  styleUrl: 'bc-field-multicheck.css',
  shadow: false,
})
export class BcFieldMulticheck {
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '[]';
  @Prop() options: string = '[]';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  private getOptions(): string[] { try { return JSON.parse(this.options); } catch { return []; } }
  private getValues(): string[] { try { return JSON.parse(this.value); } catch { return []; } }

  private toggle(opt: string) {
    if (this.readonly || this.disabled) return;
    const old = this.value;
    const vals = this.getValues();
    const idx = vals.indexOf(opt);
    if (idx >= 0) vals.splice(idx, 1); else vals.push(opt);
    this.value = JSON.stringify(vals);
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
  }

  render() {
    const opts = this.getOptions();
    const vals = this.getValues();
    return (
      <div class="bc-field bc-multicheck-wrap">
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}</label>}
        <div class="bc-multicheck-list">
          {opts.map(opt => (
            <label class="bc-multicheck-item">
              <input type="checkbox" checked={vals.includes(opt)} disabled={this.disabled} onChange={() => this.toggle(opt)} />
              <span>{opt}</span>
            </label>
          ))}
        </div>
      </div>
    );
  }
}

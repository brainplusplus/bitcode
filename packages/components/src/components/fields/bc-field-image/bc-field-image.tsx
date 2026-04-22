import { Component, Prop, Event, EventEmitter, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';

@Component({
  tag: 'bc-field-image',
  styleUrl: 'bc-field-image.css',
  shadow: true,
})
export class BcFieldImage {
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() accept: string = 'image/*';
  @Prop() maxSize: string = '10MB';
  @Prop() required: boolean = false;
  @Prop() disabled: boolean = false;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  private handleChange(e: Event) {
    const target = e.target as HTMLInputElement;
    const file = target.files?.[0];
    if (file) {
      const oldValue = this.value;
      this.value = file.name;
      this.lcFieldChange.emit({ name: this.name, value: file, oldValue });
    }
  }

  render() {
    return (
      <div class="bc-field">
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}</label>}
        <div class="bc-file-wrapper">
          <input type="file" accept={this.accept} disabled={this.disabled} onChange={(e) => this.handleChange(e)} />
          {this.value && <span class="bc-file-name">{this.value}</span>}
        </div>
      </div>
    );
  }
}

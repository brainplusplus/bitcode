import { Component, Prop, State, Event, EventEmitter, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';

@Component({
  tag: 'bc-field-string',
  styleUrl: 'bc-field-string.css',
  shadow: true,
})
export class BcFieldString {
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = '';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() max: number = 0;
  @Prop() widget: string = '';

  @State() previewSrc: string = '';

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  private debounceTimer: ReturnType<typeof setTimeout> | null = null;

  private handleInput(e: Event) {
    const target = e.target as HTMLInputElement;
    const oldValue = this.value;
    this.value = target.value;
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue });

    if (this.isEmbedWidget()) {
      this.debouncePreview(this.value);
    }
  }

  private isEmbedWidget(): boolean {
    return ['youtube', 'instagram', 'tiktok'].includes(this.widget);
  }

  private debouncePreview(val: string) {
    if (this.debounceTimer) clearTimeout(this.debounceTimer);
    this.debounceTimer = setTimeout(() => {
      this.previewSrc = val.trim();
    }, 300);
  }

  private getPlaceholder(): string {
    if (this.placeholder) return this.placeholder;
    switch (this.widget) {
      case 'youtube': return 'YouTube URL or video ID';
      case 'instagram': return 'Instagram post URL';
      case 'tiktok': return 'TikTok video URL';
      default: return '';
    }
  }

  private renderEmbedPreview() {
    if (!this.previewSrc && !this.value) return null;
    const src = this.previewSrc || this.value;

    switch (this.widget) {
      case 'youtube':
        return <bc-viewer-youtube src={src} />;
      case 'instagram':
        return <bc-viewer-instagram src={src} />;
      case 'tiktok':
        return <bc-viewer-tiktok src={src} />;
      default:
        return null;
    }
  }

  render() {
    return (
      <div class="bc-field">
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}</label>}
        <input
          type="text"
          class="bc-field-input"
          value={this.value}
          placeholder={this.getPlaceholder()}
          required={this.required}
          readOnly={this.readonly}
          disabled={this.disabled}
          onInput={(e: Event) => this.handleInput(e)}
        />
        {this.isEmbedWidget() && (this.previewSrc || this.value) && (
          <div class="bc-field-embed-preview">
            {this.renderEmbedPreview()}
          </div>
        )}
      </div>
    );
  }
}

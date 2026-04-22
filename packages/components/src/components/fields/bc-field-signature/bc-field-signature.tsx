import { Component, Prop, Event, EventEmitter, Element, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';
import SignaturePad from 'signature_pad';

@Component({
  tag: 'bc-field-signature',
  styleUrl: 'bc-field-signature.css',
  shadow: false,
})
export class BcFieldSignature {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() disabled: boolean = false;
  @Prop() width: number = 400;
  @Prop() height: number = 200;

  private pad: SignaturePad | null = null;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  componentDidLoad() {
    const canvas = this.el.querySelector('canvas') as HTMLCanvasElement;
    if (!canvas) return;
    canvas.width = this.width;
    canvas.height = this.height;
    this.pad = new SignaturePad(canvas, { backgroundColor: 'rgb(255,255,255)' });
    if (this.value) { this.pad.fromDataURL(this.value); }
    if (this.disabled) { this.pad.off(); }
    this.pad.addEventListener('endStroke', () => {
      const old = this.value;
      this.value = this.pad!.toDataURL();
      this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
    });
  }

  disconnectedCallback() { this.pad?.off(); }

  private clear() {
    if (!this.pad) return;
    this.pad.clear();
    const old = this.value;
    this.value = '';
    this.lcFieldChange.emit({ name: this.name, value: '', oldValue: old });
  }

  render() {
    return (
      <div class="bc-field bc-sig-wrap">
        {this.label && <label class="bc-field-label">{this.label}</label>}
        <div class="bc-sig-canvas-wrap">
          <canvas></canvas>
        </div>
        {!this.disabled && <button type="button" class="bc-sig-clear" onClick={() => this.clear()}>Clear</button>}
      </div>
    );
  }
}

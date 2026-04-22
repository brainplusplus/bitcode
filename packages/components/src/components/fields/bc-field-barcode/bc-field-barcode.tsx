import { Component, Prop, Event, EventEmitter, Element, Watch, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';
import { i18n } from '../../../core/i18n';
import JsBarcode from 'jsbarcode';
import QRCode from 'qrcode';

@Component({
  tag: 'bc-field-barcode',
  styleUrl: 'bc-field-barcode.css',
  shadow: false,
})
export class BcFieldBarcode {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() format: string = 'code128';
  @Prop() disabled: boolean = false;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  componentDidLoad() { this.renderBarcode(); }

  @Watch('value')
  onValueChange() { this.renderBarcode(); }

  private async renderBarcode() {
    if (!this.value) return;
    const container = this.el.querySelector('.bc-barcode-display');
    if (!container) return;
    container.innerHTML = '';
    if (this.format === 'qr') {
      try {
        const url = await QRCode.toDataURL(this.value, { width: 150, margin: 1 });
        const img = document.createElement('img');
        img.src = url;
        img.alt = this.value;
        container.appendChild(img);
      } catch { container.textContent = i18n.t('barcode.qrError'); }
    } else {
      const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
      container.appendChild(svg);
      try { JsBarcode(svg, this.value, { format: this.format.toUpperCase(), height: 60, displayValue: true, fontSize: 12 }); }
      catch { container.textContent = i18n.t('barcode.barcodeError'); }
    }
  }

  private handleInput(e: Event) {
    const old = this.value;
    this.value = (e.target as HTMLInputElement).value;
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
  }

  render() {
    return (
      <div class="bc-field bc-barcode-wrap">
        {this.label && <label class="bc-field-label">{this.label}</label>}
        <input type="text" class="bc-field-input" value={this.value} disabled={this.disabled} placeholder={i18n.t('barcode.placeholder')} onInput={(e: Event) => this.handleInput(e)} />
        <div class="bc-barcode-display"></div>
      </div>
    );
  }
}

import { Component, Prop, Event, EventEmitter, Element, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';
import * as L from 'leaflet';

@Component({
  tag: 'bc-field-geo',
  styleUrl: 'bc-field-geo.css',
  shadow: false,
})
export class BcFieldGeo {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() disabled: boolean = false;
  @Prop() drawMode: string = 'point';
  @Prop() zoom: number = 13;

  private map: L.Map | null = null;
  private marker: L.Marker | null = null;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  componentDidLoad() {
    const container = this.el.querySelector('.bc-map-container') as HTMLElement;
    if (!container) return;

    let lat = -6.2088, lng = 106.8456;
    if (this.value) {
      try {
        const parsed = JSON.parse(this.value);
        lat = parsed.lat || lat;
        lng = parsed.lng || lng;
      } catch {}
    }

    this.map = L.map(container).setView([lat, lng], this.zoom);
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '&copy; OpenStreetMap contributors',
    }).addTo(this.map);

    if (this.value) {
      this.marker = L.marker([lat, lng]).addTo(this.map);
    }

    if (!this.disabled && this.drawMode === 'point') {
      this.map.on('click', (e: L.LeafletMouseEvent) => {
        const old = this.value;
        const newVal = JSON.stringify({ lat: e.latlng.lat, lng: e.latlng.lng });
        this.value = newVal;
        if (this.marker) this.marker.setLatLng(e.latlng);
        else this.marker = L.marker(e.latlng).addTo(this.map!);
        this.lcFieldChange.emit({ name: this.name, value: newVal, oldValue: old });
      });
    }
  }

  disconnectedCallback() { this.map?.remove(); }

  render() {
    return (
      <div class="bc-field bc-geo-wrap">
        {this.label && <label class="bc-field-label">{this.label}</label>}
        <div class="bc-map-container"></div>
        {this.value && <div class="bc-geo-coords">{this.value}</div>}
      </div>
    );
  }
}

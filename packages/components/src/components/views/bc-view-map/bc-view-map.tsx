import { Component, Method, Prop, State, Element, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { i18n } from '../../../core/i18n';
import * as L from 'leaflet';

@Component({ tag: 'bc-view-map', styleUrl: 'bc-view-map.css', shadow: false })
export class BcViewMap {
  @Element() el!: HTMLElement;
  @Prop() model: string = '';
  @Prop() viewTitle: string = '';
  @Prop() fields: string = '[]';
  @Prop() config: string = '{}';
  @Prop() geoField: string = 'location';
  @Prop() nameField: string = 'name';
  @State() loading: boolean = false;
  @State() recordCount: number = 0;
  private map: L.Map | null = null;

  componentWillRender() { this.el.dir = i18n.dir; }

  async componentDidLoad() {
    const container = this.el.querySelector('.bc-map-view-container') as HTMLElement;
    if (!container) return;
    this.map = L.map(container).setView([-6.2088, 106.8456], 10);
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '© OpenStreetMap contributors',
    }).addTo(this.map);
    if (this.model) {
      this.loading = true;
      try {
        const api = getApiClient();
        const res = await api.list(this.model, { pageSize: 500 });
        const bounds: L.LatLng[] = [];
        for (const row of res.data) {
          const geo = row[this.geoField];
          if (geo && typeof geo === 'object') {
            const g = geo as Record<string, number>;
            if (g['lat'] && g['lng']) {
              const ll = L.latLng(g['lat'], g['lng']);
              bounds.push(ll);
              const name = String(row[this.nameField] || row['id'] || '');
              L.marker(ll).addTo(this.map!).bindPopup('<b>' + name + '</b>');
              this.recordCount++;
            }
          }
        }
        if (bounds.length > 0) this.map!.fitBounds(L.latLngBounds(bounds), { padding: [50, 50] });
      } catch {}
      this.loading = false;
    }
  }

  disconnectedCallback() { this.map?.remove(); }  @Method() async refresh(): Promise<void> { }

  render() {
    return (
      <div class="bc-view bc-view-map">
        <div class="bc-map-header">
          <h2>{this.viewTitle || i18n.t('map.title')}</h2>
          <span class="bc-map-count">{i18n.plural('map.locations', this.recordCount)}</span>
        </div>
        {this.loading && <div class="bc-map-loading">{i18n.t('map.loadingLocations')}</div>}
        <div class="bc-map-view-container"></div>
      </div>
    );
  }
}


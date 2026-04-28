import { Component, Method, Prop, State, Element, h } from '@stencil/core';
import { i18n } from '../../../core/i18n';
import { Calendar } from '@fullcalendar/core';
import dayGridPlugin from '@fullcalendar/daygrid';
import interactionPlugin from '@fullcalendar/interaction';
import { getApiClient } from '../../../core/api-client';

@Component({ tag: 'bc-view-calendar', styleUrl: 'bc-view-calendar.css', shadow: false })
export class BcViewCalendar {
  @Element() el!: HTMLElement;
  @Prop() model: string = '';
  @Prop() viewTitle: string = '';
  @Prop() fields: string = '[]';
  @Prop() config: string = '{}';
  @Prop() dateField: string = 'date';
  @Prop() titleField: string = 'name';
  @State() loading: boolean = false;
  private calendar: Calendar | null = null;

  componentWillRender() { this.el.dir = i18n.dir; }

  async componentDidLoad() {
    const container = this.el.querySelector('.bc-cal-container') as HTMLElement;
    if (!container) return;
    let events: Array<{title: string; start: string; id: string}> = [];
    if (this.model) {
      this.loading = true;
      try {
        const api = getApiClient();
        const res = await api.list(this.model, { pageSize: 200 });
        events = res.data.map(r => ({
          id: String(r['id'] || ''),
          title: String(r[this.titleField] || r['name'] || ''),
          start: String(r[this.dateField] || ''),
        }));
      } catch {}
      this.loading = false;
    }
    this.calendar = new Calendar(container, {
      plugins: [dayGridPlugin, interactionPlugin],
      initialView: 'dayGridMonth',
      events,
      headerToolbar: { left: 'prev,next today', center: 'title', right: 'dayGridMonth,dayGridWeek' },
      editable: true,
      selectable: true,
      height: 'auto',
      eventClick: (info) => { console.log('Event clicked:', info.event.id); },
      dateClick: (info) => { console.log('Date clicked:', info.dateStr); },
    });
    this.calendar.render();
  }

  disconnectedCallback() { this.calendar?.destroy(); }  @Method() async refresh(): Promise<void> { }

  render() {
    return (
      <div class="bc-view bc-view-calendar">
        {this.loading && <div class="bc-cal-loading">{i18n.t('calendar.loadingEvents')}</div>}
        <div class="bc-cal-container"></div>
      </div>
    );
  }
}


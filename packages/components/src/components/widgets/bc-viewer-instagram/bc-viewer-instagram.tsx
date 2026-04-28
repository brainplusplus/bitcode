import { Component, Prop, State, Watch, Element, Method, h } from '@stencil/core';

@Component({
  tag: 'bc-viewer-instagram',
  styleUrl: 'bc-viewer-instagram.css',
  shadow: false,
})
export class BcViewerInstagram {
  @Prop() src: string = '';
  @Prop() width: string = '400px';
  @Prop() captioned: boolean = true;

  @State() postUrl: string = '';
  @State() embedFailed: boolean = false;

  @Element() el!: HTMLElement;

  componentWillLoad() {
    this.parseSource();
  }

  @Watch('src')
  handleSrcChange() {
    this.parseSource();
    this.loadEmbed();
  }

  componentDidLoad() {
    this.loadEmbed();
  }

  private parseSource() {
    if (!this.src) {
      this.postUrl = '';
      return;
    }

    const src = this.src.trim();

    const match = src.match(/instagram\.com\/(p|reel|tv)\/([A-Za-z0-9_-]+)/);
    if (match) {
      this.postUrl = `https://www.instagram.com/${match[1]}/${match[2]}/`;
      return;
    }

    if (/^[A-Za-z0-9_-]{8,}$/.test(src)) {
      this.postUrl = `https://www.instagram.com/p/${src}/`;
      return;
    }

    this.postUrl = '';
  }

  private loadEmbed() {
    if (!this.postUrl) return;

    this.embedFailed = false;

    const timeout = setTimeout(() => {
      const blockquote = this.el.querySelector('.instagram-media');
      if (blockquote && !blockquote.querySelector('iframe')) {
        this.embedFailed = true;
      }
    }, 5000);

    const existingScript = document.querySelector('script[src*="instagram.com/embed.js"]');
    if (existingScript) {
      const win = window as unknown as Record<string, unknown>;
      if (win.instgrm && typeof (win.instgrm as Record<string, unknown>).Embeds === 'object') {
        const embeds = (win.instgrm as Record<string, Record<string, () => void>>).Embeds;
        if (typeof embeds.process === 'function') {
          setTimeout(() => embeds.process(), 100);
        }
      }
      return;
    }

    const script = document.createElement('script');
    script.src = 'https://www.instagram.com/embed.js';
    script.async = true;
    script.onload = () => clearTimeout(timeout);
    script.onerror = () => {
      clearTimeout(timeout);
      this.embedFailed = true;
    };
    document.body.appendChild(script);
  }  @Prop() loading: boolean = false;

  @Method() async refresh(): Promise<void> { }

  render() {
    if (!this.src) {
      return (
        <div class="bc-viewer-ig bc-viewer-empty">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <rect x="2" y="2" width="20" height="20" rx="5" ry="5" />
            <circle cx="12" cy="12" r="4" />
            <line x1="17.5" y1="6.5" x2="17.51" y2="6.5" />
          </svg>
          <span>No Instagram URL provided</span>
        </div>
      );
    }

    if (!this.postUrl) {
      return (
        <div class="bc-viewer-ig bc-viewer-empty">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <circle cx="12" cy="12" r="10" />
            <line x1="12" y1="8" x2="12" y2="12" />
            <line x1="12" y1="16" x2="12.01" y2="16" />
          </svg>
          <span>Invalid Instagram URL</span>
        </div>
      );
    }

    if (this.embedFailed) {
      return (
        <div class="bc-viewer-ig bc-viewer-fallback" style={{ maxWidth: this.width }}>
          <div class="bc-viewer-fallback-card">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <rect x="2" y="2" width="20" height="20" rx="5" ry="5" />
              <circle cx="12" cy="12" r="4" />
              <line x1="17.5" y1="6.5" x2="17.51" y2="6.5" />
            </svg>
            <span>Instagram embed unavailable</span>
            <a href={this.postUrl} target="_blank" rel="noopener noreferrer" class="bc-viewer-btn">
              Open in Instagram
            </a>
          </div>
        </div>
      );
    }

    return (
      <div class="bc-viewer-ig" style={{ maxWidth: this.width }}>
        <blockquote
          class="instagram-media"
          data-instgrm-permalink={this.postUrl}
          data-instgrm-version="14"
          data-instgrm-captioned={this.captioned ? '' : undefined}
          style={{
            background: '#FFF',
            border: '0',
            borderRadius: '3px',
            boxShadow: '0 0 1px 0 rgba(0,0,0,0.5), 0 1px 10px 0 rgba(0,0,0,0.15)',
            margin: '0',
            maxWidth: '540px',
            minWidth: '326px',
            padding: '0',
            width: '100%',
          }}
        >
          <a href={this.postUrl} target="_blank" rel="noopener noreferrer">
            View this post on Instagram
          </a>
        </blockquote>
      </div>
    );
  }
}


import { Component, Prop, State, Watch, Element, Method, h } from '@stencil/core';

@Component({
  tag: 'bc-viewer-tiktok',
  styleUrl: 'bc-viewer-tiktok.css',
  shadow: false,
})
export class BcViewerTiktok {
  @Prop() src: string = '';
  @Prop() width: string = '325px';

  @State() videoId: string = '';
  @State() videoUrl: string = '';
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
      this.videoId = '';
      this.videoUrl = '';
      return;
    }

    const src = this.src.trim();

    const fullMatch = src.match(/tiktok\.com\/@[^/]+\/video\/(\d+)/);
    if (fullMatch) {
      this.videoId = fullMatch[1];
      this.videoUrl = src;
      return;
    }

    const shortMatch = src.match(/vm\.tiktok\.com\/([A-Za-z0-9]+)/);
    if (shortMatch) {
      this.videoUrl = src;
      this.videoId = shortMatch[1];
      return;
    }

    if (/^\d{15,25}$/.test(src)) {
      this.videoId = src;
      this.videoUrl = `https://www.tiktok.com/video/${src}`;
      return;
    }

    this.videoId = '';
    this.videoUrl = '';
  }

  private loadEmbed() {
    if (!this.videoId) return;

    this.embedFailed = false;

    const timeout = setTimeout(() => {
      this.embedFailed = true;
    }, 8000);

    const existingScript = document.querySelector('script[src*="tiktok.com/embed.js"]');
    if (existingScript) {
      clearTimeout(timeout);
      return;
    }

    const script = document.createElement('script');
    script.src = 'https://www.tiktok.com/embed.js';
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
        <div class="bc-viewer-tt bc-viewer-empty">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M9 12a4 4 0 1 0 4 4V4a5 5 0 0 0 5 5" />
          </svg>
          <span>No TikTok URL provided</span>
        </div>
      );
    }

    if (!this.videoId) {
      return (
        <div class="bc-viewer-tt bc-viewer-empty">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <circle cx="12" cy="12" r="10" />
            <line x1="12" y1="8" x2="12" y2="12" />
            <line x1="12" y1="16" x2="12.01" y2="16" />
          </svg>
          <span>Invalid TikTok URL</span>
        </div>
      );
    }

    if (this.embedFailed) {
      return (
        <div class="bc-viewer-tt bc-viewer-fallback" style={{ maxWidth: this.width }}>
          <div class="bc-viewer-fallback-card">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M9 12a4 4 0 1 0 4 4V4a5 5 0 0 0 5 5" />
            </svg>
            <span>TikTok embed unavailable</span>
            <a href={this.videoUrl || this.src} target="_blank" rel="noopener noreferrer" class="bc-viewer-btn">
              Open in TikTok
            </a>
          </div>
        </div>
      );
    }

    return (
      <div class="bc-viewer-tt" style={{ maxWidth: this.width }}>
        <blockquote
          class="tiktok-embed"
          cite={this.videoUrl || this.src}
          data-video-id={this.videoId}
          style={{ maxWidth: '605px', minWidth: '325px' }}
        >
          <section>
            <a href={this.videoUrl || this.src} target="_blank" rel="noopener noreferrer">
              View on TikTok
            </a>
          </section>
        </blockquote>
      </div>
    );
  }
}


import { Component, Prop, State, Watch, h } from '@stencil/core';

@Component({
  tag: 'bc-viewer-youtube',
  styleUrl: 'bc-viewer-youtube.css',
  shadow: true,
})
export class BcViewerYoutube {
  @Prop() src: string = '';
  @Prop() width: string = '100%';
  @Prop() height: string = 'auto';
  @Prop() autoplay: boolean = false;
  @Prop() controls: boolean = true;
  @Prop() start: number = 0;

  @State() videoId: string = '';
  @State() isShort: boolean = false;

  componentWillLoad() {
    this.parseSource();
  }

  @Watch('src')
  handleSrcChange() {
    this.parseSource();
  }

  private parseSource() {
    if (!this.src) {
      this.videoId = '';
      this.isShort = false;
      return;
    }

    const src = this.src.trim();

    if (/^[a-zA-Z0-9_-]{11}$/.test(src)) {
      this.videoId = src;
      this.isShort = false;
      return;
    }

    try {
      const url = new URL(src);
      this.isShort = url.pathname.includes('/shorts/');

      if (url.hostname === 'youtu.be') {
        this.videoId = url.pathname.slice(1).split('/')[0];
        return;
      }

      if (url.hostname.includes('youtube.com') || url.hostname.includes('youtube-nocookie.com')) {
        if (url.pathname.includes('/embed/')) {
          this.videoId = url.pathname.split('/embed/')[1]?.split('/')[0] || '';
          return;
        }
        if (url.pathname.includes('/shorts/')) {
          this.videoId = url.pathname.split('/shorts/')[1]?.split('/')[0] || '';
          return;
        }
        if (url.pathname === '/watch') {
          this.videoId = url.searchParams.get('v') || '';
          return;
        }
        const vParam = url.searchParams.get('v');
        if (vParam) {
          this.videoId = vParam;
          return;
        }
      }
    } catch {
    }

    if (/^[a-zA-Z0-9_-]{10,12}$/.test(src)) {
      this.videoId = src;
      this.isShort = false;
    }
  }

  private getEmbedUrl(): string {
    if (!this.videoId) return '';
    const params = new URLSearchParams();
    if (this.autoplay) params.set('autoplay', '1');
    if (!this.controls) params.set('controls', '0');
    if (this.start > 0) params.set('start', String(this.start));
    params.set('rel', '0');
    const qs = params.toString();
    return `https://www.youtube.com/embed/${this.videoId}${qs ? '?' + qs : ''}`;
  }

  render() {
    if (!this.src) {
      return (
        <div class="bc-viewer-yt bc-viewer-empty">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <polygon points="5 3 19 12 5 21 5 3" />
          </svg>
          <span>No YouTube URL provided</span>
        </div>
      );
    }

    if (!this.videoId) {
      return (
        <div class="bc-viewer-yt bc-viewer-empty">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <circle cx="12" cy="12" r="10" />
            <line x1="12" y1="8" x2="12" y2="12" />
            <line x1="12" y1="16" x2="12.01" y2="16" />
          </svg>
          <span>Invalid YouTube URL or video ID</span>
        </div>
      );
    }

    return (
      <div
        class={{ 'bc-viewer-yt': true, 'bc-viewer-yt-short': this.isShort }}
        style={{ width: this.width }}
      >
        <div class="bc-viewer-yt-wrapper">
          <iframe
            class="bc-viewer-yt-frame"
            src={this.getEmbedUrl()}
            allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
            allowFullScreen
          />
        </div>
      </div>
    );
  }
}

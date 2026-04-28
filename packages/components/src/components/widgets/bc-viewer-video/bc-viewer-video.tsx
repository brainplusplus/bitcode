import { Component, Prop, State, Method, h } from '@stencil/core';
import { BcSetup } from '../../../core/bc-setup';

@Component({
  tag: 'bc-viewer-video',
  styleUrl: 'bc-viewer-video.css',
  shadow: false,
})
export class BcViewerVideo {
  @Prop({ mutable: true }) src: string = '';
  @Prop() type: string = '';
  @Prop() poster: string = '';
  @Prop() controls: boolean = true;
  @Prop() autoplay: boolean = false;
  @Prop() loop: boolean = false;
  @Prop() muted: boolean = false;
  @Prop() width: string = '100%';
  @Prop() height: string = 'auto';
  @Prop() download: boolean = true;
  @Prop({ mutable: true }) loading: boolean = false;
  @Prop() dataSource: string = '';
  @Prop() srcField: string = 'url';

  @State() isPlaying: boolean = false;
  @State() currentTime: number = 0;
  @State() duration: number = 0;
  @State() volume: number = 1;
  @State() isMuted: boolean = false;
  @State() isFullscreen: boolean = false;
  @State() loadError: boolean = false;

  private videoEl!: HTMLVideoElement;
  private containerEl!: HTMLDivElement;

  private detectMimeType(): string {
    if (this.type) return this.type;
    const ext = this.src.split('?')[0].split('.').pop()?.toLowerCase() || '';
    const mimeMap: Record<string, string> = {
      mp4: 'video/mp4',
      webm: 'video/webm',
      ogg: 'video/ogg',
      ogv: 'video/ogg',
    };
    return mimeMap[ext] || 'video/mp4';
  }

  private togglePlay() {
    if (!this.videoEl) return;
    if (this.videoEl.paused) {
      this.videoEl.play();
    } else {
      this.videoEl.pause();
    }
  }

  private handleTimeUpdate() {
    if (!this.videoEl) return;
    this.currentTime = this.videoEl.currentTime;
  }

  private handleLoadedMetadata() {
    if (!this.videoEl) return;
    this.duration = this.videoEl.duration;
    this.isMuted = this.videoEl.muted;
  }

  private handleSeek(e: Event) {
    const input = e.target as HTMLInputElement;
    if (this.videoEl) {
      this.videoEl.currentTime = parseFloat(input.value);
    }
  }

  private handleVolumeChange(e: Event) {
    const input = e.target as HTMLInputElement;
    const vol = parseFloat(input.value);
    if (this.videoEl) {
      this.videoEl.volume = vol;
      this.videoEl.muted = vol === 0;
    }
    this.volume = vol;
    this.isMuted = vol === 0;
  }

  private toggleMute() {
    if (!this.videoEl) return;
    this.videoEl.muted = !this.videoEl.muted;
    this.isMuted = this.videoEl.muted;
    if (!this.isMuted && this.volume === 0) {
      this.volume = 0.5;
      this.videoEl.volume = 0.5;
    }
  }

  private toggleFullscreen() {
    if (!this.containerEl) return;
    if (!document.fullscreenElement) {
      this.containerEl.requestFullscreen?.();
      this.isFullscreen = true;
    } else {
      document.exitFullscreen?.();
      this.isFullscreen = false;
    }
  }

  private handleDownload() {
    const a = document.createElement('a');
    a.href = this.src;
    a.download = this.src.split('/').pop() || 'video';
    a.target = '_blank';
    a.click();
  }

  private formatTime(seconds: number): string {
    if (!isFinite(seconds)) return '0:00';
    const m = Math.floor(seconds / 60);
    const s = Math.floor(seconds % 60);
    return `${m}:${s.toString().padStart(2, '0')}`;
  }

  componentDidLoad() { if (this.dataSource && !this.src) this._fetchSrc(); }
  private async _fetchSrc() { this.loading = true; try { const baseUrl = BcSetup.getBaseUrl(); const url = this.dataSource.startsWith('http') ? this.dataSource : baseUrl + this.dataSource; const res = await fetch(url, { headers: BcSetup.getHeaders() }); const json = await res.json(); this.src = String(json[this.srcField] || json.src || json.url || ''); } catch {} this.loading = false; }
  @Method() async refresh(): Promise<void> { if (this.dataSource) await this._fetchSrc(); }

  render() {
    if (!this.src) {
      return (
        <div class="bc-viewer-video bc-viewer-empty">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <polygon points="5 3 19 12 5 21 5 3" />
          </svg>
          <span>No video source provided</span>
        </div>
      );
    }

    return (
      <div
        class="bc-viewer-video"
        style={{ width: this.width }}
        ref={(el) => this.containerEl = el as HTMLDivElement}
      >
        <div class="bc-viewer-video-wrapper">
          {!this.loadError ? (
            <video
              ref={(el) => this.videoEl = el as HTMLVideoElement}
              poster={this.poster || undefined}
              autoplay={this.autoplay}
              loop={this.loop}
              muted={this.muted}
              playsinline
              onClick={() => this.togglePlay()}
              onTimeUpdate={() => this.handleTimeUpdate()}
              onLoadedMetaData={() => this.handleLoadedMetadata()}
              onPlay={() => { this.isPlaying = true; }}
              onPause={() => { this.isPlaying = false; }}
              onError={() => { this.loadError = true; }}
            >
              <source src={this.src} type={this.detectMimeType()} />
            </video>
          ) : (
            <div class="bc-viewer-video-error">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <circle cx="12" cy="12" r="10" />
                <line x1="15" y1="9" x2="9" y2="15" />
                <line x1="9" y1="9" x2="15" y2="15" />
              </svg>
              <span>Failed to load video</span>
            </div>
          )}
        </div>

        {this.controls && !this.loadError && (
          <div class="bc-viewer-video-controls">
            <button class="bc-control-btn" onClick={() => this.togglePlay()}>
              {this.isPlaying ? (
                <svg viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="4" width="4" height="16" /><rect x="14" y="4" width="4" height="16" /></svg>
              ) : (
                <svg viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3" /></svg>
              )}
            </button>

            <span class="bc-control-time">{this.formatTime(this.currentTime)}</span>

            <input
              type="range"
              class="bc-control-seek"
              min="0"
              max={this.duration || 0}
              value={this.currentTime}
              step="0.1"
              onInput={(e) => this.handleSeek(e)}
            />

            <span class="bc-control-time">{this.formatTime(this.duration)}</span>

            <button class="bc-control-btn" onClick={() => this.toggleMute()}>
              {this.isMuted ? (
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5" /><line x1="23" y1="9" x2="17" y2="15" /><line x1="17" y1="9" x2="23" y2="15" /></svg>
              ) : (
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5" /><path d="M19.07 4.93a10 10 0 0 1 0 14.14M15.54 8.46a5 5 0 0 1 0 7.07" /></svg>
              )}
            </button>

            <input
              type="range"
              class="bc-control-volume"
              min="0"
              max="1"
              value={this.isMuted ? 0 : this.volume}
              step="0.05"
              onInput={(e) => this.handleVolumeChange(e)}
            />

            {this.download && (
              <button class="bc-control-btn" onClick={() => this.handleDownload()} title="Download">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                  <polyline points="7 10 12 15 17 10" />
                  <line x1="12" y1="15" x2="12" y2="3" />
                </svg>
              </button>
            )}

            <button class="bc-control-btn" onClick={() => this.toggleFullscreen()}>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="15 3 21 3 21 9" /><polyline points="9 21 3 21 3 15" />
                <line x1="21" y1="3" x2="14" y2="10" /><line x1="3" y1="21" x2="10" y2="14" />
              </svg>
            </button>
          </div>
        )}
      </div>
    );
  }
}




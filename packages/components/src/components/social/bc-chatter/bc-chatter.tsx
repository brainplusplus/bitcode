import { Component, Prop, State, Event, EventEmitter, Element, h } from '@stencil/core';
import { i18n } from '../../../core/i18n';

interface Message { id: string; user: string; text: string; date: string; type: 'message' | 'note' | 'system'; }

@Component({ tag: 'bc-chatter', styleUrl: 'bc-chatter.css', shadow: false })
export class BcChatter {
  @Element() el!: HTMLElement;
  @Prop() recordId: string = '';
  @Prop() model: string = '';
  @State() messages: Message[] = [];
  @State() newMessage: string = '';
  @State() messageType: 'message' | 'note' = 'message';
  @Event() lcChatterSend!: EventEmitter<{text: string; type: string}>;

  componentWillRender() { this.el.dir = i18n.dir; }

  private handleSend() {
    if (!this.newMessage.trim()) return;
    const msg: Message = {
      id: Date.now().toString(36),
      user: 'You',
      text: this.newMessage,
      date: new Date().toISOString(),
      type: this.messageType,
    };
    this.messages = [msg, ...this.messages];
    this.lcChatterSend.emit({ text: this.newMessage, type: this.messageType });
    this.newMessage = '';
  }

  private formatDate(d: string): string {
    try { return i18n.tf.date(d, { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }); }
    catch { return d; }
  }

  render() {
    return (
      <div class="bc-chatter">
        <div class="bc-chatter-header">
          <h4>Messages</h4>
          <span class="bc-chatter-count">{this.messages.length}</span>
        </div>
        <div class="bc-chatter-compose">
          <div class="bc-chatter-tabs">
            <button type="button" class={{'bc-ch-tab': true, 'active': this.messageType === 'message'}} onClick={() => { this.messageType = 'message'; }}>Send Message</button>
            <button type="button" class={{'bc-ch-tab': true, 'active': this.messageType === 'note'}} onClick={() => { this.messageType = 'note'; }}>Log Note</button>
          </div>
          <textarea class="bc-chatter-input" placeholder={this.messageType === 'message' ? 'Write a message...' : 'Log an internal note...'} value={this.newMessage} onInput={(e: Event) => { this.newMessage = (e.target as HTMLTextAreaElement).value; }}></textarea>
          <button type="button" class="bc-chatter-send" onClick={() => this.handleSend()} disabled={!this.newMessage.trim()}>Send</button>
        </div>
        <div class="bc-chatter-thread">
          {this.messages.map(msg => (
            <div class={{'bc-chatter-msg': true, 'is-note': msg.type === 'note', 'is-system': msg.type === 'system'}}>
              <div class="bc-msg-avatar">{msg.user.charAt(0).toUpperCase()}</div>
              <div class="bc-msg-body">
                <div class="bc-msg-header">
                  <span class="bc-msg-user">{msg.user}</span>
                  <span class="bc-msg-date">{this.formatDate(msg.date)}</span>
                  {msg.type === 'note' && <span class="bc-msg-badge">Note</span>}
                </div>
                <div class="bc-msg-text">{msg.text}</div>
              </div>
            </div>
          ))}
          {this.messages.length === 0 && <div class="bc-chatter-empty">No messages yet</div>}
        </div>
      </div>
    );
  }
}

import { Component, Prop, Event, EventEmitter, Element, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';
import { Editor } from '@tiptap/core';
import StarterKit from '@tiptap/starter-kit';

@Component({
  tag: 'bc-field-richtext',
  styleUrl: 'bc-field-richtext.css',
  shadow: false,
})
export class BcFieldRichtext {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = 'Start typing...';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() toolbar: string = 'full';

  private editor: Editor | null = null;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  componentDidLoad() {
    const el = this.el.querySelector('.tiptap-editor') as HTMLElement;
    if (!el) return;
    this.editor = new Editor({
      element: el,
      extensions: [StarterKit],
      content: this.value,
      editable: !this.readonly && !this.disabled,
      onUpdate: ({ editor }) => {
        const old = this.value;
        this.value = editor.getHTML();
        this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
      },
    });
  }

  disconnectedCallback() { this.editor?.destroy(); }

  private cmd(c: string) {
    if (!this.editor) return;
    const chain = this.editor.chain().focus();
    switch(c) {
      case 'bold': chain.toggleBold().run(); break;
      case 'italic': chain.toggleItalic().run(); break;
      case 'strike': chain.toggleStrike().run(); break;
      case 'bulletList': chain.toggleBulletList().run(); break;
      case 'orderedList': chain.toggleOrderedList().run(); break;
      case 'blockquote': chain.toggleBlockquote().run(); break;
      case 'codeBlock': chain.toggleCodeBlock().run(); break;
      case 'undo': chain.undo().run(); break;
      case 'redo': chain.redo().run(); break;
    }
  }

  render() {
    const full = this.toolbar === 'full';
    return (
      <div class="bc-field bc-richtext-wrap">
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}</label>}
        {!this.readonly && !this.disabled && (
          <div class="bc-rt-toolbar">
            <button type="button" onClick={() => this.cmd('bold')}><b>B</b></button>
            <button type="button" onClick={() => this.cmd('italic')}><i>I</i></button>
            <button type="button" onClick={() => this.cmd('strike')}><s>S</s></button>
            {full && <button type="button" onClick={() => this.cmd('bulletList')}>•</button>}
            {full && <button type="button" onClick={() => this.cmd('orderedList')}>1.</button>}
            {full && <button type="button" onClick={() => this.cmd('blockquote')}>"</button>}
            {full && <button type="button" onClick={() => this.cmd('codeBlock')}>&lt;/&gt;</button>}
            <button type="button" onClick={() => this.cmd('undo')}>↩</button>
            <button type="button" onClick={() => this.cmd('redo')}>↪</button>
          </div>
        )}
        <div class="tiptap-editor"></div>
      </div>
    );
  }
}

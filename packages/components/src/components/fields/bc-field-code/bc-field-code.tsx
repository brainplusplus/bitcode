import { Component, Prop, Event, EventEmitter, Element, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';
import { EditorState } from '@codemirror/state';
import { EditorView } from '@codemirror/view';
import { basicSetup } from '@codemirror/basic-setup';
import { javascript } from '@codemirror/lang-javascript';
import { python } from '@codemirror/lang-python';
import { json as jsonLang } from '@codemirror/lang-json';
import { html as htmlLang } from '@codemirror/lang-html';
import { sql } from '@codemirror/lang-sql';

@Component({
  tag: 'bc-field-code',
  styleUrl: 'bc-field-code.css',
  shadow: false,
})
export class BcFieldCode {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = '';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() language: string = 'javascript';
  @Prop() toolbar: string = 'full';

  private view: EditorView | null = null;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  private getLangExtension() {
    switch (this.language) {
      case 'python': return python();
      case 'json': return jsonLang();
      case 'html': case 'xml': return htmlLang();
      case 'sql': return sql();
      default: return javascript();
    }
  }

  componentDidLoad() {
    const container = this.el.querySelector('.cm-container') as HTMLElement;
    if (!container) return;
    const state = EditorState.create({
      doc: this.value,
      extensions: [
        basicSetup,
        this.getLangExtension(),
        EditorView.editable.of(!this.readonly && !this.disabled),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            const old = this.value;
            this.value = update.state.doc.toString();
            this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
          }
        }),
      ],
    });
    this.view = new EditorView({ state, parent: container });
  }

  disconnectedCallback() { this.view?.destroy(); }

  render() {
    return (
      <div class="bc-field bc-code-wrap">
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}</label>}
        <div class="cm-container"></div>
      </div>
    );
  }
}

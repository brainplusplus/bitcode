import { Component, Prop, Event, EventEmitter, Element, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';
import { EditorState } from '@codemirror/state';
import { EditorView } from '@codemirror/view';
import { basicSetup } from '@codemirror/basic-setup';
import { json } from '@codemirror/lang-json';

@Component({
  tag: 'bc-field-json',
  styleUrl: 'bc-field-json.css',
  shadow: false,
})
export class BcFieldJson {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = '';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() language: string = '';
  @Prop() toolbar: string = 'full';

  private view: EditorView | null = null;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  componentDidLoad() {
    const container = this.el.querySelector('.cm-json-container') as HTMLElement;
    if (!container) return;
    const state = EditorState.create({
      doc: this.value || '{}',
      extensions: [
        basicSetup,
        json(),
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
      <div class="bc-field bc-json-wrap">
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}</label>}
        <div class="cm-json-container"></div>
      </div>
    );
  }
}

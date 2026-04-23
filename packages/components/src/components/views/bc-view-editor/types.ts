export interface EditorField {
  field: string;
  width: number;
  readonly?: boolean;
  widget?: string;
  formula?: string;
}

export interface EditorRow {
  id: string;
  fields: EditorField[];
}

export interface EditorSection {
  id: string;
  title: string;
  collapsible?: boolean;
  rows: EditorRow[];
}

export interface EditorTab {
  id: string;
  label: string;
  view?: string;
  fields?: string[];
}

export interface EditorLayout {
  sections: EditorSection[];
  tabs: EditorTab[];
  hasChatter: boolean;
}

export interface ModelFieldInfo {
  name: string;
  type: string;
  label?: string;
  required?: boolean;
}

let _idCounter = 0;
export function genId(): string {
  return 'ed_' + (++_idCounter) + '_' + Math.random().toString(36).slice(2, 6);
}

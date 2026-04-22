declare module 'frappe-gantt' {
  interface Task {
    id: string;
    name: string;
    start: string;
    end: string;
    progress: number;
    dependencies?: string;
    custom_class?: string;
  }

  interface GanttOptions {
    view_mode?: 'Quarter Day' | 'Half Day' | 'Day' | 'Week' | 'Month' | 'Year';
    date_format?: string;
    popup_trigger?: string;
    custom_popup_html?: (task: Task) => string;
    on_click?: (task: Task) => void;
    on_date_change?: (task: Task, start: Date, end: Date) => void;
    on_progress_change?: (task: Task, progress: number) => void;
    on_view_change?: (mode: string) => void;
  }

  class Gantt {
    constructor(element: SVGElement | string, tasks: Task[], options?: GanttOptions);
    change_view_mode(mode: string): void;
    refresh(tasks: Task[]): void;
  }

  export default Gantt;
}

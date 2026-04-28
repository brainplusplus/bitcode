import { Component, Method, Prop, State, Element, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { i18n } from '../../../core/i18n';

interface TreeNode { id: string; name: string; children: TreeNode[]; expanded: boolean; level: number; }

@Component({ tag: 'bc-view-tree', styleUrl: 'bc-view-tree.css', shadow: false })
export class BcViewTree {
  @Element() el!: HTMLElement;
  @Prop() model: string = '';
  @Prop() viewTitle: string = '';
  @Prop() fields: string = '[]';
  @Prop() config: string = '{}';
  @Prop() parentField: string = 'parent_id';
  @State() tree: TreeNode[] = [];
  @State() loading: boolean = false;

  componentWillRender() { this.el.dir = i18n.dir; }

  async componentDidLoad() {
    if (!this.model) return;
    this.loading = true;
    try {
      const api = getApiClient();
      const res = await api.list(this.model, { pageSize: 500 });
      this.tree = this.buildTree(res.data);
    } catch { this.tree = []; }
    this.loading = false;
  }

  private buildTree(data: Array<Record<string, unknown>>): TreeNode[] {
    const map = new Map<string, TreeNode>();
    const roots: TreeNode[] = [];
    for (const row of data) {
      const id = String(row['id'] || '');
      map.set(id, { id, name: String(row['name'] || id), children: [], expanded: false, level: 0 });
    }
    for (const row of data) {
      const id = String(row['id'] || '');
      const pid = String(row[this.parentField] || '');
      const node = map.get(id)!;
      if (pid && map.has(pid)) {
        node.level = (map.get(pid)!.level || 0) + 1;
        map.get(pid)!.children.push(node);
      } else { roots.push(node); }
    }
    return roots;
  }

  private toggleNode(node: TreeNode) {
    node.expanded = !node.expanded;
    this.tree = [...this.tree];
  }

  private renderNode(node: TreeNode): any {
    const hasChildren = node.children.length > 0;
    return [
      <div class="bc-tree-row" style={{ paddingLeft: (node.level * 24 + 8) + 'px' }} onClick={() => hasChildren && this.toggleNode(node)}>
        <span class={{'bc-tree-icon': true, 'has-children': hasChildren}}>
          {hasChildren ? (node.expanded ? '\u25BC' : '\u25B6') : '\u2022'}
        </span>
        <span class="bc-tree-name">{node.name}</span>
        {hasChildren && <span class="bc-tree-count">({node.children.length})</span>}
      </div>,
      node.expanded && node.children.map(child => this.renderNode(child)),
    ];
  }  @Method() async refresh(): Promise<void> { }

  render() {
    return (
      <div class="bc-view bc-view-tree">
        <div class="bc-tree-header"><h2>{this.viewTitle || this.model}</h2></div>
        {this.loading && <div class="bc-tree-loading">{i18n.t('common.loading')}</div>}
        <div class="bc-tree-body">{this.tree.map(n => this.renderNode(n))}</div>
      </div>
    );
  }
}


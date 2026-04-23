import { ListParams, ListResponse } from './types';

export class LcApiClient {
  private baseUrl: string;
  private headers: Record<string, string> = {};

  constructor(baseUrl: string = '') {
    this.baseUrl = baseUrl;
  }

  setAuthToken(token: string): void {
    this.headers['Authorization'] = `Bearer ${token}`;
  }

  clearAuthToken(): void {
    delete this.headers['Authorization'];
  }

  private async request<T>(url: string, options: RequestInit = {}): Promise<T> {
    const res = await fetch(`${this.baseUrl}${url}`, {
      ...options,
      headers: {
        ...this.headers,
        ...options.headers as Record<string, string>,
      },
    });
    if (!res.ok) {
      const body = await res.text();
      throw new Error(`HTTP ${res.status}: ${body}`);
    }
    return res.json();
  }

  async list(model: string, params?: ListParams): Promise<ListResponse> {
    const query = new URLSearchParams();
    if (params?.page) query.set('page', String(params.page));
    if (params?.pageSize) query.set('page_size', String(params.pageSize));
    if (params?.sort) query.set('sort', params.sort);
    if (params?.order) query.set('order', params.order);
    if (params?.q) query.set('q', params.q);
    if (params?.filters) {
      for (const [key, val] of Object.entries(params.filters)) {
        query.set(key, String(val));
      }
    }
    const qs = query.toString();
    return this.request<ListResponse>(`/api/${model}s${qs ? '?' + qs : ''}`);
  }

  async read(model: string, id: string): Promise<Record<string, unknown>> {
    return this.request<Record<string, unknown>>(`/api/${model}s/${id}`);
  }

  async create(model: string, data: Record<string, unknown>): Promise<Record<string, unknown>> {
    return this.request<Record<string, unknown>>(`/api/${model}s`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
  }

  async update(model: string, id: string, data: Record<string, unknown>): Promise<Record<string, unknown>> {
    return this.request<Record<string, unknown>>(`/api/${model}s/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
  }

  async remove(model: string, id: string): Promise<void> {
    await this.request<void>(`/api/${model}s/${id}`, { method: 'DELETE' });
  }

  async action(model: string, id: string, actionName: string): Promise<Record<string, unknown>> {
    return this.request<Record<string, unknown>>(`/api/${model}s/${id}/${actionName}`, {
      method: 'POST',
    });
  }

  async search(model: string, query: string): Promise<Record<string, unknown>[]> {
    const res = await this.list(model, { q: query, pageSize: 20 });
    return res.data;
  }

  async upload(file: File): Promise<{ url: string }> {
    const form = new FormData();
    form.append('file', file);
    const res = await fetch(`${this.baseUrl}/api/upload`, {
      method: 'POST',
      headers: this.headers,
      body: form,
    });
    if (!res.ok) throw new Error(`Upload failed: ${res.status}`);
    return res.json();
  }

  async uploadFile(formData: FormData): Promise<{
    id: string;
    name: string;
    url: string;
    thumbnail_url: string | null;
    size: number;
    mime_type: string;
    ext: string;
    hash: string;
    version: number;
    created_at: string;
  }> {
    const res = await fetch(`${this.baseUrl}/api/upload`, {
      method: 'POST',
      headers: this.headers,
      body: formData,
    });
    if (!res.ok) {
      const body = await res.text();
      throw new Error(`Upload failed: ${res.status} ${body}`);
    }
    return res.json();
  }

  async uploadFiles(formData: FormData): Promise<{
    files: Array<{
      id: string;
      name: string;
      url: string;
      thumbnail_url: string | null;
      size: number;
      mime_type: string;
    }>;
    total: number;
    errors: string[];
  }> {
    const res = await fetch(`${this.baseUrl}/api/uploads`, {
      method: 'POST',
      headers: this.headers,
      body: formData,
    });
    if (!res.ok) {
      const body = await res.text();
      throw new Error(`Upload failed: ${res.status} ${body}`);
    }
    return res.json();
  }

  async getFileMetadata(fileId: string): Promise<Record<string, unknown>> {
    return this.request<Record<string, unknown>>(`/api/files/${fileId}`);
  }

  async deleteFile(fileId: string): Promise<void> {
    await this.request<void>(`/api/files/${fileId}`, { method: 'DELETE' });
  }

  async getFileVersions(fileId: string): Promise<Record<string, unknown>[]> {
    return this.request<Record<string, unknown>[]>(`/api/files/${fileId}/versions`);
  }
}

let _client: LcApiClient | undefined;

export function getApiClient(): LcApiClient {
  if (!_client) {
    let baseUrl = '';
    if (typeof window !== 'undefined') {
      const win = window as unknown as Record<string, unknown>;
      if (typeof win.__lc_base_url === 'string') {
        baseUrl = win.__lc_base_url;
      }
    }
    _client = new LcApiClient(baseUrl);
  }
  return _client;
}

export function setApiClient(client: LcApiClient): void {
  _client = client;
}

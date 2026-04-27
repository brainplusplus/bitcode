import { FetchParams, FetchResult, DataFetcher, OptionsFetcher } from './types';
import { BcSetup } from './bc-setup';

export function resolveUrl(template: string, values: Record<string, unknown>): string {
  return template.replace(/\{(\w+)\}/g, (_match, key) => {
    const val = values[key];
    if (val === undefined || val === null) return '';
    return encodeURIComponent(String(val));
  });
}

export function buildHeaders(customHeaders?: string | Record<string, string>): Record<string, string> {
  const setupHeaders = BcSetup.getHeaders();
  let custom: Record<string, string> = {};
  if (typeof customHeaders === 'string' && customHeaders) {
    try { custom = JSON.parse(customHeaders); } catch { /* ignore */ }
  } else if (customHeaders && typeof customHeaders === 'object') {
    custom = customHeaders;
  }
  return { ...setupHeaders, ...custom };
}

export function normalizeResponse(response: unknown): FetchResult {
  const transformer = BcSetup.getConfig().responseTransformer;
  if (transformer) {
    return transformer(response);
  }

  if (Array.isArray(response)) {
    return { data: response, total: response.length };
  }

  if (response && typeof response === 'object') {
    const obj = response as Record<string, unknown>;

    const dataKeys = ['data', 'results', 'items', 'records', 'rows'];
    for (const key of dataKeys) {
      if (Array.isArray(obj[key])) {
        const totalKeys = ['total', 'total_count', 'totalCount', 'count', 'total_records', 'totalRecords'];
        let total = (obj[key] as unknown[]).length;
        for (const tk of totalKeys) {
          if (typeof obj[tk] === 'number') {
            total = obj[tk] as number;
            break;
          }
        }
        return { data: obj[key] as unknown[], total };
      }
    }
  }

  return { data: [], total: 0 };
}

function appendPaginationParams(url: string, params?: FetchParams): string {
  if (!params) return url;
  const sep = url.includes('?') ? '&' : '?';
  const parts: string[] = [];
  if (params.page) parts.push(`page=${params.page}`);
  if (params.pageSize) parts.push(`page_size=${params.pageSize}`);
  if (params.search) parts.push(`q=${encodeURIComponent(params.search)}`);
  if (params.sort && params.sort.length > 0) {
    parts.push(`sort=${params.sort[0].field}`);
    parts.push(`order=${params.sort[0].direction}`);
  }
  if (params.filters) {
    for (const [key, val] of Object.entries(params.filters)) {
      parts.push(`${encodeURIComponent(key)}=${encodeURIComponent(String(val))}`);
    }
  }
  return parts.length > 0 ? `${url}${sep}${parts.join('&')}` : url;
}

async function doFetch(url: string, headers: Record<string, string>, element?: HTMLElement): Promise<FetchResult> {
  let resolvedUrl = url;
  let resolvedHeaders = { ...headers };

  if (element) {
    const beforeEvent = new CustomEvent('lcBeforeFetch', {
      detail: { url: resolvedUrl, headers: resolvedHeaders, params: {} },
      bubbles: true,
      cancelable: true,
    });
    element.dispatchEvent(beforeEvent);
    resolvedUrl = beforeEvent.detail.url;
    resolvedHeaders = beforeEvent.detail.headers;
  }

  const res = await fetch(resolvedUrl, { headers: resolvedHeaders });
  if (!res.ok) {
    throw new Error(`Fetch failed: ${res.status} ${res.statusText}`);
  }
  const json = await res.json();

  if (element) {
    const afterEvent = new CustomEvent('lcAfterFetch', {
      detail: { response: json, data: null as unknown[] | null, total: 0 },
      bubbles: true,
    });
    element.dispatchEvent(afterEvent);
    if (afterEvent.detail.data) {
      return { data: afterEvent.detail.data, total: afterEvent.detail.total };
    }
  }

  return normalizeResponse(json);
}

export async function fetchData(opts: {
  fetcher?: DataFetcher;
  element?: HTMLElement;
  dataSource?: string;
  localData?: string | unknown[];
  model?: string;
  fetchHeaders?: string | Record<string, string>;
  params?: FetchParams;
}): Promise<FetchResult> {
  if (opts.fetcher) {
    return opts.fetcher(opts.params || {});
  }

  if (opts.dataSource) {
    const baseUrl = BcSetup.getBaseUrl();
    const dependValues = opts.params?.dependValues || {};
    let url = resolveUrl(opts.dataSource, dependValues);
    if (url && !url.startsWith('http') && baseUrl) {
      url = baseUrl + url;
    }
    url = appendPaginationParams(url, opts.params);
    const headers = buildHeaders(opts.fetchHeaders);
    return doFetch(url, headers, opts.element);
  }

  if (opts.localData) {
    const data = typeof opts.localData === 'string' ? JSON.parse(opts.localData) : opts.localData;
    return { data: Array.isArray(data) ? data : [], total: Array.isArray(data) ? data.length : 0 };
  }

  if (opts.model) {
    try {
      const { getApiClient } = await import('./api-client');
      const api = getApiClient();
      const result = await api.list(opts.model, {
        page: opts.params?.page,
        pageSize: opts.params?.pageSize,
        sort: opts.params?.sort?.[0]?.field,
        order: opts.params?.sort?.[0]?.direction,
        q: opts.params?.search,
        filters: opts.params?.filters as Record<string, unknown>,
      });
      return { data: result.data, total: result.total };
    } catch {
      // api-client not available — standalone mode
    }
  }

  return { data: [], total: 0 };
}

export async function fetchOptions(opts: {
  fetcher?: OptionsFetcher;
  element?: HTMLElement;
  dataSource?: string;
  localOptions?: string | unknown[];
  model?: string;
  query?: string;
  fetchHeaders?: string | Record<string, string>;
  params?: FetchParams;
}): Promise<unknown[]> {
  if (opts.fetcher) {
    return opts.fetcher(opts.query || '', opts.params || {});
  }

  if (opts.dataSource) {
    const baseUrl = BcSetup.getBaseUrl();
    const dependValues = opts.params?.dependValues || {};
    let url = resolveUrl(opts.dataSource, dependValues);
    if (url && !url.startsWith('http') && baseUrl) {
      url = baseUrl + url;
    }
    if (opts.query) {
      const sep = url.includes('?') ? '&' : '?';
      url = `${url}${sep}q=${encodeURIComponent(opts.query)}`;
    }
    const headers = buildHeaders(opts.fetchHeaders);
    const result = await doFetch(url, headers, opts.element);
    return result.data;
  }

  if (opts.localOptions) {
    const options = typeof opts.localOptions === 'string' ? JSON.parse(opts.localOptions) : opts.localOptions;
    if (!Array.isArray(options)) return [];
    if (!opts.query) return options;
    const q = opts.query.toLowerCase();
    return options.filter((opt: unknown) => {
      if (typeof opt === 'string') return opt.toLowerCase().includes(q);
      if (opt && typeof opt === 'object') {
        const o = opt as Record<string, unknown>;
        const label = String(o.label || o.name || o.text || '');
        return label.toLowerCase().includes(q);
      }
      return false;
    });
  }

  if (opts.model) {
    try {
      const { getApiClient } = await import('./api-client');
      const api = getApiClient();
      return api.search(opts.model, opts.query || '');
    } catch {
      // standalone mode
    }
  }

  return [];
}

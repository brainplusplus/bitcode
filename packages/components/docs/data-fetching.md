# Data Fetching — 4-Level Strategy

Components that need data (select, datatable, charts) support 4 levels of data supply. All levels available simultaneously — pick what fits, or combine.

## Level 1: Local Data

Data directly in the prop. No fetch.

```html
<bc-field-select name="status" options='[{"label":"Active","value":"active"},{"label":"Inactive","value":"inactive"}]' />
<bc-datatable columns='[...]' data='[{"id":1,"name":"John"}]' />
```

## Level 2: URL Endpoint

Component fetches data via native `fetch()`. Auto-detects response format.

```html
<bc-field-select name="city" data-source="/api/cities" />
<bc-datatable columns='[...]' data-source="/api/users" server-side />
```

Supported response formats (auto-detected):
- `{ data: [...] }`
- `{ results: [...] }`
- `{ items: [...] }`
- `{ records: [...] }`
- `{ rows: [...] }`
- `[...]` (plain array)

Total count detected from: `total`, `total_count`, `totalCount`, `count`, `total_records`.

Override auto-detection globally:

```javascript
BcSetup.configure({
  responseTransformer: (res) => ({ data: res.payload, total: res.meta.count })
});
```

## Level 3: URL + Event Intercept

Modify request before fetch, or transform response after fetch.

```html
<bc-datatable id="t" data-source="/api/users" server-side />

<script>
document.getElementById('t').addEventListener('lcBeforeFetch', (e) => {
  e.detail.headers['X-Custom'] = 'value';
  e.detail.url = e.detail.url + '&extra=param';
});

document.getElementById('t').addEventListener('lcAfterFetch', (e) => {
  e.detail.data = e.detail.response.items.map(i => ({ id: i.ID, name: i.FullName }));
  e.detail.total = e.detail.response.totalCount;
});
</script>
```

## Level 4: Custom Fetcher Function

Full control. Set a JavaScript function on the element.

```html
<bc-datatable id="t" columns='[...]' />

<script>
document.getElementById('t').dataFetcher = async (params) => {
  // params = { page, pageSize, sort, filters, search }
  const token = await getAuthToken();
  const res = await fetch('https://my-api.com/data', {
    method: 'POST',
    headers: { 'Authorization': `Bearer ${token}` },
    body: JSON.stringify({ offset: (params.page - 1) * params.pageSize, limit: params.pageSize })
  });
  const json = await res.json();
  return { data: json.records, total: json.meta.total_count };
};
</script>
```

For select-family components:

```javascript
document.getElementById('citySelect').optionsFetcher = async (query, params) => {
  // query = search text, params = { dependValues: { province: 'jabar' } }
  const res = await fetch(`/api/cities?q=${query}&prov=${params.dependValues?.province || ''}`);
  return await res.json();
};
```

## Resolution Priority

```
1. dataFetcher/optionsFetcher (JS property)  → Level 4
2. lcBeforeFetch/lcAfterFetch listeners      → Level 3 (with dataSource)
3. dataSource prop                           → Level 2 (native fetch)
4. data/options prop                         → Level 1 (local)
5. model prop + BitCode api-client           → BitCode fallback
6. nothing                                   → empty state
```

## Auth & Headers

All fetch requests automatically include headers from `BcSetup.configure()`:

```javascript
BcSetup.configure({
  baseUrl: '/api',
  auth: { type: 'bearer', token: () => localStorage.getItem('jwt') },
  headers: { 'X-Tenant': 'company-a' }
});
```

Per-component custom headers via `fetch-headers` prop:

```html
<bc-datatable data-source="/api/data" fetch-headers='{"X-Custom":"value"}' />
```

## Cascading / Dependent Data

Use `depend-on` + `data-source` with `{field}` placeholders:

```html
<bc-field-select name="province" options='[...]' />
<bc-field-select name="city" depend-on="province" data-source="/api/cities?province={province}" />
<bc-field-select name="district" depend-on="city" data-source="/api/districts?city={city}" />
```

When province changes, city auto-fetches new options and clears its value. Cascades further to district.

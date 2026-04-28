# bc-datatable

> Full-featured enterprise data table with server-side pagination, sorting, filtering, and 4-level data fetching.

## Quick Start

```html
<bc-datatable columns='[{"field":"name","label":"Name"},{"field":"email","label":"Email"}]' data-source="/api/users" server-side />
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| model | string | '' | Model name for API |
| columns | string (JSON) | '[]' | Column definitions |
| api-url | string | '' | API endpoint |
| data-source | string | '' | Data source URL (4-level) |
| local-data | string | '' | Local data JSON (Level 1) |
| page-size | number | 20 | Rows per page |
| server-side | boolean | true | Server-side mode |
| selectable | boolean | true | Row selection |
| export-xls | boolean | true | Enable XLS export |
| fetch-headers | string | '' | Custom headers JSON |
| empty-text | string | '' | Empty state text |

## Events

| Event | Payload |
|-------|---------|
| lcRowClick | {record} |
| lcSelectionChange | {ids} |
| lcBulkAction | {action, ids} |
| lcPageChange | {page, pageSize} |
| lcSortChange | {sorts} |
| lcFilterChange | {filters} |
| lcBeforeFetch | {url, headers, params} |
| lcAfterFetch | {response, data, total} |

## Methods

| Method | Returns |
|--------|---------|
| refresh() | Promise<void> |
| getData() | Promise<Array> |
| setData(data) | Promise<void> |
| getSelected() | Promise<string[]> |
| clearSelection() | Promise<void> |
| selectAll() | Promise<void> |
| goToPage(page) | Promise<void> |
| sortBy(column, direction) | Promise<void> |
| exportCSV() | Promise<void> |
| scrollToRow(id) | Promise<void> |

## 4-Level Data

See [data-fetching guide](../data-fetching.md).

```javascript
// Level 4: Custom fetcher
document.querySelector('bc-datatable').dataFetcher = async (params) => {
  const res = await fetch('/my-api', { method: 'POST', body: JSON.stringify(params) });
  const json = await res.json();
  return { data: json.records, total: json.count };
};
```

See [theming](../theming.md).


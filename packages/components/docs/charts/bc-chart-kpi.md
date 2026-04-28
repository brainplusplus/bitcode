# bc-chart-kpi

> KPI card (pure HTML)

## Quick Start

```html
<bc-chart-kpi data='[{"name":"A","value":10},{"name":"B","value":20}]' />
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| data | string (JSON) | '[]' | Chart data |
| chart-title | string | '' | Chart title |

Enterprise props: trend, valuePrefix, valueSuffix, color

## Events

| Event | Payload |
|-------|---------|
| lcChartClick | {name, value, dataIndex} |

## Methods

| Method | Returns |
|--------|---------|
| updateData(data) | Promise<void> |
| setData(data) | Promise<void> |
| refresh() | Promise<void> |
| resize() | Promise<void> |
| exportImage(format?) | Promise<string> |

See [theming](../theming.md), [data-fetching](../data-fetching.md).


const CURRENCY_LOCALES: Record<string, string> = {
  IDR: 'id-ID',
  USD: 'en-US',
  EUR: 'de-DE',
  GBP: 'en-GB',
  JPY: 'ja-JP',
  CNY: 'zh-CN',
  SGD: 'en-SG',
  MYR: 'ms-MY',
  AUD: 'en-AU',
};

export function formatCurrency(value: number, currency: string = 'USD', precision: number = 2): string {
  const locale = CURRENCY_LOCALES[currency] || 'en-US';
  try {
    return new Intl.NumberFormat(locale, {
      style: 'currency',
      currency,
      minimumFractionDigits: precision,
      maximumFractionDigits: precision,
    }).format(value);
  } catch {
    return `${currency} ${formatNumber(value, precision)}`;
  }
}

export function formatNumber(value: number, precision: number = 0): string {
  return new Intl.NumberFormat('en-US', {
    minimumFractionDigits: precision,
    maximumFractionDigits: precision,
  }).format(value);
}

export function formatPercent(value: number, precision: number = 0): string {
  return `${formatNumber(value, precision)}%`;
}

export function formatDate(value: string | Date, style: 'short' | 'medium' | 'long' = 'medium'): string {
  const date = typeof value === 'string' ? new Date(value) : value;
  if (isNaN(date.getTime())) return String(value);

  const options: Intl.DateTimeFormatOptions = {};
  switch (style) {
    case 'short': options.dateStyle = 'short'; break;
    case 'medium': options.dateStyle = 'medium'; break;
    case 'long': options.dateStyle = 'long'; break;
  }
  return new Intl.DateTimeFormat('en-GB', options).format(date);
}

export function formatTime(value: string | Date, use24h: boolean = true): string {
  const date = typeof value === 'string' ? new Date(`1970-01-01T${value}`) : value;
  if (isNaN(date.getTime())) return String(value);
  return new Intl.DateTimeFormat('en-GB', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: !use24h,
  }).format(date);
}

export function formatDateTime(value: string | Date): string {
  const date = typeof value === 'string' ? new Date(value) : value;
  if (isNaN(date.getTime())) return String(value);
  return new Intl.DateTimeFormat('en-GB', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date);
}

export function formatDuration(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;

  const parts: string[] = [];
  if (days > 0) parts.push(`${days}d`);
  if (hours > 0) parts.push(`${hours}h`);
  if (minutes > 0) parts.push(`${minutes}m`);
  if (secs > 0 || parts.length === 0) parts.push(`${secs}s`);
  return parts.join(' ');
}

export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  const size = bytes / Math.pow(1024, i);
  return `${formatNumber(size, i > 0 ? 1 : 0)} ${units[i]}`;
}

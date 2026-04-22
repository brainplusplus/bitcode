export function validateEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
}

export function validateUrl(value: string): boolean {
  try {
    new URL(value);
    return true;
  } catch {
    return false;
  }
}

export function validatePhone(value: string): boolean {
  return /^[+]?[\d\s()\-]{7,20}$/.test(value);
}

export function validateRequired(value: unknown): boolean {
  if (value === null || value === undefined) return false;
  if (typeof value === 'string') return value.trim().length > 0;
  if (Array.isArray(value)) return value.length > 0;
  return true;
}

export function validateMaxLength(value: string, max: number): boolean {
  return value.length <= max;
}

export function validateMinLength(value: string, min: number): boolean {
  return value.length >= min;
}

export function validateMax(value: number, max: number): boolean {
  return value <= max;
}

export function validateMin(value: number, min: number): boolean {
  return value >= min;
}

export function validatePattern(value: string, pattern: string): boolean {
  try {
    return new RegExp(pattern).test(value);
  } catch {
    return false;
  }
}

export function validateFileSize(file: File, maxSizeStr: string): boolean {
  const match = maxSizeStr.match(/^(\d+)\s*(B|KB|MB|GB)$/i);
  if (!match) return true;
  const size = parseInt(match[1], 10);
  const unit = match[2].toUpperCase();
  const multipliers: Record<string, number> = { B: 1, KB: 1024, MB: 1048576, GB: 1073741824 };
  const maxBytes = size * (multipliers[unit] || 1);
  return file.size <= maxBytes;
}

export function validateFileType(file: File, accept: string): boolean {
  if (!accept) return true;
  const allowed = accept.split(',').map(s => s.trim().toLowerCase());
  const ext = '.' + file.name.split('.').pop()?.toLowerCase();
  const mime = file.type.toLowerCase();
  return allowed.some(a => {
    if (a.startsWith('.')) return ext === a;
    if (a.endsWith('/*')) return mime.startsWith(a.replace('/*', '/'));
    return mime === a;
  });
}

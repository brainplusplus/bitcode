// Hybrid Logical Clock (HLC) — deterministic event ordering across devices.
// Format: "{wall_time_base36}:{logical_counter_base36}:{device_id}"
// Example: "01jk5p9q:0001:DEV-A"

const MAX_CLOCK_SKEW_MS = 60_000;

let _hlcWallTime = 0;
let _hlcLogical = 0;
let _hlcDeviceId = '';

export interface HLCTimestamp {
  wallTime: number;
  logical: number;
  deviceId: string;
}

export function parseHLC(hlc: string): HLCTimestamp {
  if (!hlc || typeof hlc !== 'string') {
    throw new Error(`[HLC] Invalid HLC value: "${hlc}"`);
  }

  const firstColon = hlc.indexOf(':');
  if (firstColon === -1) {
    throw new Error(`[HLC] Malformed HLC — missing first colon: "${hlc}"`);
  }

  const secondColon = hlc.indexOf(':', firstColon + 1);
  if (secondColon === -1) {
    throw new Error(`[HLC] Malformed HLC — missing second colon: "${hlc}"`);
  }

  const wallTimePart = hlc.substring(0, firstColon);
  const logicalPart = hlc.substring(firstColon + 1, secondColon);
  const deviceIdPart = hlc.substring(secondColon + 1);

  const wallTime = parseInt(wallTimePart, 36);
  const logical = parseInt(logicalPart, 36);

  if (isNaN(wallTime) || isNaN(logical)) {
    throw new Error(`[HLC] Malformed HLC — non-numeric parts: "${hlc}"`);
  }

  if (!deviceIdPart) {
    throw new Error(`[HLC] Malformed HLC — empty device_id: "${hlc}"`);
  }

  return { wallTime, logical, deviceId: deviceIdPart };
}

export function formatHLC(ts: HLCTimestamp): string {
  const wallStr = ts.wallTime.toString(36);
  const logicalStr = ts.logical.toString(36).padStart(4, '0');
  return `${wallStr}:${logicalStr}:${ts.deviceId}`;
}

export function hlcSetDeviceId(deviceId: string): void {
  _hlcDeviceId = deviceId;
}

export function hlcGetDeviceId(): string {
  return _hlcDeviceId;
}

export function hlcNow(nowMs?: number): string {
  if (!_hlcDeviceId) {
    throw new Error('[HLC] Device ID not set — call hlcSetDeviceId() first');
  }

  const physicalNow = nowMs ?? Date.now();

  if (physicalNow > _hlcWallTime) {
    _hlcWallTime = physicalNow;
    _hlcLogical = 0;
  } else {
    _hlcLogical++;
  }

  return formatHLC({ wallTime: _hlcWallTime, logical: _hlcLogical, deviceId: _hlcDeviceId });
}

// Lamport-style merge: maxWall = max(physical, local, remote).
// Logical counter: if walls tie → max(logicals)+1; if one wall wins → that logical+1; else 0.
export function hlcReceive(remoteHlc: string, nowMs?: number): string {
  if (!_hlcDeviceId) {
    throw new Error('[HLC] Device ID not set — call hlcSetDeviceId() first');
  }

  const remote = parseHLC(remoteHlc);
  const physicalNow = nowMs ?? Date.now();

  const maxWall = Math.max(physicalNow, _hlcWallTime, remote.wallTime);

  if (maxWall - physicalNow > MAX_CLOCK_SKEW_MS) {
    throw new Error(
      `[HLC] Clock skew too large: ${maxWall - physicalNow}ms exceeds max ${MAX_CLOCK_SKEW_MS}ms. ` +
      `Remote wall=${remote.wallTime}, local wall=${_hlcWallTime}, physical=${physicalNow}`,
    );
  }

  let newLogical: number;

  if (maxWall === _hlcWallTime && maxWall === remote.wallTime) {
    newLogical = Math.max(_hlcLogical, remote.logical) + 1;
  } else if (maxWall === _hlcWallTime) {
    newLogical = _hlcLogical + 1;
  } else if (maxWall === remote.wallTime) {
    newLogical = remote.logical + 1;
  } else {
    newLogical = 0;
  }

  _hlcWallTime = maxWall;
  _hlcLogical = newLogical;

  return formatHLC({ wallTime: _hlcWallTime, logical: _hlcLogical, deviceId: _hlcDeviceId });
}

// Compare: wall time → logical counter → device_id (lexicographic tie-breaker).
export function hlcCompare(a: string, b: string): -1 | 0 | 1 {
  const pa = parseHLC(a);
  const pb = parseHLC(b);

  if (pa.wallTime < pb.wallTime) return -1;
  if (pa.wallTime > pb.wallTime) return 1;

  if (pa.logical < pb.logical) return -1;
  if (pa.logical > pb.logical) return 1;

  if (pa.deviceId < pb.deviceId) return -1;
  if (pa.deviceId > pb.deviceId) return 1;

  return 0;
}

export function hlcReset(): void {
  _hlcWallTime = 0;
  _hlcLogical = 0;
  _hlcDeviceId = '';
}

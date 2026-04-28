import { parseHLC, formatHLC, hlcSetDeviceId, hlcNow, hlcReceive, hlcCompare, hlcReset } from './hlc';

describe('HLC', () => {
  beforeEach(() => {
    hlcReset();
  });

  describe('parseHLC / formatHLC', () => {
    it('round-trips a valid HLC string', () => {
      const hlc = '1a2b3c:0001:DEV-A';
      const parsed = parseHLC(hlc);
      expect(parsed.deviceId).toBe('DEV-A');
      expect(formatHLC(parsed)).toBe(hlc);
    });

    it('throws on empty string', () => {
      expect(() => parseHLC('')).toThrow('Invalid HLC');
    });

    it('throws on missing second colon', () => {
      expect(() => parseHLC('abc:0001')).toThrow('missing second colon');
    });

    it('throws on non-numeric wall time', () => {
      expect(() => parseHLC('!!!:0001:DEV-A')).toThrow('non-numeric');
    });

    it('throws on empty device_id', () => {
      expect(() => parseHLC('abc:0001:')).toThrow('empty device_id');
    });

    it('handles device_id containing colons', () => {
      const parsed = parseHLC('abc:0001:DEV:WITH:COLONS');
      expect(parsed.deviceId).toBe('DEV:WITH:COLONS');
    });
  });

  describe('hlcNow', () => {
    it('throws if device ID not set', () => {
      expect(() => hlcNow()).toThrow('Device ID not set');
    });

    it('generates monotonically increasing timestamps', () => {
      hlcSetDeviceId('DEV-A');
      const t1 = hlcNow(1000);
      const t2 = hlcNow(2000);
      expect(hlcCompare(t1, t2)).toBe(-1);
    });

    it('increments logical counter when wall time does not advance', () => {
      hlcSetDeviceId('DEV-A');
      const t1 = hlcNow(1000);
      const t2 = hlcNow(1000);
      const t3 = hlcNow(1000);

      const p1 = parseHLC(t1);
      const p2 = parseHLC(t2);
      const p3 = parseHLC(t3);

      expect(p1.wallTime).toBe(1000);
      expect(p1.logical).toBe(0);
      expect(p2.logical).toBe(1);
      expect(p3.logical).toBe(2);
    });

    it('resets logical counter when wall time advances', () => {
      hlcSetDeviceId('DEV-A');
      hlcNow(1000);
      hlcNow(1000);
      const t3 = hlcNow(2000);

      const p3 = parseHLC(t3);
      expect(p3.wallTime).toBe(2000);
      expect(p3.logical).toBe(0);
    });

    it('handles clock going backwards (keeps old wall time, increments logical)', () => {
      hlcSetDeviceId('DEV-A');
      hlcNow(5000);
      const t2 = hlcNow(3000);

      const p2 = parseHLC(t2);
      expect(p2.wallTime).toBe(5000);
      expect(p2.logical).toBe(1);
    });
  });

  describe('hlcReceive', () => {
    it('throws if device ID not set', () => {
      expect(() => hlcReceive('abc:0001:DEV-B')).toThrow('Device ID not set');
    });

    it('advances wall time from remote', () => {
      hlcSetDeviceId('DEV-A');
      hlcNow(1000);

      const result = hlcReceive(formatHLC({ wallTime: 5000, logical: 0, deviceId: 'DEV-B' }), 1000);
      const parsed = parseHLC(result);

      expect(parsed.wallTime).toBe(5000);
      expect(parsed.logical).toBe(1);
      expect(parsed.deviceId).toBe('DEV-A');
    });

    it('merges when all wall times are equal', () => {
      hlcSetDeviceId('DEV-A');
      hlcNow(1000);
      hlcNow(1000);

      const remoteHlc = formatHLC({ wallTime: 1000, logical: 5, deviceId: 'DEV-B' });
      const result = hlcReceive(remoteHlc, 1000);
      const parsed = parseHLC(result);

      expect(parsed.wallTime).toBe(1000);
      expect(parsed.logical).toBe(6);
    });

    it('uses physical time when it is the max', () => {
      hlcSetDeviceId('DEV-A');
      hlcNow(1000);

      const remoteHlc = formatHLC({ wallTime: 500, logical: 3, deviceId: 'DEV-B' });
      const result = hlcReceive(remoteHlc, 9000);
      const parsed = parseHLC(result);

      expect(parsed.wallTime).toBe(9000);
      expect(parsed.logical).toBe(0);
    });

    it('rejects clock skew exceeding 60 seconds', () => {
      hlcSetDeviceId('DEV-A');
      const futureRemote = formatHLC({ wallTime: 200_000, logical: 0, deviceId: 'DEV-B' });

      expect(() => hlcReceive(futureRemote, 1000)).toThrow('Clock skew too large');
    });

    it('accepts clock skew within 60 seconds', () => {
      hlcSetDeviceId('DEV-A');
      const nearFuture = formatHLC({ wallTime: 50_000, logical: 0, deviceId: 'DEV-B' });

      expect(() => hlcReceive(nearFuture, 1000)).not.toThrow();
    });
  });

  describe('hlcCompare', () => {
    it('compares by wall time first', () => {
      const a = formatHLC({ wallTime: 1000, logical: 99, deviceId: 'ZZZ' });
      const b = formatHLC({ wallTime: 2000, logical: 0, deviceId: 'AAA' });
      expect(hlcCompare(a, b)).toBe(-1);
      expect(hlcCompare(b, a)).toBe(1);
    });

    it('compares by logical counter when wall times are equal', () => {
      const a = formatHLC({ wallTime: 1000, logical: 1, deviceId: 'ZZZ' });
      const b = formatHLC({ wallTime: 1000, logical: 5, deviceId: 'AAA' });
      expect(hlcCompare(a, b)).toBe(-1);
    });

    it('uses device_id as tie-breaker', () => {
      const a = formatHLC({ wallTime: 1000, logical: 1, deviceId: 'DEV-A' });
      const b = formatHLC({ wallTime: 1000, logical: 1, deviceId: 'DEV-B' });
      expect(hlcCompare(a, b)).toBe(-1);
      expect(hlcCompare(b, a)).toBe(1);
    });

    it('returns 0 for identical HLCs', () => {
      const a = formatHLC({ wallTime: 1000, logical: 1, deviceId: 'DEV-A' });
      expect(hlcCompare(a, a)).toBe(0);
    });

    it('is deterministic — same inputs always produce same result', () => {
      const a = formatHLC({ wallTime: 5000, logical: 3, deviceId: 'DEV-X' });
      const b = formatHLC({ wallTime: 5000, logical: 3, deviceId: 'DEV-Y' });

      for (let i = 0; i < 100; i++) {
        expect(hlcCompare(a, b)).toBe(-1);
      }
    });
  });
});

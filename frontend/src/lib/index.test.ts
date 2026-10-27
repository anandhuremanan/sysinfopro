import { describe, expect, it } from 'vitest';
import { formatBytes, isBatteryCharging, shortCpuName } from './index';

describe('formatBytes', () => {
	it('handles nothing and negatives', () => {
		expect(formatBytes(0)).toBe('0 B');
		expect(formatBytes(-5)).toBe('0 B');
	});

	it('uses binary units, as Windows Explorer does', () => {
		expect(formatBytes(512)).toBe('512 B');
		expect(formatBytes(1536)).toBe('1.5 KB');
		expect(formatBytes(1024 ** 3)).toBe('1.0 GB');
		expect(formatBytes(250 * 1024 ** 3)).toBe('250 GB');
	});

	it('rolls 1000-1023 of a unit over to the next one', () => {
		expect(formatBytes(1000)).toBe('1.0 KB');
		expect(formatBytes(1023 * 1024 ** 2)).toBe('1.0 GB');
	});
});

describe('shortCpuName', () => {
	it('drops trademarks, generation and clock speed', () => {
		expect(shortCpuName('12th Gen Intel(R) Core(TM) i5-1235U')).toBe('Intel Core i5-1235U');
		expect(shortCpuName('Intel(R) Core(TM) i7-1260P CPU @ 2.10GHz')).toBe('Intel Core i7-1260P');
		expect(shortCpuName('AMD Ryzen 7 5800X 8-Core Processor')).toBe('AMD Ryzen 7 5800X 8-Core');
		expect(shortCpuName('')).toBe('');
	});
});

describe('isBatteryCharging', () => {
	it('tells charging from discharging and not charging', () => {
		expect(isBatteryCharging('Charging')).toBe(true);
		expect(isBatteryCharging('Discharging')).toBe(false);
		expect(isBatteryCharging('Not charging')).toBe(false);
		expect(isBatteryCharging(undefined)).toBe(false);
	});
});

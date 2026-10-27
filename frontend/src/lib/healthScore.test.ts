import { describe, expect, it } from 'vitest';
import { computeScore, deductions, scoreWord } from './healthScore';

const GB = 1024 ** 3;
const allGood = {
	freeBytes: 100 * GB,
	totalBytes: 200 * GB,
	memoryUsedPct: 40,
	drive: { status: 'Healthy', life_left_pct: 95 },
	batteryHealthPct: 98,
	security: { checks: [{ status: 'pass' }, { status: 'pass' }] }
};

describe('computeScore', () => {
	it('gives 100 when every reading is fine', () => {
		const s = computeScore(allGood);
		expect(s.score).toBe(100);
		expect(s.factors).toHaveLength(5);
		expect(deductions(s)).toEqual([]);
	});

	it('is unknown (-1) until anything is measured', () => {
		const s = computeScore({});
		expect(s.score).toBe(-1);
		expect(s.missing).toContain('free space');
	});

	it('leaves out a battery the PC does not have, and scales the rest to 100', () => {
		const s = computeScore({ ...allGood, batteryHealthPct: undefined });
		expect(s.factors.map((f) => f.id)).not.toContain('battery');
		expect(s.score).toBe(100);
	});

	it('costs points for a nearly full drive and full memory', () => {
		const s = computeScore({ ...allGood, freeBytes: 5 * GB, memoryUsedPct: 95 });
		expect(s.score).toBeLessThan(70);
		const ids = deductions(s).map((d) => d.factor.id);
		expect(ids[0]).toBe('space');
		expect(ids).toContain('memory');
	});

	it('does not count unknown security checks against the score', () => {
		const s = computeScore({
			...allGood,
			security: { checks: [{ status: 'pass' }, { status: 'unknown' }] }
		});
		expect(s.factors.find((f) => f.id === 'security')?.points).toBe(20);
	});

	it('an unknown drive is left out, not scored as healthy', () => {
		const s = computeScore({ ...allGood, drive: { status: 'Unknown', life_left_pct: -1 } });
		expect(s.factors.map((f) => f.id)).not.toContain('drive');
		expect(s.missing).toContain('drive health');
	});
});

describe('deductions', () => {
	it('add up to what was lost, give or take rounding', () => {
		const s = computeScore({
			...allGood,
			freeBytes: 12 * GB,
			memoryUsedPct: 85,
			batteryHealthPct: 70,
			security: { checks: [{ status: 'pass' }, { status: 'warn' }, { status: 'fail' }] }
		});
		const lost = deductions(s).reduce((sum, d) => sum + d.lost, 0);
		expect(Math.abs(100 - s.score - lost)).toBeLessThanOrEqual(3);
	});

	it('are sorted biggest first', () => {
		const d = deductions(computeScore({ ...allGood, freeBytes: 2 * GB, memoryUsedPct: 80 }));
		for (let i = 1; i < d.length; i++) expect(d[i - 1].lost).toBeGreaterThanOrEqual(d[i].lost);
	});
});

describe('scoreWord', () => {
	it('uses the documented bands', () => {
		expect(scoreWord(80)).toBe('Good');
		expect(scoreWord(79)).toBe('Fair');
		expect(scoreWord(60)).toBe('Fair');
		expect(scoreWord(59)).toBe('Poor');
	});
});

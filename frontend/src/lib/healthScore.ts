// The home screen's health score, worked out only from measured values, with
// every point lost explained. Processor load is left out: it changes every second.

export type Factor = {
	id: 'space' | 'memory' | 'drive' | 'battery' | 'security';
	max: number; // points this reading is worth
	points: number; // points it earns now
	detail: string; // the measured value, in plain words
	view: string; // the tool that explains it
};

export type ScoreInput = {
	freeBytes?: number;
	totalBytes?: number;
	memoryUsedPct?: number;
	// the system drive's own health report
	drive?: { status: string; life_left_pct: number } | null;
	// share of original capacity the battery holds now; undefined without a battery
	batteryHealthPct?: number;
	security?: { checks: { status: string }[] } | null;
};

export type Score = {
	score: number; // 0-100; -1 until anything is measured
	factors: Factor[];
	missing: string[]; // readings not in yet or not available
};

const clamp = (v: number, lo: number, hi: number) => Math.max(lo, Math.min(hi, v));
// straight line from (x0, y0) to (x1, y1), flat outside it
const ramp = (x: number, x0: number, y0: number, x1: number, y1: number) =>
	y0 + (y1 - y0) * clamp((x - x0) / (x1 - x0), 0, 1);

export function computeScore(input: ScoreInput, formatBytes: (b: number) => string): Score {
	const factors: Factor[] = [];
	const missing: string[] = [];

	if (input.totalBytes && input.freeBytes !== undefined) {
		const freePct = (input.freeBytes / input.totalBytes) * 100;
		// 20% free or more is fine; below 10% Windows itself starts to struggle
		const points = freePct >= 10 ? ramp(freePct, 10, 12.5, 20, 25) : ramp(freePct, 0, 0, 10, 12.5);
		factors.push({
			id: 'space',
			max: 25,
			points,
			detail: `Space: ${Math.round(freePct)}% free`,
			view: 'space'
		});
	} else missing.push('free space');

	if (input.memoryUsedPct !== undefined) {
		factors.push({
			id: 'memory',
			max: 20,
			points: ramp(input.memoryUsedPct, 70, 20, 95, 0),
			detail: `Memory: ${Math.round(input.memoryUsedPct)}% used`,
			view: 'slow'
		});
	} else missing.push('memory');

	if (input.drive && input.drive.status !== 'Unknown') {
		const d = input.drive;
		let points = d.status === 'Healthy' ? 20 : d.status === 'Warning' ? 8 : 0;
		// rated life below half starts to count, even on a healthy drive
		if (d.status === 'Healthy' && d.life_left_pct >= 0)
			points = ramp(d.life_left_pct, 0, 8, 50, 20);
		factors.push({
			id: 'drive',
			max: 20,
			points,
			detail:
				d.life_left_pct >= 0
					? `Drive: ${d.life_left_pct}% life left`
					: `Drive: ${d.status.toLowerCase()}`,
			view: 'health'
		});
	} else missing.push('drive health');

	if (input.batteryHealthPct !== undefined && input.batteryHealthPct > 0) {
		factors.push({
			id: 'battery',
			max: 15,
			points: ramp(input.batteryHealthPct, 60, 0, 90, 15),
			detail: `Battery: ${Math.round(input.batteryHealthPct)}% of new`,
			view: 'battery'
		});
	}

	const known = (input.security?.checks ?? []).filter((c) => c.status !== 'unknown');
	if (known.length) {
		const pass = known.filter((c) => c.status === 'pass').length;
		const warn = known.filter((c) => c.status === 'warn').length;
		factors.push({
			id: 'security',
			max: 20,
			points: (20 * (pass + warn * 0.25)) / known.length,
			detail: `Security: ${pass} of ${known.length}`,
			view: 'security'
		});
	} else missing.push('security checks');

	const max = factors.reduce((s, f) => s + f.max, 0);
	const got = factors.reduce((s, f) => s + f.points, 0);
	return { score: max ? Math.round((got / max) * 100) : -1, factors, missing };
}

// points lost, scaled the same way as the score, so the deductions add up to 100 - score
export function deductions(s: Score): { factor: Factor; lost: number }[] {
	const max = s.factors.reduce((t, f) => t + f.max, 0);
	if (!max) return [];
	return s.factors
		.map((f) => ({ factor: f, lost: Math.round(((f.max - f.points) / max) * 100) }))
		.filter((d) => d.lost >= 1)
		.sort((a, b) => b.lost - a.lost);
}

export const scoreTone = (score: number) =>
	score < 0
		? 'text-muted'
		: score >= 80
			? 'text-green-600 dark:text-green-400'
			: score >= 60
				? 'text-amber-500'
				: 'text-red-500';

export const scoreWord = (score: number) => (score >= 80 ? 'Good' : score >= 60 ? 'Fair' : 'Poor');

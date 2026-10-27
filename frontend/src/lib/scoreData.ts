import { load } from '$lib/cache';
import type { ScoreInput } from '$lib/healthScore';
import {
	GetBatteryReport,
	GetDriveHealth,
	GetDrives,
	GetMemoryStats,
	GetSecurityReport
} from '$lib/wailsjs/go/main/App';

// Everything the health score is worked out from, read fresh (the slow checks
// come from the shared cache when their screens ran recently)
export async function loadScoreInput(): Promise<ScoreInput> {
	const quiet = <T>(p: Promise<T>) => p.catch(() => undefined);
	const [drives, memory, battery, driveHealth, security] = await Promise.all([
		quiet(GetDrives()),
		quiet(GetMemoryStats()),
		// capacity as Windows' battery report records it, the Battery tool's own source
		quiet(load('battery-report', GetBatteryReport, 10 * 60_000)),
		quiet(load('drive-health', GetDriveHealth, 5 * 60_000)),
		quiet(load('security', GetSecurityReport, 5 * 60_000))
	]);
	const system = (drives as any[] | undefined)?.[0];
	const health = (driveHealth as any[] | undefined) ?? [];
	return {
		freeBytes: system?.free_bytes,
		totalBytes: system?.total_bytes,
		memoryUsedPct: (memory as any)?.usagePercent,
		drive: health.find((d) => /^c:/i.test(d.drive ?? '')) ?? health[0] ?? null,
		// left out (not counted as 100%) when the battery does not report its capacity
		batteryHealthPct:
			(battery as any)?.has_battery && (battery as any).health_pct > 0
				? (battery as any).health_pct
				: undefined,
		security: (security as any) ?? null
	};
}

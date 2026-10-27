import { load } from '$lib/cache';
import type { ScoreInput } from '$lib/healthScore';
import {
	GetBatteryDetails,
	GetDriveHealth,
	GetDrives,
	GetMemoryStats,
	GetSecurityReport
} from '$lib/wailsjs/go/main/App';

// Everything the health score is worked out from, read fresh (the slow checks
// come from the shared cache when their screens ran recently)
export async function loadScoreInput(): Promise<ScoreInput> {
	const quiet = <T>(p: Promise<T>) => p.catch(() => undefined);
	const [drives, memory, batteryJson, driveHealth, security] = await Promise.all([
		quiet(GetDrives()),
		quiet(GetMemoryStats()),
		quiet(GetBatteryDetails()),
		quiet(load('drive-health', GetDriveHealth, 5 * 60_000)),
		quiet(load('security', GetSecurityReport, 5 * 60_000))
	]);
	let battery: any = null;
	try {
		battery = batteryJson ? JSON.parse(batteryJson) : null;
	} catch {
		battery = null;
	}
	const system = (drives as any[] | undefined)?.[0];
	const health = (driveHealth as any[] | undefined) ?? [];
	return {
		freeBytes: system?.free_bytes,
		totalBytes: system?.total_bytes,
		memoryUsedPct: (memory as any)?.usagePercent,
		drive: health.find((d) => /^c:/i.test(d.drive ?? '')) ?? health[0] ?? null,
		batteryHealthPct:
			battery && battery.design_capacity > 0 ? 100 - (battery.degradation_pct ?? 0) : undefined,
		security: (security as any) ?? null
	};
}

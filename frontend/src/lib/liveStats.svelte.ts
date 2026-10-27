import { GetCPUStats, GetMemoryStats } from '$lib/wailsjs/go/main/App';
import type { functions } from '$lib/wailsjs/go/models';
import { EventsOn } from '$lib/wailsjs/runtime/runtime';

// The latest processor and memory readings, pushed by the backend every few
// seconds. One store for the whole app, so the cards that show them can come and
// go without each opening (and possibly leaking) its own listener.
export const live = $state<{
	cpu: functions.CPUStats | null;
	memory: functions.MemoryStats | null;
}>({ cpu: null, memory: null });

let listening = false;

// safe to call many times; it starts listening once
export function startLiveStats() {
	if (listening) return;
	listening = true;
	EventsOn('cpu-stats-update', (data: functions.CPUStats) => (live.cpu = data));
	EventsOn('ram-stats-update', (data: functions.MemoryStats) => (live.memory = data));
	// a first reading at once, rather than waiting for the first event
	GetCPUStats()
		.then((d) => (live.cpu ??= d))
		.catch(() => {});
	GetMemoryStats()
		.then((d) => (live.memory ??= d))
		.catch(() => {});
}

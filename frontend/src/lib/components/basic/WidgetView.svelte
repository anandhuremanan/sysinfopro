<script lang="ts">
	import { formatBytes, isBatteryCharging, usageBarColor } from '$lib';
	import {
		GetBatteryDetails,
		GetCPUStats,
		GetDrives,
		GetMemoryStats
	} from '$lib/wailsjs/go/main/App';
	import { EventsOn } from '$lib/wailsjs/runtime/runtime';
	import { exitWidget, widget, type WidgetMetric } from '$lib/widget.svelte';
	import { onDestroy } from 'svelte';
	import Logo from '../ui/Logo.svelte';
	import StrokeIcon from '../ui/StrokeIcon.svelte';

	const DISK_REFRESH_MS = 5000;
	const BATTERY_REFRESH_MS = 30000;

	let cpu = $state<number | undefined>(undefined);
	let memory = $state<{ usagePercent: number; usedRAMMB: number; totalRAMMB: number } | undefined>(
		undefined
	);
	let disk = $state<{ free_bytes: number; used_pct: number } | undefined>(undefined);
	let battery: any = $state(undefined);
	let batteryLoaded = $state(false);

	type Row = { id: WidgetMetric; label: string; value: string; detail: string; pct: number; color: string };

	// one row per metric the user chose, in a fixed order
	let rows = $derived(
		widget.metrics.map((id): Row => {
			if (id === 'cpu') {
				const pct = cpu ?? 0;
				return {
					id,
					label: 'CPU',
					value: cpu === undefined ? '…' : `${pct.toFixed(0)}%`,
					detail: '',
					pct,
					color: pct > 80 ? 'bg-red-500' : pct > 60 ? 'bg-amber-500' : 'bg-accent'
				};
			}
			if (id === 'memory') {
				const pct = memory?.usagePercent ?? 0;
				return {
					id,
					label: 'Memory',
					value: memory ? `${pct.toFixed(0)}%` : '…',
					detail: memory ? `${(memory.usedRAMMB / 1024).toFixed(1)} of ${(memory.totalRAMMB / 1024).toFixed(1)} GB` : '',
					pct,
					color: usageBarColor(pct)
				};
			}
			if (id === 'disk') {
				const pct = disk?.used_pct ?? 0;
				return {
					id,
					label: 'Disk',
					value: disk ? `${pct.toFixed(0)}%` : '…',
					detail: disk ? `${formatBytes(disk.free_bytes)} free` : '',
					pct,
					color: usageBarColor(pct)
				};
			}
			const pct = battery?.percentage ?? 0;
			return {
				id,
				label: 'Battery',
				value: battery ? `${pct.toFixed(0)}%` : batteryLoaded ? 'None' : '…',
				detail: battery ? (isBatteryCharging(battery.state) ? 'Charging' : battery.state || '') : '',
				pct,
				color: !battery ? 'bg-track' : pct <= 20 ? 'bg-red-500' : pct <= 40 ? 'bg-amber-500' : 'bg-accent'
			};
		})
	);

	async function refreshDisk() {
		try {
			// the system drive is listed first
			const drives = await GetDrives();
			if (drives?.[0]) disk = drives[0];
		} catch (err) {
			console.error('Failed to refresh disk usage:', err);
		}
	}

	async function refreshBattery() {
		try {
			const res = await GetBatteryDetails();
			battery = res ? JSON.parse(res) : null;
		} catch (err) {
			console.error('Failed to get battery details:', err);
		} finally {
			batteryLoaded = true;
		}
	}

	GetCPUStats()
		.then((stats) => (cpu = stats?.usage ?? 0))
		.catch((err) => console.error('Failed to get CPU stats:', err));
	GetMemoryStats()
		.then((stats) => (memory = stats as any))
		.catch((err) => console.error('Failed to get RAM stats:', err));

	const stopCpu = EventsOn('cpu-stats-update', (stats: { usage: number }) => {
		if (stats) cpu = stats.usage || 0;
	});
	const stopMemory = EventsOn('ram-stats-update', (stats: any) => {
		if (stats) memory = stats;
	});

	$effect(() => {
		refreshDisk();
		refreshBattery();
		const diskTimer = setInterval(refreshDisk, DISK_REFRESH_MS);
		const batteryTimer = setInterval(refreshBattery, BATTERY_REFRESH_MS);
		return () => {
			clearInterval(diskTimer);
			clearInterval(batteryTimer);
		};
	});

	onDestroy(() => {
		stopCpu();
		stopMemory();
	});
</script>

<section
	class="h-screen w-screen flex flex-col px-2 pt-1 pb-1.5 bg-ground text-ink text-xs overflow-hidden select-none"
>
	<header class="flex items-center justify-between h-5 shrink-0">
		<Logo size={14} showWordmark={false} />
		<button
			aria-label="Back to full window"
			title="Back to full window"
			class="w-5 h-5 rounded-md flex items-center justify-center text-muted cursor-pointer hover:text-ink hover:bg-line transition-colors"
			onclick={exitWidget}
		>
			<StrokeIcon d="M4 14v6h6M20 10V4h-6M4 20l7-7M20 4l-7 7" size={12} />
		</button>
	</header>

	<!-- one compact line per reading: name, bar, value -->
	{#each rows as row (row.id)}
		<div class="flex-1 min-h-0 flex items-center gap-2" title={row.detail || undefined}>
			<span class="w-[50px] shrink-0 text-muted font-medium">{row.label}</span>
			<span class="flex-1 h-1 rounded-full bg-track">
				<span
					class="block h-1 rounded-full transition-all duration-700 ease-out {row.color}"
					style="width: {Math.min(100, Math.max(0, row.pct))}%"
				></span>
			</span>
			<span class="w-9 shrink-0 text-right font-semibold tabular-nums">{row.value}</span>
		</div>
	{/each}
</section>

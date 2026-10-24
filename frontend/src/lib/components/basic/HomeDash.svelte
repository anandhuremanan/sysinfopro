<script lang="ts">
	import { formatBytes, isBatteryCharging, usageBarColor } from '$lib';
	import { ALL_TOOLS_ICON, getTool, pinnedTools, tools, type ToolStatus } from '$lib/tools';
	import {
		GetBatteryDetails,
		GetDiagnosticReport,
		GetDrives,
		GetTopProcesses
	} from '$lib/wailsjs/go/main/App';
	import BatteryGauge from '../ui/BatteryGauge.svelte';
	import ToolTile from '../ui/ToolTile.svelte';
	import CpuStats from './CpuStats.svelte';
	import RamStats from './RamStats.svelte';

	let {
		systemInfo,
		toolStatus,
		onOpen,
		onSpeedTest
	}: {
		systemInfo: any;
		toolStatus: Record<string, ToolStatus>;
		onOpen: (view: string) => void;
		onSpeedTest: () => void;
	} = $props();

	let report: any = $state(undefined);
	let battery: any = $state(undefined);
	let batteryLoaded = $state(false);
	let topProcess: any = $state(undefined);

	const RING = 2 * Math.PI * 53; // r = 53

	// Every tile on the home grid opens something
	const tile =
		'bg-tile rounded-[14px] min-w-0 text-left cursor-pointer transition-transform active:scale-[0.99] hover:ring-1 hover:ring-inset hover:ring-line';

	let score = $derived(Math.round(report?.overall_score ?? 0));
	let issueCount = $derived(report?.recommendations?.length ?? 0);
	let scoreColor = $derived(
		!report
			? 'text-muted'
			: score >= 80
				? 'text-accent'
				: score >= 60
					? 'text-amber-500'
					: 'text-red-500'
	);

	const GB = 1024 * 1024 * 1024;
	const DISK_REFRESH_MS = 5000;

	// Latest reading of the system drive; refreshed while the home screen is open
	let liveDisk = $state<{ total_bytes: number; free_bytes: number } | undefined>(undefined);

	// live reading first, then the exact bytes from startup, then whole gigabytes
	let diskTotalBytes = $derived(
		liveDisk?.total_bytes ?? systemInfo.total_disk_bytes ?? systemInfo.total_disk_gb * GB
	);
	let diskFreeBytes = $derived(
		liveDisk?.free_bytes ?? systemInfo.free_disk_bytes ?? systemInfo.free_disk_gb * GB
	);
	// "250 MB" -> ["250", "MB"], so a nearly full drive never reads as "0 GB"
	let diskFree = $derived(formatBytes(diskFreeBytes).split(' '));

	let diskUsedPct = $derived(
		diskTotalBytes > 0 ? Math.min(100, Math.max(0, 100 - (diskFreeBytes / diskTotalBytes) * 100)) : 0
	);

	async function loadReport() {
		try {
			report = await GetDiagnosticReport();
		} catch (err) {
			console.error('Failed to get diagnostic report:', err);
		}
	}

	async function loadBattery() {
		try {
			const res = await GetBatteryDetails();
			battery = res ? JSON.parse(res) : null;
		} catch (err) {
			console.error('Failed to get battery details:', err);
		} finally {
			batteryLoaded = true;
		}
	}

	async function loadTopProcess() {
		try {
			const res = await GetTopProcesses(1);
			topProcess = res?.[0];
		} catch (err) {
			console.error('Failed to get processes:', err);
		}
	}

	async function refreshDisk() {
		try {
			// the system drive is listed first, the same one the startup snapshot describes
			const drives = await GetDrives();
			if (drives?.[0]) liveDisk = drives[0];
		} catch (err) {
			console.error('Failed to refresh disk usage:', err);
		}
	}

	$effect(() => {
		loadReport();
		loadBattery();
		loadTopProcess();
	});

	$effect(() => {
		refreshDisk();
		const timer = setInterval(refreshDisk, DISK_REFRESH_MS);
		return () => clearInterval(timer);
	});
</script>

<div class="h-full grid grid-cols-4 grid-rows-[minmax(0,1fr)_minmax(0,1fr)_82px] gap-3">
	<!-- Health score (hero) -->
	<button class="{tile} row-span-2 p-4 flex flex-col gap-2" onclick={() => onOpen('health')}>
		<span class="text-muted font-medium">Health score</span>
		<span class="relative w-[124px] h-[124px] self-center my-auto {scoreColor}">
			<svg width="124" height="124" viewBox="0 0 124 124" fill="none" class="-rotate-90" aria-hidden="true">
				<circle cx="62" cy="62" r="53" stroke-width="10" class="stroke-track" />
				<circle
					cx="62"
					cy="62"
					r="53"
					stroke="currentColor"
					stroke-width="10"
					stroke-linecap="round"
					stroke-dasharray="{(RING * score) / 100} {RING}"
					class="transition-all duration-700 ease-out"
				/>
			</svg>
			<span class="absolute inset-0 flex flex-col items-center justify-center text-ink">
				<span class="text-[34px] font-semibold tracking-tight leading-none">
					{report ? score : '–'}
				</span>
				<span class="text-muted">of 100</span>
			</span>
		</span>
		<span class="flex flex-col gap-0.5 min-w-0 w-full">
			<span class="flex items-center gap-1.5 font-semibold min-w-0">
				<span class="w-1.5 h-1.5 rounded-full bg-current shrink-0 {scoreColor}"></span>
				<span class="truncate">{report?.status_text || 'System active'}</span>
			</span>
			{#if issueCount > 0}
				<span class="flex items-center gap-1.5 min-w-0">
					<span
						class="shrink-0 min-w-[18px] h-[18px] px-[5px] rounded-full bg-amber-200 text-amber-900 text-[11px] font-bold leading-[18px] text-center"
					>
						{issueCount}
					</span>
					<span class="text-muted truncate">to review</span>
				</span>
			{/if}
		</span>
	</button>

	<!-- Live gauges -->
	<button class="{tile} px-4 py-3.5 flex flex-col" title="Run a stress test" onclick={() => onOpen('stress')}>
		<CpuStats />
	</button>

	<button class="{tile} px-4 py-3.5 flex flex-col" onclick={() => onOpen('processes')}>
		<RamStats />
	</button>

	<button class="{tile} px-4 py-3.5 flex flex-col justify-between" onclick={() => onOpen('space')}>
		<span class="flex justify-between text-muted font-medium w-full">
			<span>Storage</span>
			<span>{systemInfo.disk_type}</span>
		</span>
		<span class="text-[28px] font-semibold tracking-tight leading-none tabular-nums">
			{diskFree[0]}<span class="text-sm text-muted ml-1">{diskFree[1]} free</span>
		</span>
		<span class="flex flex-col gap-[5px] w-full">
			<span class="block h-[5px] rounded-full bg-track">
				<span
					class="block h-[5px] rounded-full transition-all duration-700 ease-out {usageBarColor(
						diskUsedPct
					)}"
					style="width: {diskUsedPct}%"
				></span>
			</span>
			<span class="text-muted">of {formatBytes(diskTotalBytes)}</span>
		</span>
	</button>

	<button class="{tile} px-4 py-3.5 flex flex-col justify-between" onclick={() => onOpen('processes')}>
		<span class="text-muted font-medium">Processes</span>
		{#if topProcess}
			<span class="flex flex-col min-w-0 w-full">
				<span class="text-[15px] font-semibold truncate" title={topProcess.name}>
					{topProcess.name}
				</span>
				<span class="text-muted truncate">
					Top consumer · {topProcess.memory_mb.toFixed(0)} MB
				</span>
			</span>
		{:else}
			<span class="text-[15px] font-semibold">Top consumers</span>
		{/if}
		<span class="font-semibold">View top processes</span>
	</button>

	<button class="{tile} px-4 py-3.5 flex flex-col justify-between" onclick={onSpeedTest}>
		<span class="text-muted font-medium">Network</span>
		<span class="text-xl font-semibold tracking-tight leading-none capitalize truncate w-full">
			{systemInfo.network_status}
		</span>
		<span class="font-semibold">Run speed test</span>
	</button>

	<button class="{tile} px-4 py-3.5 flex flex-col justify-between" onclick={() => onOpen('health')}>
		<span class="text-muted font-medium">Battery</span>
		{#if battery}
			<span class="flex items-center justify-between gap-2 w-full">
				<span class="text-xl font-semibold tracking-tight leading-none">
					{battery.percentage.toFixed(0)}%<span
						class="text-xs font-medium text-muted capitalize ml-1.5">{battery.state}</span
					>
				</span>
				<BatteryGauge
					width={40}
					percent={battery.percentage}
					charging={isBatteryCharging(battery.state)}
				/>
			</span>
			<span class="text-muted truncate w-full">
				{#if battery.health_status}
					{battery.health_status} health{battery.degradation_pct !== undefined
						? ` · ${battery.degradation_pct}% wear`
						: ''}
				{:else}
					{battery.current_capacity} of {battery.design_capacity} mWh
				{/if}
			</span>
		{:else}
			<span class="text-[15px] font-semibold">
				{batteryLoaded ? 'No battery' : 'Checking...'}
			</span>
			<span class="text-muted truncate w-full">
				{batteryLoaded ? 'Laptop hardware only' : ''}
			</span>
		{/if}
	</button>

	<!-- Pinned tools -->
	<nav aria-label="Tools" class="col-span-4 grid grid-cols-6 gap-3">
		{#each pinnedTools as id (id)}
			{@const tool = getTool(id)}
			<ToolTile
				compact
				name={tool.name}
				hint={toolStatus[id]?.hint ?? tool.hint}
				icon={tool.icon}
				count={toolStatus[id]?.count ?? 0}
				onclick={() => onOpen(id)}
			/>
		{/each}
		<ToolTile
			compact
			name="All tools"
			hint="{tools.length} tools"
			icon={ALL_TOOLS_ICON}
			onclick={() => onOpen('tools')}
		/>
	</nav>
</div>

<script lang="ts">
	import { formatBytes, isBatteryCharging, usageBarColor } from '$lib';
	import { ALL_TOOLS_ICON, getTool, pinnedTools, tools, type ToolStatus } from '$lib/tools';
	import {
		CheckInternetConnection,
		GetBatteryDetails,
		GetDriveHealth,
		GetDrives,
		GetMemoryStats,
		GetBatteryReport,
		GetSecurityReport,
		GetTopProcesses
	} from '$lib/wailsjs/go/main/App';
	import { load } from '$lib/cache';
	import { computeScore, deductions, scoreTone, scoreWord } from '$lib/healthScore';
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

	let battery: any = $state(undefined);
	let batteryLoaded = $state(false);
	// share of original capacity, from Windows' battery report (the Battery tool's source)
	let batteryHealth = $state<number | undefined>(undefined);
	let topProcess: any = $state(undefined);
	// rechecked while home is open; until the first check, the startup reading
	let online = $state<boolean | null>(null);
	async function checkOnline() {
		online = await CheckInternetConnection().catch(() => null);
	}
	let memoryUsedPct = $state<number | undefined>(undefined);
	let driveHealth = $state<any[] | null>(null);
	let security = $state<any>(null);

	// Every tile on the home grid opens something
	const tile =
		'bg-tile rounded-[14px] min-w-0 text-left cursor-pointer transition-transform active:scale-[0.99] hover:ring-1 hover:ring-inset hover:ring-line';

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
		diskTotalBytes > 0
			? Math.min(100, Math.max(0, 100 - (diskFreeBytes / diskTotalBytes) * 100))
			: 0
	);

	const RING = 2 * Math.PI * 53; // r = 53
	// The score is worked out only from measured values: free space on the system drive
	// (25 points), memory in use (20), drive health (20), battery capacity (15) and
	// security checks (20). Processor load is left out because it changes every second.

	let scored = $derived(
		computeScore(
			{
				freeBytes: diskFreeBytes,
				totalBytes: diskTotalBytes,
				memoryUsedPct,
				drive: driveHealth?.find((d) => /^c:/i.test(d.drive ?? '')) ?? driveHealth?.[0] ?? null,
				batteryHealthPct: batteryHealth,
				security
			},
			formatBytes
		)
	);
	let lost = $derived(deductions(scored));
	// what lowers the score most, in two or three words
	const CAUSE: Record<string, string> = {
		space: 'Low free space',
		memory: 'Memory in use',
		drive: 'Drive wear',
		battery: 'Battery wear',
		security: 'Security checks'
	};

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

	async function refreshMemory() {
		try {
			memoryUsedPct = (await GetMemoryStats())?.usagePercent;
		} catch (err) {
			console.error('Failed to read memory:', err);
		}
	}

	// slow checks, shared with their own screens through the cache
	function loadSlowChecks() {
		load('drive-health', GetDriveHealth, 5 * 60_000)
			.then((d) => (driveHealth = (d || []) as any[]))
			.catch(() => {});
		load('security', GetSecurityReport, 5 * 60_000)
			.then((r) => (security = r))
			.catch(() => {});
		load('battery-report', GetBatteryReport, 10 * 60_000)
			.then(
				(r: any) => (batteryHealth = r?.has_battery && r.health_pct > 0 ? r.health_pct : undefined)
			)
			.catch(() => {});
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
		loadBattery();
		loadSlowChecks();
		loadTopProcess();
	});

	$effect(() => {
		refreshDisk();
		refreshMemory();
		checkOnline();
		const netTimer = setInterval(checkOnline, 30_000);
		const timer = setInterval(() => {
			refreshDisk();
			refreshMemory();
		}, DISK_REFRESH_MS);
		return () => {
			clearInterval(timer);
			clearInterval(netTimer);
		};
	});
</script>

<div class="h-full grid grid-cols-4 grid-rows-[minmax(0,1fr)_minmax(0,1fr)_82px] gap-3">
	<!-- Health score (hero): worked out from measured values only; point at it for the working -->
	<button class="{tile} row-span-2 p-4 flex flex-col gap-2" onclick={() => onOpen('score')}>
		<span class="text-muted font-medium">Health score</span>
		<span class="relative w-[124px] h-[124px] self-center my-auto {scoreTone(scored.score)}">
			<svg
				width="124"
				height="124"
				viewBox="0 0 124 124"
				fill="none"
				class="-rotate-90"
				aria-hidden="true"
			>
				<circle cx="62" cy="62" r="53" stroke-width="10" class="stroke-track" />
				<circle
					cx="62"
					cy="62"
					r="53"
					stroke="currentColor"
					stroke-width="10"
					stroke-linecap="round"
					stroke-dasharray="{(RING * Math.max(0, scored.score)) / 100} {RING}"
					class="transition-all duration-700 ease-out"
				/>
			</svg>
			<span class="absolute inset-0 flex flex-col items-center justify-center text-ink">
				<span class="text-[34px] font-semibold tracking-tight leading-none">
					{scored.score < 0 ? '–' : scored.score}
				</span>
				<span class="text-muted">of 100</span>
			</span>
		</span>
		<span class="flex flex-col gap-0.5 min-w-0 w-full">
			<span class="flex items-center gap-1.5 font-semibold min-w-0">
				<span class="w-1.5 h-1.5 rounded-full bg-current shrink-0 {scoreTone(scored.score)}"></span>
				<span class="truncate">{scored.score < 0 ? 'Checking...' : scoreWord(scored.score)}</span>
			</span>
			<span class="text-muted truncate">
				{#if scored.score < 0}
					&nbsp;
				{:else if lost.length}
					{CAUSE[lost[0].factor.id]} −{lost[0].lost}
				{:else}
					Nothing is lowering it
				{/if}
			</span>
		</span>
	</button>

	<!-- Live gauges -->
	<!-- the whole card explains what is using the processor; the link runs a stress test -->
	<div class="relative {tile} px-4 py-3.5 flex flex-col">
		<button
			class="absolute inset-0 rounded-[14px] cursor-pointer"
			aria-label="Open processor details"
			onclick={() => onOpen('cpu')}
		></button>
		<CpuStats onStress={() => onOpen('stress')} />
	</div>

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

	<button
		class="{tile} px-4 py-3.5 flex flex-col justify-between"
		onclick={() => onOpen('processes')}
	>
		<span class="text-muted font-medium">Processes</span>
		{#if topProcess}
			<span class="flex flex-col min-w-0 w-full">
				<span class="text-[15px] font-semibold truncate" title={topProcess.name}>
					{topProcess.name}
				</span>
				<span class="text-muted truncate">
					Most memory · {topProcess.memory_mb.toFixed(0)} MB
				</span>
			</span>
		{:else}
			<span class="text-[15px] font-semibold">Top consumers</span>
		{/if}
		<span class="font-semibold text-accent">View top processes →</span>
	</button>

	<button class="{tile} px-4 py-3.5 flex flex-col justify-between" onclick={onSpeedTest}>
		<span class="text-muted font-medium">Network</span>
		<span class="text-xl font-semibold tracking-tight leading-none capitalize truncate w-full">
			{online === null ? systemInfo.network_status : online ? 'Connected' : 'No internet'}
		</span>
		<span class="font-semibold text-accent">Run speed test →</span>
	</button>

	<button
		class="{tile} px-4 py-3.5 flex flex-col justify-between"
		onclick={() => onOpen('battery')}
	>
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
				<!-- capacity from Windows' battery report, worded like the Battery tool -->
				{#if batteryHealth !== undefined}
					Health {Math.round(batteryHealth)}% of new
				{:else}
					Battery health: see Battery
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

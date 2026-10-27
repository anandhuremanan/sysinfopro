<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { GetUptimeSeconds } from '$lib/wailsjs/go/main/App';
	import { live, startLiveStats } from '$lib/liveStats.svelte';
	import type { functions } from '$lib/wailsjs/go/models';

	let { onStress }: { onStress?: () => void } = $props();

	// time since Windows last started; with Fast Startup, "Shut down" does not reset it
	let uptimeSec = $state(0);
	let uptimeText = $derived.by(() => {
		const h = Math.floor(uptimeSec / 3600);
		if (h < 1) return `${Math.max(1, Math.floor(uptimeSec / 60))} min`;
		if (h < 48) return `${h} h`;
		return `${Math.floor(h / 24)} days`;
	});
	let longUptime = $derived(uptimeSec > 14 * 86400);
	async function readUptime() {
		uptimeSec = (await GetUptimeSeconds().catch(() => 0)) || 0;
	}

	let cpuUsage = $state(0);
	let freqGHz = $state(0);
	let isThrottling = $state(false);
	let sustainedLoad = $state(false);

	let timestamps = $state<string[]>([]);
	let usageHistory = $state<number[]>([0, 0, 0, 0, 0, 0, 0, 0, 0, 0]);
	const MAX_HISTORY_LENGTH = 30;

	let formattedCpuUsage = $derived(cpuUsage.toFixed(1));

	let cpuColor = $derived(
		cpuUsage > 80 ? 'text-fail' : cpuUsage > 60 ? 'text-warn' : 'text-accent'
	);

	let sparklinePath = $derived.by(() => {
		if (usageHistory.length < 2) return '';
		const width = 72;
		const height = 26;
		const step = width / (usageHistory.length - 1);

		return usageHistory.reduce((acc, val, idx) => {
			const x = idx * step;
			const y = height - (val / 100) * height;
			return idx === 0 ? `M ${x} ${y}` : `${acc} L ${x} ${y}`;
		}, '');
	});

	function updateStats(stats: functions.CPUStats) {
		if (!stats) return;
		cpuUsage = stats.usage || 0;
		freqGHz = stats.frequency_ghz || 0;
		isThrottling = !!stats.is_throttling;
		sustainedLoad = !!stats.sustained_load;

		timestamps = [
			...timestamps,
			new Date((stats.timestamp || Date.now() / 1000) * 1000).toLocaleTimeString()
		];
		usageHistory = [...usageHistory, stats.usage || 0];

		if (usageHistory.length > MAX_HISTORY_LENGTH) {
			timestamps = timestamps.slice(1);
			usageHistory = usageHistory.slice(1);
		}
	}

	// each new reading from the shared store extends the sparkline
	$effect(() => {
		const stats = live.cpu;
		if (stats) untrack(() => updateStats(stats));
	});

	onMount(() => {
		startLiveStats();
		readUptime();
		const timer = setInterval(readUptime, 60_000);
		return () => clearInterval(timer);
	});
</script>

<div class="flex-1 w-full flex flex-col justify-between">
	<div class="flex justify-between text-muted font-medium">
		<span>CPU</span>
		{#if freqGHz > 0}
			<span>{freqGHz} GHz</span>
		{/if}
	</div>

	<div class="flex items-end justify-between gap-2">
		<div class="text-[28px] font-semibold tracking-tight leading-none tabular-nums">
			{formattedCpuUsage}<span class="text-sm text-muted">%</span>
		</div>

		<!-- Mini inline 30-second SVG sparkline graph -->
		<svg width="72" height="26" class="overflow-visible shrink-0" aria-hidden="true">
			<path
				d={sparklinePath}
				fill="none"
				stroke="currentColor"
				stroke-width="1.75"
				stroke-linecap="round"
				stroke-linejoin="round"
				class="{cpuColor} transition-all duration-500 ease-in-out"
			/>
		</svg>
	</div>

	<div class="flex flex-col gap-0.5">
		{#if isThrottling}
			<span
				class="font-semibold text-warn"
				title="Busy, yet running below its base speed: usually heat or a power-saving setting"
				>Running below base speed</span
			>
		{:else if sustainedLoad}
			<span class="font-semibold text-warn">Busy for a while now</span>
		{:else if cpuUsage > 80}
			<span class="font-semibold text-warn">Busy right now</span>
		{:else}
			<span class="text-muted">Normal load</span>
		{/if}
		<span class="flex items-center justify-between gap-2 text-[11px]">
			<span
				class={longUptime ? 'text-warn' : 'text-muted'}
				title={longUptime
					? 'A restart clears memory and finishes updates. With Fast Startup on, Shut down does not count as a restart.'
					: 'Time since Windows last started'}
			>
				{uptimeSec ? `Up ${uptimeText}${longUptime ? ' · restart?' : ''}` : ''}
			</span>
			{#if onStress}
				<button
					class="relative z-10 font-semibold text-accent cursor-pointer hover:underline shrink-0"
					onclick={onStress}
				>
					Stress test →
				</button>
			{/if}
		</span>
	</div>
</div>

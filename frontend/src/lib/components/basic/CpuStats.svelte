<script lang="ts">
	import { GetCPUStats } from '../../wailsjs/go/main/App';

	interface CPUStats {
		usage: number;
		timestamp: number;
		frequency_ghz?: number;
		is_throttling?: boolean;
		sustained_load?: boolean;
	}

	let cpuUsage = $state(0);
	let freqGHz = $state(0);
	let isThrottling = $state(false);
	let sustainedLoad = $state(false);

	let timestamps = $state<string[]>([]);
	let usageHistory = $state<number[]>([0, 0, 0, 0, 0, 0, 0, 0, 0, 0]);
	const MAX_HISTORY_LENGTH = 30;
	let unsubscribe = $state<(() => void) | undefined>(undefined);

	let formattedCpuUsage = $derived(cpuUsage.toFixed(1));

	let cpuColor = $derived(
		cpuUsage > 80 ? 'text-red-500' : cpuUsage > 60 ? 'text-amber-500' : 'text-green-500'
	);

	let sparklinePath = $derived.by(() => {
		if (usageHistory.length < 2) return '';
		const width = 80;
		const height = 28;
		const step = width / (usageHistory.length - 1);

		return usageHistory.reduce((acc, val, idx) => {
			const x = idx * step;
			const y = height - (val / 100) * height;
			return idx === 0 ? `M ${x} ${y}` : `${acc} L ${x} ${y}`;
		}, '');
	});

	function updateStats(stats: CPUStats) {
		if (!stats) return;
		cpuUsage = stats.usage || 0;
		freqGHz = stats.frequency_ghz || 0;
		isThrottling = !!stats.is_throttling;
		sustainedLoad = !!stats.sustained_load;

		timestamps = [...timestamps, new Date((stats.timestamp || Date.now() / 1000) * 1000).toLocaleTimeString()];
		usageHistory = [...usageHistory, stats.usage || 0];

		if (usageHistory.length > MAX_HISTORY_LENGTH) {
			timestamps = timestamps.slice(1);
			usageHistory = usageHistory.slice(1);
		}
	}

	async function init() {
		try {
			const initialStats = await GetCPUStats();
			updateStats(initialStats);

			// @ts-ignore
			if (window.runtime) {
				// @ts-ignore
				unsubscribe = window.runtime.EventsOn('cpu-stats-update', (data: CPUStats) => {
					updateStats(data);
				});
			}
		} catch (error) {
			console.error('Failed to get CPU stats:', error);
		}
	}

	$effect(() => {
		init();
	});

	$effect(() => {
		return () => {
			if (unsubscribe) {
				unsubscribe();
			}
		};
	});
</script>

<div class="flex flex-col gap-1">
	<div class="flex justify-between items-center text-xs">
		<span>CPU Utilization</span>
		{#if freqGHz > 0}
			<span class="text-[10px] text-gray-400 font-mono">{freqGHz} GHz</span>
		{/if}
	</div>
	
	<div class="flex items-center gap-3">
		<div class="{cpuColor} fill-current text-5xl font-bold tracking-tight">
			{formattedCpuUsage}%
		</div>

		<!-- Mini inline 30-second SVG sparkline graph -->
		<div class="w-20 h-8 flex items-center justify-center opacity-80">
			<svg width="80" height="28" class="overflow-visible">
				<path
					d={sparklinePath}
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
					class="{cpuColor} transition-all duration-500 ease-in-out"
				/>
			</svg>
		</div>
	</div>

	<!-- Intelligent Diagnostics Badges -->
	{#if isThrottling}
		<div class="text-[10px] bg-red-500/20 text-red-400 px-1.5 py-0.5 rounded font-semibold w-fit border border-red-500/30">
			⚠️ Thermal Throttling Detected
		</div>
	{:else if sustainedLoad}
		<div class="text-[10px] bg-amber-500/20 text-amber-400 px-1.5 py-0.5 rounded font-semibold w-fit border border-amber-500/30">
			⚡ Sustained Heavy Load
		</div>
	{/if}
</div>

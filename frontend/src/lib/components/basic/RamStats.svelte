<script lang="ts">
	import { GetMemoryStats, GetMemoryVelocity } from '$lib/wailsjs/go/main/App';

	type MemoryStats = {
		totalRAMMB: number;
		freeRAMMB: number;
		usedRAMMB: number;
		usagePercent: number;
		timestamp: number;
		swapTotalMB?: number;
		swapUsedMB?: number;
		pressureLevel?: string;
	};

	type VelocitySample = {
		velocity_mbps: number;
		status_text: string;
		estimated_oom_sec: number;
	};

	let memStats: MemoryStats | null = $state(null);
	let velocitySample: VelocitySample | null = $state(null);
	let unsubscribe: (() => void) | null = $state(null);
	let velocityInterval: any = $state();

	function formatRAM(mb: number): string {
		if (!mb) return '0 MB';
		if (mb > 1024) {
			return (mb / 1024).toFixed(2) + ' GB';
		}
		return mb.toFixed(0) + ' MB';
	}

	function updateStats(data: MemoryStats) {
		memStats = data;
	}

	async function pollVelocity() {
		try {
			// @ts-ignore
			const res = await GetMemoryVelocity();
			if (res) {
				velocitySample = res;
			}
		} catch (err) {
			console.error('Velocity poll error:', err);
		}
	}

	async function init() {
		try {
			const initialStats = await GetMemoryStats();
			updateStats(initialStats);
			await pollVelocity();

			// @ts-ignore
			if (window.runtime) {
				// @ts-ignore
				unsubscribe = window.runtime.EventsOn('ram-stats-update', (data: MemoryStats) => {
					updateStats(data);
				});
			}

			velocityInterval = setInterval(pollVelocity, 3000);
		} catch (error) {
			console.error('Failed to get RAM stats:', error);
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
			if (velocityInterval) {
				clearInterval(velocityInterval);
			}
		};
	});

	function getUsageColor(percent: number): string {
		if (percent < 50) return 'stroke-green-500';
		if (percent < 80) return 'stroke-yellow-500';
		return 'stroke-red-500';
	}

	const circumference = 2 * Math.PI * 45; // r = 45
</script>

<div class="">
	{#if memStats}
		<!-- Circular Arc -->
		<div class="flex items-center text-xs gap-1">
			<div class="relative w-14 h-14 mx-auto">
				<svg class="w-full h-full transform -rotate-90" viewBox="0 0 100 100">
					<circle cx="50" cy="50" r="45" class="stroke-zinc-300 dark:stroke-zinc-700" stroke-width="10" fill="none" />
					<circle
						cx="50"
						cy="50"
						r="45"
						stroke-width="10"
						fill="none"
						stroke-linecap="round"
						stroke-dasharray={circumference}
						stroke-dashoffset={circumference - (circumference * memStats.usagePercent) / 100}
						class={`transition-all duration-700 ease-out ${getUsageColor(memStats.usagePercent)}`}
					/>
				</svg>
				<div class="absolute inset-0 flex items-center justify-center font-semibold text-xs">
					{memStats.usagePercent.toFixed(1)}%
				</div>
			</div>
			<div class="ml-1 text-[11px] leading-tight flex-1">
				<div class="flex justify-between items-center">
					<span>Total Ram: <span class="font-bold">{formatRAM(memStats.totalRAMMB)}</span></span>
				</div>
				<div>Used Ram: <span class="font-bold">{formatRAM(memStats.usedRAMMB)}</span></div>
				<div>Free Ram: <span class="font-bold">{formatRAM(memStats.freeRAMMB)}</span></div>
				{#if memStats.swapTotalMB && memStats.swapTotalMB > 0}
					<div class="text-[10px] text-gray-400 mt-0.5">
						Swap: {formatRAM(memStats.swapUsedMB || 0)} / {formatRAM(memStats.swapTotalMB)}
					</div>
				{/if}

				<!-- Live Velocity Indicator -->
				{#if velocitySample}
					<div class="mt-1 text-[9px] font-semibold flex items-center gap-1">
						{#if velocitySample.velocity_mbps > 50}
							<span class="text-amber-400 animate-pulse">⚡ Velocity: +{velocitySample.velocity_mbps} MB/s</span>
							{#if velocitySample.estimated_oom_sec > 0}
								<span class="text-red-400">(OOM: {velocitySample.estimated_oom_sec}s)</span>
							{/if}
						{:else if velocitySample.velocity_mbps < -50}
							<span class="text-green-400">⚡ Velocity: {velocitySample.velocity_mbps} MB/s</span>
						{:else}
							<span class="text-gray-500">⚡ Velocity: Stable</span>
						{/if}
					</div>
				{/if}
			</div>
		</div>
	{:else}
		<div class="text-gray-500 text-sm text-center">Loading RAM stats...</div>
	{/if}
</div>

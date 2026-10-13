<script lang="ts">
	import { usageBarColor } from '$lib';
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

	let swapTitle = $derived.by(() => {
		const stats = memStats as MemoryStats | null;
		return stats?.swapTotalMB && stats.swapTotalMB > 0
			? `Swap: ${formatRAM(stats.swapUsedMB || 0)} / ${formatRAM(stats.swapTotalMB)}`
			: undefined;
	});
</script>

<div class="flex-1 w-full flex flex-col justify-between" title={swapTitle}>
	{#if memStats}
		<div class="flex justify-between gap-2 text-muted font-medium">
			<span class="shrink-0">Memory</span>
			<!-- Live Velocity Indicator -->
			{#if velocitySample}
				{#if velocitySample.velocity_mbps > 50 && velocitySample.estimated_oom_sec > 0}
					<span
						class="text-red-500 animate-pulse truncate"
						title="Growing by {velocitySample.velocity_mbps} MB/s; memory runs out in about {velocitySample.estimated_oom_sec} seconds at this rate"
					>
						Full in {velocitySample.estimated_oom_sec}s
					</span>
				{:else if velocitySample.velocity_mbps > 50}
					<span class="text-amber-600 dark:text-amber-400 animate-pulse truncate">
						+{Math.round(velocitySample.velocity_mbps)} MB/s
					</span>
				{:else if velocitySample.velocity_mbps < -50}
					<span class="text-accent truncate">
						{Math.round(velocitySample.velocity_mbps)} MB/s
					</span>
				{:else}
					<span>Stable</span>
				{/if}
			{/if}
		</div>

		<div class="text-[28px] font-semibold tracking-tight leading-none tabular-nums">
			{memStats.usagePercent.toFixed(1)}<span class="text-sm text-muted">%</span>
		</div>

		<div class="flex flex-col gap-[5px]">
			<div class="h-[5px] rounded-full bg-track">
				<div
					class="h-[5px] rounded-full transition-all duration-700 ease-out {usageBarColor(
						memStats.usagePercent
					)}"
					style="width: {Math.min(100, memStats.usagePercent)}%"
				></div>
			</div>
			<div class="text-muted truncate">
				{formatRAM(memStats.usedRAMMB)} of {formatRAM(memStats.totalRAMMB)}
			</div>
		</div>
	{:else}
		<div class="text-muted m-auto">Loading RAM stats...</div>
	{/if}
</div>

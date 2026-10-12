<script lang="ts">
	import { RunThermalStressTest } from '$lib/wailsjs/go/main/App';

	type StressResult = {
		duration_sec: number;
		ops_completed: number;
		thermal_score: number;
		baseline_load_pct: number;
		peak_load_pct: number;
		status_text: string;
	};

	let isTesting = $state(false);
	let countdown = $state(10);
	let result = $state<StressResult | null>(null);
	let timerInterval: any = $state();

	async function startBenchmark() {
		if (isTesting) return;
		isTesting = true;
		countdown = 10;
		result = null;

		timerInterval = setInterval(() => {
			if (countdown > 1) {
				countdown -= 1;
			} else {
				clearInterval(timerInterval);
			}
		}, 1000);

		try {
			// @ts-ignore
			const res = await RunThermalStressTest();
			if (res) {
				result = res;
			}
		} catch (err) {
			console.error('Thermal stress test failed:', err);
		} finally {
			clearInterval(timerInterval);
			isTesting = false;
		}
	}
</script>

<div class="flex flex-col justify-between h-full space-y-2 p-1">
	<!-- Header Bar -->
	<div
		class="flex justify-between items-center pb-2 border-b border-line"
	>
		<div>
			<div class="font-bold text-base text-ink flex items-center gap-2">
				<span>10-Second Thermal & CPU Stress Benchmark</span>
			</div>
			<div class="text-xs text-muted">
				Lightweight multi-threaded CPU stress burn measuring thermal headroom & clock throttling
			</div>
		</div>

		<button
			class="px-4 py-1.5 text-xs bg-accent text-on-accent font-medium rounded-lg cursor-pointer hover:opacity-90 transition-transform active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed shadow-xs"
			onclick={startBenchmark}
			disabled={isTesting}
		>
			{isTesting ? `Testing (${countdown}s)...` : 'Run 10s Stress Test'}
		</button>
	</div>

	<!-- Main Burn Content -->
	<div class="flex-1 flex flex-col items-center justify-center py-2">
		{#if isTesting}
			<div class="flex flex-col items-center space-y-4">
				<div class="relative w-24 h-24 flex items-center justify-center">
					<div
						class="w-full h-full border-4 border-amber-500 border-t-transparent rounded-full animate-spin"
					></div>
					<div class="absolute font-bold text-2xl text-amber-700 dark:text-amber-400 font-mono">{countdown}s</div>
				</div>
				<div class="text-xs text-amber-700 dark:text-amber-400 font-semibold animate-pulse">
					Running multi-threaded floating-point CPU stress burn...
				</div>
			</div>
		{:else if result}
			<div
				class="w-full max-w-lg space-y-4 bg-ground p-4 rounded-xl border border-line"
			>
				<!-- Score Hero Card -->
				<div class="flex justify-between items-center">
					<div>
						<div class="text-xs text-muted uppercase font-semibold">Thermal Headroom Rating</div>
						<div class="text-3xl font-bold font-mono text-green-700 dark:text-green-400 mt-0.5">
							{result.thermal_score} <span class="text-sm font-sans text-muted">/ 100</span>
						</div>
					</div>
					<div
						class="px-3 py-1 rounded-full bg-green-500/20 text-green-700 dark:text-green-400 font-semibold border border-green-500/30 text-xs"
					>
						{result.status_text}
					</div>
				</div>

				<!-- Stats Grid -->
				<div
					class="grid grid-cols-3 gap-3 text-xs pt-2 border-t border-line"
				>
					<div>
						<div class="text-muted text-[10px]">Baseline Load</div>
						<div class="font-bold text-ink mt-0.5">
							{result.baseline_load_pct}%
						</div>
					</div>
					<div>
						<div class="text-muted text-[10px]">Peak Stress Load</div>
						<div class="font-bold text-amber-700 dark:text-amber-400 mt-0.5">{result.peak_load_pct}%</div>
					</div>
					<div>
						<div class="text-muted text-[10px]">Completed Ops</div>
						<div class="font-bold text-ink mt-0.5 font-mono">
							{(result.ops_completed / 1000000).toFixed(1)}M
						</div>
					</div>
				</div>
			</div>
		{:else}
			<div class="text-center space-y-2 max-w-md">
				<div class="w-11 h-11 mx-auto rounded-xl bg-accent text-on-accent flex items-center justify-center">
					<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M13 2L4 14h7l-1 8 9-12h-7l1-8z" /></svg>
				</div>
				<div class="font-bold text-sm text-ink">
					Ready for Thermal Stress Benchmark
				</div>
				<p class="text-xs text-muted leading-relaxed">
					Tests multi-core sustained floating-point throughput for 10 seconds. Evaluates whether
					your CPU suffers from fan lag or thermal throttling.
				</p>
			</div>
		{/if}
	</div>

	<!-- Footer Note -->
	<div
		class="text-[11px] text-muted pt-1.5 border-t border-line flex justify-between items-center"
	>
		<span>Safe, non-destructive 10-second micro-burn</span>
		<span class="text-[10px] text-muted">Auto-stops after 10 seconds</span>
	</div>
</div>



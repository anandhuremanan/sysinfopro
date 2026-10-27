<script lang="ts">
	import type { hardware } from '$lib/wailsjs/go/models';
	import ToolHeader from '$lib/components/ui/ToolHeader.svelte';
	import { RunThermalStressTest } from '$lib/wailsjs/go/main/App';

	// The backend also returns a "thermal score", a status line and a peak load read
	// after the test ends; none of them measure heat, so only the real numbers are shown.
	type StressResult = hardware.StressResult;
	type Saved = { perSec: number; at: number };

	const KEY = 'sysinfopro-stress-last';
	const ICON = 'M13 2L4 14h7l-1 8 9-12h-7l1-8z';

	let isTesting = $state(false);
	let countdown = $state(10);
	let result = $state<StressResult | null>(null);
	let previous = $state<Saved | null>(readSaved());

	function readSaved(): Saved | null {
		try {
			return JSON.parse(localStorage.getItem(KEY) ?? 'null');
		} catch {
			return null;
		}
	}

	let perSec = $derived(result ? result.ops_completed / Math.max(0.1, result.duration_sec) : 0);
	let change = $derived(previous && perSec ? (perSec / previous.perSec - 1) * 100 : null);

	async function start() {
		if (isTesting) return;
		isTesting = true;
		countdown = 10;
		previous = readSaved();
		result = null;
		const timer = setInterval(() => countdown > 1 && (countdown -= 1), 1000);
		try {
			const res = (await RunThermalStressTest()) as StressResult;
			if (res) {
				result = res;
				try {
					localStorage.setItem(
						KEY,
						JSON.stringify({
							perSec: res.ops_completed / Math.max(0.1, res.duration_sec),
							at: Date.now()
						})
					);
				} catch {
					/* not remembered; the next run just has nothing to compare with */
				}
			}
		} catch (err) {
			console.error('Stress test failed:', err);
		} finally {
			clearInterval(timer);
			isTesting = false;
		}
	}

	const millions = (n: number) => `${(n / 1e6).toFixed(1)} million`;
	const dateText = (ms: number) =>
		new Date(ms).toLocaleDateString(undefined, { day: 'numeric', month: 'short' });
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<ToolHeader
		title="Stress test"
		subtitle="Runs the processor flat out for 10 seconds to see if it overheats or slows down"
	>
		{#snippet actions()}
			<button class="btn btn-primary shrink-0" onclick={start} disabled={isTesting}>
				{isTesting ? `Running... ${countdown}s` : result ? 'Run again' : 'Start'}
			</button>
		{/snippet}
	</ToolHeader>

	<div class="flex-1 min-h-0 flex flex-col items-center justify-center px-4">
		{#if isTesting}
			<div class="flex flex-col items-center gap-3">
				<div class="relative size-20 grid place-items-center">
					<div
						class="absolute inset-0 border-4 border-track border-t-accent rounded-full motion-safe:animate-spin motion-reduce:animate-none"
					></div>
					<span class="font-bold text-xl text-ink tabular-nums">{countdown}s</span>
				</div>
				<p class="text-xs text-muted">Every core is busy. The PC may get loud for a moment.</p>
			</div>
		{:else if result}
			<div class="w-full max-w-lg flex flex-col gap-3">
				<div class="rounded-xl bg-tile border border-line px-4 py-3">
					<div class="text-[11px] text-muted">Calculations a second, on all cores</div>
					<div class="text-2xl font-bold text-ink tabular-nums mt-0.5">{millions(perSec)}</div>
					{#if change !== null && previous}
						<p class="text-xs mt-1 {change < -15 ? 'text-warn font-semibold' : 'text-muted'}">
							{#if Math.abs(change) < 5}
								About the same as last time ({dateText(previous.at)}).
							{:else if change < 0}
								{Math.round(-change)}% slower than last time ({dateText(previous.at)}).
								{#if change < -15}
									A big drop like this usually means the processor got hot and slowed itself down,
									or the PC was on battery or busy with something else.
								{/if}
							{:else}
								{Math.round(change)}% faster than last time ({dateText(previous.at)}).
							{/if}
						</p>
					{:else}
						<p class="text-xs text-muted mt-1">
							Run it again later, or when the PC feels slow, to compare with this result.
						</p>
					{/if}
				</div>
				<p class="text-[11px] text-muted px-1">
					The processor was {Math.round(result.baseline_load_pct)}% busy before the test. This
					measures how much work got done, not the temperature.
				</p>
			</div>
		{:else}
			<div class="text-center flex flex-col items-center gap-2 max-w-md">
				<div class="size-11 rounded-xl bg-accent/15 text-accent grid place-items-center">
					<svg
						width="22"
						height="22"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						aria-hidden="true"><path d={ICON} /></svg
					>
				</div>
				<p class="text-xs text-muted leading-relaxed">
					It is safe and stops by itself. Run it now and again later: if the result drops a lot, the
					processor is slowing itself down, usually because it is too hot.
				</p>
			</div>
		{/if}
	</div>
</div>

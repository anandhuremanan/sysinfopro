<script lang="ts">
	import ToolHeader from '$lib/components/ui/ToolHeader.svelte';
	import { onMount } from 'svelte';
	import { computeScore, deductions, scoreTone, scoreWord, type Score } from '$lib/healthScore';
	import { loadScoreInput } from '$lib/scoreData';
	import { getTool, type ToolId } from '$lib/tools';
	import StrokeIcon from '../ui/StrokeIcon.svelte';

	let { onOpen }: { onOpen: (view: string) => void } = $props();

	// every reading, in the order shown, with why it matters
	const PARTS = [
		{
			id: 'space',
			label: 'Free space',
			max: 25,
			icon: 'M12 3v9h9a9 9 0 1 1-9-9z',
			why: 'Windows needs room for updates and its page file. Below about 10% free, everything slows down.'
		},
		{
			id: 'memory',
			label: 'Memory',
			max: 20,
			icon: 'M4 7h16v10H4zM8 7V4M12 7V4M16 7V4M8 20v-3M12 20v-3M16 20v-3',
			why: 'When memory is nearly full, Windows moves things to the drive, which is much slower.'
		},
		{
			id: 'drive',
			label: 'Drive health',
			max: 20,
			icon: 'M4 6h16v12H4zM8 14h.01M12 14h4',
			why: "The drive's own report of wear and errors. A worn or failing drive can lose files."
		},
		{
			id: 'battery',
			label: 'Battery',
			max: 15,
			icon: 'M3 8h15a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V9a1 1 0 0 1 1-1zM22 11v2',
			why: 'How much charge the battery holds compared with when it was new.'
		},
		{
			id: 'security',
			label: 'Security',
			max: 20,
			icon: 'M12 3l7 3v5c0 4.5-3 8.5-7 10-4-1.5-7-5.5-7-10V6l7-3z',
			why: 'Antivirus, firewall, Windows updates, drive encryption and similar protections.'
		}
	] as const;

	let scored = $state<Score>({ score: -1, factors: [], missing: [] });
	let isLoading = $state(true);

	let rows = $derived(
		PARTS.map((p) => ({ ...p, factor: scored.factors.find((f) => f.id === p.id) }))
	);
	let lost = $derived(deductions(scored));

	const RING = 2 * Math.PI * 44;
	const tone = (ratio: number) =>
		ratio >= 0.8 ? 'bg-green-500' : ratio >= 0.5 ? 'bg-amber-500' : 'bg-red-500';

	async function refresh() {
		isLoading = true;
		try {
			scored = computeScore(await loadScoreInput());
		} finally {
			isLoading = false;
		}
	}

	onMount(() => {
		refresh();
	});
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<ToolHeader
		title="Health score"
		subtitle="Worked out only from what this PC reports right now. Nothing is guessed."
	>
		{#snippet actions()}
			<button class="btn btn-secondary shrink-0" disabled={isLoading} onclick={refresh}
				>Check again</button
			>
		{/snippet}
	</ToolHeader>

	<div class="flex-1 min-h-0 grid grid-cols-[180px_1fr] gap-4 pt-1">
		<!-- the score -->
		<div class="flex flex-col items-center gap-3 pt-2">
			<span class="relative w-[104px] h-[104px] {scoreTone(scored.score)}">
				<svg
					width="104"
					height="104"
					viewBox="0 0 104 104"
					fill="none"
					class="-rotate-90"
					aria-hidden="true"
				>
					<circle cx="52" cy="52" r="44" stroke-width="9" class="stroke-track" />
					<circle
						cx="52"
						cy="52"
						r="44"
						stroke="currentColor"
						stroke-width="9"
						stroke-linecap="round"
						stroke-dasharray="{(RING * Math.max(0, scored.score)) / 100} {RING}"
						class="transition-all duration-700 ease-out"
					/>
				</svg>
				<span class="absolute inset-0 flex flex-col items-center justify-center text-ink">
					<span class="text-[30px] font-semibold tracking-tight leading-none">
						{scored.score < 0 ? '–' : scored.score}
					</span>
					<span class="text-[11px] text-muted">of 100</span>
				</span>
			</span>
			<span class="font-semibold {scoreTone(scored.score)}">
				{scored.score < 0 ? 'Measuring...' : scoreWord(scored.score)}
			</span>
			<p class="text-[11px] text-muted text-center leading-relaxed">
				{#if scored.score < 0}
					&nbsp;
				{:else if lost.length}
					{PARTS.find((p) => p.id === lost[0].factor.id)?.label} ({lost[0].factor.detail.replace(
						/^[^:]+:\s*/,
						''
					)}) costs the most: {lost[0].lost} points.
				{:else}
					Every reading earns full points.
				{/if}
			</p>
			<p class="text-[11px] text-muted text-center leading-relaxed mt-auto pb-1">
				80 and up is good, 60 to 79 fair, below 60 poor. Processor load is not counted: it changes
				every second.
			</p>
		</div>

		<!-- each reading -->
		<div class="min-h-0 overflow-y-auto overscroll-contain pr-1.5 pb-2 flex flex-col gap-1.5">
			{#each rows as r (r.id)}
				{@const f = r.factor}
				{@const tool = f ? getTool(f.view as ToolId) : null}
				<div class="rounded-xl bg-ground px-3 py-2 flex gap-3">
					<span
						class="mt-0.5 w-7 h-7 rounded-lg grid place-items-center bg-accent/15 text-accent shrink-0"
					>
						<StrokeIcon d={r.icon} size={15} />
					</span>
					<div class="flex-1 min-w-0">
						<div class="flex items-baseline gap-2">
							<span class="font-semibold text-ink">{r.label}</span>
							<span class="text-xs text-muted truncate">
								{f
									? f.detail.replace(/^[^:]+:\s*/, '')
									: isLoading
										? 'Checking...'
										: 'Not available on this PC'}
							</span>
							<span class="ml-auto font-mono text-xs text-ink shrink-0">
								{f ? Math.round(f.points) : '–'}<span class="text-muted">&nbsp;/&nbsp;{r.max}</span>
							</span>
						</div>
						<div class="h-1.5 mt-1.5 rounded-full bg-track overflow-hidden">
							{#if f}
								<div
									class="h-full rounded-full transition-[width] duration-700 {tone(
										f.points / f.max
									)}"
									style="width: {(f.points / f.max) * 100}%"
								></div>
							{/if}
						</div>
						<div class="flex items-start justify-between gap-3 mt-1">
							<span class="text-[11px] text-muted leading-snug">{r.why}</span>
							{#if tool}
								<button
									class="text-[11px] font-semibold text-accent cursor-pointer hover:underline shrink-0"
									onclick={() => onOpen(tool.id)}
								>
									Open {tool.name} →
								</button>
							{/if}
						</div>
					</div>
				</div>
			{/each}
		</div>
	</div>
</div>

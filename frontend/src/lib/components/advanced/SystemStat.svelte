<script lang="ts">
	import { GetDiagnosticReport } from '$lib/wailsjs/go/main/App';
	import Card from '../ui/Card.svelte';
	import Modal from '../ui/Modal.svelte';

	let { overallScore } = $props();

	type Recommendation = {
		id: string;
		severity: string;
		title: string;
		description: string;
		action_label?: string;
	};

	type DiagnosticReport = {
		overall_score: number;
		status_text: string;
		recommendations: Recommendation[];
	};

	let report = $state<DiagnosticReport | null>(null);
	let showRecommendationsModal = $state(false);

	async function loadReport() {
		try {
			// @ts-ignore
			const res = await GetDiagnosticReport();
			if (res) {
				report = res;
			}
		} catch (err) {
			console.error('Failed to get diagnostic report:', err);
		}
	}

	$effect(() => {
		loadReport();
	});

	let displayScore = $derived(report?.overall_score || overallScore);
	let scoreColor = $derived(
		displayScore >= 80 ? 'text-green-700 dark:text-green-400' : displayScore >= 60 ? 'text-amber-700 dark:text-amber-400' : 'text-red-500'
	);
</script>

<Card isLoading={false}>
	<div class="h-full flex flex-col justify-between">
		<!-- Header Row with unified baseline height -->
		<div class="h-6 flex justify-between items-center w-full">
			<div class="flex items-center gap-2">
				<span class="font-bold text-xs text-ink">System Health Score</span>
				{#if report?.status_text}
					<span class="text-[10px] px-1.5 py-0.5 rounded bg-tile border border-line font-semibold">
						{report.status_text}
					</span>
				{/if}
			</div>
			{#if report?.recommendations && report.recommendations.length > 0}
				<button
					class="text-[10px] bg-accent text-on-accent font-medium px-2 py-0.5 rounded hover:opacity-90 transition-colors cursor-pointer shrink-0"
					onclick={() => (showRecommendationsModal = true)}
				>
					Actions ({report.recommendations.length})
				</button>
			{/if}
		</div>

		<!-- Middle Body -->
		<div class="flex justify-between items-center gap-2 my-auto">
			<div>
				<div class="text-3xl font-bold {scoreColor}">
					{#if Number.isFinite(displayScore)}
						{displayScore}
					{:else}
						<span class="text-xl">Not Available</span>
					{/if}
				</div>
			</div>
			<img src="/pc.png" alt="system health" class="w-14 h-14 shrink-0 object-contain" />
		</div>

		<!-- Footer text -->
		<p class="text-[11px] text-muted">
			Evaluates CPU load, memory pressure, storage, battery wear & network status.
		</p>
	</div>
</Card>

<!-- Diagnostic Recommendations Modal -->
<Modal
	bind:showModal={showRecommendationsModal}
	width="max-w-lg"
	title="Diagnostic Recommendations"
	description="Actionable steps to optimize performance & health"
>
	<div class="flex flex-col gap-3">
		{#if report?.status_text}
			<div class="flex items-center gap-2">
				<span class="text-muted">Health score</span>
				<span class="font-bold {scoreColor}">{displayScore}</span>
				<span class="px-1.5 py-0.5 rounded bg-ground border border-line font-semibold">
					{report.status_text}
				</span>
			</div>
		{/if}

		<!-- Recommendations List -->
		{#if report?.recommendations && report.recommendations.length > 0}
			{#each report.recommendations as rec}
				<div class="p-3 rounded-xl bg-ground border border-line flex flex-col gap-1.5">
					<div class="flex justify-between items-center gap-2">
						<div class="flex items-center gap-2 font-bold text-ink min-w-0">
							{#if rec.severity === 'critical'}
								<span class="w-2 h-2 rounded-full bg-red-500 shrink-0"></span>
							{:else if rec.severity === 'warning'}
								<span class="w-2 h-2 rounded-full bg-amber-500 shrink-0"></span>
							{:else}
								<span class="w-2 h-2 rounded-full bg-accent shrink-0"></span>
							{/if}
							<span class="truncate">{rec.title}</span>
						</div>
						<span
							class="text-[10px] uppercase font-bold px-2 py-0.5 rounded-md shrink-0 border {rec.severity ===
							'critical'
								? 'bg-red-500/15 text-red-600 dark:text-red-400 border-red-500/30'
								: rec.severity === 'warning'
									? 'bg-amber-500/20 text-amber-700 dark:text-amber-400 border-amber-500/30'
									: 'bg-tile text-muted border-line'}"
						>
							{rec.severity}
						</span>
					</div>
					<p class="text-muted leading-relaxed pl-4">{rec.description}</p>
				</div>
			{/each}
		{:else}
			<div class="py-6 text-center text-muted">
				No diagnostic issues detected. System is running optimally!
			</div>
		{/if}
	</div>

	{#snippet actions()}
		<button class="btn btn-primary" onclick={() => (showRecommendationsModal = false)}>Done</button>
	{/snippet}
</Modal>

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
		displayScore >= 80 ? 'text-green-500' : displayScore >= 60 ? 'text-amber-500' : 'text-red-500'
	);
</script>

<Card isLoading={false}>
	<div class="h-full flex flex-col justify-between">
		<!-- Header Row with unified baseline height -->
		<div class="h-6 flex justify-between items-center w-full">
			<div class="flex items-center gap-2">
				<span class="font-bold text-xs text-slate-800 dark:text-slate-100">System Health Score</span>
				{#if report?.status_text}
					<span class="text-[10px] px-1.5 py-0.5 rounded bg-slate-300 dark:bg-slate-700 font-semibold">
						{report.status_text}
					</span>
				{/if}
			</div>
			{#if report?.recommendations && report.recommendations.length > 0}
				<button
					class="text-[10px] bg-blue-500/90 text-white font-medium px-2 py-0.5 rounded hover:bg-blue-600 transition-colors cursor-pointer shrink-0"
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
		<p class="text-[11px] text-gray-400">
			Evaluates CPU load, memory pressure, storage, battery wear & network status.
		</p>
	</div>
</Card>

<!-- Diagnostic Recommendations Modal -->
<Modal bind:showModal={showRecommendationsModal} showClose={true}>
	<div class="flex flex-col space-y-4 max-h-[70vh]">
		<!-- Modal Header -->
		<div class="flex justify-between items-center pb-3 border-b border-slate-300 dark:border-slate-700/80 pr-6">
			<div>
				<div class="font-bold text-sm text-slate-800 dark:text-slate-100">
					Diagnostic Recommendations
				</div>
				<div class="text-xs text-gray-400 mt-0.5">
					Actionable steps to optimize performance & health
				</div>
			</div>
			{#if report?.status_text}
				<div class="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-slate-300 dark:bg-slate-700/80 text-xs font-semibold shrink-0">
					<span class="text-gray-400">Score:</span>
					<span class={scoreColor}>{displayScore}</span>
				</div>
			{/if}
		</div>

		<!-- Recommendations List -->
		{#if report?.recommendations && report.recommendations.length > 0}
			<div class="space-y-3 overflow-y-auto max-h-[45vh] pr-1">
				{#each report.recommendations as rec}
					<div class="p-3.5 rounded-xl bg-slate-300/40 dark:bg-slate-800/80 border border-slate-300 dark:border-slate-700 flex flex-col space-y-2">
						<div class="flex justify-between items-center gap-2">
							<div class="flex items-center gap-2 font-bold text-xs text-slate-800 dark:text-slate-100">
								{#if rec.severity === 'critical'}
									<span class="text-red-500">🚨</span>
								{:else if rec.severity === 'warning'}
									<span class="text-amber-500">⚠️</span>
								{:else}
									<span class="text-blue-500">ℹ️</span>
								{/if}
								<span>{rec.title}</span>
							</div>
							<span class="text-[10px] uppercase font-bold px-2 py-0.5 rounded-md bg-amber-500/20 text-amber-400 border border-amber-500/30 shrink-0">
								{rec.severity}
							</span>
						</div>
						<p class="text-xs text-gray-400 leading-relaxed pl-6">{rec.description}</p>
					</div>
				{/each}
			</div>
		{:else}
			<div class="py-6 text-center text-xs text-gray-400">
				No diagnostic issues detected. System is running optimally!
			</div>
		{/if}

		<!-- Modal Footer -->
		<div class="pt-3 border-t border-slate-300 dark:border-slate-700/80 flex justify-end">
			<button
				class="px-4 py-1.5 text-xs bg-slate-300 dark:bg-slate-700 hover:bg-slate-400 dark:hover:bg-slate-600 text-slate-800 dark:text-slate-100 font-medium rounded-lg transition-colors cursor-pointer"
				onclick={() => (showRecommendationsModal = false)}
			>
				Done
			</button>
		</div>
	</div>
</Modal>

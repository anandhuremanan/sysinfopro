<script lang="ts">
	import type { history } from '$lib/wailsjs/go/models';
	import ToolHeader from '$lib/components/ui/ToolHeader.svelte';
	import { onMount } from 'svelte';
	import BusyArt from '../ui/BusyArt.svelte';
	import { GetChanges, SaveChangesBaseline } from '$lib/wailsjs/go/main/App';

	type CategoryDiff = history.CategoryDiff;

	type Changes = history.Changes;

	let changes = $state<Changes | undefined>(undefined);
	let isLoading = $state(false);
	let isSaving = $state(false);
	let loadError = $state<string | undefined>(undefined);

	let sinceLabel = $derived(
		changes
			? new Date(changes.since).toLocaleString(undefined, {
					dateStyle: 'medium',
					timeStyle: 'short'
				})
			: ''
	);
	let changedCategories = $derived(
		(changes?.categories ?? []).filter(
			(c) => c.added.length + c.removed.length + c.changed.length > 0
		)
	);

	async function load() {
		isLoading = true;
		loadError = undefined;
		try {
			changes = (await GetChanges()) as Changes;
		} catch (err) {
			console.error('Failed to compare snapshots:', err);
			loadError = `${err}`;
		} finally {
			isLoading = false;
		}
	}

	async function markReviewed() {
		isSaving = true;
		try {
			await SaveChangesBaseline();
			await load();
		} catch (err) {
			loadError = `${err}`;
		} finally {
			isSaving = false;
		}
	}

	onMount(() => {
		load();
	});
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<!-- Header Bar -->
	<ToolHeader title="What changed">
		{#snippet sub()}
			{#if changes?.has_baseline}
				Compared with {sinceLabel}
			{:else}
				Startup programs, installed software, listening ports and PATH, compared over time
			{/if}
		{/snippet}
		{#snippet actions()}
			<div class="flex items-center gap-2 shrink-0">
				<button class="btn btn-secondary" onclick={load} disabled={isLoading}>
					{isLoading ? 'Comparing...' : 'Compare again'}
				</button>
				{#if changes?.has_baseline && changes.total > 0}
					<button
						class="btn btn-primary"
						onclick={markReviewed}
						disabled={isSaving}
						title="Start comparing from now"
					>
						Mark as reviewed
					</button>
				{/if}
			</div>
		{/snippet}
	</ToolHeader>

	{#if isLoading && !changes}
		<div class="flex-1 flex items-center justify-center">
			<BusyArt kind="check" size={64} label="Reading the current state of this PC..." />
		</div>
	{:else if loadError}
		<div class="flex-1 flex flex-col items-center justify-center gap-2 text-center">
			<div class="text-xs text-red-500 max-w-md">{loadError}</div>
			<button class="btn btn-secondary" onclick={load}>Try again</button>
		</div>
	{:else if changes && !changes.has_baseline}
		<div class="flex-1 flex flex-col items-center justify-center gap-2 text-center">
			<div class="text-sm font-bold text-ink">First snapshot saved</div>
			<p class="text-xs text-muted max-w-md leading-relaxed">
				This is the starting point. Open this screen again after installing, updating or changing
				something, and it will list what is different from {sinceLabel}.
			</p>
		</div>
	{:else if changes && changes.total === 0}
		<div class="flex-1 flex flex-col items-center justify-center gap-1 text-center">
			<div class="text-sm font-bold text-ink">Nothing has changed</div>
			<p class="text-xs text-muted">Same as on {sinceLabel}.</p>
		</div>
	{:else if changes}
		<div
			class="flex-1 min-h-0 overflow-y-auto overscroll-contain pb-5 [mask-image:linear-gradient(to_bottom,#000_calc(100%-20px),transparent)] pr-1.5 flex flex-col gap-2.5"
		>
			{#each changedCategories as category (category.id)}
				<section class="bg-ground border border-line rounded-xl px-3 py-2.5">
					<h2
						class="flex items-center justify-between text-[10px] uppercase tracking-wider font-semibold text-muted pb-1"
					>
						<span>{category.title}</span>
						<span class="normal-case tracking-normal">
							{category.added.length + category.removed.length + category.changed.length}
							{category.added.length + category.removed.length + category.changed.length === 1
								? 'change'
								: 'changes'}
						</span>
					</h2>
					{#each category.added as item}
						<div class="flex gap-2 py-0.5">
							<span class="shrink-0 w-14 font-semibold text-ok">Added</span>
							<span class="min-w-0 break-words text-ink select-text">{item}</span>
						</div>
					{/each}
					{#each category.changed as item}
						<div class="flex gap-2 py-0.5">
							<span class="shrink-0 w-14 font-semibold text-ink">Changed</span>
							<span class="min-w-0 break-words text-ink select-text">{item}</span>
						</div>
					{/each}
					{#each category.removed as item}
						<div class="flex gap-2 py-0.5">
							<span class="shrink-0 w-14 font-semibold text-fail">Removed</span>
							<span class="min-w-0 break-words text-ink select-text">{item}</span>
						</div>
					{/each}
				</section>
			{/each}
		</div>

		<div class="text-[11px] text-muted pt-1 shrink-0">
			{changes.total}
			{changes.total === 1 ? 'change' : 'changes'} since {sinceLabel}. Ports come and go as programs
			start and stop.
		</div>
	{/if}
</div>

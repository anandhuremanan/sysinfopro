<script lang="ts">
	import BusyArt from '../ui/BusyArt.svelte';
	import {
		GetEnvPathAnalysis,
		RemoveUserPathEntries,
		UndoPathChange
	} from '$lib/wailsjs/go/main/App';
	import Modal from '../ui/Modal.svelte';

	type PathEntry = {
		path: string;
		raw: string;
		scope: 'system' | 'user' | 'process';
		index: number;
		is_valid: boolean;
		is_duplicate: boolean;
		status_text: string;
		fixable: boolean;
	};
	type Analysis = {
		total_entries: number;
		valid_entries: number;
		invalid_entries: number;
		duplicates_count: number;
		path_entries: PathEntry[];
		key_variables: { key: string; value: string }[];
		user_path_raw: string;
		can_undo: boolean;
		editable: boolean;
	};

	const SCOPES = [
		{ id: 'user', label: 'Your PATH', note: 'Yours to change' },
		{ id: 'system', label: 'System PATH', note: 'Needs administrator rights to change' },
		{ id: 'process', label: 'PATH', note: '' }
	] as const;

	let analysis = $state<Analysis | null>(null);
	// a missing folder or a repeat; an entry using a variable nobody defines cannot
	// be checked, so it is not counted as a problem (and is never offered for removal)
	const isProblem = (e: { is_valid: boolean; is_duplicate: boolean; status_text: string }) =>
		e.is_duplicate || (!e.is_valid && e.status_text !== 'Uses an undefined variable');
	// a list with problems comes first, so they are on screen without scrolling
	let scopes = $derived(
		[...SCOPES].sort((a, b) => {
			const bad = (id: string) =>
				analysis?.path_entries.some((e) => e.scope === id && isProblem(e)) ? 0 : 1;
			return bad(a.id) - bad(b.id);
		})
	);
	let isLoading = $state(false);
	let isSaving = $state(false);
	let showConfirm = $state(false);
	let statusMessage = $state<string | undefined>(undefined);
	let selected = $state<Record<number, boolean>>({});

	let fixable = $derived(analysis?.path_entries.filter((e) => e.fixable) ?? []);
	let chosen = $derived(fixable.filter((e) => selected[e.index]));
	let problems = $derived((analysis?.invalid_entries ?? 0) + (analysis?.duplicates_count ?? 0));

	function showStatus(message: string) {
		statusMessage = message;
		setTimeout(() => (statusMessage = undefined), 6000);
	}

	// every fixable entry starts ticked, so one click fixes everything that can be fixed
	function useAnalysis(next: Analysis) {
		analysis = next;
		selected = Object.fromEntries(
			next.path_entries.filter((e) => e.fixable).map((e) => [e.index, true])
		);
	}

	async function load() {
		isLoading = true;
		try {
			useAnalysis((await GetEnvPathAnalysis()) as Analysis);
		} catch (err) {
			showStatus(`Failed to read PATH: ${err}`);
		} finally {
			isLoading = false;
		}
	}

	async function removeChosen() {
		if (!analysis) return;
		isSaving = true;
		const count = chosen.length;
		try {
			const next = (await RemoveUserPathEntries(
				analysis.user_path_raw,
				chosen.map((e) => e.index)
			)) as Analysis;
			useAnalysis(next);
			showStatus(
				`Removed ${count} ${count === 1 ? 'entry' : 'entries'}. Terminals opened from now on use the new PATH.`
			);
		} catch (err) {
			showStatus(`${err}`);
			await load();
		} finally {
			isSaving = false;
			showConfirm = false;
		}
	}

	async function undo() {
		isSaving = true;
		try {
			useAnalysis((await UndoPathChange()) as Analysis);
			showStatus('Your PATH is back as it was.');
		} catch (err) {
			showStatus(`${err}`);
		} finally {
			isSaving = false;
		}
	}

	$effect(() => {
		load();
	});
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<div class="flex justify-between items-center gap-3 pb-2 border-b border-line shrink-0">
		<div class="min-w-0">
			<div class="font-bold text-base text-ink">PATH</div>
			<div class="text-xs text-muted truncate">
				Folders Windows searches for programs, checked for missing folders and repeats
			</div>
		</div>
		<div class="flex items-center gap-2 shrink-0">
			{#if analysis?.can_undo}
				<button class="btn btn-secondary" disabled={isSaving} onclick={undo}>Undo last fix</button>
			{/if}
			{#if fixable.length > 0}
				<!-- removal can be undone, so it is the main action, not a red one -->
				<button
					class="btn btn-primary"
					disabled={chosen.length === 0 || isSaving}
					onclick={() => (showConfirm = true)}
				>
					Remove {chosen.length}
					{chosen.length === 1 ? 'entry' : 'entries'}
				</button>
			{/if}
			<button class="btn btn-secondary" disabled={isLoading || isSaving} onclick={load}
				>Refresh</button
			>
		</div>
	</div>

	{#if statusMessage}
		<div
			class="text-xs font-semibold text-green-700 dark:text-green-400 px-1 shrink-0 truncate"
			title={statusMessage}
		>
			{statusMessage}
		</div>
	{/if}

	{#if isLoading && !analysis}
		<div class="flex-1 flex items-center justify-center">
			<BusyArt kind="read" size={64} label="Reading PATH..." />
		</div>
	{:else if analysis}
		<div class="text-muted shrink-0">
			{analysis.total_entries} entries ·
			{#if problems === 0}
				<span class="text-green-700 dark:text-green-400 font-semibold">all good</span>
			{:else}
				<span class="text-amber-700 dark:text-amber-400 font-semibold">
					{analysis.invalid_entries} missing, {analysis.duplicates_count} repeated
				</span>
				{#if fixable.length < problems && analysis.editable}
					· {problems - fixable.length} in the system PATH
					{problems - fixable.length === 1 ? 'needs' : 'need'} administrator rights
				{/if}
			{/if}
		</div>

		<div
			class="flex-1 min-h-0 overflow-y-auto overscroll-contain pb-5 [mask-image:linear-gradient(to_bottom,#000_calc(100%-20px),transparent)] pr-1.5"
		>
			{#each scopes as scope (scope.id)}
				<!-- missing and repeated folders first, so the problems are on screen -->
				{@const entries = analysis.path_entries
					.filter((e) => e.scope === scope.id)
					.sort((a, b) => Number(!isProblem(a)) - Number(!isProblem(b)))}
				{#if entries.length > 0}
					<div
						class="flex items-baseline justify-between gap-2 pt-2 pb-1 sticky top-0 bg-tile z-10"
					>
						<span class="text-[10px] uppercase tracking-wider font-semibold text-muted"
							>{scope.label}</span
						>
						<span class="text-[10px] text-muted">{scope.note}</span>
					</div>
					{#each entries as entry (entry.scope + entry.index)}
						<label
							class="flex items-center gap-2.5 h-8 border-b border-line {entry.fixable
								? 'cursor-pointer'
								: ''}"
						>
							{#if entry.fixable}
								<input
									type="checkbox"
									class="rounded border-line bg-tile checked:bg-accent text-accent focus:ring-accent"
									disabled={isSaving}
									bind:checked={selected[entry.index]}
								/>
							{:else if isProblem(entry)}
								<!-- a problem that needs administrator rights to fix -->
								<span
									class="size-4 shrink-0 grid place-items-center text-amber-600 dark:text-amber-400 font-bold"
									title="Needs administrator rights to remove"
									aria-hidden="true">!</span
								>
							{:else}
								<span class="size-4 shrink-0" aria-hidden="true"></span>
							{/if}
							<span
								class="flex-1 min-w-0 font-mono text-[11px] truncate {!isProblem(entry)
									? 'text-ink'
									: 'text-muted line-through decoration-muted/50'}"
								title={entry.raw === entry.path ? entry.path : `${entry.raw}\n${entry.path}`}
							>
								{entry.raw}
							</span>
							{#if entry.is_duplicate}
								<span
									class="shrink-0 text-[10px] px-1.5 py-0.5 rounded-md bg-amber-500/15 text-amber-700 dark:text-amber-400 font-semibold"
								>
									Repeated
								</span>
							{:else if !entry.is_valid}
								<span
									class="shrink-0 text-[10px] px-1.5 py-0.5 rounded-md bg-red-500/15 text-red-700 dark:text-red-400 font-semibold"
								>
									Missing folder
								</span>
							{/if}
						</label>
					{/each}
				{/if}
			{/each}
		</div>
	{/if}
</div>

<Modal
	bind:showModal={showConfirm}
	width="max-w-md"
	showClose={!isSaving}
	title="Remove {chosen.length} {chosen.length === 1 ? 'entry' : 'entries'} from your PATH?"
	description="Only your own PATH changes. You can undo this afterwards."
>
	<div class="flex flex-col gap-2">
		<div class="max-h-32 overflow-y-auto rounded-lg bg-ground border border-line px-2.5 py-1.5">
			{#each chosen as entry (entry.index)}
				<div class="flex justify-between gap-3 py-0.5">
					<span class="font-mono text-[11px] truncate" title={entry.raw}>{entry.raw}</span>
					<span class="shrink-0 text-muted">{entry.is_duplicate ? 'repeated' : 'missing'}</span>
				</div>
			{/each}
		</div>
		<p class="text-muted leading-relaxed">
			Programs already running keep the PATH they started with. Open a new terminal to see the
			change.
		</p>
	</div>
	{#snippet actions()}
		<button class="btn btn-secondary" disabled={isSaving} onclick={() => (showConfirm = false)}
			>Cancel</button
		>
		<button class="btn btn-danger" disabled={isSaving} onclick={removeChosen}>
			{isSaving ? 'Removing...' : 'Remove'}
		</button>
	{/snippet}
</Modal>

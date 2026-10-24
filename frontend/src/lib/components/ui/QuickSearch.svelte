<script lang="ts">
	import { matchTool, tools } from '$lib/tools';
	import { tick } from 'svelte';
	import StrokeIcon from './StrokeIcon.svelte';

	// Ctrl+K: type part of a tool's name, or a word like "node_modules", and press Enter
	let {
		open = $bindable(false),
		onOpen,
		onAction
	}: {
		open?: boolean;
		onOpen: (view: string) => void;
		// app actions that are not tools
		onAction: (id: 'settings' | 'widget' | 'processes' | 'home') => void;
	} = $props();

	type Entry = { id: string; name: string; hint: string; icon: string; action?: boolean };

	const ACTIONS: (Entry & { keywords: string })[] = [
		{ id: 'processes', name: 'Processes', hint: 'Running programs', icon: 'M4 6h16M4 12h16M4 18h10', keywords: 'task manager memory cpu leak', action: true },
		{ id: 'widget', name: 'Open widget', hint: 'Small window on top', icon: 'M3 5h18v14H3zM12 12h7v5h-7z', keywords: 'floating mini always on top', action: true },
		{ id: 'settings', name: 'Settings', hint: 'Theme, widget, alerts', icon: 'M4 6h9M17 6h3M4 12h3M11 12h9M4 18h11M19 18h1', keywords: 'preferences theme dark light alerts shortcut', action: true },
		{ id: 'home', name: 'Home', hint: 'Overview', icon: 'M4 11l8-7 8 7v9H4z', keywords: 'dashboard start', action: true }
	];

	let query = $state('');
	let active = $state(0);
	let input = $state<HTMLInputElement | undefined>(undefined);

	let results = $derived.by(() => {
		const scored: { entry: Entry; score: number }[] = [];
		for (const tool of tools) {
			const score = matchTool(tool, query);
			if (score > 0) scored.push({ entry: tool, score });
		}
		for (const action of ACTIONS) {
			const score = matchTool({ ...action, group: '' } as any, query);
			if (score > 0) scored.push({ entry: action, score: score - 0.5 });
		}
		// best match first; with nothing typed, keep the usual order
		return query.trim() ? scored.sort((a, b) => b.score - a.score).map((s) => s.entry) : scored.map((s) => s.entry);
	});

	$effect(() => {
		if (open) {
			query = '';
			active = 0;
			tick().then(() => input?.focus());
		}
	});

	$effect(() => {
		// a new search starts at the top
		query;
		active = 0;
	});

	function choose(entry: Entry | undefined) {
		if (!entry) return;
		open = false;
		if (entry.action) onAction(entry.id as 'settings' | 'widget' | 'processes' | 'home');
		else onOpen(entry.id);
	}

	function onKeydown(event: KeyboardEvent) {
		if (event.key === 'ArrowDown') {
			event.preventDefault();
			active = Math.min(results.length - 1, active + 1);
		} else if (event.key === 'ArrowUp') {
			event.preventDefault();
			active = Math.max(0, active - 1);
		} else if (event.key === 'Enter') {
			event.preventDefault();
			choose(results[active]);
		} else if (event.key === 'Escape') {
			event.preventDefault();
			open = false;
		}
	}

	$effect(() => {
		// keep the highlighted row in view
		document.getElementById(`quick-${active}`)?.scrollIntoView({ block: 'nearest' });
	});
</script>

{#if open}
	<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
	<div
		class="fixed z-50 inset-0 flex justify-center items-start pt-14 px-4 bg-black/40 backdrop-blur-xs"
		onclick={(event) => {
			if (event.target === event.currentTarget) open = false;
		}}
	>
		<div
			role="dialog"
			aria-modal="true"
			aria-label="Quick search"
			class="w-full max-w-md flex flex-col bg-tile text-ink text-xs rounded-2xl border border-line shadow-2xl overflow-hidden"
		>
			<div class="flex items-center gap-2.5 px-4 h-12 border-b border-line">
				<span class="text-muted"><StrokeIcon d="M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14zM20 20l-4-4" size={15} /></span>
				<input
					bind:this={input}
					bind:value={query}
					onkeydown={onKeydown}
					type="text"
					role="combobox"
					aria-expanded="true"
					aria-controls="quick-results"
					aria-activedescendant={results.length ? `quick-${active}` : undefined}
					placeholder="Go to a tool, or search for node_modules, ports, temp..."
					class="flex-1 min-w-0 h-full bg-transparent border-0 p-0 text-sm text-ink placeholder:text-muted focus:outline-none focus:ring-0"
				/>
				<kbd class="shrink-0 text-[10px] text-muted border border-line rounded px-1.5 py-0.5">Esc</kbd>
			</div>
			<div id="quick-results" role="listbox" class="max-h-72 overflow-y-auto p-1.5">
				{#each results as entry, i (entry.id)}
					<div
						id="quick-{i}"
						role="option"
						aria-selected={i === active}
						tabindex="-1"
						class="flex items-center gap-3 h-10 px-2.5 rounded-lg cursor-pointer {i === active
							? 'bg-ground'
							: ''}"
						onmouseenter={() => (active = i)}
						onclick={() => choose(entry)}
					>
						<span
							class="shrink-0 w-7 h-7 rounded-lg flex items-center justify-center {entry.action
								? 'bg-line text-muted'
								: 'bg-accent text-on-accent'}"
						>
							<StrokeIcon d={entry.icon} size={14} />
						</span>
						<span class="flex-1 min-w-0 font-semibold truncate">{entry.name}</span>
						<span class="shrink-0 text-muted truncate">{entry.hint}</span>
					</div>
				{:else}
					<div class="h-16 flex items-center justify-center text-muted">Nothing matches "{query}"</div>
				{/each}
			</div>
		</div>
	</div>
{/if}

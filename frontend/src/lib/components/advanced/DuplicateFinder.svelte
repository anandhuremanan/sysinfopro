<script lang="ts">
	import { onMount } from 'svelte';
	import { formatBytes } from '$lib';
	import {
		GetDrives,
		GetDuplicateFolders,
		PickFolder,
		RemoveDuplicates,
		RevealInExplorer
	} from '$lib/wailsjs/go/main/App';
	import {
		cancelDupeSearch,
		dropRemoved,
		dupeSearch,
		searchDupes,
		type DupeGroup
	} from '$lib/dupeSearch.svelte';
	import BusyArt from '../ui/BusyArt.svelte';
	import Modal from '../ui/Modal.svelte';
	import Segmented from '../ui/Segmented.svelte';
	import StrokeIcon from '../ui/StrokeIcon.svelte';

	type Drive = { path: string; total_bytes: number; free_bytes: number; used_pct: number };
	type Place = { label: string; path: string };

	const REVEAL_ICON = 'M14 4h6v6M20 4l-9 9M18 14v4a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4';
	const MB = 1024 * 1024;
	const SIZES = [
		{ id: 100 * 1024, label: '100 KB' },
		{ id: MB, label: '1 MB' },
		{ id: 10 * MB, label: '10 MB' },
		{ id: 100 * MB, label: '100 MB' }
	];
	const PAGE = 100;
	const ROWS = 4; // copies shown per set before "more"

	let drives = $state<Drive[]>([]);
	let places = $state<Place[]>([]);
	let minSize = $state(MB);
	let keep = $state<'oldest' | 'newest'>('oldest');
	let showRisky = $state(false);
	let selected = $state(new Set<string>());
	let shown = $state(PAGE);
	let opened = $state(new Set<string>());
	let confirmOpen = $state(false);
	let isRemoving = $state(false);
	let status = $state<{ text: string; ok: boolean } | null>(null);

	let result = $derived(dupeSearch.result);
	let riskyCount = $derived(result?.groups.filter((g) => g.risky).length ?? 0);
	let groups = $derived((result?.groups ?? []).filter((g) => showRisky || !g.risky));
	let visibleWasted = $derived(groups.reduce((sum, g) => sum + g.wasted, 0));
	let sizeOf = $derived(
		new Map((result?.groups ?? []).flatMap((g) => g.files.map((f) => [f.path, g.size] as const)))
	);
	let selectedBytes = $derived([...selected].reduce((sum, p) => sum + (sizeOf.get(p) ?? 0), 0));
	let progress = $derived(dupeSearch.progress);

	const folderName = (path: string) =>
		path
			.replace(/[\\/]$/, '')
			.split(/[\\/]/)
			.pop() || path;
	const folderOf = (path: string) => path.slice(0, path.lastIndexOf('\\')) || path;
	// where a copy is, inside the folder searched: the part that tells copies apart
	function placeOf(path: string) {
		const root = (result?.root ?? '').replace(/[\\/]+$/, '');
		const folder = folderOf(path);
		if (root && folder.toLowerCase().startsWith(root.toLowerCase())) {
			const rest = folder.slice(root.length).replace(/^[\\/]+/, '');
			return rest || `${folderName(root) || root} (top folder)`;
		}
		return folder;
	}
	const dateOf = (ms: number) =>
		new Date(ms).toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' });

	function start(root: string) {
		selected = new Set();
		opened = new Set();
		shown = PAGE;
		status = null;
		searchDupes(root, minSize);
	}

	async function chooseFolder() {
		const path = await PickFolder('Choose a folder to search for duplicates').catch(() => '');
		if (path) start(path);
	}

	// the copy kept when selecting automatically
	function keeper(g: DupeGroup) {
		return keep === 'oldest' ? g.files[0].path : g.files[g.files.length - 1].path;
	}

	function selectExtras() {
		const next = new Set<string>();
		for (const g of groups) {
			if (g.risky) continue;
			const k = keeper(g);
			for (const f of g.files) if (f.path !== k) next.add(f.path);
		}
		selected = next;
	}

	// a copy can be ticked only while another copy in its set stays
	function canTick(g: DupeGroup, path: string) {
		if (selected.has(path)) return true;
		return g.files.filter((f) => !selected.has(f.path)).length > 1;
	}

	function toggle(g: DupeGroup, path: string) {
		const next = new Set(selected);
		if (next.has(path)) next.delete(path);
		else if (canTick(g, path)) next.add(path);
		selected = next;
	}

	async function remove() {
		isRemoving = true;
		const paths = [...selected];
		try {
			const r = await RemoveDuplicates(paths);
			const failed = new Set(r.failed);
			const gone = new Set(paths.filter((p) => !failed.has(p)));
			dropRemoved(gone);
			selected = new Set([...selected].filter((p) => failed.has(p)));
			status = {
				ok: r.failed.length === 0,
				text:
					`${r.removed} ${r.removed === 1 ? 'file' : 'files'} moved to the Recycle Bin, ${formatBytes(r.freed)} freed.` +
					(r.failed.length
						? ` ${r.failed.length} could not be moved (in use, or gone).`
						: ' Empty the Recycle Bin to get the space back.')
			};
		} catch (err) {
			status = { ok: false, text: `${err}` };
		} finally {
			isRemoving = false;
			confirmOpen = false;
		}
	}

	onMount(() => {
		GetDrives()
			.then((d) => (drives = (d || []) as Drive[]))
			.catch(() => {});
		GetDuplicateFolders()
			.then((p) => (places = (p || []) as Place[]))
			.catch(() => {});
	});
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<div class="flex justify-between items-center gap-3 pb-2 border-b border-line shrink-0">
		<div class="min-w-0">
			<div class="font-bold text-base text-ink">Duplicates</div>
			<div class="text-xs text-muted truncate">
				{#if dupeSearch.isSearching}
					Searching {dupeSearch.root}
				{:else if result}
					<span title={result.root}
						>{result.files.toLocaleString()} files compared in {folderName(result.root)}</span
					>
				{:else}
					Files with exactly the same contents, wherever they are
				{/if}
			</div>
		</div>
		{#if dupeSearch.isSearching}
			<button class="btn btn-secondary shrink-0" onclick={cancelDupeSearch}>Stop</button>
		{:else if result}
			<button
				class="text-[11px] text-muted font-medium cursor-pointer hover:text-ink shrink-0"
				onclick={() => (dupeSearch.result = null)}
			>
				New search
			</button>
		{/if}
	</div>

	{#if status}
		<div
			class="text-xs font-semibold px-1 shrink-0 {status.ok
				? 'text-green-700 dark:text-green-400'
				: 'text-amber-700 dark:text-amber-400'}"
		>
			{status.text}
		</div>
	{/if}

	{#if dupeSearch.isSearching}
		<div class="flex-1 flex flex-col items-center justify-center gap-3 text-center">
			<BusyArt kind="search" size={64} />
			<div class="text-xs font-semibold text-ink">
				{#if progress?.phase === 'comparing'}
					Comparing files that look alike...
				{:else}
					Listing files{progress ? `: ${progress.files.toLocaleString()} so far` : '...'}
				{/if}
			</div>
			{#if progress?.phase === 'comparing' && progress.total}
				<div class="w-64 h-1.5 rounded-full bg-track overflow-hidden">
					<div
						class="h-full bg-accent rounded-full transition-[width] duration-200"
						style="width: {Math.min(100, (progress.done / progress.total) * 100)}%"
					></div>
				</div>
			{/if}
			<div class="text-[11px] text-muted max-w-sm truncate" title={progress?.current}>
				{progress?.current ?? ''}
			</div>
			<div class="text-[11px] text-muted">You can use other tools meanwhile; it keeps going.</div>
		</div>
	{:else if !result}
		<!-- what to search -->
		<div class="flex-1 min-h-0 overflow-y-auto flex flex-col gap-4 pt-1">
			{#if dupeSearch.error}
				<p class="text-xs text-red-700 dark:text-red-400">{dupeSearch.error}</p>
			{/if}
			<div class="flex items-center gap-3">
				<span class="text-xs text-muted">Files larger than</span>
				<Segmented
					options={SIZES}
					value={minSize}
					onchange={(v) => (minSize = v)}
					label="Smallest file size"
					size="sm"
				/>
			</div>
			{#if places.length}
				<section class="flex flex-col gap-2">
					<h3 class="text-[10px] uppercase tracking-wider font-semibold text-muted">
						Your folders
					</h3>
					<div class="flex flex-wrap gap-2">
						{#each places as place (place.path)}
							<button
								class="btn btn-secondary"
								title={place.path}
								onclick={() => start(place.path)}
							>
								{place.label}
							</button>
						{/each}
						<button class="btn btn-primary" onclick={chooseFolder}>Choose a folder...</button>
					</div>
				</section>
			{/if}
			{#if drives.length}
				<section class="flex flex-col gap-2">
					<h3 class="text-[10px] uppercase tracking-wider font-semibold text-muted">Whole drive</h3>
					<div class="grid grid-cols-3 gap-2">
						{#each drives as drive (drive.path)}
							{@const letter = drive.path.replace(/[\\/]$/, '')}
							<button
								class="text-left px-3 py-2.5 rounded-xl border border-line bg-tile cursor-pointer transition-colors hover:border-accent"
								onclick={() => start(letter + '\\')}
							>
								<span class="flex items-baseline justify-between gap-2">
									<span class="font-bold text-ink text-sm">{letter}</span>
									<span class="text-[11px] text-muted"
										>{formatBytes(drive.total_bytes - drive.free_bytes)} used</span
									>
								</span>
								<span class="block mt-1 text-[11px] text-muted">Takes a few minutes</span>
							</button>
						{/each}
					</div>
				</section>
			{/if}
			<p class="text-[11px] text-muted leading-relaxed max-w-2xl">
				Windows, program folders, app data and project folders such as node_modules and .git are
				skipped, and so are OneDrive files that are only online. Nothing is removed unless you
				choose it, and removed files go to the Recycle Bin.
			</p>
		</div>
	{:else if result.groups.length === 0}
		<div class="flex-1 flex flex-col items-center justify-center gap-1 text-center px-8">
			<div class="text-sm font-semibold text-ink">No duplicates in {folderName(result.root)}</div>
			<div class="text-xs text-muted">
				Every file over {formatBytes(minSize)} there is one of a kind.
				{#if result.cloud_skipped}{result.cloud_skipped} online-only files were not checked.{/if}
			</div>
		</div>
	{:else}
		<!-- what was found -->
		<div class="flex items-center gap-3 shrink-0 px-1 flex-wrap">
			<div class="text-sm">
				<span class="font-bold text-ink">{formatBytes(visibleWasted)}</span>
				<span class="text-muted"
					>in extra copies · {groups.length} {groups.length === 1 ? 'set' : 'sets'}</span
				>
			</div>
			<div class="ml-auto flex items-center gap-2">
				<span class="text-[11px] text-muted">Keep the</span>
				<Segmented
					options={[
						{ id: 'oldest', label: 'Oldest' },
						{ id: 'newest', label: 'Newest' }
					]}
					value={keep}
					onchange={(v) => (keep = v as 'oldest' | 'newest')}
					label="Which copy to keep"
					size="sm"
				/>
				<button class="btn btn-secondary" onclick={selectExtras}>Select extra copies</button>
				<button
					class="btn btn-primary"
					disabled={selected.size === 0}
					onclick={() => (confirmOpen = true)}
				>
					Remove {selected.size ? `${selected.size} · ${formatBytes(selectedBytes)}` : ''}
				</button>
			</div>
		</div>
		{#if riskyCount}
			<label
				class="flex items-center gap-2 text-[11px] text-muted px-1 shrink-0 cursor-pointer w-fit"
			>
				<input
					type="checkbox"
					class="rounded border-line bg-tile checked:bg-accent text-accent focus:ring-accent"
					bind:checked={showRisky}
				/>
				Show {riskyCount}
				{riskyCount === 1 ? 'set' : 'sets'} of program files (.dll and the like): programs often need
				their own copy
			</label>
		{/if}

		<div class="flex-1 min-h-0 overflow-y-auto overscroll-contain pr-1.5 flex flex-col gap-1.5">
			{#each groups.slice(0, shown) as g (g.id)}
				{@const all = opened.has(g.id)}
				<div
					class="rounded-xl bg-tile border px-3 py-2 {g.risky
						? 'border-amber-500/40'
						: 'border-line'}"
				>
					<div class="flex items-baseline gap-2 min-w-0">
						<span
							class="text-xs font-semibold text-ink truncate"
							title={folderName(g.files[0].path)}
						>
							{folderName(g.files[0].path)}
						</span>
						<span class="text-[11px] text-muted shrink-0"
							>{formatBytes(g.size)} × {g.files.length}</span
						>
						{#if g.risky}<span class="text-[11px] text-amber-700 dark:text-amber-400 shrink-0"
								>program file</span
							>{/if}
						<span class="ml-auto text-[11px] font-mono text-muted shrink-0"
							>{formatBytes(g.wasted)} extra</span
						>
					</div>
					<div class="mt-1 flex flex-col">
						{#each all ? g.files : g.files.slice(0, ROWS) as f (f.path)}
							{@const ticked = selected.has(f.path)}
							{@const locked = !canTick(g, f.path)}
							<div class="flex items-center gap-2 py-0.5 min-w-0 group">
								<input
									type="checkbox"
									class="rounded border-line bg-tile checked:bg-accent text-accent focus:ring-accent disabled:opacity-30 shrink-0"
									checked={ticked}
									disabled={locked}
									title={locked ? 'The last copy is always kept' : ''}
									aria-label="Remove {f.path}"
									onchange={() => toggle(g, f.path)}
								/>
								<span
									class="flex-1 min-w-0 text-[11px] font-mono truncate {ticked
										? 'text-muted line-through'
										: 'text-ink'}"
									title={f.path}
								>
									{placeOf(f.path)}{#if folderName(f.path) !== folderName(g.files[0].path)}<span
											class="text-muted"
										>
											›
										</span>{folderName(f.path)}{/if}
								</span>
								{#if locked}<span
										class="text-[10px] font-semibold uppercase tracking-wider text-accent shrink-0"
										>Kept</span
									>{/if}
								<span class="text-[11px] text-muted shrink-0">{dateOf(f.modified)}</span>
								<button
									class="text-muted hover:text-ink cursor-pointer shrink-0 opacity-60 group-hover:opacity-100"
									title="Show in File Explorer"
									aria-label="Show in File Explorer"
									onclick={() => RevealInExplorer(f.path)}
								>
									<StrokeIcon d={REVEAL_ICON} size={13} />
								</button>
							</div>
						{/each}
						{#if g.files.length > ROWS && !all}
							<button
								class="self-start text-[11px] font-semibold text-accent cursor-pointer hover:underline mt-0.5"
								onclick={() => (opened = new Set([...opened, g.id]))}
							>
								{g.files.length - ROWS} more copies
							</button>
						{/if}
					</div>
				</div>
			{/each}
			{#if groups.length > shown}
				<button class="btn btn-secondary self-center my-1" onclick={() => (shown += PAGE)}>
					Show more ({groups.length - shown} left)
				</button>
			{/if}
			{#if result.truncated}
				<p class="text-[11px] text-muted text-center pb-1">
					Only the {result.groups.length} biggest sets are listed.
				</p>
			{/if}
		</div>
	{/if}
</div>

<Modal
	bind:showModal={confirmOpen}
	width="max-w-sm"
	showClose={!isRemoving}
	title="Move {selected.size} {selected.size === 1 ? 'file' : 'files'} to the Recycle Bin?"
	description="{formatBytes(
		selectedBytes
	)} of extra copies. At least one copy of each file stays where it is."
>
	<p class="text-muted leading-relaxed">
		You can restore them from the Recycle Bin. On drives without one (some USB drives), Windows asks
		before deleting for good.
	</p>
	{#snippet actions()}
		<button class="btn btn-secondary" disabled={isRemoving} onclick={() => (confirmOpen = false)}
			>Cancel</button
		>
		<button class="btn btn-primary" disabled={isRemoving} onclick={remove}>
			{isRemoving ? 'Moving...' : 'Move to Recycle Bin'}
		</button>
	{/snippet}
</Modal>

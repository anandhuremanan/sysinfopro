<script lang="ts">
	import { formatBytes } from '$lib';
	import {
		CancelDevCacheDelete,
		CancelNodeModulesSearch,
		DeleteDevCaches,
		FindNodeModules,
		PickFolder,
		RevealInExplorer,
		ScanDevCaches
	} from '$lib/wailsjs/go/main/App';
	import { EventsOn } from '$lib/wailsjs/runtime/runtime';
	import { onDestroy } from 'svelte';
	import Modal from '../ui/Modal.svelte';
	import StrokeIcon from '../ui/StrokeIcon.svelte';

	type Item = {
		id: string;
		name: string;
		kind: 'cache' | 'node_modules';
		path: string;
		size_bytes: number;
		files: number;
		note: string;
		deletable: boolean;
		last_used: string;
	};

	// "Unused for" choices, in months
	const AGE_OPTIONS = [
		{ months: 1, label: '1 mo' },
		{ months: 3, label: '3 mo' },
		{ months: 6, label: '6 mo' },
		{ months: 12, label: '1 yr' }
	];

	type DeleteProgress = {
		current: string;
		folder: number;
		folders: number;
		freed_bytes: number;
		files_deleted: number;
	};

	const REVEAL_ICON =
		'M14 4h6v6M20 4l-9 9M18 14v4a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4';

	let caches = $state<Item[]>([]);
	let modules = $state<Item[]>([]);
	let searchRoot = $state('');
	let searchCurrent = $state('');
	let isScanningCaches = $state(false);
	let isSearching = $state(false);
	let isDeleting = $state(false);
	let isStopping = $state(false);
	// live state of a running delete, from "devcache-delete-progress" events
	let deleteProgress = $state<DeleteProgress | null>(null);
	// bytes expected to be freed, fixed when the delete starts
	let deleteTotal = $state(0);
	let showConfirm = $state(false);
	let statusMessage = $state<string | undefined>(undefined);
	let selected = $state<Record<string, boolean>>({});
	let unusedMonths = $state(3);

	// Bumped on every search/cancel so a superseded search's result is ignored
	let searchId = 0;

	let allItems = $derived([...caches, ...modules]);
	let selectedItems = $derived(allItems.filter((item) => item.deletable && selected[item.path]));
	let selectedBytes = $derived(selectedItems.reduce((sum, item) => sum + item.size_bytes, 0));
	let totalBytes = $derived(allItems.reduce((sum, item) => sum + item.size_bytes, 0));
	let deletePercent = $derived(
		deleteProgress && deleteTotal > 0
			? Math.min(100, Math.round((deleteProgress.freed_bytes / deleteTotal) * 100))
			: 0
	);
	let deletingItem = $derived(allItems.find((item) => item.path === deleteProgress?.current));

	// node_modules whose project has not been touched since the cutoff; folders with
	// no known date are never included
	let staleModules = $derived.by(() => {
		const cutoff = new Date();
		cutoff.setMonth(cutoff.getMonth() - unusedMonths);
		return modules.filter((item) => {
			const used = parseDay(item.last_used);
			return used !== null && used < cutoff;
		});
	});
	let staleBytes = $derived(staleModules.reduce((sum, item) => sum + item.size_bytes, 0));

	// "YYYY-MM-DD" as a local date
	function parseDay(day: string): Date | null {
		const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(day ?? '');
		return match ? new Date(+match[1], +match[2] - 1, +match[3]) : null;
	}

	function usedAgo(day: string): string {
		const used = parseDay(day);
		if (!used) return '';
		const days = Math.floor((Date.now() - used.getTime()) / 86_400_000);
		if (days < 1) return 'Used today';
		if (days < 2) return 'Used yesterday';
		if (days < 31) return `Used ${days} days ago`;
		const months = Math.floor(days / 30.44);
		if (months < 12) return `Used ${months} ${months === 1 ? 'month' : 'months'} ago`;
		const years = Math.floor(days / 365.25);
		return `Used ${years} ${years === 1 ? 'year' : 'years'} ago`;
	}

	// select exactly the stale folders and ask to delete them
	function freeStale() {
		selected = Object.fromEntries(staleModules.map((item) => [item.path, true]));
		showConfirm = true;
	}

	function showStatus(message: string) {
		statusMessage = message;
		setTimeout(() => {
			statusMessage = undefined;
		}, 6000);
	}

	async function loadCaches() {
		isScanningCaches = true;
		try {
			caches = ((await ScanDevCaches()) || []) as Item[];
		} catch (err) {
			console.error('Failed to scan caches:', err);
			showStatus(`Failed to scan caches: ${err}`);
		} finally {
			isScanningCaches = false;
		}
	}

	async function chooseFolderAndSearch() {
		let folder = '';
		try {
			folder = await PickFolder('Choose a folder to search for node_modules');
		} catch (err) {
			showStatus(`${err}`);
			return;
		}
		if (!folder) return;
		await searchModules(folder);
	}

	async function searchModules(folder: string) {
		const id = ++searchId;
		searchRoot = folder;
		searchCurrent = folder;
		isSearching = true;
		modules = [];

		try {
			const found = ((await FindNodeModules(folder)) || []) as Item[];
			if (id !== searchId) return;
			modules = found;
		} catch (err) {
			if (id !== searchId) return;
			showStatus(`${err}`);
		} finally {
			if (id === searchId) isSearching = false;
		}
	}

	function cancelSearch() {
		if (!isSearching) return;
		searchId++;
		isSearching = false;
		CancelNodeModulesSearch();
	}

	function itemLabel(item: Item) {
		return item.kind === 'node_modules' ? `${item.name} / node_modules` : item.name;
	}

	// the dialog stays open and shows progress until the delete finishes or is stopped
	async function deleteSelected() {
		isDeleting = true;
		isStopping = false;
		deleteTotal = selectedBytes;
		deleteProgress = {
			current: '',
			folder: 0,
			folders: selectedItems.length,
			freed_bytes: 0,
			files_deleted: 0
		};
		const paths = selectedItems.map((item) => item.path);

		try {
			const result = await DeleteDevCaches(paths);
			finishDelete();
			const failed = result.failed ?? [];
			const stopped = failed.some((line) => /: (stopped|cancelled)$/.test(line));
			const broken = failed.filter((line) => !/: (stopped|cancelled)$/.test(line));
			showStatus(
				(stopped ? 'Stopped. ' : '') +
					`Freed ${formatBytes(result.freed_bytes)}` +
					(broken.length > 0
						? ` · ${broken.length} could not be fully removed (files in use?)`
						: '')
			);
			if (broken.length > 0) console.error('Could not remove:', broken);

			selected = {};
			const gone = new Set(result.deleted ?? []);
			modules = modules.filter((item) => !gone.has(item.path));
			// folders that were only partly deleted are smaller now
			if (failed.length > 0 && searchRoot) searchModules(searchRoot);
			await loadCaches();
		} catch (err) {
			finishDelete();
			showStatus(`Failed to delete: ${err}`);
		}
	}

	// close the dialog before the selection is cleared, so it never shows an empty list
	function finishDelete() {
		showConfirm = false;
		isDeleting = false;
		isStopping = false;
		deleteProgress = null;
	}

	function stopDelete() {
		isStopping = true;
		CancelDevCacheDelete();
	}

	async function reveal(path: string) {
		try {
			await RevealInExplorer(path);
		} catch (err) {
			showStatus(`${err}`);
		}
	}

	const unsubscribe = EventsOn('devcache-progress', (current: string) => {
		if (isSearching && current) searchCurrent = current;
	});
	const unsubscribeDelete = EventsOn('devcache-delete-progress', (progress: DeleteProgress) => {
		if (isDeleting) deleteProgress = progress;
	});

	$effect(() => {
		loadCaches();
	});

	onDestroy(() => {
		cancelSearch();
		unsubscribe();
		unsubscribeDelete();
	});
</script>

{#snippet row(item: Item, title: string, subtitle: string, isStale = false)}
	<label
		class="flex items-center gap-2.5 h-10 border-b border-line {item.deletable
			? 'cursor-pointer'
			: ''}"
		title={item.note}
	>
		<input
			type="checkbox"
			class="rounded border-line bg-tile text-accent focus:ring-accent disabled:opacity-30"
			disabled={!item.deletable || isDeleting}
			bind:checked={selected[item.path]}
		/>
		<span class="flex-1 min-w-0 flex flex-col leading-tight">
			<span class="font-semibold text-ink truncate">{title}</span>
			<span
				class="text-[11px] truncate {isStale
					? 'text-amber-700 dark:text-amber-400 font-medium'
					: 'text-muted'}"
				title={item.last_used ? `${item.path}\nProject last changed ${item.last_used}` : item.path}
				>{subtitle}</span
			>
		</span>
		<span class="shrink-0 font-mono font-bold text-ink">{formatBytes(item.size_bytes)}</span>
		<button
			aria-label="Show {title} in file manager"
			title="Show in file manager"
			class="shrink-0 w-6 h-6 rounded-md flex items-center justify-center text-muted cursor-pointer hover:text-ink hover:bg-ground"
			onclick={(event) => {
				event.preventDefault();
				reveal(item.path);
			}}
		>
			<StrokeIcon d={REVEAL_ICON} size={13} />
		</button>
	</label>
{/snippet}

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<!-- Header Bar -->
	<div class="flex justify-between items-center gap-3 pb-2 border-b border-line shrink-0">
		<div class="min-w-0">
			<div class="font-bold text-base text-ink">Dev Cache Cleaner</div>
			<div class="text-xs text-muted truncate">
				Reclaim space from package caches and old node_modules folders
			</div>
		</div>

		<div class="flex items-center gap-3 shrink-0">
			{#if totalBytes > 0}
				<span class="text-muted">{formatBytes(totalBytes)} found</span>
			{/if}
			<button
				class="btn btn-danger"
				disabled={selectedItems.length === 0 || isDeleting}
				onclick={() => (showConfirm = true)}
			>
				{isDeleting
					? `Deleting ${deletePercent}%`
					: selectedItems.length > 0
						? `Free ${formatBytes(selectedBytes)}`
						: 'Free selected'}
			</button>
		</div>
	</div>

	{#if statusMessage}
		<div
			class="text-xs text-green-700 dark:text-green-400 font-semibold px-1 shrink-0 truncate"
			title={statusMessage}
		>
			{statusMessage}
		</div>
	{/if}

	<div class="flex-1 min-h-0 grid grid-cols-2 gap-4">
		<!-- Package manager and build caches -->
		<div class="min-h-0 flex flex-col">
			<div class="flex items-center justify-between pb-1 shrink-0">
				<span class="text-[10px] uppercase tracking-wider font-semibold text-muted">
					Package and build caches
				</span>
				<button
					class="text-[11px] text-muted font-medium cursor-pointer hover:text-ink"
					onclick={loadCaches}
					disabled={isScanningCaches}
				>
					Rescan
				</button>
			</div>
			<div class="flex-1 min-h-0 overflow-y-auto overscroll-contain pr-1.5">
				{#if isScanningCaches}
					<div class="h-full flex items-center justify-center text-muted animate-pulse">
						Measuring caches...
					</div>
				{:else}
					{#each caches as item (item.path)}
						{@render row(item, item.name, item.deletable ? item.path : item.note)}
					{:else}
						<div class="h-full flex items-center justify-center text-muted text-center px-4">
							No developer caches found on this PC.
						</div>
					{/each}
				{/if}
			</div>
		</div>

		<!-- node_modules folders -->
		<div class="min-h-0 flex flex-col">
			<div class="flex items-center justify-between gap-2 pb-1 shrink-0">
				<span
					class="text-[10px] uppercase tracking-wider font-semibold text-muted truncate"
					title={searchRoot}
				>
					node_modules{searchRoot ? ` in ${searchRoot}` : ''}
				</span>
				{#if isSearching}
					<button
						class="text-[11px] text-muted font-medium cursor-pointer hover:text-ink shrink-0"
						onclick={cancelSearch}
					>
						Cancel
					</button>
				{:else if searchRoot}
					<button
						class="text-[11px] text-muted font-medium cursor-pointer hover:text-ink shrink-0"
						onclick={chooseFolderAndSearch}
					>
						Change folder
					</button>
				{/if}
			</div>
			{#if !isSearching && modules.length > 0}
				<div class="flex items-center gap-2 pb-1.5 shrink-0">
					<span class="text-[11px] text-muted shrink-0">Unused for</span>
					<div
						class="flex rounded-lg bg-ground border border-line p-0.5"
						role="radiogroup"
						aria-label="Unused for"
					>
						{#each AGE_OPTIONS as option (option.months)}
							<button
								role="radio"
								aria-checked={unusedMonths === option.months}
								class="h-5 px-1.5 rounded-md text-[11px] font-medium cursor-pointer transition-colors {unusedMonths ===
								option.months
									? 'bg-tile text-ink shadow-sm'
									: 'text-muted hover:text-ink'}"
								onclick={() => (unusedMonths = option.months)}
							>
								{option.label}
							</button>
						{/each}
					</div>
					<button
						class="btn btn-danger ml-auto h-6 px-2"
						disabled={staleModules.length === 0 || isDeleting}
						title={staleModules.length > 0
							? `Delete ${staleModules.length} node_modules not used in ${unusedMonths} ${unusedMonths === 1 ? 'month' : 'months'}`
							: 'No node_modules older than this'}
						onclick={freeStale}
					>
						{staleModules.length > 0
							? `Free ${formatBytes(staleBytes)} (${staleModules.length})`
							: 'None older'}
					</button>
				</div>
			{/if}
			<div class="flex-1 min-h-0 overflow-y-auto overscroll-contain pr-1.5">
				{#if isSearching}
					<div class="h-full flex flex-col items-center justify-center gap-2 text-center">
						<div
							class="w-7 h-7 border-3 border-accent border-t-transparent rounded-full animate-spin"
						></div>
						<div class="text-[11px] text-muted max-w-full truncate px-2" title={searchCurrent}>
							{searchCurrent}
						</div>
					</div>
				{:else if !searchRoot}
					<div class="h-full flex flex-col items-center justify-center gap-2 text-center px-4">
						<p class="text-muted leading-relaxed">
							Pick the folder that holds your projects to find every node_modules inside it.
						</p>
						<button class="btn btn-primary" onclick={chooseFolderAndSearch}>Choose folder</button>
					</div>
				{:else}
					{#each modules as item (item.path)}
						{@render row(
							item,
							item.name,
							item.last_used ? usedAgo(item.last_used) : item.path,
							staleModules.includes(item)
						)}
					{:else}
						<div class="h-full flex items-center justify-center text-muted text-center px-4">
							No node_modules folders found here.
						</div>
					{/each}
				{/if}
			</div>
		</div>
	</div>
</div>

<Modal
	bind:showModal={showConfirm}
	width="max-w-sm"
	showClose={!isDeleting}
	title={isDeleting
		? `Deleting ${selectedItems.length} ${selectedItems.length === 1 ? 'folder' : 'folders'}`
		: `Delete ${selectedItems.length} ${selectedItems.length === 1 ? 'folder' : 'folders'}?`}
	description={isDeleting
		? isStopping
			? 'Stopping after the current file...'
			: 'Keep the app open until this finishes.'
		: `${formatBytes(selectedBytes)} will be freed. This cannot be undone.`}
>
	<div class="flex flex-col gap-2.5">
		{#if isDeleting && deleteProgress}
			<div class="flex flex-col gap-1.5">
				<div class="flex items-baseline justify-between gap-3">
					<span class="font-semibold text-ink">
						{formatBytes(deleteProgress.freed_bytes)}
						<span class="font-normal text-muted">of {formatBytes(deleteTotal)} freed</span>
					</span>
					<span class="font-mono font-bold text-ink">{deletePercent}%</span>
				</div>
				<div
					class="h-2 rounded-full bg-track overflow-hidden"
					role="progressbar"
					aria-label="Delete progress"
					aria-valuemin={0}
					aria-valuemax={100}
					aria-valuenow={deletePercent}
				>
					<div
						class="h-full rounded-full bg-accent transition-[width] duration-200 ease-out"
						style="width: {deletePercent}%"
					></div>
				</div>
				<div class="text-[11px] text-muted truncate" title={deleteProgress.current}>
					{deleteProgress.files_deleted.toLocaleString()} files removed{deletingItem
						? ` · now ${itemLabel(deletingItem)}`
						: ''}
				</div>
			</div>
		{/if}

		<div class="max-h-28 overflow-y-auto rounded-lg bg-ground border border-line px-2.5 py-1.5">
			{#each selectedItems as item, index (item.path)}
				{@const position = index + 1}
				{@const isDone = isDeleting && deleteProgress && position < deleteProgress.folder}
				{@const isCurrent = isDeleting && deleteProgress && position === deleteProgress.folder}
				<div class="flex items-center justify-between gap-3 py-0.5">
					<span
						class="flex items-center gap-1.5 min-w-0 {isDone
							? 'text-muted'
							: isCurrent
								? 'text-ink font-semibold'
								: ''} {isDeleting && !isDone && !isCurrent ? 'opacity-60' : ''}"
						title={item.path}
					>
						{#if isDone}
							<span class="shrink-0 text-accent" aria-label="Deleted">
								<StrokeIcon d="M5 12.5l4.5 4.5L19 7.5" size={12} />
							</span>
						{:else if isCurrent}
							<span
								class="shrink-0 w-3 h-3 border-2 border-accent border-t-transparent rounded-full animate-spin"
								aria-label="Deleting"
							></span>
						{/if}
						<span class="truncate">{itemLabel(item)}</span>
					</span>
					<span class="shrink-0 font-mono text-muted">{formatBytes(item.size_bytes)}</span>
				</div>
			{/each}
		</div>

		{#if !isDeleting}
			<p class="text-muted leading-relaxed">
				Nothing here is your own work: caches are downloaded again when needed, and node_modules
				comes back when you run install in that project. Close editors and dev servers first.
			</p>
		{/if}
	</div>

	{#snippet actions()}
		{#if isDeleting}
			<button class="btn btn-secondary" disabled={isStopping} onclick={stopDelete}>
				{isStopping ? 'Stopping...' : 'Stop'}
			</button>
		{:else}
			<button class="btn btn-secondary" onclick={() => (showConfirm = false)}>Cancel</button>
			<button class="btn btn-danger" onclick={deleteSelected}>Delete</button>
		{/if}
	{/snippet}
</Modal>

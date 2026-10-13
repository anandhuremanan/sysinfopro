<script lang="ts">
	import { formatBytes } from '$lib';
	import {
		AnalyzeDiskUsage,
		CancelDiskScan,
		GetWatchFolders,
		PickFolder,
		RevealInExplorer
	} from '$lib/wailsjs/go/main/App';
	import { onDestroy } from 'svelte';
	import StrokeIcon from '../ui/StrokeIcon.svelte';

	type WatchFolder = { label: string; path: string };
	type Entry = { name: string; path: string; is_dir: boolean; size_bytes: number };
	type Row = Entry & { delta: number; perMinute: number };

	const CHECK_EVERY_S = 30;
	const REVEAL_ICON =
		'M14 4h6v6M20 4l-9 9M18 14v4a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4';

	let folders = $state<WatchFolder[]>([]);
	let watchPath = $state('');
	let isWatching = $state(false);
	let isScanning = $state(false);
	let checks = $state(0);
	let nextCheckIn = $state(0);
	let startedAt = $state(0);
	let lastCheckAt = $state(0);
	let rows = $state<Row[]>([]);
	let totalDelta = $state(0);
	let statusMessage = $state<string | undefined>(undefined);

	// sizes at the first scan, by item name
	let baseline = new Map<string, number>();
	// bumped on every start/stop so an old watch loop exits
	let watchId = 0;

	let changedRows = $derived(rows.filter((row) => row.delta !== 0));

	function showStatus(message: string) {
		statusMessage = message;
		setTimeout(() => {
			statusMessage = undefined;
		}, 5000);
	}

	function signed(bytes: number) {
		return (bytes > 0 ? '+' : bytes < 0 ? '−' : '') + formatBytes(Math.abs(bytes));
	}

	async function scan(path: string): Promise<Entry[] | undefined> {
		isScanning = true;
		try {
			const usage = await AnalyzeDiskUsage(path);
			return usage.cancelled ? undefined : (usage.entries as Entry[]);
		} finally {
			isScanning = false;
		}
	}

	const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

	async function start(path: string) {
		stop();
		const id = ++watchId;
		watchPath = path;
		isWatching = true;
		checks = 0;
		rows = [];
		totalDelta = 0;

		try {
			const first = await scan(path);
			if (id !== watchId || !first) return;
			baseline = new Map(first.map((entry) => [entry.name, entry.size_bytes]));
			startedAt = Date.now();
			lastCheckAt = startedAt;

			while (id === watchId) {
				for (nextCheckIn = CHECK_EVERY_S; nextCheckIn > 0; nextCheckIn--) {
					await sleep(1000);
					if (id !== watchId) return;
				}

				const current = await scan(path);
				if (id !== watchId || !current) return;

				lastCheckAt = Date.now();
				const minutes = Math.max((lastCheckAt - startedAt) / 60000, 1 / 60);
				rows = current
					.map((entry) => {
						// an item that appeared after the first scan counts in full
						const delta = entry.size_bytes - (baseline.get(entry.name) ?? 0);
						return { ...entry, delta, perMinute: delta / minutes };
					})
					.sort((a, b) => Math.abs(b.delta) - Math.abs(a.delta));
				totalDelta = rows.reduce((sum, row) => sum + row.delta, 0);
				checks++;
			}
		} catch (err) {
			if (id === watchId) {
				showStatus(`${err}`);
				isWatching = false;
			}
		}
	}

	function stop() {
		if (!isWatching) return;
		watchId++;
		isWatching = false;
		if (isScanning) CancelDiskScan();
	}

	async function chooseFolder() {
		try {
			const path = await PickFolder('Choose a folder to watch for growth');
			if (path) start(path);
		} catch (err) {
			showStatus(`${err}`);
		}
	}

	async function reveal(path: string) {
		try {
			await RevealInExplorer(path);
		} catch (err) {
			showStatus(`${err}`);
		}
	}

	$effect(() => {
		GetWatchFolders()
			.then((result) => (folders = result || []))
			.catch((err) => console.error('Failed to get watch folders:', err));
	});

	onDestroy(stop);
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<!-- Header Bar -->
	<div class="flex justify-between items-center gap-3 pb-2 border-b border-line shrink-0">
		<div class="min-w-0">
			<div class="font-bold text-base text-ink">Disk Growth Watch</div>
			<div class="text-xs text-muted truncate" title={watchPath}>
				{isWatching ? `Watching ${watchPath}` : 'See which folders are growing right now'}
			</div>
		</div>

		{#if isWatching}
			<div class="flex items-center gap-3 shrink-0">
				<span class="text-muted tabular-nums">
					{#if isScanning}
						Measuring...
					{:else}
						Next check in {nextCheckIn}s
					{/if}
				</span>
				<button class="btn btn-secondary" onclick={stop}>Stop</button>
			</div>
		{/if}
	</div>

	{#if statusMessage}
		<div class="text-xs text-amber-700 dark:text-amber-400 font-semibold px-1 shrink-0 truncate">
			{statusMessage}
		</div>
	{/if}

	{#if !isWatching}
		<!-- Ready State -->
		<div class="flex-1 flex flex-col items-center justify-center gap-3 text-center">
			<p class="text-xs text-muted max-w-md leading-relaxed">
				Pick a place to watch. It is measured once as a starting point, then again every
				{CHECK_EVERY_S} seconds, and the folders that grew are listed with how fast. Nothing is
				changed.
			</p>
			<div class="flex flex-wrap justify-center gap-2 max-w-xl">
				{#each folders as folder (folder.path)}
					<button class="btn btn-secondary" title={folder.path} onclick={() => start(folder.path)}>
						{folder.label}
					</button>
				{/each}
				<button class="btn btn-primary" onclick={chooseFolder}>Choose folder</button>
			</div>
		</div>
	{:else if checks === 0}
		<div class="flex-1 flex flex-col items-center justify-center gap-2 text-center">
			<div class="w-9 h-9 border-3 border-accent border-t-transparent rounded-full animate-spin"></div>
			<div class="text-xs text-ink font-semibold">
				{isScanning ? 'Taking the starting measurement...' : 'Starting point recorded'}
			</div>
			<div class="text-[11px] text-muted">
				{isScanning ? 'Large folders take a while the first time' : `First comparison in ${nextCheckIn}s`}
			</div>
		</div>
	{:else}
		<div class="flex-1 min-h-0 overflow-y-auto overscroll-contain pr-1.5">
			{#each changedRows as row (row.path)}
				<div class="flex items-center gap-3 h-9 border-b border-line">
					<span class="flex-1 min-w-0 font-semibold text-ink truncate" title={row.path}>
						{row.name}
					</span>
					<span class="shrink-0 w-20 text-right text-muted tabular-nums">
						{formatBytes(row.size_bytes)}
					</span>
					<span
						class="shrink-0 w-24 text-right font-mono font-bold {row.delta > 0
							? 'text-amber-700 dark:text-amber-400'
							: 'text-green-700 dark:text-green-400'}"
					>
						{signed(row.delta)}
					</span>
					<span class="shrink-0 w-28 text-right text-muted tabular-nums">
						{signed(row.perMinute)}/min
					</span>
					<button
						aria-label="Show {row.name} in file manager"
						title="Show in file manager"
						class="shrink-0 w-6 h-6 rounded-md flex items-center justify-center text-muted cursor-pointer hover:text-ink hover:bg-ground"
						onclick={() => reveal(row.path)}
					>
						<StrokeIcon d={REVEAL_ICON} size={13} />
					</button>
				</div>
			{:else}
				<div class="h-full flex flex-col items-center justify-center gap-1 text-center">
					<div class="font-semibold text-ink">No change so far</div>
					<p class="text-muted">Nothing in this folder has grown or shrunk since the watch began.</p>
				</div>
			{/each}
		</div>

		<!-- Footer Note -->
		<div
			class="text-[11px] text-muted pt-1.5 border-t border-line flex justify-between items-center gap-3 shrink-0"
		>
			<span>
				Net change {signed(totalDelta)} over {Math.max(
					1,
					Math.round((lastCheckAt - startedAt) / 1000)
				)}s · {checks}
				{checks === 1 ? 'check' : 'checks'}
			</span>
			<span class="shrink-0">Sizes are compared with the start of the watch</span>
		</div>
	{/if}
</div>

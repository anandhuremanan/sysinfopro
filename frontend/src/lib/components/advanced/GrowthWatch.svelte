<script lang="ts">
	import BusyArt from '../ui/BusyArt.svelte';
	import { formatBytes, usageBarColor } from '$lib';
	import {
		AnalyzeDiskUsage,
		CancelDiskScan,
		GetDrives,
		GetWatchFolders,
		PickFolder,
		RevealInExplorer
	} from '$lib/wailsjs/go/main/App';
	import { diskScan } from '$lib/diskScan.svelte';
	import { onDestroy } from 'svelte';
	import StrokeIcon from '../ui/StrokeIcon.svelte';

	type WatchFolder = { label: string; path: string };
	type Drive = { path: string; total_bytes: number; free_bytes: number; used_pct: number };
	type Entry = { name: string; path: string; is_dir: boolean; size_bytes: number };
	type Row = Entry & { delta: number; perMinute: number };

	const CHECK_EVERY_S = 30;
	const REVEAL_ICON = 'M14 4h6v6M20 4l-9 9M18 14v4a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4';

	let folders = $state<WatchFolder[]>([]);
	let drives = $state<Drive[]>([]);
	// watching a whole drive: free space is tracked too, and checks wait for each pass
	let watchDrive = $state('');
	let freeAtStart = $state(0);
	let freeNow = $state(0);
	let waitSeconds = $state(CHECK_EVERY_S);
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

	// "C:" -> "C:\" ; a drive root, or "" for a folder
	const driveOf = (path: string) =>
		/^[A-Za-z]:[\\/]?$/.test(path) ? path.slice(0, 2).toUpperCase() : '';
	const rootOf = (drive: string) => drive + '\\';

	async function readFree(drive: string) {
		const list = ((await GetDrives().catch(() => [])) || []) as Drive[];
		return list.find((d) => d.path.toUpperCase().startsWith(drive))?.free_bytes ?? 0;
	}

	async function start(path: string) {
		// the measuring engine does one job at a time; never cancel Disk usage's analysis from here
		if (diskScan.isScanning) {
			showStatus(
				'Disk usage is analysing a drive. Start the watch when it finishes, or cancel it there.'
			);
			return;
		}
		stop();
		const id = ++watchId;
		watchPath = path;
		watchDrive = driveOf(path);
		isWatching = true;
		checks = 0;
		rows = [];
		totalDelta = 0;
		waitSeconds = CHECK_EVERY_S;

		try {
			if (watchDrive) freeAtStart = freeNow = await readFree(watchDrive);
			const began = Date.now();
			const first = await scan(path);
			if (id !== watchId) return;
			if (!first) {
				showStatus('The watch stopped because another analysis started.');
				isWatching = false;
				return;
			}
			// a whole drive takes minutes to measure, so wait at least that long between checks
			waitSeconds = Math.max(CHECK_EVERY_S, Math.round((Date.now() - began) / 1000));
			baseline = new Map(first.map((entry) => [entry.name, entry.size_bytes]));
			startedAt = Date.now();
			lastCheckAt = startedAt;

			while (id === watchId) {
				for (nextCheckIn = waitSeconds; nextCheckIn > 0; nextCheckIn--) {
					await sleep(1000);
					if (id !== watchId) return;
				}

				if (diskScan.isScanning) continue; // Disk usage is busy; try again next round
				if (watchDrive) freeNow = await readFree(watchDrive);
				const current = await scan(path);
				if (id !== watchId) return;
				if (!current) {
					showStatus('The watch stopped because another analysis started.');
					isWatching = false;
					return;
				}

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
		GetDrives()
			.then((result) => (drives = (result || []) as Drive[]))
			.catch((err) => console.error('Failed to list drives:', err));
	});

	onDestroy(stop);
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<!-- Header Bar -->
	<div class="flex justify-between items-center gap-3 pb-2 border-b border-line shrink-0">
		<div class="min-w-0">
			<div class="font-bold text-base text-ink">Disk growth</div>
			<div class="text-xs text-muted truncate" title={watchPath}>
				{isWatching
					? `Watching ${watchDrive ? `all of ${watchDrive}` : watchPath}`
					: 'Pick a drive or folder to see what is growing in it'}
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
		<!-- Ready State: what to watch -->
		<div class="flex-1 min-h-0 overflow-y-auto flex flex-col gap-4 pt-1">
			<p class="text-xs text-muted leading-relaxed max-w-2xl">
				Measured now, then checked again and again; folders that grow are listed with how fast.
				Nothing is changed.
			</p>

			{#if drives.length}
				<section class="flex flex-col gap-2">
					<h3 class="text-[10px] uppercase tracking-wider font-semibold text-muted">Drives</h3>
					<div class="grid grid-cols-3 gap-2">
						{#each drives as drive (drive.path)}
							{@const letter = drive.path.replace(/[\\/]$/, '')}
							<button
								class="text-left px-3 py-2.5 rounded-xl border border-line bg-tile cursor-pointer transition-colors hover:border-accent"
								onclick={() => start(rootOf(letter))}
							>
								<span class="flex items-baseline justify-between gap-2">
									<span class="font-bold text-ink text-sm">{letter}</span>
									<span class="text-[11px] text-muted">{formatBytes(drive.free_bytes)} free</span>
								</span>
								<span class="block h-1 mt-2 rounded-full bg-track">
									<span
										class="block h-1 rounded-full {usageBarColor(drive.used_pct)}"
										style="width: {Math.min(100, drive.used_pct)}%"
									></span>
								</span>
								<span class="block mt-1.5 text-[11px] text-muted"
									>Whole drive · first pass takes minutes</span
								>
							</button>
						{/each}
					</div>
				</section>
			{/if}

			<section class="flex flex-col gap-2">
				<h3 class="text-[10px] uppercase tracking-wider font-semibold text-muted">
					Common folders
				</h3>
				<div class="flex flex-wrap gap-2">
					{#each folders as folder (folder.path)}
						<button
							class="btn btn-secondary"
							title={folder.path}
							onclick={() => start(folder.path)}
						>
							{folder.label}
						</button>
					{/each}
					<button class="btn btn-secondary" onclick={chooseFolder}>Choose a folder...</button>
				</div>
			</section>
		</div>
	{:else if checks === 0}
		<div class="flex-1 flex flex-col items-center justify-center gap-2 text-center">
			<BusyArt kind="measure" size={64} />
			<div class="text-xs text-ink font-semibold">
				{isScanning ? 'Taking the starting measurement...' : 'Starting point recorded'}
			</div>
			<div class="text-[11px] text-muted">
				{isScanning
					? watchDrive
						? 'A whole drive takes a few minutes the first time'
						: 'Large folders take a while the first time'
					: `First comparison in ${nextCheckIn}s`}
			</div>
		</div>
	{:else}
		{#if watchDrive}
			<div class="flex items-baseline gap-2 shrink-0 px-1">
				<span class="text-muted">Free space on {watchDrive}</span>
				<span class="font-bold text-ink">{formatBytes(freeNow)}</span>
				<span
					class="font-mono font-bold {freeNow < freeAtStart
						? 'text-amber-700 dark:text-amber-400'
						: 'text-green-700 dark:text-green-400'}"
				>
					{freeNow === freeAtStart ? 'no change' : signed(freeNow - freeAtStart)}
				</span>
				<span class="text-muted">since the watch began</span>
			</div>
		{/if}
		<div
			class="flex-1 min-h-0 overflow-y-auto overscroll-contain pb-5 [mask-image:linear-gradient(to_bottom,#000_calc(100%-20px),transparent)] pr-1.5"
		>
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
					<p class="text-muted">
						Nothing in this folder has grown or shrunk since the watch began.
					</p>
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

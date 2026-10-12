<script lang="ts">
	import { formatBytes, usageBarColor } from '$lib';
	import {
		AnalyzeDiskUsage,
		CancelDiskScan,
		GetDrives,
		RevealInExplorer
	} from '$lib/wailsjs/go/main/App';
	import { EventsOn } from '$lib/wailsjs/runtime/runtime';
	import { onDestroy } from 'svelte';
	import StrokeIcon from '../ui/StrokeIcon.svelte';

	type Drive = {
		path: string;
		fs_type: string;
		total_bytes: number;
		free_bytes: number;
		used_bytes: number;
		used_pct: number;
	};

	type UsageEntry = {
		name: string;
		path: string;
		is_dir: boolean;
		size_bytes: number;
		files: number;
	};

	type FolderUsage = {
		path: string;
		total_bytes: number;
		files: number;
		skipped: number;
		entries: UsageEntry[];
		largest_files: UsageEntry[];
		duration_ms: number;
		cancelled: boolean;
	};

	type ScanProgress = { files: number; bytes: number; current: string };

	const FOLDER_ICON =
		'M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7z';
	const FILE_ICON = 'M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8l-5-5zM14 3v5h5';
	const REVEAL_ICON =
		'M14 4h6v6M20 4l-9 9M18 14v4a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4';

	let drives = $state<Drive[]>([]);
	let selectedDrive = $state<Drive | undefined>(undefined);
	let currentPath = $state('');
	let usage = $state<FolderUsage | undefined>(undefined);
	let isScanning = $state(false);
	let scanProgress = $state<ScanProgress | undefined>(undefined);
	let statusMessage = $state<string | undefined>(undefined);

	// Finished scans, so going back up a level is instant
	const cache = new Map<string, FolderUsage>();
	// Bumped on every scan/cancel so a superseded scan's result is ignored
	let scanId = 0;

	// "C:" would mean the current directory on C, so drive roots always end in a separator
	function driveRoot(drive: Drive) {
		return /^[A-Za-z]:$/.test(drive.path) ? drive.path + '\\' : drive.path;
	}

	// Clickable path from the drive root down to the folder being shown
	let crumbs = $derived.by(() => {
		if (!selectedDrive || !currentPath) return [];
		const root = driveRoot(selectedDrive);
		const sep = root.includes('\\') ? '\\' : '/';
		const result = [{ label: root.replace(/[\\/]$/, '') || root, path: root }];
		let acc = root;
		for (const part of currentPath.slice(root.length).split(/[\\/]/).filter(Boolean)) {
			acc = acc.endsWith(sep) ? acc + part : acc + sep + part;
			result.push({ label: part, path: acc });
		}
		return result;
	});

	let largestEntry = $derived(usage?.entries[0]?.size_bytes || 0);

	function showStatus(message: string) {
		statusMessage = message;
		setTimeout(() => {
			statusMessage = undefined;
		}, 4000);
	}

	async function loadDrives() {
		try {
			drives = (await GetDrives()) || [];
			if (!selectedDrive && drives.length > 0) {
				selectDrive(drives[0]);
			}
		} catch (err) {
			console.error('Failed to list drives:', err);
			showStatus(`Failed to list drives: ${err}`);
		}
	}

	function selectDrive(drive: Drive) {
		cancelScan();
		selectedDrive = drive;
		currentPath = driveRoot(drive);
		usage = cache.get(currentPath);
	}

	async function scan(path: string, force = false) {
		cancelScan();
		currentPath = path;

		const cached = cache.get(path);
		if (cached && !force) {
			usage = cached;
			return;
		}

		const id = ++scanId;
		isScanning = true;
		scanProgress = undefined;
		usage = undefined;

		try {
			const result = (await AnalyzeDiskUsage(path)) as FolderUsage;
			if (id !== scanId) return;
			if (result.cancelled) return;
			cache.set(path, result);
			usage = result;
		} catch (err) {
			if (id !== scanId) return;
			showStatus(`${err}`);
		} finally {
			if (id === scanId) isScanning = false;
		}
	}

	function cancelScan() {
		if (!isScanning) return;
		scanId++;
		isScanning = false;
		CancelDiskScan();
	}

	async function reveal(path: string) {
		try {
			await RevealInExplorer(path);
		} catch (err) {
			showStatus(`${err}`);
		}
	}

	function share(bytes: number) {
		if (!usage || usage.total_bytes <= 0) return '0%';
		const pct = (bytes / usage.total_bytes) * 100;
		return pct > 0 && pct < 0.1 ? '<0.1%' : `${pct.toFixed(1)}%`;
	}

	const unsubscribe = EventsOn('disk-scan-progress', (data: ScanProgress) => {
		if (isScanning && data) scanProgress = data;
	});

	$effect(() => {
		loadDrives();
	});

	onDestroy(() => {
		cancelScan();
		unsubscribe();
	});
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<!-- Header Bar: drives -->
	<div class="flex justify-between items-center gap-3 pb-2 border-b border-line shrink-0">
		<div class="min-w-0">
			<div class="font-bold text-base text-ink">Disk Usage Analyser</div>
			<div class="text-xs text-muted truncate">
				See which folders and files take up space on each drive
			</div>
		</div>

		<div class="flex items-center gap-2 shrink-0">
			{#each drives as drive (drive.path)}
				<button
					class="w-[132px] px-2.5 py-1.5 rounded-lg border text-left cursor-pointer transition-colors {selectedDrive?.path ===
					drive.path
						? 'border-accent bg-ground'
						: 'border-line hover:bg-ground'}"
					aria-pressed={selectedDrive?.path === drive.path}
					onclick={() => selectDrive(drive)}
				>
					<span class="flex justify-between items-baseline gap-2">
						<span class="font-bold text-ink">{drive.path.replace(/[\\/]$/, '')}</span>
						<span class="text-[10px] text-muted truncate">{formatBytes(drive.free_bytes)} free</span>
					</span>
					<span class="block h-1 mt-1 rounded-full bg-track">
						<span
							class="block h-1 rounded-full {usageBarColor(drive.used_pct)}"
							style="width: {Math.min(100, drive.used_pct)}%"
						></span>
					</span>
				</button>
			{/each}
		</div>
	</div>

	<!-- Path bar -->
	{#if selectedDrive}
		<div class="flex items-center justify-between gap-3 shrink-0">
			<nav aria-label="Folder path" class="flex items-center gap-1 min-w-0 overflow-hidden">
				{#each crumbs as crumb, i (crumb.path)}
					{#if i > 0}
						<span class="text-muted">/</span>
					{/if}
					{#if i === crumbs.length - 1}
						<span class="font-bold text-ink truncate" aria-current="page">{crumb.label}</span>
					{:else}
						<button
							class="text-muted font-medium cursor-pointer hover:text-ink shrink-0 max-w-[120px] truncate"
							onclick={() => scan(crumb.path)}
						>
							{crumb.label}
						</button>
					{/if}
				{/each}
			</nav>
			{#if usage}
				<button
					class="text-xs bg-ground px-3 py-1 rounded-lg cursor-pointer hover:bg-line font-medium border border-line shrink-0"
					onclick={() => scan(currentPath, true)}
				>
					Rescan
				</button>
			{/if}
		</div>
	{/if}

	{#if statusMessage}
		<div class="text-xs text-amber-700 dark:text-amber-400 font-semibold px-1 shrink-0 truncate">
			{statusMessage}
		</div>
	{/if}

	<!-- Main area -->
	{#if isScanning}
		<div class="flex-1 flex flex-col items-center justify-center gap-2 min-h-0">
			<div class="w-9 h-9 border-3 border-accent border-t-transparent rounded-full animate-spin"></div>
			<div class="text-xs text-ink font-semibold tabular-nums">
				{#if scanProgress}
					{scanProgress.files.toLocaleString()} files · {formatBytes(scanProgress.bytes)} scanned
				{:else}
					Starting scan...
				{/if}
			</div>
			<div class="text-[11px] text-muted max-w-[520px] truncate" title={scanProgress?.current}>
				{scanProgress?.current || currentPath}
			</div>
			<button
				class="mt-1 text-xs bg-ground px-3 py-1.5 rounded-lg cursor-pointer hover:bg-line font-medium border border-line"
				onclick={cancelScan}
			>
				Cancel
			</button>
		</div>
	{:else if usage}
		<div class="flex-1 min-h-0 grid grid-cols-5 gap-3">
			<!-- Folders and files in this location -->
			<div class="col-span-3 min-h-0 flex flex-col">
				<div class="text-[10px] uppercase tracking-wider font-semibold text-muted pb-1 shrink-0">
					In this folder
				</div>
				<div class="flex-1 min-h-0 overflow-y-auto overscroll-contain pr-1.5">
					{#each usage.entries as entry (entry.path)}
						<div class="group flex items-center gap-2 h-8 border-b border-line">
							<span class="shrink-0 {entry.is_dir ? 'text-accent' : 'text-muted'}">
								<StrokeIcon d={entry.is_dir ? FOLDER_ICON : FILE_ICON} size={14} />
							</span>
							{#if entry.is_dir && entry.size_bytes > 0}
								<button
									class="flex-1 min-w-0 text-left font-semibold text-ink truncate cursor-pointer hover:underline"
									title="Open {entry.path}"
									onclick={() => scan(entry.path)}
								>
									{entry.name}
								</button>
							{:else}
								<span class="flex-1 min-w-0 text-ink truncate" title={entry.path}>{entry.name}</span>
							{/if}
							<span class="shrink-0 w-16 h-1 rounded-full bg-track">
								<span
									class="block h-1 rounded-full bg-accent"
									style="width: {largestEntry > 0 ? (entry.size_bytes / largestEntry) * 100 : 0}%"
								></span>
							</span>
							<span class="shrink-0 w-10 text-right text-muted tabular-nums">
								{share(entry.size_bytes)}
							</span>
							<span class="shrink-0 w-16 text-right font-mono font-bold text-ink">
								{formatBytes(entry.size_bytes)}
							</span>
							<button
								aria-label="Show {entry.name} in file manager"
								title="Show in file manager"
								class="shrink-0 w-6 h-6 rounded-md flex items-center justify-center text-muted cursor-pointer hover:text-ink hover:bg-ground"
								onclick={() => reveal(entry.path)}
							>
								<StrokeIcon d={REVEAL_ICON} size={13} />
							</button>
						</div>
					{:else}
						<div class="h-full flex items-center justify-center text-muted">This folder is empty.</div>
					{/each}
				</div>
			</div>

			<!-- Largest files anywhere below this location -->
			<div class="col-span-2 min-h-0 flex flex-col">
				<div class="text-[10px] uppercase tracking-wider font-semibold text-muted pb-1 shrink-0">
					Largest files here and below
				</div>
				<div class="flex-1 min-h-0 overflow-y-auto overscroll-contain pr-1.5">
					{#each usage.largest_files as file (file.path)}
						<div class="flex items-center gap-2 h-8 border-b border-line">
							<span class="flex-1 min-w-0 text-ink truncate" title={file.path}>{file.name}</span>
							<span class="shrink-0 font-mono font-bold text-ink">{formatBytes(file.size_bytes)}</span>
							<button
								aria-label="Show {file.name} in file manager"
								title="Show in file manager"
								class="shrink-0 w-6 h-6 rounded-md flex items-center justify-center text-muted cursor-pointer hover:text-ink hover:bg-ground"
								onclick={() => reveal(file.path)}
							>
								<StrokeIcon d={REVEAL_ICON} size={13} />
							</button>
						</div>
					{:else}
						<div class="h-full flex items-center justify-center text-muted">No files found.</div>
					{/each}
				</div>
			</div>
		</div>

		<!-- Footer Note -->
		<div
			class="text-[11px] text-muted pt-1.5 border-t border-line flex justify-between items-center gap-3 shrink-0"
		>
			<span class="truncate">
				{formatBytes(usage.total_bytes)} in {usage.files.toLocaleString()} files · scanned in {(
					usage.duration_ms / 1000
				).toFixed(1)}s
			</span>
			{#if usage.skipped > 0}
				<span
					class="shrink-0 text-amber-700 dark:text-amber-400"
					title="Usually protected system folders. Run SysInfo Pro as administrator to include them."
				>
					{usage.skipped.toLocaleString()} items could not be read
				</span>
			{/if}
		</div>
	{:else if selectedDrive}
		<!-- Ready State -->
		<div class="flex-1 flex flex-col items-center justify-center gap-2 text-center min-h-0">
			<div class="text-sm font-bold text-ink">
				{formatBytes(selectedDrive.used_bytes)} used of {formatBytes(selectedDrive.total_bytes)}
			</div>
			<p class="text-xs text-muted max-w-sm leading-relaxed">
				Scanning reads file sizes only and changes nothing. A full drive can take a few minutes,
				longer on a hard disk.
			</p>
			<button
				class="mt-1 px-4 py-1.5 text-xs bg-accent hover:opacity-90 text-on-accent font-medium rounded-lg cursor-pointer"
				onclick={() => scan(currentPath)}
			>
				Analyse {currentPath}
			</button>
		</div>
	{:else}
		<div class="flex-1 flex items-center justify-center text-xs text-muted">Looking for drives...</div>
	{/if}
</div>

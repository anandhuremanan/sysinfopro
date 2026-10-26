<script lang="ts">
	import BusyArt from '../ui/BusyArt.svelte';
	import { formatBytes } from '$lib';
	import { load, recall } from '$lib/cache';
	import { recordFreed } from '$lib/rating.svelte';
	import {
		CancelCleanJunk,
		CleanJunk,
		GetDriveHealth,
		ScanJunk
	} from '$lib/wailsjs/go/main/App';
	import { EventsOn } from '$lib/wailsjs/runtime/runtime';
	import { onDestroy, onMount } from 'svelte';
	import Card from '../ui/Card.svelte';
	import DriveScan from '../ui/DriveScan.svelte';
	import Modal from '../ui/Modal.svelte';

	type NVMe = {
		percent_used: number;
		temperature_c: number;
		power_on_hours: number;
		tb_written: number;
	};
	type Drive = {
		drive: string;
		model: string;
		type: string;
		status: 'Healthy' | 'Warning' | 'Unhealthy' | 'Unknown';
		reason: string;
		nvme: NVMe | null;
		life_left_pct: number;
	};
	// drives on the same physical disk share its health, so they are shown together
	type Disk = Drive & { letters: string[] };
	type Category = { name: string; size_bytes: number; items: number };
	type Progress = {
		category: string;
		freed_bytes: number;
		total_bytes: number;
		files_deleted: number;
	};

	const STATUS_STYLE: Record<Drive['status'], string> = {
		Healthy: 'bg-green-500/15 text-green-700 dark:text-green-400',
		Warning: 'bg-amber-500/15 text-amber-700 dark:text-amber-400',
		Unhealthy: 'bg-red-500/15 text-red-700 dark:text-red-400',
		Unknown: 'bg-line text-muted'
	};

	// drives on one physical disk share its health, so they are shown together
	function groupByDisk(drives: Drive[]): Disk[] {
		const byDisk = new Map<string, Disk>();
		for (const drive of drives) {
			const key = `${drive.model}|${drive.type}`;
			const disk = byDisk.get(key);
			if (disk) disk.letters.push(drive.drive);
			else byDisk.set(key, { ...drive, letters: [drive.drive] });
		}
		return [...byDisk.values()];
	}

	const cachedHealth = recall<Drive[]>('drive-health');
	const cachedJunk = recall<{ categories: Category[]; total_bytes: number }>('junk');

	let disks = $state<Disk[]>(groupByDisk(cachedHealth ?? []));
	let healthState = $state<'loading' | 'ready' | 'failed'>(cachedHealth ? 'ready' : 'loading');

	let categories = $state<Category[]>((cachedJunk?.categories ?? []).filter((c) => c.size_bytes > 0));
	let totalBytes = $state(cachedJunk?.total_bytes ?? 0);
	let scanState = $state<'scanning' | 'ready' | 'failed'>(cachedJunk ? 'ready' : 'scanning');

	let showConfirm = $state(false);
	let isCleaning = $state(false);
	let isStopping = $state(false);
	let progress = $state<Progress | null>(null);
	let result = $state<string | undefined>(undefined);

	let percent = $derived(
		progress && progress.total_bytes > 0
			? Math.min(100, Math.round((progress.freed_bytes / progress.total_bytes) * 100))
			: 0
	);

	// a reading up to five minutes old is shown as is; Refresh reads again
	async function loadHealth(force = false) {
		if (force || healthState !== 'ready') healthState = 'loading';
		try {
			const drives = ((await load('drive-health', GetDriveHealth, 5 * 60_000, force)) || []) as Drive[];
			disks = groupByDisk(drives);
			healthState = 'ready';
		} catch (err) {
			console.error('Failed to read drive health:', err);
			healthState = 'failed';
		}
	}

	// temporary files change often, so a scan is reused for one minute
	async function scan(force = false) {
		if (force || scanState !== 'ready') scanState = 'scanning';
		try {
			const found = await load('junk', ScanJunk, 60_000, force);
			categories = (found.categories || []).filter((c) => c.size_bytes > 0);
			totalBytes = found.total_bytes;
			scanState = 'ready';
		} catch (err) {
			console.error('Failed to scan for junk files:', err);
			scanState = 'failed';
		}
	}

	async function clean() {
		isCleaning = true;
		isStopping = false;
		result = undefined;
		progress = { category: '', freed_bytes: 0, total_bytes: totalBytes, files_deleted: 0 };
		try {
			const summary = await CleanJunk();
			recordFreed(summary.freed_bytes);
			result =
				(summary.stopped ? 'Stopped. ' : '') +
				`Freed ${formatBytes(summary.freed_bytes)}` +
				(summary.skipped > 0 ? ` · ${summary.skipped.toLocaleString()} files in use were left` : '');
		} catch (err) {
			result = `Cleaning failed: ${err}`;
		} finally {
			showConfirm = false;
			isCleaning = false;
			isStopping = false;
			progress = null;
			scan(true);
		}
	}

	function stop() {
		isStopping = true;
		CancelCleanJunk();
	}

	function hours(h: number) {
		return h >= 10000 ? `${(h / 1000).toFixed(0)}k h` : `${h.toLocaleString()} h`;
	}

	const unsubscribe = EventsOn('cleaner-progress', (p: Progress) => {
		if (isCleaning) progress = p;
	});

	// load once when the screen opens
	onMount(() => {
		loadHealth();
		scan();
	});

	onDestroy(() => {
		if (isCleaning) CancelCleanJunk();
		unsubscribe();
	});
</script>

<Card>
	<div class="h-full flex flex-col gap-3 min-h-0">
		<!-- Drive health -->
		<section class="flex flex-col gap-2 shrink-0">
			<div class="h-6 flex items-center justify-between">
				<span class="font-bold text-ink">Drive health</span>
				<button
					class="text-[11px] text-muted font-medium cursor-pointer hover:text-ink disabled:opacity-50"
					disabled={healthState === 'loading'}
					onclick={() => loadHealth(true)}
				>
					Refresh
				</button>
			</div>

			{#if healthState === 'loading'}
				<div class="h-16 flex items-center"><BusyArt kind="read" size={40} label="Reading drive health..." /></div>
			{:else if healthState === 'failed' || disks.length === 0}
				<div class="h-16 flex items-center text-muted">Windows did not report drive health.</div>
			{:else}
				<div class="flex flex-col gap-2.5 max-h-36 overflow-y-auto pr-1">
					{#each disks as disk (disk.model + disk.type)}
						<div class="flex flex-col gap-1.5">
							<div class="flex items-center gap-2 min-w-0">
								<div class="flex-1 min-w-0 leading-tight">
									<div class="font-semibold text-ink truncate" title={disk.model}>
										{disk.model || 'Drive'}
									</div>
									<div class="text-[11px] text-muted truncate">
										{disk.letters.join(', ')}{disk.type ? ` · ${disk.type}` : ''}
									</div>
								</div>
								<span
									class="shrink-0 text-[10px] px-2 py-0.5 rounded-md font-semibold {STATUS_STYLE[
										disk.status
									]}"
								>
									{disk.status}
								</span>
							</div>

							{#if disk.nvme}
								<div class="grid grid-cols-4 gap-1.5">
									{#each [{ value: `${disk.life_left_pct}%`, label: 'Life left', title: `The drive reports ${disk.nvme.percent_used}% of its rated write endurance used` }, { value: `${disk.nvme.temperature_c} °C`, label: 'Temperature', title: 'Current temperature reported by the drive' }, { value: `${disk.nvme.tb_written} TB`, label: 'Written', title: 'Total data written over the drive’s life' }, { value: hours(disk.nvme.power_on_hours), label: 'Powered on', title: `${disk.nvme.power_on_hours.toLocaleString()} hours in total` }] as stat (stat.label)}
										<div class="rounded-lg bg-tile px-2 py-1.5 min-w-0" title={stat.title}>
											<div class="font-bold text-ink font-mono truncate">{stat.value}</div>
											<div class="text-[10px] text-muted truncate">{stat.label}</div>
										</div>
									{/each}
								</div>
							{:else}
								<p class="text-[11px] text-muted leading-snug">
									This drive does not share wear or temperature without administrator rights.
								</p>
							{/if}

							{#if disk.reason}
								<p class="text-[11px] font-medium text-amber-700 dark:text-amber-400 leading-snug">
									{disk.reason}
								</p>
							{/if}
						</div>
					{/each}
				</div>
			{/if}
		</section>

		<!-- Cleanup -->
		<section class="flex-1 min-h-0 flex flex-col gap-2 pt-3 border-t border-line">
			<div class="h-6 flex items-center justify-between gap-2 shrink-0">
				<span class="font-bold text-ink">Temporary files</span>
				<button
					class="text-[11px] text-muted font-medium cursor-pointer hover:text-ink disabled:opacity-50"
					disabled={scanState === 'scanning' || isCleaning}
					onclick={() => scan(true)}
				>
					Rescan
				</button>
			</div>

			<div class="flex-1 min-h-0 overflow-y-auto pr-1">
				{#if scanState === 'scanning'}
					<div class="h-full flex flex-col items-center justify-center gap-2 text-muted">
						<DriveScan size={44} active />
						Measuring temporary files and caches...
					</div>
				{:else if scanState === 'failed'}
					<div class="h-full flex items-center justify-center text-muted">Could not scan.</div>
				{:else if categories.length === 0}
					<div class="h-full flex items-center justify-center text-muted text-center px-4">
						Nothing to clean right now.
					</div>
				{:else}
					{#each categories as category (category.name)}
						<div class="py-1">
							<div class="flex justify-between gap-2">
								<span class="text-ink truncate">{category.name}</span>
								<span class="shrink-0 font-mono font-bold text-ink">
									{formatBytes(category.size_bytes)}
								</span>
							</div>
							<div class="mt-1 h-1 rounded-full bg-track">
								<div
									class="h-full rounded-full bg-accent"
									style="width: {Math.max(2, (category.size_bytes / totalBytes) * 100)}%"
								></div>
							</div>
						</div>
					{/each}
				{/if}
			</div>

			<div class="flex items-center justify-between gap-3 shrink-0">
				{#if result}
					<span
						class="text-[11px] font-semibold text-green-700 dark:text-green-400 truncate"
						title={result}
					>
						{result}
					</span>
				{:else}
					<span class="text-[11px] text-muted truncate" title="Files changed in the last hour, and files a program has open, are left alone">
						Files changed in the last hour are kept
					</span>
				{/if}
				<button
					class="btn btn-danger shrink-0"
					disabled={scanState !== 'ready' || totalBytes === 0 || isCleaning}
					onclick={() => (showConfirm = true)}
				>
					{isCleaning ? `Cleaning ${percent}%` : `Free ${formatBytes(totalBytes)}`}
				</button>
			</div>
		</section>
	</div>
</Card>

<Modal
	bind:showModal={showConfirm}
	width="max-w-sm"
	showClose={!isCleaning}
	title={isCleaning ? 'Cleaning temporary files' : `Free ${formatBytes(totalBytes)}?`}
	description={isCleaning
		? isStopping
			? 'Stopping after the current file...'
			: 'Keep the app open until this finishes.'
		: 'Temporary files and browser caches will be deleted. This cannot be undone.'}
>
	{#if isCleaning && progress}
		<div class="flex flex-col gap-1.5">
			<div class="flex items-baseline justify-between gap-3">
				<span class="font-semibold text-ink">
					{formatBytes(progress.freed_bytes)}
					<span class="font-normal text-muted">of {formatBytes(progress.total_bytes)} freed</span>
				</span>
				<span class="font-mono font-bold text-ink">{percent}%</span>
			</div>
			<div
				class="h-2 rounded-full bg-track overflow-hidden"
				role="progressbar"
				aria-label="Cleaning progress"
				aria-valuemin={0}
				aria-valuemax={100}
				aria-valuenow={percent}
			>
				<div
					class="h-full rounded-full bg-accent transition-[width] duration-200 ease-out"
					style="width: {percent}%"
				></div>
			</div>
			<div class="text-[11px] text-muted truncate">
				{progress.files_deleted.toLocaleString()} files removed{progress.category
					? ` · now ${progress.category.toLowerCase()}`
					: ''}
			</div>
		</div>
	{:else}
		<p class="text-muted leading-relaxed">
			Browsers rebuild their caches as you browse; you stay signed in and keep your history.
			Files a program has open, and anything changed in the last hour, are left alone.
		</p>
	{/if}

	{#snippet actions()}
		{#if isCleaning}
			<button class="btn btn-secondary" disabled={isStopping} onclick={stop}>
				{isStopping ? 'Stopping...' : 'Stop'}
			</button>
		{:else}
			<button class="btn btn-secondary" onclick={() => (showConfirm = false)}>Cancel</button>
			<button class="btn btn-danger" onclick={clean}>Delete</button>
		{/if}
	{/snippet}
</Modal>

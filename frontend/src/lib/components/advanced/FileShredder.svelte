<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import type { shred, storage } from '$lib/wailsjs/go/models';
	import ToolHeader from '$lib/components/ui/ToolHeader.svelte';
	import Segmented from '$lib/components/ui/Segmented.svelte';
	import Modal from '$lib/components/ui/Modal.svelte';
	import StrokeIcon from '$lib/components/ui/StrokeIcon.svelte';
	import { formatBytes } from '$lib';
	import {
		CancelShred,
		CancelWipe,
		DescribeShredDrives,
		GetDrives,
		MeasureFolder,
		PickFiles,
		PickFolder,
		ShredFiles,
		WipeFreeSpace
	} from '$lib/wailsjs/go/main/App';
	import { EventsOn } from '$lib/wailsjs/runtime/runtime';

	type Chosen = { path: string; size: number };
	// sent as events, so written by hand
	type Progress = {
		file: string;
		file_index: number;
		files: number;
		pass: number;
		passes: number;
		done_bytes: number;
		total_bytes: number;
	};
	type WipeProgress = { drive: string; pass: number };

	let chosen = $state<Chosen[]>([]);
	let passes = $state(1);
	let drives = $state<shred.Drive[]>([]);
	let confirm = $state(false);
	let running = $state(false);
	let progress = $state<Progress | null>(null);
	let result = $state<shred.Result | null>(null);
	let error = $state('');

	// free space wiping
	let allDrives = $state<storage.DriveInfo[]>([]);
	let wipeDrive = $state('');
	let confirmWipe = $state(false);
	let wiping = $state(false);
	let wipePass = $state(0);
	let wipeMessage = $state('');

	const METHODS = [
		{ id: 1, label: '1 pass' },
		{ id: 3, label: '3 passes' },
		{ id: 7, label: '7 passes' }
	];
	const WIPE_PASSES = ['Writing zeros', 'Writing ones', 'Writing random data'];

	let totalBytes = $derived(chosen.reduce((sum, c) => sum + Math.max(0, c.size), 0));
	let pct = $derived(
		progress && progress.total_bytes > 0
			? Math.min(100, Math.round((progress.done_bytes / progress.total_bytes) * 100))
			: 0
	);

	const name = (path: string) => path.split(/[\\/]/).pop() || path;

	async function add(kind: 'files' | 'folder') {
		error = '';
		try {
			const paths =
				kind === 'files'
					? await PickFiles('Choose files to shred')
					: [await PickFolder('Choose a folder to shred, with everything in it')].filter(Boolean);
			const fresh = (paths || []).filter((p) => !chosen.some((c) => c.path === p));
			if (!fresh.length) return;
			result = null;
			chosen = [...chosen, ...fresh.map((path) => ({ path, size: -1 }))];
			for (const p of fresh) {
				MeasureFolder(p).then((size) => {
					chosen = chosen.map((c) => (c.path === p ? { ...c, size } : c));
				});
			}
			drives = (await DescribeShredDrives(chosen.map((c) => c.path))) || [];
		} catch (err) {
			error = String(err);
		}
	}

	function remove(path: string) {
		chosen = chosen.filter((c) => c.path !== path);
		DescribeShredDrives(chosen.map((c) => c.path)).then((d) => (drives = d || []));
	}

	async function start() {
		confirm = false;
		running = true;
		progress = null;
		error = '';
		try {
			result = await ShredFiles(
				chosen.map((c) => c.path),
				passes
			);
			chosen = [];
			drives = [];
		} catch (err) {
			error = String(err);
		} finally {
			running = false;
		}
	}

	// what overwriting can promise on each drive, in plain words
	function driveNote(d: shred.Drive): { tone: string; text: string } {
		const ssd = d.kind.includes('SSD');
		if (ssd && d.bitlocker === 'on')
			return {
				tone: 'ok',
				text: `${d.drive} is an SSD protected by BitLocker. Recovery tools cannot bring the files back, and any old copies the SSD keeps internally stay encrypted.`
			};
		if (ssd)
			return {
				tone: 'warn',
				text: `${d.drive} is an SSD. SSDs choose where data goes and keep spare space Windows cannot reach, so pieces of the old data can survive an overwrite until the drive reuses that space. Recovery tools will not find the file, but a lab could. For files that must never be recovered, turn on BitLocker or Device encryption.`
			};
		if (d.kind === 'HDD')
			return {
				tone: 'ok',
				text: `${d.drive} is a hard drive. One pass replaces the old data for good; more passes only take longer.`
			};
		return {
			tone: 'info',
			text: `Windows did not say what kind of drive ${d.drive} is. On a hard drive one pass is enough; on an SSD some old data can survive any overwrite unless the drive is encrypted.`
		};
	}

	async function wipe() {
		confirmWipe = false;
		wiping = true;
		wipePass = 0;
		wipeMessage = '';
		try {
			await WipeFreeSpace(wipeDrive);
			wipeMessage = `Free space on ${wipeDrive} has been wiped.`;
		} catch (err) {
			wipeMessage = String(err);
		} finally {
			wiping = false;
			loadDrives();
		}
	}

	function loadDrives() {
		GetDrives()
			.then((list) => {
				allDrives = (list || []) as storage.DriveInfo[];
				if (!wipeDrive && allDrives.length) wipeDrive = allDrives[0].path.slice(0, 2);
			})
			.catch(() => {});
	}

	let offProgress: () => void = () => {};
	let offWipe: () => void = () => {};
	onMount(() => {
		loadDrives();
		offProgress = EventsOn('shred-progress', (p: Progress) => (progress = p));
		offWipe = EventsOn('wipe-progress', (p: WipeProgress) => (wipePass = p.pass));
	});
	onDestroy(() => {
		offProgress();
		offWipe();
	});
</script>

<div class="flex flex-col h-full min-h-0 gap-3 p-1">
	<ToolHeader
		title="File shredder"
		subtitle="Overwrite files before deleting them, so recovery tools cannot bring them back"
	>
		{#snippet actions()}
			<div class="flex items-center gap-2 shrink-0">
				<button class="btn btn-secondary" onclick={() => add('folder')} disabled={running}>
					Add folder
				</button>
				<button class="btn btn-primary" onclick={() => add('files')} disabled={running}>
					Add files
				</button>
			</div>
		{/snippet}
	</ToolHeader>

	<div class="flex-1 min-h-0 overflow-y-auto overscroll-contain pr-1.5 flex flex-col gap-4">
		{#if running}
			<div class="flex flex-col gap-2 rounded-xl border border-line p-4" role="status">
				<div class="flex items-center justify-between gap-3">
					<span class="font-semibold truncate" title={progress?.file}>
						{progress ? `Shredding ${name(progress.file)}` : 'Starting...'}
					</span>
					<button class="btn btn-secondary shrink-0" onclick={CancelShred}>Stop</button>
				</div>
				<span class="block h-1.5 rounded-full bg-track">
					<span
						class="block h-1.5 rounded-full bg-accent transition-[width] duration-300"
						style="width: {pct}%"
					></span>
				</span>
				{#if progress}
					<span class="text-muted tabular-nums">
						File {progress.file_index} of {progress.files} · pass {progress.pass} of {progress.passes}
						· {pct}%
					</span>
				{/if}
			</div>
		{:else if result}
			<div class="flex flex-col gap-2 rounded-xl border border-line p-4">
				<span class="font-semibold text-[13px]">
					{result.shredded
						? `Shredded ${result.shredded} ${result.shredded === 1 ? 'file' : 'files'} (${formatBytes(result.bytes)})`
						: 'Nothing was shredded'}{result.stopped ? ' before it was stopped' : ''}
				</span>
				{#each result.skipped as skip (skip.path + skip.reason)}
					<span class="text-warn leading-snug"
						><span class="font-semibold">{name(skip.path) || 'Skipped'}:</span> {skip.reason}</span
					>
				{/each}
				{#each result.notes as note (note)}
					<span class="text-muted leading-snug">{note}</span>
				{/each}
			</div>
		{/if}

		{#if !running}
			{#if chosen.length}
				<div class="grid grid-cols-[minmax(0,1fr)_260px] gap-4">
					<div class="flex flex-col min-w-0">
						{#each chosen as item (item.path)}
							<div class="flex items-center gap-2 py-1.5 border-b border-line">
								<span class="flex flex-col flex-1 min-w-0" title={item.path}>
									<span class="font-semibold truncate">{name(item.path)}</span>
									<span class="text-muted text-[11px] truncate">{item.path}</span>
								</span>
								<span class="shrink-0 text-muted tabular-nums">
									{item.size < 0 ? '...' : formatBytes(item.size)}
								</span>
								<button
									class="w-6 h-6 rounded-md grid place-items-center text-muted hover:text-ink hover:bg-line cursor-pointer"
									aria-label="Remove {name(item.path)} from the list"
									onclick={() => remove(item.path)}
								>
									<StrokeIcon d="M6 6l12 12M18 6L6 18" size={14} />
								</button>
							</div>
						{/each}
					</div>

					<div class="flex flex-col gap-3">
						<div class="flex flex-col gap-1.5">
							<span class="font-semibold">Overwrite with</span>
							<Segmented
								options={METHODS}
								value={passes}
								onchange={(v) => (passes = v)}
								label="Number of passes"
								size="sm"
							/>
							<span class="text-muted leading-snug">
								One pass of random data is enough for any drive. More passes only matter to people
								who want them on old hard drives, and take longer.
							</span>
						</div>
						<button class="btn btn-danger h-9" onclick={() => (confirm = true)}>
							Shred {chosen.length}
							{chosen.length === 1 ? 'item' : 'items'}{totalBytes
								? ` · ${formatBytes(totalBytes)}`
								: ''}
						</button>
					</div>
				</div>

				{#each drives as d (d.drive)}
					{@const note = driveNote(d)}
					<p
						class="leading-relaxed rounded-xl px-3 py-2 {note.tone === 'warn'
							? 'bg-warn/10'
							: note.tone === 'ok'
								? 'bg-ok/10'
								: 'bg-info/10'}"
					>
						{note.text}
					</p>
				{/each}
				<p class="text-muted leading-relaxed">
					Shredding removes this copy only. Copies in OneDrive or other cloud folders, backups,
					restore points and email attachments are separate and stay where they are.
				</p>
			{:else if !result}
				<div class="flex flex-col items-center justify-center gap-2 text-center py-6">
					<div class="text-sm font-bold">Delete it so it stays deleted</div>
					<p class="text-muted max-w-md leading-relaxed">
						A normal delete only forgets where a file is; recovery tools can often bring it back.
						Shredding writes over its contents first. Add the files or folders you want gone for
						good. Nothing happens until you confirm.
					</p>
				</div>
			{/if}

			{#if error}
				<p class="text-fail" role="alert">{error}</p>
			{/if}

			<!-- files deleted earlier still sit in the free space -->
			<div class="flex flex-col gap-2 pt-3 border-t border-line">
				<div class="flex items-center justify-between gap-4">
					<div class="min-w-0">
						<div class="font-semibold">Wipe free space</div>
						<div class="text-muted leading-snug">
							For files you deleted before: overwrites the drive's empty space three times with
							Windows' own cipher tool. Your current files are not touched.
						</div>
					</div>
					<div class="flex items-center gap-2 shrink-0">
						{#if allDrives.length > 1}
							<Segmented
								options={allDrives.map((d) => ({
									id: d.path.slice(0, 2),
									label: d.path.slice(0, 2)
								}))}
								value={wipeDrive}
								onchange={(v) => (wipeDrive = v)}
								label="Drive"
								size="sm"
								disabled={wiping}
							/>
						{/if}
						{#if wiping}
							<button class="btn btn-secondary" onclick={CancelWipe}>Stop</button>
						{:else}
							<button
								class="btn btn-secondary"
								disabled={!wipeDrive}
								onclick={() => (confirmWipe = true)}
							>
								Wipe {wipeDrive}
							</button>
						{/if}
					</div>
				</div>
				{#if wiping}
					<div class="flex flex-col gap-1.5" role="status">
						<span class="block h-1.5 rounded-full bg-track overflow-hidden">
							<span
								class="block h-1.5 rounded-full bg-accent"
								style="width: {Math.max(4, (wipePass / 3) * 100)}%"
							></span>
						</span>
						<span class="text-muted">
							{wipePass ? `Pass ${wipePass} of 3: ${WIPE_PASSES[wipePass - 1]}` : 'Starting...'}
							· {wipeDrive} fills up while this runs and is freed again at the end
						</span>
					</div>
				{:else if wipeMessage}
					<p class="text-muted">{wipeMessage}</p>
				{/if}
			</div>
		{/if}
	</div>
</div>

<Modal
	bind:showModal={confirm}
	width="max-w-sm"
	title="Shred {chosen.length} {chosen.length === 1 ? 'item' : 'items'} for good?"
	description="They are overwritten and deleted, and do not go to the Recycle Bin. This cannot be undone."
>
	{#snippet actions()}
		<button class="btn btn-secondary" onclick={() => (confirm = false)}>Cancel</button>
		<button class="btn btn-danger" onclick={start}>Shred</button>
	{/snippet}
</Modal>

<Modal
	bind:showModal={confirmWipe}
	width="max-w-sm"
	title="Wipe free space on {wipeDrive}?"
	description="This writes over all of the drive's empty space three times, which can take an hour or more on a large drive. The drive fills up while it runs; low-space alerts for it pause until it is done. Your files are not touched."
>
	{#snippet actions()}
		<button class="btn btn-secondary" onclick={() => (confirmWipe = false)}>Cancel</button>
		<button class="btn btn-primary" onclick={wipe}>Wipe {wipeDrive}</button>
	{/snippet}
</Modal>

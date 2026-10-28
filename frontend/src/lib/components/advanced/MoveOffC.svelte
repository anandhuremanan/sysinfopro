<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import type { main, relocate, storage } from '$lib/wailsjs/go/models';
	import ToolHeader from '$lib/components/ui/ToolHeader.svelte';
	import Modal from '$lib/components/ui/Modal.svelte';
	import BusyArt from '$lib/components/ui/BusyArt.svelte';
	import Segmented from '$lib/components/ui/Segmented.svelte';
	import { formatBytes } from '$lib';
	import {
		CancelMove,
		GetDrives,
		ListMovable,
		MeasureFolder,
		MoveDestinations,
		MoveItem,
		MoveItemBack,
		PickMoveFolder
	} from '$lib/wailsjs/go/main/App';
	import { EventsOn } from '$lib/wailsjs/runtime/runtime';

	type Item = relocate.Item;
	// sent as events, so written by hand
	type Progress = { id: string; phase: string; done: number; total: number };

	const GROUPS = [
		{ id: 'dev', title: 'Developer caches' },
		{ id: 'wsl', title: 'WSL and Docker' },
		{ id: 'browser', title: 'Browsers' }
	];
	const PHASE: Record<string, string> = {
		copying: 'Copying',
		switching: 'Switching over',
		cleaning: 'Removing the old copy'
	};

	let items = $state<Item[]>([]);
	let loading = $state(true);
	let destinations = $state<main.MoveDestination[]>([]);
	let base = $state('');
	let systemDrive = $state<storage.DriveInfo | null>(null);
	let busyId = $state('');
	let progress = $state<Progress | null>(null);
	let message = $state<{ text: string; error: boolean } | null>(null);
	// the item waiting for confirmation, and whether it is a move back
	let pending = $state<{ item: Item; back: boolean } | null>(null);
	let showConfirm = $state(false);
	let openGuide = $state('');

	let movableBytes = $derived(
		items
			.filter((i) => i.on_system_drive && !i.moved && !i.guide && i.size_bytes > 0)
			.reduce((sum, i) => sum + i.size_bytes, 0)
	);
	let pct = $derived(
		progress && progress.total > 0
			? Math.min(100, Math.round((progress.done / progress.total) * 100))
			: 0
	);

	const driveOf = (path: string) => path.slice(0, 2).toUpperCase();
	const canMoveIn = (group: string) =>
		items.some((i) => i.group === group && i.on_system_drive && !i.moved && !i.guide);

	async function load() {
		loading = true;
		try {
			const [list, dests, drives] = await Promise.all([
				ListMovable(),
				MoveDestinations(),
				GetDrives()
			]);
			items = (list || []) as Item[];
			destinations = dests || [];
			systemDrive = ((drives || []) as storage.DriveInfo[]).find((d) => d.is_system) ?? null;
			if (!base && destinations.length) base = destinations[0].folder;
		} catch (err) {
			message = { text: String(err), error: true };
		} finally {
			loading = false;
		}
		// sizes after the list, a few at a time, since big caches take a while
		const queue = items.filter((i) => i.size_bytes < 0 && !i.guide);
		const worker = async () => {
			for (let next = queue.shift(); next; next = queue.shift()) {
				const size = await MeasureFolder(next.path).catch(() => -1);
				const id = next.id;
				items = items.map((i) => (i.id === id ? ({ ...i, size_bytes: size } as Item) : i));
			}
		};
		await Promise.all([worker(), worker(), worker()]);
	}

	async function chooseFolder() {
		try {
			const picked = await PickMoveFolder();
			if (picked) base = picked;
		} catch (err) {
			message = { text: String(err), error: true };
		}
	}

	function ask(item: Item, back = false) {
		pending = { item, back };
		showConfirm = true;
	}

	async function run() {
		if (!pending) return;
		const { item, back } = pending;
		showConfirm = false;
		busyId = item.id;
		progress = null;
		message = null;
		try {
			const result = back ? await MoveItemBack(item.id) : await MoveItem(item.id, base);
			message = {
				text: `${item.name} ${back ? 'is back in' : 'now lives in'} ${result.to}.${result.note ? ' ' + result.note : ''}`,
				error: false
			};
		} catch (err) {
			message = { text: String(err), error: true };
		} finally {
			busyId = '';
			pending = null;
			await load();
		}
	}

	function confirmText(item: Item, back: boolean): string {
		const size = item.size_bytes > 0 ? formatBytes(item.size_bytes) : 'it';
		const sys = systemDrive ? systemDrive.path.slice(0, 2) : 'the system drive';
		if (back)
			return `This copies ${size} back to ${item.moved_from} and points everything there again.`;
		const note = item.note ? ` ${item.note}` : '';
		switch (item.method) {
			case 'setting':
				return `This copies ${size} to ${base}, points ${item.name.split(' ')[0]} at the new place using its own setting, then removes the old copy from ${sys}. You can move it back any time.${note}`;
			case 'junction': {
				// the program that uses it: "Firefox cache" is used by Firefox
				const app = item.name === 'Microsoft Edge' ? 'Edge' : item.name.split(' ')[0];
				return `This copies ${size} to ${base} and leaves a folder link in the old place, so ${app} still finds everything. Keep ${app} closed until it is done. You can move it back any time.${note}`;
			}
			default:
				return `WSL moves the disk of ${item.name} (${size}) to ${base} itself. ${item.note}`;
		}
	}

	let off: () => void = () => {};
	onMount(() => {
		load();
		off = EventsOn('move-progress', (p: Progress) => (progress = p));
	});
	onDestroy(() => off());
</script>

<div class="flex flex-col h-full min-h-0 gap-3 p-1">
	<ToolHeader title="Move off C:" subtitle="Move big caches and app data to another drive">
		{#snippet actions()}
			{#if destinations.length}
				<div class="flex items-center gap-2 shrink-0">
					<span class="text-muted">Move to</span>
					<Segmented
						options={destinations.map((d) => ({
							id: d.folder,
							label: `${d.drive} · ${formatBytes(d.free_bytes)} free`
						}))}
						value={destinations.some((d) => d.folder === base) ? base : ''}
						onchange={(v) => (base = v)}
						label="Drive to move to"
						size="sm"
						disabled={!!busyId}
					/>
					<button class="btn btn-secondary" onclick={chooseFolder} disabled={!!busyId}>
						Other folder...
					</button>
				</div>
			{/if}
		{/snippet}
	</ToolHeader>

	{#if message}
		<p class="leading-snug shrink-0 {message.error ? 'text-fail' : 'text-ok'}" role="status">
			{message.text}
		</p>
	{/if}

	{#if loading && !items.length}
		<div class="flex-1 flex items-center justify-center">
			<BusyArt kind="search" size={64} label="Asking each tool where it keeps its files..." />
		</div>
	{:else if !destinations.length}
		<div class="flex-1 flex flex-col items-center justify-center gap-2 text-center">
			<div class="text-sm font-bold">No other drive to move to</div>
			<p class="text-muted max-w-md leading-relaxed">
				Moving needs a second drive inside this PC. USB drives and memory cards are not supported,
				because everything moved there stops working whenever they are unplugged.
			</p>
		</div>
	{:else}
		{#if systemDrive}
			<div class="shrink-0 text-muted">
				{systemDrive.path.slice(0, 2)} has {formatBytes(systemDrive.free_bytes)} free of
				{formatBytes(systemDrive.total_bytes)}{movableBytes
					? ` · ${formatBytes(movableBytes)} below can move`
					: ''} · moves go into <span class="font-mono text-ink">{base}</span>
			</div>
		{/if}

		<div
			class="flex-1 min-h-0 overflow-y-auto overscroll-contain pr-1.5 pb-4 flex flex-col gap-4 [mask-image:linear-gradient(to_bottom,#000_calc(100%-20px),transparent)]"
		>
			<!-- groups with something left to move come first -->
			{#each [...GROUPS].sort((a, b) => Number(canMoveIn(b.id)) - Number(canMoveIn(a.id))) as group (group.id)}
				<!-- what can still move comes first; what is already elsewhere after it -->
				{@const rows = items
					.filter((i) => i.group === group.id)
					.sort(
						(a, b) => Number(b.on_system_drive && !b.moved) - Number(a.on_system_drive && !a.moved)
					)}
				{#if rows.length}
					<section class="flex flex-col">
						<h3 class="text-[11px] font-bold uppercase tracking-wider text-muted mb-1">
							{group.title}
						</h3>
						{#each rows as item (item.id)}
							<div class="flex flex-col border-b border-line py-2 gap-1">
								<div class="flex items-center gap-3">
									<span class="flex flex-col min-w-0 flex-1">
										<span class="font-semibold truncate">{item.name}</span>
										<span class="text-muted truncate text-[11px] font-mono" title={item.path}
											>{item.path}</span
										>
									</span>
									<span class="shrink-0 w-20 text-right tabular-nums text-muted">
										{item.size_bytes < 0 ? '...' : formatBytes(item.size_bytes)}
									</span>
									<span class="shrink-0 w-[170px] flex justify-end">
										{#if busyId === item.id}
											{#if progress?.phase === 'copying'}
												<button class="btn btn-secondary" onclick={CancelMove}>Stop</button>
											{:else}
												<span class="text-muted">Working...</span>
											{/if}
										{:else if item.guide}
											<button
												class="btn btn-secondary"
												aria-expanded={openGuide === item.id}
												onclick={() => (openGuide = openGuide === item.id ? '' : item.id)}
											>
												How to move it
											</button>
										{:else if item.moved}
											<span class="flex items-center gap-2">
												<span class="pill-ok px-2 py-0.5 rounded-md text-[11px] font-semibold"
													>On {driveOf(item.path)}</span
												>
												<button
													class="btn btn-secondary"
													disabled={!!busyId || !!item.blocker}
													title={item.blocker}
													onclick={() => ask(item, true)}>Move back</button
												>
											</span>
										{:else if !item.on_system_drive}
											<span
												class="pill-info px-2 py-0.5 rounded-md text-[11px] font-semibold"
												title={item.blocker || ''}>Already on {driveOf(item.path)}</span
											>
										{:else if item.blocker}
											<span
												class="pill-warn px-2 py-0.5 rounded-md text-[11px] font-semibold truncate max-w-[170px]"
												title={item.blocker}>{item.blocker}</span
											>
										{:else}
											<button
												class="btn btn-primary"
												disabled={!!busyId || !base}
												onclick={() => ask(item)}>Move</button
											>
										{/if}
									</span>
								</div>
								{#if busyId === item.id}
									<div class="flex flex-col gap-1" role="status">
										<span class="block h-1.5 rounded-full bg-track">
											<span
												class="block h-1.5 rounded-full bg-accent transition-[width] duration-300"
												style="width: {pct}%"
											></span>
										</span>
										<span class="text-muted tabular-nums"
											>{progress
												? `${PHASE[progress.phase] ?? progress.phase} · ${pct}%`
												: 'Starting...'}</span
										>
									</div>
								{/if}
								{#if openGuide === item.id}
									<p class="text-muted leading-relaxed">{item.guide}</p>
								{/if}
							</div>
						{/each}
					</section>
				{/if}
			{/each}

			<p class="text-muted leading-relaxed">
				Programs installed in Program Files are not moved: that needs administrator rights and
				breaks their updates. To move one, uninstall it and install it again on the other drive.
			</p>
		</div>
	{/if}
</div>

<Modal
	bind:showModal={showConfirm}
	width="max-w-sm"
	title={pending
		? pending.back
			? `Move ${pending.item.name} back?`
			: `Move ${pending.item.name}?`
		: ''}
	description={pending ? confirmText(pending.item, pending.back) : ''}
>
	{#snippet actions()}
		<button class="btn btn-secondary" onclick={() => (showConfirm = false)}>Cancel</button>
		<button class="btn btn-primary" onclick={run}>{pending?.back ? 'Move back' : 'Move'}</button>
	{/snippet}
</Modal>

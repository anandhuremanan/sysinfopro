<script lang="ts">
	import { onMount } from 'svelte';
	import { formatBytes } from '$lib';
	import {
		GetDockerUsage,
		ListVirtualDisks,
		PruneDocker,
		RevealInExplorer
	} from '$lib/wailsjs/go/main/App';
	import { ClipboardSetText } from '$lib/wailsjs/runtime/runtime';
	import { load, recall } from '$lib/cache';
	import Modal from '../ui/Modal.svelte';
	import StrokeIcon from '../ui/StrokeIcon.svelte';

	type Disk = {
		name: string;
		kind: 'wsl' | 'docker';
		path: string;
		default: boolean;
		version: number;
		size_bytes: number;
		on_disk_bytes: number;
		sparse: boolean;
		missing: boolean;
	};
	type DockerUsage = {
		installed: boolean;
		running: boolean;
		rows: { type: string; count: string; size: string; reclaimable: string }[];
	};

	const REVEAL_ICON =
		'M14 4h6v6M20 4l-9 9M18 14v4a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4';

	let disks = $state<Disk[]>(recall('vdisks') ?? []);
	let docker = $state<DockerUsage | null>(recall('docker') ?? null);
	let isLoading = $state(!recall('vdisks'));
	let selectedPath = $state('');
	let copied = $state('');
	let showPrune = $state(false);
	let isPruning = $state(false);
	let statusMessage = $state<string | undefined>(undefined);

	let selected = $derived(disks.find((d) => d.path === selectedPath) ?? disks[0]);

	// a result up to a minute old is shown as is; Refresh and cleanup look again
	async function refresh(force = false) {
		isLoading = force || disks.length === 0;
		try {
			disks = ((await load('vdisks', ListVirtualDisks, 60_000, force)) || []) as Disk[];
		} catch (err) {
			console.error('Failed to list virtual disks:', err);
		} finally {
			isLoading = false;
		}
		// Docker can take a few seconds to answer; the disks show first
		docker = (await load('docker', GetDockerUsage, 60_000, force).catch(() => null)) as DockerUsage | null;
	}

	function diskpartScript(path: string) {
		return [
			`select vdisk file="${path}"`,
			'attach vdisk readonly',
			'compact vdisk',
			'detach vdisk',
			'exit'
		].join('\n');
	}

	async function copy(id: string, text: string) {
		await ClipboardSetText(text);
		copied = id;
		setTimeout(() => {
			if (copied === id) copied = '';
		}, 2000);
	}

	async function prune() {
		isPruning = true;
		try {
			const reclaimed = await PruneDocker();
			statusMessage = `Docker freed ${reclaimed}. Shrink its disk to give the space back to Windows.`;
			await refresh(true);
		} catch (err) {
			statusMessage = `Docker cleanup failed: ${err}`;
		} finally {
			isPruning = false;
			showPrune = false;
		}
	}

	// load once when the screen opens
	onMount(() => {
		refresh();
	});
</script>

{#snippet command(id: string, text: string)}
	<div class="flex items-start gap-2 rounded-lg bg-ground border border-line px-2.5 py-1.5">
		<pre class="flex-1 min-w-0 font-mono text-[11px] text-ink whitespace-pre-wrap break-all">{text}</pre>
		<button
			class="shrink-0 text-[11px] font-semibold text-accent cursor-pointer hover:underline"
			onclick={() => copy(id, text)}
		>
			{copied === id ? 'Copied' : 'Copy'}
		</button>
	</div>
{/snippet}

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<div class="flex justify-between items-center gap-3 pb-2 border-b border-line shrink-0">
		<div class="min-w-0">
			<div class="font-bold text-base text-ink">WSL & Docker</div>
			<div class="text-xs text-muted truncate">
				Their virtual disks grow as you use them and never give space back on their own
			</div>
		</div>
		<button
			class="text-[11px] text-muted font-medium cursor-pointer hover:text-ink shrink-0"
			disabled={isLoading}
			onclick={() => refresh(true)}
		>
			Refresh
		</button>
	</div>

	{#if statusMessage}
		<div class="text-xs font-semibold text-green-700 dark:text-green-400 px-1 shrink-0 truncate" title={statusMessage}>
			{statusMessage}
		</div>
	{/if}

	{#if isLoading}
		<div class="flex-1 flex items-center justify-center text-muted animate-pulse">
			Looking for WSL and Docker disks...
		</div>
	{:else if disks.length === 0 && !docker?.installed}
		<div class="flex-1 flex items-center justify-center text-muted text-center px-8">
			No WSL distributions or Docker disks on this PC.
		</div>
	{:else}
		<div class="flex-1 min-h-0 grid grid-cols-2 gap-4">
			<!-- disks, then Docker's own usage -->
			<div class="min-h-0 flex flex-col gap-3 overflow-y-auto overscroll-contain pr-1.5">
				<div class="flex flex-col">
					{#each disks as disk (disk.path)}
						<button
							class="flex items-center gap-2.5 h-11 px-2 rounded-lg text-left cursor-pointer transition-colors {selected?.path ===
							disk.path
								? 'bg-ground'
								: 'hover:bg-ground/60'}"
							onclick={() => (selectedPath = disk.path)}
						>
							<span class="flex-1 min-w-0 flex flex-col leading-tight">
								<span class="flex items-center gap-1.5 min-w-0">
									<span class="font-semibold text-ink truncate">{disk.name}</span>
									{#if disk.default}
										<span class="shrink-0 text-[10px] px-1.5 rounded bg-line text-muted font-semibold">Default</span>
									{/if}
								</span>
								<span class="text-[11px] text-muted truncate">
									{disk.kind === 'docker' ? 'Docker' : `WSL ${disk.version}`}
									{disk.sparse ? ' · shrinks by itself' : ''}
								</span>
							</span>
							{#if disk.missing}
								<span class="shrink-0 text-[10px] px-1.5 py-0.5 rounded-md bg-amber-500/15 text-amber-700 dark:text-amber-400 font-semibold">
									Disk missing
								</span>
							{:else if disk.version !== 1}
								<span class="shrink-0 font-mono font-bold text-ink">{formatBytes(disk.on_disk_bytes)}</span>
							{/if}
						</button>
					{:else}
						<div class="text-muted py-2">No WSL distributions on this PC.</div>
					{/each}
				</div>

				{#if docker?.installed}
					<div class="flex flex-col gap-1.5 pt-3 border-t border-line">
						<div class="flex items-center justify-between gap-2">
							<span class="text-[10px] uppercase tracking-wider font-semibold text-muted">Docker data</span>
							{#if docker.running}
								<button class="btn btn-danger h-6 px-2" disabled={isPruning} onclick={() => (showPrune = true)}>
									Remove unused
								</button>
							{/if}
						</div>
						{#if !docker.running}
							<p class="text-muted leading-snug">Start Docker Desktop to see how much it could give back.</p>
						{:else}
							{#each docker.rows as row (row.type)}
								<div class="flex justify-between gap-3">
									<span class="text-ink">{row.type} <span class="text-muted">({row.count})</span></span>
									<span class="shrink-0">
										<span class="font-mono font-semibold text-ink">{row.reclaimable.replace(/\s*\(.*\)/, '')}</span>
										<span class="text-muted">of {row.size} unused</span>
									</span>
								</div>
							{/each}
						{/if}
					</div>
				{/if}
			</div>

			<!-- what to do about the selected disk -->
			<div class="min-h-0 overflow-y-auto overscroll-contain pr-1.5 flex flex-col gap-2.5">
				{#if selected}
					<div class="flex items-start justify-between gap-2">
						<div class="min-w-0">
							<div class="font-bold text-ink truncate">{selected.name}</div>
							<div class="text-[11px] text-muted truncate" title={selected.path}>{selected.path}</div>
						</div>
						{#if !selected.missing}
							<button
								aria-label="Show in file manager"
								title="Show in file manager"
								class="shrink-0 w-6 h-6 rounded-md flex items-center justify-center text-muted cursor-pointer hover:text-ink hover:bg-ground"
								onclick={() => RevealInExplorer(selected.path)}
							>
								<StrokeIcon d={REVEAL_ICON} size={13} />
							</button>
						{/if}
					</div>

					{#if selected.missing}
						<p class="leading-snug text-amber-700 dark:text-amber-400 font-medium">
							WSL still lists this distribution, but its disk file is gone.
						</p>
						<p class="text-muted leading-snug">
							If it lives on a drive that is not connected right now, connect that drive. If you
							deleted it, remove the leftover entry from WSL:
						</p>
						{@render command('unregister', `wsl --unregister ${selected.name}`)}
					{:else if selected.version === 1}
						<p class="text-muted leading-snug">
							WSL 1 keeps its files in an ordinary folder, so deleting files inside the distribution
							frees space right away. Use Disk usage to see what is large inside it.
						</p>
					{:else}
						<div class="flex items-baseline gap-2">
							<span class="text-xl font-bold text-ink font-mono">{formatBytes(selected.on_disk_bytes)}</span>
							<span class="text-muted">
								on disk{selected.sparse && selected.size_bytes > selected.on_disk_bytes
									? ` of a ${formatBytes(selected.size_bytes)} file`
									: ''}
							</span>
						</div>
						{#if selected.sparse}
							<p class="text-muted leading-snug">
								This disk is sparse: space freed inside it goes back to Windows on its own.
							</p>
						{:else}
							<p class="text-muted leading-snug">
								Deleting files inside {selected.kind === 'docker' ? 'Docker' : 'the distribution'} does
								not make this file smaller. To give the space back to Windows:
							</p>
							<ol class="flex flex-col gap-2 list-decimal pl-4 text-ink leading-snug">
								<li>
									{selected.kind === 'docker'
										? 'Remove unused Docker data (the button on the left).'
										: 'Delete what you no longer need inside the distribution, then run:'}
								</li>
								{#if selected.kind === 'wsl'}
									{@render command('trim', 'sudo fstrim -av')}
								{/if}
								<li>Close everything that uses WSL or Docker, then run:</li>
								{@render command('shutdown', 'wsl --shutdown')}
								<li>
									In a terminal opened as administrator, run <span class="font-mono">diskpart</span>
									and paste:
								</li>
								{@render command('diskpart', diskpartScript(selected.path))}
							</ol>
						{/if}
					{/if}
				{:else}
					<div class="h-full flex items-center justify-center text-muted text-center px-4">
						Docker Desktop keeps its data in WSL. Start it once to create its disk.
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>

<Modal
	bind:showModal={showPrune}
	width="max-w-sm"
	showClose={!isPruning}
	title="Remove unused Docker data?"
	description="Stopped containers, unused networks, dangling images and build cache will be deleted."
>
	<p class="text-muted leading-relaxed">
		Volumes are kept, so databases and other data stored in them are safe. Images you still use
		stay; anything removed is downloaded or rebuilt again when needed.
	</p>
	{#snippet actions()}
		{#if isPruning}
			<span class="text-muted animate-pulse self-center">Docker is cleaning up...</span>
		{:else}
			<button class="btn btn-secondary" onclick={() => (showPrune = false)}>Cancel</button>
			<button class="btn btn-danger" onclick={prune}>Remove</button>
		{/if}
	{/snippet}
</Modal>

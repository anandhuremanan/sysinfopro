<script lang="ts">
	import { onMount } from 'svelte';
	import { formatBytes } from '$lib';
	import { GetDrives, GetDuplicateFolders, PickFolder } from '$lib/wailsjs/go/main/App';
	import type { dupes, storage } from '$lib/wailsjs/go/models';

	// Where to search: the user's own folders, whole drives, or any folder.
	// "cards" shows drives as a row of cards under their own heading; "inline" puts
	// "All of D:" buttons in the same row as the folders.
	let {
		onPick,
		chooseTitle,
		layout = 'cards',
		heading = 'Your folders',
		folders = (p) => p,
		onError
	}: {
		onPick: (root: string) => void;
		chooseTitle: string; // the folder dialog's title
		layout?: 'cards' | 'inline';
		heading?: string;
		// narrow the suggested folders, e.g. only the user folder and Documents
		folders?: (places: dupes.Place[]) => dupes.Place[];
		onError?: (message: string) => void;
	} = $props();

	let places = $state<dupes.Place[]>([]);
	let drives = $state<storage.DriveInfo[]>([]);

	const letterOf = (drive: storage.DriveInfo) => drive.path.replace(/[\\/]$/, '');

	async function choose() {
		try {
			const path = await PickFolder(chooseTitle);
			if (path) onPick(path);
		} catch (err) {
			onError?.(`${err}`);
		}
	}

	onMount(() => {
		GetDuplicateFolders()
			.then((p) => (places = folders(p || [])))
			.catch(() => {});
		GetDrives()
			.then((d) => (drives = d || []))
			.catch(() => {});
	});
</script>

{#if layout === 'inline'}
	<section class="flex flex-col gap-2">
		<h3 class="text-[10px] uppercase tracking-wider font-semibold text-muted">{heading}</h3>
		<div class="flex flex-wrap gap-2">
			{#each places as place (place.path)}
				<button class="btn btn-secondary" title={place.path} onclick={() => onPick(place.path)}>
					{place.label}
				</button>
			{/each}
			{#each drives as drive (drive.path)}
				<button class="btn btn-secondary" onclick={() => onPick(letterOf(drive) + '\\')}>
					All of {letterOf(drive)}
				</button>
			{/each}
			<button class="btn btn-secondary" onclick={choose}>Choose a folder...</button>
		</div>
	</section>
{:else}
	{#if places.length}
		<section class="flex flex-col gap-2">
			<h3 class="text-[10px] uppercase tracking-wider font-semibold text-muted">{heading}</h3>
			<div class="flex flex-wrap gap-2">
				{#each places as place (place.path)}
					<button class="btn btn-secondary" title={place.path} onclick={() => onPick(place.path)}>
						{place.label}
					</button>
				{/each}
				<button class="btn btn-secondary" onclick={choose}>Choose a folder...</button>
			</div>
		</section>
	{/if}
	{#if drives.length}
		<section class="flex flex-col gap-2">
			<h3 class="text-[10px] uppercase tracking-wider font-semibold text-muted">Whole drive</h3>
			<div class="grid grid-cols-3 gap-2">
				{#each drives as drive (drive.path)}
					<button
						class="text-left px-3 py-2.5 rounded-xl border border-line bg-tile cursor-pointer transition-colors hover:border-accent"
						onclick={() => onPick(letterOf(drive) + '\\')}
					>
						<span class="flex items-baseline justify-between gap-2">
							<span class="font-bold text-ink text-sm">{letterOf(drive)}</span>
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
{/if}

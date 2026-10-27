<script lang="ts">
	import BusyArt from '../ui/BusyArt.svelte';
	import { onMount, untrack } from 'svelte';
	import {
		GetSoftwareHealthDetails,
		GetWingetUpgrades,
		UpgradeWithWinget
	} from '$lib/wailsjs/go/main/App';
	import { BrowserOpenURL } from '$lib/wailsjs/runtime/runtime';
	import { load, recall } from '$lib/cache';
	import Segmented from '../ui/Segmented.svelte';
	import StrokeIcon from '../ui/StrokeIcon.svelte';

	type SoftwareItem = {
		name: string;
		current_version: string;
		latest_version: string;
		is_outdated: boolean;
		release_date: string;
		official_download: string;
	};
	type WingetPackage = {
		name: string;
		id: string;
		version: string;
		available: string;
		source: string;
		truncated: boolean;
	};
	type WingetResult = {
		status: 'ok' | 'missing' | 'blocked' | 'failed';
		message: string;
		packages: WingetPackage[];
	};
	type UpdateState = { state: 'updating' | 'done' | 'failed'; error?: string };

	// App Installer, which provides winget
	const APP_INSTALLER_URL = 'https://apps.microsoft.com/detail/9NBLGGH4NNS1';

	let tab = $state<'dev' | 'apps'>('dev');

	let items = $state<SoftwareItem[]>(recall('software') ?? []);
	let isLoading = $state(false);
	let searchTerm = $state('');

	let winget = $state<WingetResult | null>(recall('winget') ?? null);
	let isLoadingWinget = $state(false);
	let updates = $state<Record<string, UpdateState>>({});
	let isUpdatingAll = $state(false);

	// outdated tools first, so what needs doing is on screen
	let filteredItems = $derived(
		items
			.filter(
				(i) =>
					i.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
					i.current_version.toLowerCase().includes(searchTerm.toLowerCase())
			)
			.sort((a, b) => Number(b.is_outdated) - Number(a.is_outdated))
	);
	let outdatedCount = $derived(items.filter((i) => i.is_outdated).length);

	// whether the latest release was actually looked up; apps found only in the
	// installed-programs list are "Vendor Managed" and never compared
	const isChecked = (item: SoftwareItem) =>
		!!item.latest_version && item.latest_version !== 'Vendor Managed';
	// what "Update all" will try: apps not yet updated; a failed one is retried on its own
	let updatable = $derived((winget?.packages ?? []).filter((p) => !p.truncated && !updates[p.id]));

	// versions are looked up online, so a result is reused for half an hour; Refresh looks again
	async function loadSoftware(force = false) {
		isLoading = force || items.length === 0;
		try {
			items = ((await load('software', GetSoftwareHealthDetails, 30 * 60_000, force)) ||
				[]) as SoftwareItem[];
		} catch (err) {
			console.error('Failed to get software health details:', err);
		} finally {
			isLoading = false;
		}
	}

	async function loadWinget(force = false) {
		isLoadingWinget = true;
		updates = {};
		try {
			winget = (await load('winget', GetWingetUpgrades, 10 * 60_000, force)) as WingetResult;
		} catch (err) {
			winget = { status: 'failed', message: `${err}`, packages: [] };
		} finally {
			isLoadingWinget = false;
		}
	}

	async function update(pkg: WingetPackage) {
		updates[pkg.id] = { state: 'updating' };
		try {
			await UpgradeWithWinget(pkg.id);
			updates[pkg.id] = { state: 'done' };
		} catch (err) {
			updates[pkg.id] = { state: 'failed', error: `${err}` };
		}
	}

	async function updateAll() {
		isUpdatingAll = true;
		// one at a time: installers often cannot run side by side
		for (const pkg of updatable) await update(pkg);
		isUpdatingAll = false;
	}

	function refresh() {
		if (tab === 'dev') loadSoftware(true);
		else loadWinget(true);
	}

	// load once when the screen opens
	onMount(() => {
		loadSoftware();
	});

	$effect(() => {
		if (tab === 'apps' && !winget && !isLoadingWinget) untrack(() => loadWinget());
	});

	let anyUpdating = $derived(Object.values(updates).some((u) => u.state === 'updating'));
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<div class="flex justify-between items-center gap-3 pb-2 border-b border-line shrink-0">
		<div class="min-w-0">
			<div class="font-bold text-base text-ink">Software</div>
			<div class="text-xs text-muted truncate">
				{tab === 'dev'
					? 'Developer tools compared with their latest official releases'
					: 'Apps that Windows Package Manager (winget) can update'}
			</div>
		</div>
		<div class="flex items-center gap-2 shrink-0">
			<Segmented
				options={[
					{ id: 'dev', label: 'Developer tools' },
					{ id: 'apps', label: 'All apps' }
				]}
				value={tab}
				onchange={(t) => (tab = t as 'dev' | 'apps')}
				label="Software lists"
				tabs
				size="sm"
			/>
			<button
				class="btn btn-secondary"
				disabled={(tab === 'dev' ? isLoading : isLoadingWinget) || anyUpdating}
				onclick={refresh}
			>
				Refresh
			</button>
		</div>
	</div>

	{#if tab === 'dev'}
		{#if isLoading}
			<div class="flex-1 flex items-center justify-center">
				<BusyArt kind="measure" size={64} label="Checking installed versions..." />
			</div>
		{:else}
			<div class="flex items-center justify-between gap-3 shrink-0">
				<span class="text-muted">
					{items.length} tools ·
					<span class={outdatedCount > 0 ? 'text-amber-700 dark:text-amber-400 font-semibold' : ''}>
						{outdatedCount} outdated
					</span>
				</span>
				<input
					type="search"
					placeholder="Search"
					aria-label="Search developer tools"
					bind:value={searchTerm}
					class="text-xs w-40 h-7 px-2.5 py-0 rounded-lg border border-line bg-tile text-ink placeholder:text-muted focus:outline-none focus:ring-1 focus:ring-accent focus:border-accent"
				/>
			</div>
			<div
				class="flex-1 min-h-0 overflow-y-auto overscroll-contain pb-5 [mask-image:linear-gradient(to_bottom,#000_calc(100%-20px),transparent)] pr-1.5"
			>
				{#each filteredItems as item (item.name)}
					<div class="flex items-center gap-3 h-10 border-b border-line">
						<span class="flex-1 min-w-0 font-semibold text-ink truncate" title={item.name}
							>{item.name}</span
						>
						<span class="w-36 shrink-0 font-mono text-[11px] text-right truncate">
							{#if item.is_outdated}
								<span class="text-muted">{item.current_version}</span>
								<span class="text-muted">→</span>
								<span class="text-ink font-semibold">{item.latest_version}</span>
							{:else}
								<span class="text-muted">{item.current_version}</span>
							{/if}
						</span>
						<span class="w-16 shrink-0 text-right">
							{#if item.is_outdated}
								<span
									class="text-[10px] px-1.5 py-0.5 rounded-md bg-amber-500/15 text-amber-700 dark:text-amber-400 font-semibold"
								>
									Outdated
								</span>
							{:else if isChecked(item)}
								<span class="text-[10px] px-1.5 py-0.5 rounded-md text-muted font-semibold">
									Up to date
								</span>
							{/if}
						</span>
						<button
							class="w-16 shrink-0 text-right text-[11px] font-semibold cursor-pointer hover:underline disabled:opacity-40 disabled:no-underline {item.is_outdated
								? 'text-accent'
								: 'text-muted hover:text-ink'}"
							disabled={!item.official_download}
							onclick={() => BrowserOpenURL(item.official_download)}
						>
							{item.official_download.includes('google.com/search') ? 'Search' : 'Download'}
						</button>
					</div>
				{:else}
					<div class="h-full flex items-center justify-center text-muted">No matching tools.</div>
				{/each}
			</div>
		{/if}
	{:else if isLoadingWinget || !winget}
		<div class="flex-1 flex items-center justify-center">
			<BusyArt
				kind="search"
				size={64}
				label="Asking winget for updates. This can take half a minute..."
			/>
		</div>
	{:else if winget.status === 'blocked'}
		<div
			class="flex-1 flex items-center justify-center text-muted text-center px-10 leading-relaxed"
		>
			Your organisation has turned off Windows Package Manager (winget) on this PC, so apps cannot
			be updated from here.
		</div>
	{:else if winget.status === 'missing'}
		<div class="flex-1 flex flex-col items-center justify-center gap-3 text-center px-10">
			<p class="text-muted leading-relaxed">
				Windows Package Manager (winget) is not installed. It comes with App Installer from the
				Microsoft Store.
			</p>
			<button class="btn btn-primary" onclick={() => BrowserOpenURL(APP_INSTALLER_URL)}>
				Get App Installer
			</button>
		</div>
	{:else if winget.status === 'failed'}
		<div class="flex-1 flex items-center justify-center text-muted text-center px-10">
			winget could not list updates{winget.message ? `: ${winget.message}` : '.'}
		</div>
	{:else if winget.packages.length === 0}
		<div class="flex-1 flex items-center justify-center text-muted">
			Everything winget manages is up to date.
		</div>
	{:else}
		<div class="flex items-center justify-between gap-3 shrink-0">
			<span class="text-muted">
				{winget.packages.length}
				{winget.packages.length === 1 ? 'update' : 'updates'} available · some installers ask for administrator
				permission
			</span>
			<button
				class="btn btn-primary"
				disabled={updatable.length === 0 || anyUpdating}
				onclick={updateAll}
			>
				{isUpdatingAll ? 'Updating...' : `Update all (${updatable.length})`}
			</button>
		</div>
		<div
			class="flex-1 min-h-0 overflow-y-auto overscroll-contain pb-5 [mask-image:linear-gradient(to_bottom,#000_calc(100%-20px),transparent)] pr-1.5"
		>
			{#each winget.packages as pkg (pkg.id + pkg.name)}
				{@const u = updates[pkg.id]}
				<div class="flex items-center gap-3 h-10 border-b border-line">
					<span class="flex-1 min-w-0 flex flex-col leading-tight">
						<span class="font-semibold text-ink truncate" title={pkg.name}>{pkg.name}</span>
						<span class="text-[11px] text-muted truncate" title={pkg.id}>{pkg.id}</span>
					</span>
					<span class="w-40 shrink-0 font-mono text-[11px] text-right truncate">
						<span class="text-muted">{pkg.version}</span>
						<span class="text-muted">→</span>
						<span class="text-accent font-semibold">{pkg.available}</span>
					</span>
					<span class="w-20 shrink-0 flex justify-end">
						{#if u?.state === 'updating'}
							<span
								class="w-4 h-4 border-2 border-accent border-t-transparent rounded-full animate-spin"
								aria-label="Updating"
							></span>
						{:else if u?.state === 'done'}
							<span
								class="flex items-center gap-1 text-green-700 dark:text-green-400 font-semibold text-[11px]"
							>
								<StrokeIcon d="M5 12.5l4.5 4.5L19 7.5" size={12} /> Updated
							</span>
						{:else}
							<button
								class="btn btn-secondary h-6 px-2.5 {u?.state === 'failed'
									? 'text-red-600 dark:text-red-400'
									: ''}"
								disabled={pkg.truncated || anyUpdating}
								title={pkg.truncated
									? 'winget shortened this app’s ID; update it from a terminal with: winget upgrade'
									: u?.state === 'failed'
										? `Update failed: ${u.error}`
										: `Update ${pkg.name} to ${pkg.available}`}
								onclick={() => update(pkg)}
							>
								{u?.state === 'failed' ? 'Retry' : 'Update'}
							</button>
						{/if}
					</span>
				</div>
			{/each}
		</div>
	{/if}
</div>

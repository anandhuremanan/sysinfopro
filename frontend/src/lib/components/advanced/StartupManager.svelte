<script lang="ts">
	import type { system } from '$lib/wailsjs/go/models';
	import ToolHeader from '$lib/components/ui/ToolHeader.svelte';
	import { onMount } from 'svelte';
	import BusyArt from '../ui/BusyArt.svelte';
	import { GetStartupItems } from '$lib/wailsjs/go/main/App';

	type StartupItem = system.StartupItem;

	let items = $state<StartupItem[]>([]);
	let isLoading = $state(true);
	let searchTerm = $state('');

	async function loadStartup() {
		isLoading = true;
		try {
			items = ((await GetStartupItems()) || []) as StartupItem[];
		} catch (err) {
			console.error('Failed to get startup items:', err);
		} finally {
			isLoading = false;
		}
	}

	onMount(() => {
		loadStartup();
	});

	// "GoogleChromeAutoLaunch_4F5E2A…" -> "GoogleChromeAutoLaunch"; "Teams.lnk" -> "Teams"
	function friendlyName(name: string) {
		return name
			.replace(/[_-][0-9A-F]{8,}$/i, '')
			.replace(/\.(lnk|url|exe|bat|cmd)$/i, '')
			.trim();
	}

	// the program a command starts: the first quoted or unquoted path, file name only
	function programOf(command: string) {
		const m = command.match(/^\s*"([^"]+)"/) ?? command.match(/^\s*(\S+)/);
		const path = m?.[1] ?? '';
		return path.split(/[\\/]/).pop() ?? '';
	}

	const WHERE: Record<string, string> = {
		'HKCU Registry': 'Registry, for you',
		'HKLM Registry': 'Registry, for everyone',
		'Startup Folder': 'Startup folder'
	};

	let filteredItems = $derived(
		items.filter((i) => {
			const q = searchTerm.trim().toLowerCase();
			return !q || i.name.toLowerCase().includes(q) || i.command.toLowerCase().includes(q);
		})
	);
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<ToolHeader title="Startup">
		{#snippet sub()}
			{isLoading
				? 'Programs that start when you sign in'
				: `${items.length} ${items.length === 1 ? 'program starts' : 'programs start'} when you sign in`}
		{/snippet}
		{#snippet actions()}
			<div class="flex items-center gap-2 shrink-0">
				<input
					type="search"
					placeholder="Search..."
					aria-label="Search startup programs"
					bind:value={searchTerm}
					class="text-xs px-3 py-1.5 rounded-lg border border-line bg-tile text-ink focus:outline-none focus:ring-1 focus:ring-accent w-40"
				/>
				<button class="btn btn-secondary" onclick={loadStartup} disabled={isLoading}>Refresh</button
				>
			</div>
		{/snippet}
	</ToolHeader>

	{#if isLoading}
		<div class="flex-1 flex items-center justify-center">
			<BusyArt kind="search" size={64} label="Looking for programs that start with Windows..." />
		</div>
	{:else if filteredItems.length === 0}
		<div class="flex-1 flex items-center justify-center text-xs text-muted">
			{items.length ? 'Nothing matches that search.' : 'No programs start when you sign in.'}
		</div>
	{:else}
		<div class="flex-1 min-h-0 overflow-y-auto overscroll-contain pr-1.5 flex flex-col gap-1.5">
			{#each filteredItems as item (item.location + item.name)}
				{@const program = programOf(item.command)}
				<div class="rounded-xl bg-tile border border-line px-3 py-2 min-w-0" title={item.command}>
					<div class="flex items-baseline gap-2 min-w-0">
						<span class="text-xs font-semibold text-ink truncate">{friendlyName(item.name)}</span>
						<span class="ml-auto text-[11px] text-muted shrink-0"
							>{WHERE[item.location] ?? item.location}</span
						>
					</div>
					{#if program}
						<div class="text-[11px] text-muted font-mono truncate mt-0.5">{program}</div>
					{/if}
				</div>
			{/each}
		</div>
		<p class="text-[11px] text-muted px-1 shrink-0">
			Point at a program to see its full command. To stop one starting, use Task Manager's Startup
			apps page.
		</p>
	{/if}
</div>

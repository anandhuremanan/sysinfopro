<script lang="ts">
	import { GetSoftwareHealthDetails } from '$lib/wailsjs/go/main/App';
	import { BrowserOpenURL } from '$lib/wailsjs/runtime/runtime';

	type SoftwareItem = {
		name: string;
		current_version: string;
		latest_version: string;
		is_outdated: boolean;
		release_date: string;
		official_download: string;
	};

	let items = $state<SoftwareItem[]>([]);
	let isLoading = $state(false);
	let searchTerm = $state('');

	async function loadSoftware() {
		isLoading = true;
		try {
			// @ts-ignore
			const res = await GetSoftwareHealthDetails();
			if (res) {
				items = res;
			}
		} catch (err) {
			console.error('Failed to get software health details:', err);
		} finally {
			isLoading = false;
		}
	}

	function openDownload(url: string) {
		if (url) {
			BrowserOpenURL(url);
		}
	}

	$effect(() => {
		loadSoftware();
	});

	let filteredItems = $derived(
		items.filter(
			(i) =>
				i.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
				i.current_version.toLowerCase().includes(searchTerm.toLowerCase())
		)
	);

	let outdatedCount = $derived(items.filter((i) => i.is_outdated).length);
</script>

<div class="flex flex-col h-full min-h-0 space-y-2 p-1">
	<!-- Header Bar -->
	<div
		class="flex justify-between items-center pb-2 border-b border-slate-300 dark:border-slate-800 shrink-0"
	>
		<div>
			<div class="font-bold text-base text-slate-800 dark:text-slate-100 flex items-center gap-2">
				<span>Software Health Checker</span>
			</div>
			<div class="text-xs text-gray-500 dark:text-slate-400">
				Audit installed developer tools & system software against official latest releases
			</div>
		</div>

		<div class="flex items-center gap-3">
			<input
				type="text"
				placeholder="Search software..."
				bind:value={searchTerm}
				class="text-xs px-3 py-1.5 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-800 dark:text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500 w-44"
			/>
			<button
				class="text-xs dark:bg-zinc-800 bg-zinc-300 px-3 py-1.5 rounded-lg cursor-pointer hover:bg-zinc-400 dark:hover:bg-zinc-700 font-medium transition-transform active:scale-95 border border-slate-300 dark:border-slate-700/60"
				onclick={loadSoftware}
				disabled={isLoading}
			>
				{isLoading ? 'Scanning...' : 'Refresh'}
			</button>
		</div>
	</div>

	<!-- Main List -->
	{#if isLoading}
		<div class="flex-1 flex items-center justify-center text-xs text-slate-400 animate-pulse">
			Auditing installed software versions...
		</div>
	{:else if filteredItems.length === 0}
		<div class="flex-1 flex items-center justify-center text-xs text-slate-400">
			No software items found.
		</div>
	{:else}
		<div class="flex-1 flex flex-col min-h-0 space-y-2">
			<!-- Summary Stats Bar -->
			<div
				class="grid grid-cols-3 gap-3 bg-slate-300/30 dark:bg-slate-800/60 p-2.5 rounded-xl text-xs border border-slate-300 dark:border-slate-700/60 shrink-0"
			>
				<div>
					<div class="text-[10px] text-gray-400">Total Monitored</div>
					<div class="font-bold font-mono text-slate-800 dark:text-slate-200">
						{items.length} Apps
					</div>
				</div>
				<div>
					<div class="text-[10px] text-gray-400">Up to Date</div>
					<div class="font-bold font-mono text-green-400">
						{items.length - outdatedCount}
					</div>
				</div>
				<div>
					<div class="text-[10px] text-gray-400">Updates Pending</div>
					<div class="font-bold font-mono {outdatedCount > 0 ? 'text-amber-400' : 'text-gray-400'}">
						{outdatedCount} Outdated
					</div>
				</div>
			</div>

			<!-- Main Table Container (Single vertical scroll, 100% width fit) -->
			<div class="flex-1 basis-0 min-h-0 overflow-y-auto overflow-x-hidden pr-1 rounded-lg">
				<table class="w-full text-xs text-left border-collapse table-fixed">
					<thead class="sticky top-0 z-10 bg-slate-200 dark:bg-slate-900 shadow-xs">
						<tr
							class="border-b border-slate-300 dark:border-slate-700 text-slate-500 dark:text-slate-400 uppercase tracking-wider text-[10px]"
						>
							<th class="py-2 pl-2 text-left w-[34%]">Software Name</th>
							<th class="py-2 text-center w-[16%]">Status</th>
							<th class="py-2 text-center w-[14%]">Current</th>
							<th class="py-2 text-center w-[15%]">Latest</th>
							<th class="py-2 text-center w-[11%]">Release</th>
							<th class="py-2 pr-2 text-center w-[10%]">Link</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-slate-300/40 dark:divide-slate-800">
						{#each filteredItems as item}
							<tr class="hover:bg-slate-300/40 dark:hover:bg-slate-800/60 transition-colors">
								<td
									class="py-2 pl-2 font-semibold text-slate-800 dark:text-slate-200 truncate pr-2"
									title={item.name}
								>
									{item.name}
								</td>
								<td class="py-2 text-center">
									{#if item.is_outdated}
										<span
											class="inline-block px-1.5 py-0.5 rounded-full bg-amber-500/20 text-amber-400 font-semibold border border-amber-500/30 text-[9px] whitespace-nowrap"
										>
											↓ Outdated
										</span>
									{:else}
										<span
											class="inline-block px-1.5 py-0.5 rounded-full bg-green-500/15 text-green-400 font-medium text-[9px] whitespace-nowrap"
										>
											✓ Current
										</span>
									{/if}
								</td>
								<td class="py-2 text-center font-mono text-[10px] text-gray-400 truncate">
									{item.current_version}
								</td>
								<td class="py-2 text-center font-mono text-[10px] text-sky-400 font-semibold truncate">
									{item.latest_version}
								</td>
								<td class="py-2 text-center font-mono text-[9px] text-gray-400 truncate">
									{item.release_date}
								</td>
								<td class="py-2 pr-2 text-center">
									<button
										class="px-2 py-0.5 text-[10px] bg-blue-500 text-white rounded-md cursor-pointer hover:bg-blue-600 font-medium transition-transform active:scale-95 shadow-xs whitespace-nowrap"
										onclick={() => openDownload(item.official_download)}
									>
										Get
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}

	<!-- Footer Note -->
	<div
		class="text-[11px] text-gray-400 pt-1.5 border-t border-slate-300 dark:border-slate-800 flex justify-between items-center shrink-0"
	>
		<span>Software Health & Security Auditor</span>
		<span class="text-[10px] text-gray-500">Official vendor release channels</span>
	</div>
</div>

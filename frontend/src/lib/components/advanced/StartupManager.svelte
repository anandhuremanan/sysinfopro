<script lang="ts">
	import { GetStartupItems } from '$lib/wailsjs/go/main/App';

	type StartupItem = {
		name: string;
		command: string;
		location: string;
		impact: string;
		impact_score: number;
		is_enabled: boolean;
	};

	let items = $state<StartupItem[]>([]);
	let isLoading = $state(false);
	let searchTerm = $state('');

	async function loadStartup() {
		isLoading = true;
		try {
			// @ts-ignore
			const res = await GetStartupItems();
			if (res) {
				items = res;
			}
		} catch (err) {
			console.error('Failed to get startup items:', err);
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		loadStartup();
	});

	let filteredItems = $derived(
		items.filter(
			(i) =>
				i.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
				i.command.toLowerCase().includes(searchTerm.toLowerCase())
		)
	);
</script>

<div class="flex flex-col h-full min-h-0 space-y-2 p-1">
	<!-- Header Bar -->
	<div
		class="flex justify-between items-center pb-2 border-b border-slate-300 dark:border-slate-800 shrink-0"
	>
		<div>
			<div class="font-bold text-base text-slate-800 dark:text-slate-100 flex items-center gap-2">
				<span>Startup Impact & Boot Optimizer</span>
			</div>
			<div class="text-xs text-gray-500 dark:text-slate-400">
				Audit auto-starting registry entries & startup folder bloat causing boot delays
			</div>
		</div>

		<div class="flex items-center gap-3">
			<input
				type="text"
				placeholder="Search startup item..."
				bind:value={searchTerm}
				class="text-xs px-3 py-1.5 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-800 dark:text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500 w-48"
			/>
			<button
				class="text-xs dark:bg-zinc-800 bg-zinc-300 px-3 py-1.5 rounded-lg cursor-pointer hover:bg-zinc-400 dark:hover:bg-zinc-700 font-medium transition-transform active:scale-95 border border-slate-300 dark:border-slate-700/60"
				onclick={loadStartup}
				disabled={isLoading}
			>
				{isLoading ? 'Scanning...' : 'Refresh'}
			</button>
		</div>
	</div>

	<!-- Main List -->
	{#if isLoading}
		<div class="flex-1 flex items-center justify-center text-xs text-slate-400 animate-pulse">
			Scanning Registry & Startup triggers...
		</div>
	{:else if filteredItems.length === 0}
		<div class="flex-1 flex items-center justify-center text-xs text-slate-400">
			No startup items found.
		</div>
	{:else}
		<div class="flex-1 basis-0 min-h-0 overflow-y-auto overscroll-contain pr-2 rounded-lg">
			<table class="w-full text-xs text-left border-collapse">
				<thead class="sticky top-0 z-10 bg-slate-200 dark:bg-slate-900 shadow-xs">
					<tr
						class="border-b border-slate-300 dark:border-slate-700 text-slate-500 dark:text-slate-400 uppercase tracking-wider text-[10px]"
					>
						<th class="py-2 pl-2">Program Name</th>
						<th class="py-2">Registry / Location</th>
						<th class="py-2">Command Path</th>
						<th class="py-2 text-center">Boot Impact</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-300/40 dark:divide-slate-800">
					{#each filteredItems as item}
						<tr class="hover:bg-slate-300/40 dark:hover:bg-slate-800/60 transition-colors">
							<td
								class="py-2 pl-2 font-semibold text-slate-800 dark:text-slate-200 truncate max-w-[180px]"
								title={item.name}
							>
								{item.name}
							</td>
							<td class="py-2 text-gray-400 text-[10px]">{item.location}</td>
							<td
								class="py-2 font-mono text-[10px] text-gray-400 truncate max-w-[240px]"
								title={item.command}
							>
								{item.command}
							</td>
							<td class="py-2 text-center">
								{#if item.impact === 'High'}
									<span
										class="px-2 py-0.5 rounded-full bg-red-500/20 text-red-400 font-semibold border border-red-500/30 text-[10px]"
									>
										⚡ High ({item.impact_score})
									</span>
								{:else if item.impact === 'Medium'}
									<span
										class="px-2 py-0.5 rounded-full bg-amber-500/20 text-amber-400 font-semibold border border-amber-500/30 text-[10px]"
									>
										Med ({item.impact_score})
									</span>
								{:else}
									<span
										class="px-2 py-0.5 rounded-full bg-slate-500/15 text-gray-400 font-medium text-[10px]"
									>
										Low ({item.impact_score})
									</span>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

	<!-- Footer Note -->
	<div
		class="text-[11px] text-gray-400 pt-1.5 border-t border-slate-300 dark:border-slate-800 flex justify-between items-center shrink-0"
	>
		<span>Showing {filteredItems.length} auto-start items</span>
		<span class="text-[10px] text-gray-500">Registry & Startup Folder entries</span>
	</div>
</div>


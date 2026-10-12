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
		class="flex justify-between items-center pb-2 border-b border-line shrink-0"
	>
		<div>
			<div class="font-bold text-base text-ink flex items-center gap-2">
				<span>Startup Impact & Boot Optimizer</span>
			</div>
			<div class="text-xs text-muted">
				Audit auto-starting registry entries & startup folder bloat causing boot delays
			</div>
		</div>

		<div class="flex items-center gap-3">
			<input
				type="text"
				placeholder="Search startup item..."
				bind:value={searchTerm}
				class="text-xs px-3 py-1.5 rounded-lg border border-line bg-tile text-ink focus:outline-none focus:ring-1 focus:ring-accent w-48"
			/>
			<button
				class="text-xs bg-ground px-3 py-1.5 rounded-lg cursor-pointer hover:bg-line font-medium transition-transform active:scale-95 border border-line"
				onclick={loadStartup}
				disabled={isLoading}
			>
				{isLoading ? 'Scanning...' : 'Refresh'}
			</button>
		</div>
	</div>

	<!-- Main List -->
	{#if isLoading}
		<div class="flex-1 flex items-center justify-center text-xs text-muted animate-pulse">
			Scanning Registry & Startup triggers...
		</div>
	{:else if filteredItems.length === 0}
		<div class="flex-1 flex items-center justify-center text-xs text-muted">
			No startup items found.
		</div>
	{:else}
		<div class="flex-1 basis-0 min-h-0 overflow-y-auto overscroll-contain pr-2 rounded-lg">
			<table class="w-full text-xs text-left border-collapse">
				<thead class="sticky top-0 z-10 bg-tile shadow-xs">
					<tr
						class="border-b border-line text-muted uppercase tracking-wider text-[10px]"
					>
						<th class="py-2 pl-2">Program Name</th>
						<th class="py-2">Registry / Location</th>
						<th class="py-2">Command Path</th>
						<th class="py-2 text-center">Boot Impact</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-line">
					{#each filteredItems as item}
						<tr class="hover:bg-ground transition-colors">
							<td
								class="py-2 pl-2 font-semibold text-ink truncate max-w-[180px]"
								title={item.name}
							>
								{item.name}
							</td>
							<td class="py-2 text-muted text-[10px]">{item.location}</td>
							<td
								class="py-2 font-mono text-[10px] text-muted truncate max-w-[240px]"
								title={item.command}
							>
								{item.command}
							</td>
							<td class="py-2 text-center">
								{#if item.impact === 'High'}
									<span
										class="px-2 py-0.5 rounded-full bg-red-500/20 text-red-600 dark:text-red-400 font-semibold border border-red-500/30 text-[10px]"
									>
										High ({item.impact_score})
									</span>
								{:else if item.impact === 'Medium'}
									<span
										class="px-2 py-0.5 rounded-full bg-amber-500/20 text-amber-700 dark:text-amber-400 font-semibold border border-amber-500/30 text-[10px]"
									>
										Med ({item.impact_score})
									</span>
								{:else}
									<span
										class="px-2 py-0.5 rounded-full bg-ground text-muted font-medium text-[10px]"
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
		class="text-[11px] text-muted pt-1.5 border-t border-line flex justify-between items-center shrink-0"
	>
		<span>Showing {filteredItems.length} auto-start items</span>
		<span class="text-[10px] text-muted">Registry & Startup Folder entries</span>
	</div>
</div>


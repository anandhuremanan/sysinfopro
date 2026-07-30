<script lang="ts">
	import { GetTopProcesses, TerminateProcess } from '$lib/wailsjs/go/main/App';

	type ProcessItem = {
		pid: number;
		name: string;
		cpu_percent: number;
		memory_mb: number;
		memory_percent: number;
		energy_impact?: number;
		energy_impact_label?: string;
	};

	let processes = $state<ProcessItem[]>([]);
	let isLoading = $state(false);
	let statusMessage = $state<string | undefined>(undefined);
	let searchTerm = $state('');

	async function loadProcesses() {
		isLoading = true;
		try {
			// Fetch top 25 resource consumers
			// @ts-ignore
			const result = await GetTopProcesses(25);
			if (result) {
				processes = result;
			}
		} catch (err) {
			console.error('Failed to get processes:', err);
		} finally {
			isLoading = false;
		}
	}

	async function handleKill(pid: number, name: string) {
		try {
			// @ts-ignore
			await TerminateProcess(pid);
			statusMessage = `Terminated ${name} (PID ${pid})`;
			await loadProcesses();
		} catch (err) {
			statusMessage = `Failed to terminate: ${err}`;
		} finally {
			setTimeout(() => {
				statusMessage = undefined;
			}, 3000);
		}
	}

	$effect(() => {
		loadProcesses();
	});

	let filteredProcesses = $derived(
		processes.filter(p => p.name.toLowerCase().includes(searchTerm.toLowerCase()))
	);
</script>

<div class="flex flex-col justify-between h-full space-y-2 p-1">
	<!-- Header Bar -->
	<div class="flex justify-between items-center pb-2 border-b border-slate-300 dark:border-slate-800">
		<div>
			<div class="font-bold text-base text-slate-800 dark:text-slate-100">Top Resource Consumers</div>
			<div class="text-xs text-gray-500 dark:text-slate-400">
				Monitor memory footprints, CPU load & energy impact in real time
			</div>
		</div>

		<div class="flex items-center gap-3">
			<input
				type="text"
				placeholder="Search process..."
				bind:value={searchTerm}
				class="text-xs px-3 py-1.5 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-800 dark:text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500 w-48"
			/>
			<button
				class="text-xs dark:bg-zinc-800 bg-zinc-300 px-3 py-1.5 rounded-lg cursor-pointer hover:bg-zinc-400 dark:hover:bg-zinc-700 font-medium transition-colors border border-slate-300 dark:border-slate-700/60"
				onclick={loadProcesses}
			>
				Refresh List
			</button>
		</div>
	</div>

	{#if statusMessage}
		<div class="text-xs text-amber-500 font-semibold px-1">{statusMessage}</div>
	{/if}

	<!-- Main Table Container with Scrollbar -->
	{#if isLoading}
		<div class="flex-1 flex items-center justify-center text-xs text-slate-400 animate-pulse">
			Sampling running system processes...
		</div>
	{:else if filteredProcesses.length === 0}
		<div class="flex-1 flex items-center justify-center text-xs text-slate-400">
			No matching active processes found.
		</div>
	{:else}
		<div class="flex-1 min-h-0 overflow-y-auto pr-2 rounded-lg">
			<table class="w-full text-xs text-left border-collapse">
				<thead class="sticky top-0 z-10 bg-slate-200 dark:bg-slate-900 shadow-xs">
					<tr class="border-b border-slate-300 dark:border-slate-700 text-slate-500 dark:text-slate-400 uppercase tracking-wider text-[10px]">
						<th class="py-2 pl-2">Process Name</th>
						<th class="py-2 text-right">PID</th>
						<th class="py-2 text-right pr-4">RAM Footprint</th>
						<th class="py-2 text-right pr-4">CPU %</th>
						<th class="py-2 text-center">Energy Drain</th>
						<th class="py-2 text-center">Action</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-300/40 dark:divide-slate-800">
					{#each filteredProcesses as proc}
						<tr class="hover:bg-slate-300/40 dark:hover:bg-slate-800/60 transition-colors">
							<td class="py-2 pl-2 font-semibold text-slate-800 dark:text-slate-200 truncate max-w-[200px]" title={proc.name}>
								{proc.name}
							</td>
							<td class="py-2 text-right text-gray-400 font-mono">{proc.pid}</td>
							<td class="py-2 text-right pr-4 font-mono font-bold text-slate-800 dark:text-slate-200">
								{proc.memory_mb.toFixed(1)} MB
							</td>
							<td class="py-2 text-right pr-4 font-mono {proc.cpu_percent > 10 ? 'text-amber-500 font-bold' : ''}">
								{proc.cpu_percent.toFixed(1)}%
							</td>
							<td class="py-2 text-center font-mono text-[10px]">
								{#if proc.energy_impact_label === 'High'}
									<span class="px-2 py-0.5 rounded-full bg-red-500/20 text-red-400 font-semibold border border-red-500/30">
										⚡ High ({proc.energy_impact?.toFixed(0)})
									</span>
								{:else if proc.energy_impact_label === 'Moderate'}
									<span class="px-2 py-0.5 rounded-full bg-amber-500/20 text-amber-400 font-semibold border border-amber-500/30">
										⚡ Med ({proc.energy_impact?.toFixed(0)})
									</span>
								{:else}
									<span class="px-2 py-0.5 rounded-full bg-slate-500/15 text-gray-400 font-medium">
										Low ({proc.energy_impact ? proc.energy_impact.toFixed(0) : '0'})
									</span>
								{/if}
							</td>
							<td class="py-2 text-center">
								<button
									class="px-3 py-1 text-[11px] bg-red-500 text-white rounded-md cursor-pointer hover:bg-red-600 font-medium transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
									disabled={proc.pid <= 4}
									onclick={() => handleKill(proc.pid, proc.name)}
								>
									End Task
								</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

	<!-- Footer Note -->
	<div class="text-[11px] text-gray-400 pt-1.5 border-t border-slate-300 dark:border-slate-800 flex justify-between items-center">
		<span>Showing top {filteredProcesses.length} resource consuming processes</span>
		<span class="text-[10px] text-gray-500">System critical processes (PID &le; 4) are protected</span>
	</div>
</div>

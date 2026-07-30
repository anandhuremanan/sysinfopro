<script lang="ts">
	import { GetTopProcesses, TerminateProcess } from '$lib/wailsjs/go/main/App';
	import Modal from './Modal.svelte';

	let { showModal = $bindable(false) } = $props();

	type ProcessItem = {
		pid: number;
		name: string;
		cpu_percent: number;
		memory_mb: number;
		memory_percent: number;
	};

	let processes = $state<ProcessItem[]>([]);
	let isLoading = $state(false);
	let statusMessage = $state<string | undefined>(undefined);
	let searchTerm = $state('');

	async function loadProcesses() {
		isLoading = true;
		try {
			// @ts-ignore
			const result = await GetTopProcesses(15);
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
		if (showModal) {
			loadProcesses();
		}
	});

	let filteredProcesses = $derived(
		processes.filter(p => p.name.toLowerCase().includes(searchTerm.toLowerCase()))
	);
</script>

<Modal bind:showModal showClose={true}>
	<div class="flex flex-col space-y-3 max-h-[70vh] p-1">
		<div class="flex justify-between items-center pr-6">
			<div class="font-bold text-sm text-slate-800 dark:text-slate-100">Top Resource Consumers</div>
			<button
				class="text-xs bg-slate-300 dark:bg-slate-700 px-2.5 py-1 rounded cursor-pointer hover:bg-slate-400 dark:hover:bg-slate-600 transition-colors"
				onclick={loadProcesses}
			>
				Refresh
			</button>
		</div>

		<input
			type="text"
			placeholder="Search process..."
			bind:value={searchTerm}
			class="w-full text-xs p-2 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900/80 text-slate-800 dark:text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500"
		/>

		{#if statusMessage}
			<div class="text-xs text-amber-500 font-semibold">{statusMessage}</div>
		{/if}

		{#if isLoading}
			<div class="text-xs text-center py-8 text-slate-400">Sampling running processes...</div>
		{:else if filteredProcesses.length === 0}
			<div class="text-xs text-center py-8 text-slate-400">No matching processes found.</div>
		{:else}
			<div class="overflow-y-auto max-h-[45vh] pr-2 rounded-lg">
				<table class="w-full text-xs text-left border-collapse">
					<thead class="sticky top-0 z-10 bg-zinc-200 dark:bg-zinc-800">
						<tr class="border-b border-slate-300 dark:border-slate-700 text-slate-500 dark:text-slate-400">
							<th class="py-2 pl-1">Process</th>
							<th class="py-2 text-right">PID</th>
							<th class="py-2 text-right pr-2">RAM (MB)</th>
							<th class="py-2 text-right pr-2">CPU %</th>
							<th class="py-2 text-center">Action</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-slate-300/40 dark:divide-slate-700/40">
						{#each filteredProcesses as proc}
							<tr class="hover:bg-slate-300/30 dark:hover:bg-slate-700/30 transition-colors">
								<td class="py-2 pl-1 font-medium truncate max-w-[150px]" title={proc.name}>
									{proc.name}
								</td>
								<td class="py-2 text-right text-gray-400 font-mono">{proc.pid}</td>
								<td class="py-2 text-right pr-2 font-mono">{proc.memory_mb.toFixed(1)} MB</td>
								<td class="py-2 text-right pr-2 font-mono {proc.cpu_percent > 10 ? 'text-amber-500 font-bold' : ''}">
									{proc.cpu_percent.toFixed(1)}%
								</td>
								<td class="py-2 text-center">
									<button
										class="px-2.5 py-0.5 text-[11px] bg-red-500/90 text-white rounded cursor-pointer hover:bg-red-600 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
										disabled={proc.pid <= 4}
										onclick={() => handleKill(proc.pid, proc.name)}
									>
										End
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>
</Modal>

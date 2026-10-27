<script lang="ts">
	import type { process } from '$lib/wailsjs/go/models';
	import { onMount } from 'svelte';
	import { Status } from '$lib/status.svelte';
	import { GetTopProcesses, TerminateProcessChecked } from '$lib/wailsjs/go/main/App';

	type ProcessItem = process.ProcessItem;

	const REFRESH_MS = 5000;
	const GROWTH_WINDOW_MS = 60000;
	// a rate from less than this much history is noise
	const MIN_GROWTH_SPAN_MS = 15000;

	let processes = $state<ProcessItem[]>([]);
	let isLoading = $state(false);
	const status = new Status(5000);
	let searchTerm = $state('');
	let sortBy = $state<'memory' | 'growth'>('memory');

	// recent memory readings per process, used to work out how fast each is growing
	const samples = new Map<number, { at: number; mb: number }[]>();
	// MB per minute by PID; absent until there is enough history
	let growth = $state<Record<number, number>>({});

	function recordSamples(list: ProcessItem[]) {
		const now = Date.now();
		const alive = new Set<number>();
		const next: Record<number, number> = {};

		for (const proc of list) {
			alive.add(proc.pid);
			const history = samples.get(proc.pid) ?? [];
			history.push({ at: now, mb: proc.memory_mb });
			while (history.length > 1 && now - history[0].at > GROWTH_WINDOW_MS) history.shift();
			samples.set(proc.pid, history);

			const span = now - history[0].at;
			if (span >= MIN_GROWTH_SPAN_MS) {
				next[proc.pid] = (proc.memory_mb - history[0].mb) / (span / 60000);
			}
		}
		for (const pid of samples.keys()) {
			if (!alive.has(pid)) samples.delete(pid);
		}
		growth = next;
	}

	async function loadProcesses(silent = false) {
		if (!silent) isLoading = true;
		try {
			// Fetch top 25 resource consumers
			// @ts-ignore
			const result = await GetTopProcesses(25);
			if (result) {
				processes = result;
				recordSamples(result);
			}
		} catch (err) {
			console.error('Failed to get processes:', err);
		} finally {
			isLoading = false;
		}
	}

	// ends the program only if it is still the one shown: same id, name and start time
	async function handleKill(pid: number, name: string, startedMs: number) {
		try {
			await TerminateProcessChecked(pid, name, startedMs);
			status.text = `Terminated ${name} (PID ${pid})`;
			await loadProcesses();
		} catch (err) {
			status.text = `Failed to terminate: ${err}`;
		}
	}

	onMount(() => {
		loadProcesses();
		const timer = setInterval(() => loadProcesses(true), REFRESH_MS);
		return () => clearInterval(timer);
	});

	let filteredProcesses = $derived.by(() => {
		const list = processes.filter((p) => p.name.toLowerCase().includes(searchTerm.toLowerCase()));
		if (sortBy === 'growth') {
			return [...list].sort((a, b) => (growth[b.pid] ?? -Infinity) - (growth[a.pid] ?? -Infinity));
		}
		return list;
	});

	function growthLabel(rate: number | undefined) {
		if (rate === undefined) return '…';
		if (Math.abs(rate) < 1) return 'steady';
		return `${rate > 0 ? '+' : '−'}${Math.abs(rate).toFixed(0)} MB/min`;
	}

	function growthColor(rate: number | undefined) {
		if (rate === undefined || Math.abs(rate) < 1) return 'text-muted';
		if (rate >= 20) return 'text-warn font-bold';
		return rate > 0 ? 'text-ink' : 'text-ok';
	}
</script>

<div class="flex flex-col justify-between h-full space-y-2 p-1">
	<!-- Header Bar -->
	<div class="flex justify-between items-center pb-2 border-b border-line">
		<div>
			<div class="font-bold text-base text-ink">Top Resource Consumers</div>
			<div class="text-xs text-muted">
				Memory, memory growth and processor time, updated every 5 seconds
			</div>
		</div>

		<div class="flex items-center gap-3">
			<input
				type="text"
				placeholder="Search process..."
				bind:value={searchTerm}
				class="text-xs px-3 py-1.5 rounded-lg border border-line bg-tile text-ink focus:outline-none focus:ring-1 focus:ring-accent w-48"
			/>
			<button
				class="text-xs bg-ground px-3 py-1.5 rounded-lg cursor-pointer hover:bg-line font-medium transition-colors border border-line whitespace-nowrap shrink-0"
				title="Growth finds a process that is leaking memory"
				onclick={() => (sortBy = sortBy === 'memory' ? 'growth' : 'memory')}
			>
				Sort: {sortBy === 'memory' ? 'Memory' : 'Growth'}
			</button>
		</div>
	</div>

	{#if status.text}
		<div class="text-xs text-warn font-semibold px-1">{status.text}</div>
	{/if}

	<!-- Main Table Container with Scrollbar -->
	{#if isLoading}
		<div
			class="flex-1 flex items-center justify-center text-xs text-muted motion-safe:animate-pulse"
		>
			Sampling running system processes...
		</div>
	{:else if filteredProcesses.length === 0}
		<div class="flex-1 flex items-center justify-center text-xs text-muted">
			No matching active processes found.
		</div>
	{:else}
		<div class="flex-1 min-h-0 overflow-y-auto pr-2 rounded-lg">
			<table class="w-full text-xs text-left border-collapse">
				<thead class="sticky top-0 z-10 bg-tile shadow-xs">
					<tr class="border-b border-line text-muted uppercase tracking-wider text-[10px]">
						<th class="py-2 pl-2">Process Name</th>
						<th class="py-2 text-right">PID</th>
						<th class="py-2 text-right pr-4">RAM Footprint</th>
						<th class="py-2 text-right pr-4">Growth</th>
						<!-- each program's share of the whole processor since the list was last read -->
						<th
							class="py-2 text-right pr-4"
							title="Share of the whole processor each program used since the list was last refreshed. On processors that speed up past their base speed, Task Manager counts this differently and can show higher numbers."
							>CPU</th
						>
						<th class="py-2 text-center">Action</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-line">
					{#each filteredProcesses as proc (proc.pid)}
						<tr class="hover:bg-ground transition-colors">
							<td class="py-2 pl-2 font-semibold text-ink truncate max-w-[200px]" title={proc.name}>
								{proc.name}
							</td>
							<td class="py-2 text-right text-muted font-mono">{proc.pid}</td>
							<td class="py-2 text-right pr-4 font-mono font-bold text-ink">
								{proc.memory_mb.toFixed(1)} MB
							</td>
							<td
								class="py-2 text-right pr-4 font-mono whitespace-nowrap {growthColor(
									growth[proc.pid]
								)}"
							>
								{growthLabel(growth[proc.pid])}
							</td>
							<td
								class="py-2 text-right pr-4 font-mono {proc.cpu_percent > 10
									? 'text-warn font-bold'
									: ''}"
							>
								{proc.cpu_percent.toFixed(1)}%
							</td>
							<td class="py-2 text-center">
								<button
									class="px-3 py-1 text-[11px] border border-red-500/40 text-fail hover:bg-red-500 hover:text-white dark:hover:text-white rounded-md cursor-pointer font-medium transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
									disabled={proc.pid <= 4}
									onclick={() => handleKill(proc.pid, proc.name, proc.started_ms)}
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
	<div class="text-[11px] text-muted pt-1.5 border-t border-line flex justify-between items-center">
		<span>Showing top {filteredProcesses.length} resource consuming processes</span>
		<span class="text-[10px] text-muted">System critical processes (PID &le; 4) are protected</span>
	</div>
</div>

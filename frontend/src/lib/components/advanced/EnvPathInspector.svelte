<script lang="ts">
	import { GetEnvPathAnalysis } from '$lib/wailsjs/go/main/App';

	type PathEntry = {
		path: string;
		is_valid: boolean;
		is_duplicate: boolean;
		status_text: string;
	};

	type EnvVar = {
		key: string;
		value: string;
	};

	type Analysis = {
		total_entries: number;
		valid_entries: number;
		invalid_entries: number;
		duplicates_count: number;
		path_entries: PathEntry[];
		key_variables: EnvVar[];
	};

	let analysis = $state<Analysis | null>(null);
	let isLoading = $state(false);

	async function loadAnalysis() {
		isLoading = true;
		try {
			// @ts-ignore
			const res = await GetEnvPathAnalysis();
			if (res) {
				analysis = res;
			}
		} catch (err) {
			console.error('Failed to get PATH analysis:', err);
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		loadAnalysis();
	});
</script>

<div class="flex flex-col h-full min-h-0 space-y-2 p-1">
	<!-- Header Bar -->
	<div
		class="flex justify-between items-center pb-2 border-b border-slate-300 dark:border-slate-800 shrink-0"
	>
		<div>
			<div class="font-bold text-base text-slate-800 dark:text-slate-100 flex items-center gap-2">
				<span>Environment & PATH Debugger</span>
			</div>
			<div class="text-xs text-gray-500 dark:text-slate-400">
				Detect invalid directory paths, duplicate entries & audit developer environment variables
			</div>
		</div>

		<button
			class="text-xs dark:bg-zinc-800 bg-zinc-300 px-3 py-1.5 rounded-lg cursor-pointer hover:bg-zinc-400 dark:hover:bg-zinc-700 font-medium transition-transform active:scale-95 border border-slate-300 dark:border-slate-700/60"
			onclick={loadAnalysis}
			disabled={isLoading}
		>
			{isLoading ? 'Scanning...' : 'Refresh'}
		</button>
	</div>

	{#if isLoading}
		<div class="flex-1 flex items-center justify-center text-xs text-slate-400 animate-pulse">
			Parsing %PATH% directory entries...
		</div>
	{:else if analysis}
		<div class="flex-1 flex flex-col min-h-0 space-y-2">
			<!-- Stats Summary Bar -->
			<div
				class="grid grid-cols-4 gap-3 bg-slate-300/30 dark:bg-slate-800/60 p-2.5 rounded-xl text-xs border border-slate-300 dark:border-slate-700/60 shrink-0"
			>
				<div>
					<div class="text-[10px] text-gray-400">Total PATH Entries</div>
					<div class="font-bold font-mono text-slate-800 dark:text-slate-200">
						{analysis.total_entries}
					</div>
				</div>
				<div>
					<div class="text-[10px] text-gray-400">Valid Directories</div>
					<div class="font-bold font-mono text-green-400">{analysis.valid_entries}</div>
				</div>
				<div>
					<div class="text-[10px] text-gray-400">Invalid / Broken</div>
					<div
						class="font-bold font-mono {analysis.invalid_entries > 0
							? 'text-red-400'
							: 'text-gray-400'}"
					>
						{analysis.invalid_entries}
					</div>
				</div>
				<div>
					<div class="text-[10px] text-gray-400">Duplicates</div>
					<div
						class="font-bold font-mono {analysis.duplicates_count > 0
							? 'text-amber-400'
							: 'text-gray-400'}"
					>
						{analysis.duplicates_count}
					</div>
				</div>
			</div>

			<!-- Path Table -->
			<div class="flex-1 basis-0 min-h-0 overflow-y-auto overscroll-contain pr-2 rounded-lg">
				<table class="w-full text-xs text-left border-collapse">
					<thead class="sticky top-0 z-10 bg-slate-200 dark:bg-slate-900 shadow-xs">
						<tr
							class="border-b border-slate-300 dark:border-slate-700 text-slate-500 dark:text-slate-400 uppercase tracking-wider text-[10px]"
						>
							<th class="py-2 pl-2">System PATH Entry</th>
							<th class="py-2 text-center">Directory Health</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-slate-300/40 dark:divide-slate-800">
						{#each analysis.path_entries as entry}
							<tr class="hover:bg-slate-300/40 dark:hover:bg-slate-800/60 transition-colors">
								<td
									class="py-2 pl-2 font-mono text-[10px] text-slate-800 dark:text-slate-200 truncate max-w-[420px]"
									title={entry.path}
								>
									{entry.path}
								</td>
								<td class="py-2 text-center">
									{#if entry.is_duplicate}
										<span
											class="px-2 py-0.5 rounded-full bg-amber-500/20 text-amber-400 font-semibold border border-amber-500/30 text-[10px]"
										>
											Duplicate
										</span>
									{:else if !entry.is_valid}
										<span
											class="px-2 py-0.5 rounded-full bg-red-500/20 text-red-400 font-semibold border border-red-500/30 text-[10px]"
										>
											Missing / Broken
										</span>
									{:else}
										<span
											class="px-2 py-0.5 rounded-full bg-green-500/15 text-green-400 font-medium text-[10px]"
										>
											Valid
										</span>
									{/if}
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
		<span>Environment & PATH Inspection</span>
		<span class="text-[10px] text-gray-500">Cross-platform environment diagnostics</span>
	</div>
</div>


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
		class="flex justify-between items-center pb-2 border-b border-line shrink-0"
	>
		<div>
			<div class="font-bold text-base text-ink flex items-center gap-2">
				<span>Environment & PATH Debugger</span>
			</div>
			<div class="text-xs text-muted">
				Detect invalid directory paths, duplicate entries & audit developer environment variables
			</div>
		</div>

		<button
			class="text-xs bg-ground px-3 py-1.5 rounded-lg cursor-pointer hover:bg-line font-medium transition-transform active:scale-95 border border-line"
			onclick={loadAnalysis}
			disabled={isLoading}
		>
			{isLoading ? 'Scanning...' : 'Refresh'}
		</button>
	</div>

	{#if isLoading}
		<div class="flex-1 flex items-center justify-center text-xs text-muted animate-pulse">
			Parsing %PATH% directory entries...
		</div>
	{:else if analysis}
		<div class="flex-1 flex flex-col min-h-0 space-y-2">
			<!-- Stats Summary Bar -->
			<div
				class="grid grid-cols-4 gap-3 bg-ground p-2.5 rounded-xl text-xs border border-line shrink-0"
			>
				<div>
					<div class="text-[10px] text-muted">Total PATH Entries</div>
					<div class="font-bold font-mono text-ink">
						{analysis.total_entries}
					</div>
				</div>
				<div>
					<div class="text-[10px] text-muted">Valid Directories</div>
					<div class="font-bold font-mono text-green-700 dark:text-green-400">{analysis.valid_entries}</div>
				</div>
				<div>
					<div class="text-[10px] text-muted">Invalid / Broken</div>
					<div
						class="font-bold font-mono {analysis.invalid_entries > 0
							? 'text-red-600 dark:text-red-400'
							: 'text-muted'}"
					>
						{analysis.invalid_entries}
					</div>
				</div>
				<div>
					<div class="text-[10px] text-muted">Duplicates</div>
					<div
						class="font-bold font-mono {analysis.duplicates_count > 0
							? 'text-amber-700 dark:text-amber-400'
							: 'text-muted'}"
					>
						{analysis.duplicates_count}
					</div>
				</div>
			</div>

			<!-- Path Table -->
			<div class="flex-1 basis-0 min-h-0 overflow-y-auto overscroll-contain pr-2 rounded-lg">
				<table class="w-full text-xs text-left border-collapse">
					<thead class="sticky top-0 z-10 bg-tile shadow-xs">
						<tr
							class="border-b border-line text-muted uppercase tracking-wider text-[10px]"
						>
							<th class="py-2 pl-2">System PATH Entry</th>
							<th class="py-2 text-center">Directory Health</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-line">
						{#each analysis.path_entries as entry}
							<tr class="hover:bg-ground transition-colors">
								<td
									class="py-2 pl-2 font-mono text-[10px] text-ink truncate max-w-[420px]"
									title={entry.path}
								>
									{entry.path}
								</td>
								<td class="py-2 text-center">
									{#if entry.is_duplicate}
										<span
											class="px-2 py-0.5 rounded-full bg-amber-500/20 text-amber-700 dark:text-amber-400 font-semibold border border-amber-500/30 text-[10px]"
										>
											Duplicate
										</span>
									{:else if !entry.is_valid}
										<span
											class="px-2 py-0.5 rounded-full bg-red-500/20 text-red-600 dark:text-red-400 font-semibold border border-red-500/30 text-[10px]"
										>
											Missing / Broken
										</span>
									{:else}
										<span
											class="px-2 py-0.5 rounded-full bg-green-500/15 text-green-700 dark:text-green-400 font-medium text-[10px]"
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
		class="text-[11px] text-muted pt-1.5 border-t border-line flex justify-between items-center shrink-0"
	>
		<span>Environment & PATH Inspection</span>
		<span class="text-[10px] text-muted">Cross-platform environment diagnostics</span>
	</div>
</div>


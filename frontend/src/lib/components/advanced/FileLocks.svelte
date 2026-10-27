<script lang="ts">
	import BusyArt from '../ui/BusyArt.svelte';
	import {
		FindLockingProcesses,
		PickFile,
		PickFolder,
		TerminateProcess
	} from '$lib/wailsjs/go/main/App';

	type LockingProcess = { pid: number; name: string; kind: string };
	type LockResult = {
		path: string;
		is_dir: boolean;
		files_checked: number;
		truncated: boolean;
		processes: LockingProcess[];
	};

	let result = $state<LockResult | undefined>(undefined);
	let isChecking = $state(false);
	let statusMessage = $state<string | undefined>(undefined);

	function showStatus(message: string) {
		statusMessage = message;
		setTimeout(() => {
			statusMessage = undefined;
		}, 5000);
	}

	async function check(path: string) {
		isChecking = true;
		try {
			result = (await FindLockingProcesses(path)) as LockResult;
		} catch (err) {
			result = undefined;
			showStatus(`${err}`);
		} finally {
			isChecking = false;
		}
	}

	async function choose(kind: 'file' | 'folder') {
		try {
			const path =
				kind === 'file'
					? await PickFile('Choose the file that will not delete, move or save')
					: await PickFolder('Choose the folder that will not delete or move');
			if (path) await check(path);
		} catch (err) {
			showStatus(`${err}`);
		}
	}

	async function endProcess(proc: LockingProcess) {
		try {
			await TerminateProcess(proc.pid);
			showStatus(`Ended ${proc.name} (PID ${proc.pid})`);
			if (result) await check(result.path);
		} catch (err) {
			showStatus(`Failed to end ${proc.name}: ${err}`);
		}
	}

	// ending these would take Windows down with them
	function isProtected(proc: LockingProcess) {
		return proc.pid <= 4 || proc.kind === 'System critical';
	}
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<!-- Header Bar -->
	<div class="flex justify-between items-center gap-3 pb-2 border-b border-line shrink-0">
		<div class="min-w-0">
			<div class="font-bold text-base text-ink">File locks</div>
			<div class="text-xs text-muted truncate">
				Find the program that stops a file or folder being deleted, moved or saved
			</div>
		</div>

		<div class="flex items-center gap-2 shrink-0">
			<button class="btn btn-secondary" onclick={() => choose('file')} disabled={isChecking}>
				Choose file
			</button>
			<button class="btn btn-primary" onclick={() => choose('folder')} disabled={isChecking}>
				Choose folder
			</button>
		</div>
	</div>

	{#if statusMessage}
		<div
			class="text-xs text-amber-700 dark:text-amber-400 font-semibold px-1 shrink-0 truncate"
			title={statusMessage}
		>
			{statusMessage}
		</div>
	{/if}

	{#if isChecking}
		<div class="flex-1 flex items-center justify-center">
			<BusyArt kind="search" size={64} label="Asking Windows which programs have it open..." />
		</div>
	{:else if result}
		<div class="flex items-center justify-between gap-3 shrink-0">
			<span class="font-semibold text-ink truncate" title={result.path}>{result.path}</span>
			<button
				class="btn btn-secondary h-6 px-2 shrink-0"
				onclick={() => result && check(result.path)}
			>
				Check again
			</button>
		</div>

		<div
			class="flex-1 min-h-0 overflow-y-auto overscroll-contain pb-5 [mask-image:linear-gradient(to_bottom,#000_calc(100%-20px),transparent)] pr-1.5"
		>
			{#each result.processes as proc (proc.pid)}
				<div class="flex items-center gap-3 h-10 border-b border-line">
					<span class="flex-1 min-w-0 font-semibold text-ink truncate" title={proc.name}>
						{proc.name}
					</span>
					<span class="shrink-0 w-28 text-muted">{proc.kind}</span>
					<span class="shrink-0 w-16 text-right font-mono text-muted">{proc.pid}</span>
					{#if isProtected(proc)}
						<span class="shrink-0 w-[92px] text-right text-[11px] text-muted"
							>Needed by Windows</span
						>
					{:else}
						<!-- neutral until pointed at: ending a program is a choice, not an alarm -->
						<button
							class="btn btn-secondary h-6 px-2 w-[92px] shrink-0 hover:border-red-500/50 hover:text-red-600 dark:hover:text-red-400"
							title="Unsaved work in it is lost"
							onclick={() => endProcess(proc)}
						>
							End process
						</button>
					{/if}
				</div>
			{:else}
				<div class="h-full flex flex-col items-center justify-center gap-1 text-center">
					<div class="font-semibold text-ink">Nothing has it open</div>
					<p class="text-muted max-w-sm leading-relaxed">
						If it still will not delete, the cause is likely permissions or a read-only flag, not
						another program.
					</p>
				</div>
			{/each}
		</div>

		<!-- Footer Note -->
		<div
			class="text-[11px] text-muted pt-1.5 border-t border-line flex justify-between items-center gap-3 shrink-0"
		>
			<span>
				{result.is_dir
					? `Checked ${result.files_checked.toLocaleString()} files in this folder`
					: 'Checked 1 file'} · {result.processes.length}
				{result.processes.length === 1 ? 'program' : 'programs'} holding it
			</span>
			{#if result.truncated}
				<span class="shrink-0 text-amber-700 dark:text-amber-400">
					Large folder: only the first {result.files_checked.toLocaleString()} files were checked
				</span>
			{/if}
		</div>
	{:else}
		<!-- Ready State -->
		<div class="flex-1 flex flex-col items-center justify-center gap-2 text-center">
			<div class="text-sm font-bold text-ink">
				"The action can't be completed because the file is open"
			</div>
			<p class="text-xs text-muted max-w-md leading-relaxed">
				Choose the file or folder Windows refuses to touch. You will see every program that has it
				open, and can end the one in the way. Checking changes nothing.
			</p>
		</div>
	{/if}
</div>

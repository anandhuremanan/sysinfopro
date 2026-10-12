<script lang="ts">
	import { GetActiveSockets, TerminateProcess } from '$lib/wailsjs/go/main/App';

	type SocketItem = {
		pid: number;
		process_name: string;
		protocol: string;
		local_port: number;
		local_ip: string;
		remote_ip: string;
		remote_port: number;
		status: string;
	};

	let sockets = $state<SocketItem[]>([]);
	let isLoading = $state(false);
	let statusMessage = $state<string | undefined>(undefined);
	let searchTerm = $state('');

	async function loadSockets() {
		isLoading = true;
		try {
			// @ts-ignore
			const res = await GetActiveSockets();
			if (res) {
				sockets = res;
			}
		} catch (err) {
			console.error('Failed to get sockets:', err);
		} finally {
			isLoading = false;
		}
	}

	async function handleKillPort(pid: number, name: string, port: number) {
		try {
			// @ts-ignore
			await TerminateProcess(pid);
			statusMessage = `Released Port ${port} (Terminated ${name})`;
			await loadSockets();
		} catch (err) {
			statusMessage = `Failed to release port: ${err}`;
		} finally {
			setTimeout(() => {
				statusMessage = undefined;
			}, 3000);
		}
	}

	$effect(() => {
		loadSockets();
	});

	let filteredSockets = $derived(
		sockets.filter(
			s =>
				s.process_name.toLowerCase().includes(searchTerm.toLowerCase()) ||
				s.local_port.toString().includes(searchTerm)
		)
	);
</script>

<div class="flex flex-col h-full min-h-0 space-y-2 p-1">
	<!-- Header Bar -->
	<div class="flex justify-between items-center pb-2 border-b border-line shrink-0">
		<div>
			<div class="font-bold text-base text-ink flex items-center gap-2">
				<span>Port & Socket Inspector</span>
			</div>
			<div class="text-xs text-muted">
				Audit active listening ports & release blocked developer sockets (e.g. 3000, 8080)
			</div>
		</div>

		<div class="flex items-center gap-3">
			<input
				type="text"
				placeholder="Search port or process..."
				bind:value={searchTerm}
				class="text-xs px-3 py-1.5 rounded-lg border border-line bg-tile text-ink focus:outline-none focus:ring-1 focus:ring-accent w-48"
			/>
			<button
				class="text-xs bg-ground px-3 py-1.5 rounded-lg cursor-pointer hover:bg-line font-medium transition-transform active:scale-95 border border-line"
				onclick={loadSockets}
				disabled={isLoading}
			>
				{isLoading ? 'Scanning...' : 'Refresh'}
			</button>
		</div>
	</div>

	{#if statusMessage}
		<div class="text-xs text-amber-700 dark:text-amber-400 font-semibold px-1">{statusMessage}</div>
	{/if}

	<!-- Main Table Container -->
	{#if isLoading}
		<div class="flex-1 flex items-center justify-center text-xs text-muted animate-pulse">
			Scanning active system network sockets...
		</div>
	{:else if filteredSockets.length === 0}
		<div class="flex-1 flex items-center justify-center text-xs text-muted">
			No matching listening ports found.
		</div>
	{:else}
		<div class="flex-1 basis-0 min-h-0 overflow-y-auto overscroll-contain pr-2 rounded-lg">
			<table class="w-full text-xs text-left border-collapse">
				<thead class="sticky top-0 z-10 bg-tile shadow-xs">
					<tr class="border-b border-line text-muted uppercase tracking-wider text-[10px]">
						<th class="py-2 pl-2">Local Port</th>
						<th class="py-2">Protocol</th>
						<th class="py-2">Process Name</th>
						<th class="py-2 text-right pr-4">PID</th>
						<th class="py-2">Status</th>
						<th class="py-2 text-center">Action</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-line">
					{#each filteredSockets as sock}
						<tr class="hover:bg-ground transition-colors">
							<td class="py-2 pl-2 font-mono font-bold text-accent">
								:{sock.local_port}
							</td>
							<td class="py-2 font-mono text-muted text-[10px]">{sock.protocol}</td>
							<td class="py-2 font-semibold text-ink truncate max-w-[180px]" title={sock.process_name}>
								{sock.process_name}
							</td>
							<td class="py-2 text-right pr-4 font-mono text-muted">{sock.pid}</td>
							<td class="py-2 font-mono text-[10px]">
								<span class="px-2 py-0.5 rounded-full bg-green-500/15 text-green-700 dark:text-green-400 font-semibold border border-green-500/30">
									{sock.status}
								</span>
							</td>
							<td class="py-2 text-center">
								<button
									class="px-2.5 py-1 text-[10px] border border-red-500/40 text-red-600 dark:text-red-400 hover:bg-red-500 hover:text-white dark:hover:text-white rounded-md cursor-pointer font-medium transition-transform active:scale-95 disabled:opacity-30 disabled:cursor-not-allowed"
									disabled={sock.pid <= 4}
									onclick={() => handleKillPort(sock.pid, sock.process_name, sock.local_port)}
								>
									Release Port
								</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

	<!-- Footer Note -->
	<div class="text-[11px] text-muted pt-1.5 border-t border-line flex justify-between items-center shrink-0">
		<span>Showing active listening ports ({filteredSockets.length})</span>
		<span class="text-[10px] text-muted">Click Release Port to terminate socket owners</span>
	</div>
</div>


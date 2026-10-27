<script lang="ts">
	import type { network } from '$lib/wailsjs/go/models';
	import ToolHeader from '$lib/components/ui/ToolHeader.svelte';
	import { Status } from '$lib/status.svelte';
	import { onMount } from 'svelte';
	import BusyArt from '../ui/BusyArt.svelte';
	import Modal from '../ui/Modal.svelte';
	import { GetActiveSockets, TerminateProcess } from '$lib/wailsjs/go/main/App';

	type SocketItem = network.SocketItem;

	// "2h 10m", "35m", "just now"
	function runningFor(startedAt: number) {
		if (!startedAt) return '';
		const minutes = Math.floor((Date.now() / 1000 - startedAt) / 60);
		if (minutes < 1) return 'just now';
		if (minutes < 60) return `${minutes}m`;
		const hours = Math.floor(minutes / 60);
		if (hours < 24) return `${hours}h ${minutes % 60}m`;
		return `${Math.floor(hours / 24)}d ${hours % 24}h`;
	}

	function originTitle(sock: SocketItem) {
		return [
			sock.command_line && `Command: ${sock.command_line}`,
			sock.working_dir && `Started in: ${sock.working_dir}`,
			sock.project_root && `Project: ${sock.project_root}`
		]
			.filter(Boolean)
			.join('\n');
	}

	let sockets = $state<SocketItem[]>([]);
	let isLoading = $state(true);
	const status = new Status(4000);
	let searchTerm = $state('');
	let stopping = $state<SocketItem | null>(null);

	// Windows' own processes: stopping them is refused, or takes Windows down with them
	const SYSTEM = new Set(
		[
			'system',
			'svchost.exe',
			'lsass.exe',
			'services.exe',
			'wininit.exe',
			'spoolsv.exe',
			'csrss.exe',
			'smss.exe',
			'winlogon.exe',
			'searchindexer.exe',
			'msmpeng.exe'
		].map((n) => n.toLowerCase())
	);
	// a name Windows will not give out ("System/Unknown") is a protected process
	const canStop = (s: SocketItem) =>
		s.pid > 4 && !SYSTEM.has(s.process_name.toLowerCase()) && !s.process_name.startsWith('System');

	async function loadSockets() {
		isLoading = true;
		try {
			sockets = ((await GetActiveSockets()) || []) as SocketItem[];
		} catch (err) {
			console.error('Failed to get sockets:', err);
		} finally {
			isLoading = false;
		}
	}

	async function stop() {
		const sock = stopping;
		stopping = null;
		if (!sock) return;
		try {
			await TerminateProcess(sock.pid);
			status.text = `${sock.process_name} was stopped, so port ${sock.local_port} is free.`;
			await loadSockets();
		} catch (err) {
			status.text = `Could not stop ${sock.process_name}: ${err}`;
		}
	}

	onMount(() => {
		loadSockets();
	});

	// only ports waiting for connections (TCP listening, UDP bound); one row per port and
	// process, and ports opened from a project first: those are what a developer looks for
	let listening = $derived.by(() => {
		const byKey = new Map<string, SocketItem>();
		for (const sock of sockets) {
			if (!['LISTEN', 'NONE', ''].includes(sock.status)) continue;
			byKey.set(`${sock.protocol}/${sock.local_port}/${sock.pid}`, sock);
		}
		return [...byKey.values()].sort(
			(a, b) => Number(!!b.project_name) - Number(!!a.project_name) || a.local_port - b.local_port
		);
	});

	let filteredSockets = $derived(
		listening.filter((s) => {
			const q = searchTerm.trim().toLowerCase();
			return (
				!q ||
				s.process_name.toLowerCase().includes(q) ||
				(s.project_name || '').toLowerCase().includes(q) ||
				s.local_port.toString().includes(q)
			);
		})
	);
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<ToolHeader title="Ports">
		{#snippet sub()}
			{isLoading
				? 'Which program is using each port'
				: `${listening.length} ports open, and which program holds each`}
		{/snippet}
		{#snippet actions()}
			<div class="flex items-center gap-2 shrink-0">
				<input
					type="search"
					placeholder="Port or program..."
					aria-label="Search ports"
					bind:value={searchTerm}
					class="text-xs px-3 py-1.5 rounded-lg border border-line bg-tile text-ink focus:outline-none focus:ring-1 focus:ring-accent w-40"
				/>
				<button class="btn btn-secondary" onclick={loadSockets} disabled={isLoading}>Refresh</button
				>
			</div>
		{/snippet}
	</ToolHeader>

	{#if status.text}
		<div class="text-xs font-semibold text-muted px-1 shrink-0">{status.text}</div>
	{/if}

	{#if isLoading}
		<div class="flex-1 flex items-center justify-center">
			<BusyArt kind="search" size={64} label="Looking at open ports..." />
		</div>
	{:else if filteredSockets.length === 0}
		<div class="flex-1 flex items-center justify-center text-xs text-muted">
			{listening.length ? 'Nothing matches that search.' : 'No ports are open.'}
		</div>
	{:else}
		<div
			class="flex-1 basis-0 min-h-0 overflow-y-auto overscroll-contain pr-1.5 pb-5 [mask-image:linear-gradient(to_bottom,#000_calc(100%-20px),transparent)]"
		>
			<table class="w-full text-xs text-left border-collapse">
				<thead class="sticky top-0 z-10 bg-tile">
					<tr class="border-b border-line text-muted text-[11px]">
						<th class="py-1.5 pl-2 font-medium">Port</th>
						<th class="py-1.5 font-medium">Program</th>
						<th class="py-1.5 font-medium">Project</th>
						<th class="py-1.5 font-medium">Running for</th>
						<th class="py-1.5"><span class="sr-only">Stop</span></th>
					</tr>
				</thead>
				<tbody class="divide-y divide-line">
					{#each filteredSockets as sock (sock.protocol + sock.local_port + '/' + sock.pid)}
						<tr class="hover:bg-ground transition-colors">
							<td class="py-1.5 pl-2 font-mono font-semibold text-ink whitespace-nowrap">
								{sock.local_port}<span class="ml-1 text-muted font-normal text-[10px]">
									{sock.protocol}</span
								>
							</td>
							<td class="py-1.5 max-w-[160px]" title={originTitle(sock) || sock.process_name}>
								<div class="font-semibold text-ink truncate">{sock.process_name}</div>
								<div class="text-muted text-[10px]">PID {sock.pid}</div>
							</td>
							<td class="py-1.5 max-w-[170px]" title={originTitle(sock)}>
								{#if sock.project_name}
									<div class="font-semibold text-ink truncate">{sock.project_name}</div>
									{#if sock.git_branch}
										<div class="font-mono text-muted text-[10px] truncate">{sock.git_branch}</div>
									{/if}
								{:else}
									<span class="text-muted">&ndash;</span>
								{/if}
							</td>
							<td class="py-1.5 text-muted whitespace-nowrap">{runningFor(sock.started_at)}</td>
							<td class="py-1.5 pr-1 text-right">
								{#if canStop(sock)}
									<button
										class="btn btn-secondary !px-2.5 !py-1 !text-[11px]"
										onclick={() => (stopping = sock)}
									>
										Stop
									</button>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		<p class="text-[11px] text-muted px-1 shrink-0">
			Point at a program to see its command line and folder.
		</p>
	{/if}
</div>

<Modal
	showModal={stopping !== null}
	width="max-w-sm"
	title="Stop {stopping?.process_name}?"
	description="Anything unsaved in it is lost. Port {stopping?.local_port} becomes free."
>
	{#snippet actions()}
		<button class="btn btn-secondary" onclick={() => (stopping = null)}>Cancel</button>
		<button class="btn btn-primary" onclick={stop}>Stop it</button>
	{/snippet}
</Modal>

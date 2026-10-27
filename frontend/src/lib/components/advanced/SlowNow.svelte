<script lang="ts">
	import { onMount } from 'svelte';
	import { formatBytes } from '$lib';
	import { DiagnoseSlowness, TerminateProcess } from '$lib/wailsjs/go/main/App';
	import { recall, remember } from '$lib/cache';
	import BusyArt from '../ui/BusyArt.svelte';
	import Modal from '../ui/Modal.svelte';

	type Proc = {
		pid: number;
		name: string;
		cpu_pct: number;
		memory_bytes: number;
		io_bytes_per_sec: number;
		count: number;
	};
	type Finding = {
		id: string;
		level: 'high' | 'medium' | 'info';
		title: string;
		detail: string;
		fix: string;
		pid: number;
		process: string;
	};
	type Report = {
		findings: Finding[];
		cpu: { percent: number; performance_pct: number; top: Proc[] };
		memory: { used_pct: number; total_bytes: number; available_bytes: number; top: Proc[] };
		disk: { busy_pct: number; top: Proc[] };
		summary: string;
		calm: boolean;
		sampled_ms: number;
	};

	const LEVEL = {
		high: { pill: 'bg-red-500/15 text-red-700 dark:text-red-400', label: 'Big effect' },
		medium: { pill: 'bg-amber-500/15 text-amber-700 dark:text-amber-400', label: 'Some effect' },
		info: { pill: 'bg-ground text-muted', label: 'Worth knowing' }
	};

	// the banner sums up; the cards below give each cause
	let banner = $derived.by(() => {
		if (!report || report.calm) return 'Nothing is slowing the PC down right now.';
		const n = report.findings.filter((f) => f.level !== 'info').length;
		return n > 1
			? `${n} things are slowing the PC down right now. The biggest is first.`
			: report.summary;
	});

	// a result is only "right now" for a short while
	const fresh = recall<{ at: number; report: Report }>('slownow');
	let report = $state<Report | null>(fresh && Date.now() - fresh.at < 30_000 ? fresh.report : null);
	let isRunning = $state(false);
	let ending = $state<Finding | null>(null);
	let status = $state('');

	let meters = $derived(
		report
			? [
					{
						label: 'Processor',
						pct: report.cpu.percent,
						top: report.cpu.top,
						value: (p: Proc) => `${p.cpu_pct.toFixed(0)}%`
					},
					{
						label: 'Memory',
						pct: report.memory.used_pct,
						top: report.memory.top,
						value: (p: Proc) => formatBytes(p.memory_bytes)
					},
					{
						label: 'Drive',
						pct: report.disk.busy_pct,
						top: report.disk.top,
						value: (p: Proc) => `${formatBytes(p.io_bytes_per_sec)}/s`
					}
				]
			: []
	);

	async function run() {
		if (isRunning) return;
		isRunning = true;
		status = '';
		try {
			report = (await DiagnoseSlowness()) as Report;
			remember('slownow', { at: Date.now(), report });
		} catch (err) {
			status = `${err}`;
		} finally {
			isRunning = false;
		}
	}

	async function endIt() {
		if (!ending) return;
		const f = ending;
		ending = null;
		try {
			await TerminateProcess(f.pid);
			status = `${f.process} was closed.`;
			run();
		} catch (err) {
			status = `Could not close ${f.process}: ${err}`;
		}
	}

	const tone = (pct: number) =>
		pct >= 90 ? 'bg-red-500' : pct >= 75 ? 'bg-amber-500' : 'bg-ink/35';

	onMount(() => {
		if (!report) run();
	});
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<div class="flex justify-between items-center gap-3 pb-2 border-b border-line shrink-0">
		<div class="min-w-0">
			<div class="font-bold text-base text-ink">Why so slow</div>
			<div class="text-xs text-muted truncate">
				What is holding the PC back right now, in plain words
			</div>
		</div>
		<button class="btn btn-secondary shrink-0" disabled={isRunning} onclick={run}>
			{isRunning ? 'Watching...' : 'Check again'}
		</button>
	</div>

	{#if status}
		<div class="text-xs font-semibold text-muted px-1 shrink-0">{status}</div>
	{/if}

	{#if isRunning && !report}
		<div class="flex-1 flex items-center justify-center">
			<BusyArt kind="measure" size={64} label="Watching the PC for a few seconds..." />
		</div>
	{:else if report}
		<div
			class="mx-1 px-3 py-2 rounded-xl text-sm font-semibold shrink-0 transition-opacity {isRunning
				? 'opacity-50'
				: ''} {report.calm
				? 'bg-green-500/10 text-green-700 dark:text-green-400'
				: report.findings[0]?.level === 'high'
					? 'bg-red-500/10 text-red-700 dark:text-red-400'
					: 'bg-amber-500/10 text-amber-700 dark:text-amber-400'}"
		>
			{banner}
		</div>

		<div class="flex-1 min-h-0 grid grid-cols-[3fr_2fr] gap-3 pt-1 {isRunning ? 'opacity-50' : ''}">
			<!-- what was found -->
			<div
				class="min-h-0 overflow-y-auto overscroll-contain pb-5 [mask-image:linear-gradient(to_bottom,#000_calc(100%-20px),transparent)] pr-1 flex flex-col gap-1.5"
			>
				{#each report.findings as f (f.id + f.pid)}
					<div class="rounded-xl bg-tile border border-line px-3 py-2">
						<div class="flex items-center gap-2">
							<span class="text-xs font-semibold text-ink flex-1 min-w-0">{f.title}</span>
							<span
								class="text-[11px] font-semibold px-1.5 py-0.5 rounded-md shrink-0 {LEVEL[f.level]
									.pill}"
							>
								{LEVEL[f.level].label}
							</span>
						</div>
						<p class="text-xs text-muted mt-0.5 leading-relaxed">{f.detail}</p>
						{#if f.fix}
							<p class="text-xs text-ink mt-1 leading-relaxed">
								<span class="font-semibold">Try:</span>
								{f.fix}
							</p>
						{/if}
						{#if f.id === 'cpu-app' && f.pid > 0}
							<button
								class="mt-1.5 text-[11px] font-semibold text-accent cursor-pointer hover:underline"
								onclick={() => (ending = f)}
							>
								Close {f.process}
							</button>
						{/if}
					</div>
				{:else}
					<div class="text-xs text-muted px-1 leading-relaxed">
						The processor, memory and drive all have room to spare, and nothing in the background is
						busy. If the PC still feels slow, the cause may be the network (try Network doctor) or a
						single app.
					</div>
				{/each}
			</div>

			<!-- the three meters -->
			<div
				class="min-h-0 overflow-y-auto overscroll-contain pb-5 [mask-image:linear-gradient(to_bottom,#000_calc(100%-20px),transparent)] flex flex-col gap-2"
			>
				{#each meters as m (m.label)}
					<section class="rounded-xl bg-tile border border-line px-3 py-2 text-xs">
						<div class="flex items-baseline justify-between">
							<h3 class="text-[10px] uppercase tracking-wider font-semibold text-muted">
								{m.label}
							</h3>
							<span class="font-mono font-bold text-ink"
								>{m.pct < 0 ? 'n/a' : `${m.pct.toFixed(0)}%`}</span
							>
						</div>
						<div class="h-1.5 mt-1.5 rounded-full bg-track overflow-hidden">
							<div
								class="h-full rounded-full {tone(m.pct)}"
								style="width: {Math.max(0, Math.min(100, m.pct))}%"
							></div>
						</div>
						<ul class="mt-1.5 flex flex-col gap-0.5">
							{#each m.top.slice(0, 3) as p (p.name)}
								<li class="flex items-baseline gap-2 min-w-0">
									<span class="text-ink truncate">{p.name.replace(/\.exe$/i, '')}</span>
									{#if p.count > 1}<span class="text-[11px] text-muted shrink-0">×{p.count}</span
										>{/if}
									<span class="ml-auto font-mono text-[11px] text-muted shrink-0">{m.value(p)}</span
									>
								</li>
							{/each}
						</ul>
					</section>
				{/each}
			</div>
		</div>
	{/if}
</div>

<Modal
	showModal={ending !== null}
	width="max-w-sm"
	title="Close {ending?.process}?"
	description="Anything not saved in it will be lost."
>
	{#snippet actions()}
		<button class="btn btn-secondary" onclick={() => (ending = null)}>Cancel</button>
		<button class="btn btn-primary" onclick={endIt}>Close it</button>
	{/snippet}
</Modal>

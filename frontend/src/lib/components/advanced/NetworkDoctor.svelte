<script lang="ts">
	import ToolHeader from '$lib/components/ui/ToolHeader.svelte';
	import { onMount } from 'svelte';
	import { RunNetworkDoctor } from '$lib/wailsjs/go/main/App';
	import { EventsOn } from '$lib/wailsjs/runtime/runtime';
	import { recall, remember } from '$lib/cache';
	import StrokeIcon from '../ui/StrokeIcon.svelte';

	type Status = 'ok' | 'warn' | 'fail' | 'skip';
	type Step = { id: string; name: string; status: Status; detail: string; fix: string; ms: number };
	type Report = {
		steps: Step[];
		adapter: { name: string; kind: string; ipv4: string; gateway: string; dns: string[] } | null;
		proxy: {
			enabled: boolean;
			server: string;
			auto_config: string;
			bypass: string;
			winhttp: string;
		};
		hosts: { ip: string; names: string[]; kind: 'redirect' | 'block' | 'local' }[];
		summary: string;
		healthy: boolean;
	};

	// the chain, in the order it is checked
	const LINKS = [
		{ id: 'adapter', label: 'This PC', icon: 'M4 5h16v11H4zM8 20h8M12 16v4' },
		{
			id: 'router',
			label: 'Router',
			icon: 'M4 14h16v5H4zM8 14l-2-6M16 14l2-6M8 16.5h.01M12 16.5h.01'
		},
		{ id: 'dns', label: 'DNS', icon: 'M4 6h16M4 12h16M4 18h10M18 16l2 2-2 2' },
		{
			id: 'internet',
			label: 'Internet',
			icon: 'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zM3 12h18M12 3c2.5 2.5 3.5 5.5 3.5 9s-1 6.5-3.5 9c-2.5-2.5-3.5-5.5-3.5-9s1-6.5 3.5-9z'
		},
		{
			id: 'site',
			label: 'Site',
			icon: 'M10 14a4 4 0 0 0 5.66 0l3-3a4 4 0 0 0-5.66-5.66l-1 1M14 10a4 4 0 0 0-5.66 0l-3 3a4 4 0 0 0 5.66 5.66l1-1'
		}
	];

	const TONE: Record<Status, string> = {
		ok: 'text-ok',
		warn: 'text-warn',
		fail: 'text-fail',
		skip: 'text-muted'
	};
	const MARK: Record<Status, string> = {
		ok: 'M5 12l5 5 9-10',
		warn: 'M12 7v6M12 17h.01',
		fail: 'M7 7l10 10M17 7L7 17',
		skip: 'M7 12h10'
	};

	let report = $state<Report | null>(recall('netdoctor') ?? null);
	let live = $state<Step[]>([]);
	let isRunning = $state(false);
	let site = $state<string>(recall('netdoctor-site') ?? '');
	let testedSite = $state<string>(recall('netdoctor-site') ?? '');

	let steps = $derived(isRunning ? live : (report?.steps ?? []));
	let links = $derived(
		LINKS.filter((l) => l.id !== 'site' || (isRunning ? site.trim() : testedSite))
	);
	let hostsChanges = $derived((report?.hosts ?? []).filter((h) => h.kind !== 'local'));
	let problems = $derived(steps.filter((s) => s.status === 'warn' || s.status === 'fail'));

	const stepFor = (id: string) => steps.find((s) => s.id === id);

	async function run() {
		if (isRunning) return;
		isRunning = true;
		live = [];
		const target = site.trim();
		try {
			const result = (await RunNetworkDoctor(target)) as Report;
			report = result;
			testedSite = target;
			remember('netdoctor', result);
			remember('netdoctor-site', target);
		} catch (err) {
			console.error('Network check failed:', err);
		} finally {
			isRunning = false;
		}
	}

	onMount(() => {
		const off = EventsOn('netdoctor-step', (s: Step) => {
			if (isRunning) live = [...live, s];
		});
		if (!report) run();
		return off;
	});
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<ToolHeader
		title="Network doctor"
		subtitle="Finds where the connection breaks, one link at a time"
	>
		{#snippet actions()}
			<form
				class="flex items-center gap-2 shrink-0"
				onsubmit={(e) => {
					e.preventDefault();
					run();
				}}
			>
				<input
					type="text"
					placeholder="Also test a site, e.g. github.com"
					bind:value={site}
					spellcheck="false"
					class="text-xs px-3 py-1.5 rounded-lg border border-line bg-tile text-ink focus:outline-none focus:ring-1 focus:ring-accent w-56"
				/>
				<button class="btn btn-primary" type="submit" disabled={isRunning}>
					{isRunning ? 'Checking...' : report ? 'Check again' : 'Check'}
				</button>
			</form>
		{/snippet}
	</ToolHeader>

	<!-- the chain: each link lights up as it is checked -->
	<div class="flex items-center gap-1 px-2 pt-2 shrink-0" aria-label="Connection chain">
		{#each links as link, i (link.id)}
			{@const s = stepFor(link.id)}
			{@const pending = isRunning && !s}
			{@const current = pending && i === live.length}
			{#if i > 0}
				<div class="flex-1 h-0.5 rounded-full relative overflow-hidden bg-track">
					<div
						class="absolute inset-y-0 left-0 rounded-full transition-[width] duration-300 {s?.status ===
						'fail'
							? 'bg-red-500'
							: s?.status === 'skip'
								? 'bg-track'
								: 'bg-green-500'}"
						style="width: {s ? '100%' : '0%'}"
					></div>
					{#if current}<div
							class="flow absolute inset-y-0 w-1/3 bg-muted/60 rounded-full"
						></div>{/if}
				</div>
			{/if}
			<div class="flex flex-col items-center gap-1 w-16 shrink-0">
				<div
					class="relative size-10 rounded-xl grid place-items-center border transition-colors {s
						? s.status === 'fail'
							? 'border-red-500/60 bg-red-500/10'
							: s.status === 'warn'
								? 'border-amber-500/60 bg-amber-500/10'
								: s.status === 'ok'
									? 'border-green-500/50 bg-green-500/10'
									: 'border-line bg-tile'
						: 'border-line bg-tile'} {current ? 'pulse' : ''}"
				>
					<span class={s ? TONE[s.status] : 'text-muted'}>
						<StrokeIcon d={link.icon} size={18} />
					</span>
					{#if s}
						<span
							class="absolute -right-1.5 -bottom-1.5 size-4 rounded-full grid place-items-center bg-tile border border-line {TONE[
								s.status
							]}"
						>
							<StrokeIcon d={MARK[s.status]} size={10} />
						</span>
					{/if}
				</div>
				<span class="text-[11px] text-muted truncate max-w-full" title={s?.name ?? link.label}>
					{link.id === 'site' ? (s?.name ?? (site.trim() || link.label)) : link.label}
				</span>
			</div>
		{/each}
	</div>

	{#if report && !isRunning}
		<div
			class="mx-1 px-3 py-2 rounded-xl text-sm font-semibold shrink-0 {report.healthy &&
			problems.length === 0
				? 'bg-green-500/10 text-ok'
				: report.healthy
					? 'bg-amber-500/10 text-warn'
					: 'bg-red-500/10 text-fail'}"
		>
			{report.summary}
		</div>
	{/if}

	<div class="flex-1 min-h-0 grid grid-cols-[3fr_2fr] gap-3 pt-1">
		<!-- what each check found -->
		<div
			class="min-h-0 overflow-y-auto overscroll-contain pb-5 [mask-image:linear-gradient(to_bottom,#000_calc(100%-20px),transparent)] pr-1 flex flex-col gap-1.5"
		>
			{#each steps as s (s.id)}
				<div class="rounded-xl bg-tile border border-line px-3 py-2">
					<div class="flex items-center gap-2">
						<span class={TONE[s.status]}><StrokeIcon d={MARK[s.status]} size={13} /></span>
						<span class="text-xs font-semibold text-ink">{s.name}</span>
						{#if s.ms > 0}<span class="ml-auto text-[11px] font-mono text-muted">{s.ms} ms</span
							>{/if}
					</div>
					{#if s.status !== 'skip'}
						<p class="text-xs text-muted mt-0.5 leading-relaxed">{s.detail}</p>
					{:else}
						<p class="text-xs text-muted mt-0.5">Not checked: an earlier link is down.</p>
					{/if}
					{#if s.fix}
						<p class="text-xs text-ink mt-1 leading-relaxed">
							<span class="font-semibold">Try:</span>
							{s.fix}
						</p>
					{/if}
				</div>
			{:else}
				<div class="text-xs text-muted px-1">Checking the network adapter...</div>
			{/each}
		</div>

		<!-- settings that change where traffic goes -->
		<div
			class="min-h-0 overflow-y-auto overscroll-contain pb-5 [mask-image:linear-gradient(to_bottom,#000_calc(100%-20px),transparent)] flex flex-col gap-2"
		>
			{#if report?.adapter}
				<section class="rounded-xl bg-tile border border-line px-3 py-2 text-xs">
					<h3 class="text-[10px] uppercase tracking-wider font-semibold text-muted mb-1">
						Connection
					</h3>
					<dl class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5">
						<dt class="text-muted">Adapter</dt>
						<dd class="text-ink truncate">{report.adapter.name}</dd>
						<dt class="text-muted">Address</dt>
						<dd class="text-ink font-mono">{report.adapter.ipv4}</dd>
						<dt class="text-muted">Router</dt>
						<dd class="text-ink font-mono">{report.adapter.gateway || 'none'}</dd>
						<dt class="text-muted">DNS</dt>
						<dd class="text-ink font-mono truncate" title={report.adapter.dns.join(', ')}>
							{report.adapter.dns.join(', ') || 'none'}
						</dd>
					</dl>
				</section>
			{/if}
			{#if report}
				<section class="rounded-xl bg-tile border border-line px-3 py-2 text-xs">
					<h3 class="text-[10px] uppercase tracking-wider font-semibold text-muted mb-1">Proxy</h3>
					{#if !report.proxy.enabled && !report.proxy.auto_config && !report.proxy.winhttp}
						<p class="text-muted">None. Traffic goes straight out.</p>
					{:else}
						{#if report.proxy.enabled}
							<p class="text-ink">Browsers: <span class="font-mono">{report.proxy.server}</span></p>
						{/if}
						{#if report.proxy.auto_config}
							<p class="text-ink truncate" title={report.proxy.auto_config}>
								Script: <span class="font-mono">{report.proxy.auto_config}</span>
							</p>
						{/if}
						{#if report.proxy.winhttp}
							<p class="text-ink">
								Services: <span class="font-mono">{report.proxy.winhttp}</span>
							</p>
						{/if}
						<p class="text-muted mt-1">
							Normal on work PCs. Elsewhere, an unknown proxy can slow or break sites.
						</p>
					{/if}
				</section>
				<section class="rounded-xl bg-tile border border-line px-3 py-2 text-xs">
					<h3 class="text-[10px] uppercase tracking-wider font-semibold text-muted mb-1">
						Hosts file
					</h3>
					{#if hostsChanges.length === 0}
						<p class="text-muted">No changes. Every name is looked up normally.</p>
					{:else}
						<p class="text-muted">These names skip the normal lookup:</p>
						<ul class="flex flex-col gap-0.5 mt-1">
							{#each hostsChanges as h, i (i)}
								<li class="flex items-baseline gap-2 min-w-0">
									<span class="font-mono text-ink truncate" title={h.names.join(' ')}
										>{h.names.join(', ')}</span
									>
									<span
										class="ml-auto shrink-0 text-[11px] {h.kind === 'block'
											? 'text-fail'
											: 'text-muted'}"
									>
										{h.kind === 'block' ? 'blocked' : `→ ${h.ip}`}
									</span>
								</li>
							{/each}
						</ul>
					{/if}
				</section>
			{/if}
		</div>
	</div>
</div>

<style>
	.flow {
		animation: flow 0.9s linear infinite;
	}
	@keyframes flow {
		from {
			transform: translateX(-100%);
		}
		to {
			transform: translateX(300%);
		}
	}
	.pulse {
		animation: pulse 1s ease-in-out infinite;
	}
	@keyframes pulse {
		50% {
			box-shadow: 0 0 0 4px color-mix(in srgb, var(--accent) 25%, transparent);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.flow,
		.pulse {
			animation: none;
		}
	}
</style>

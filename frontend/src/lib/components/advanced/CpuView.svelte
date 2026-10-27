<script lang="ts">
	import type { cpuinfo } from '$lib/wailsjs/go/models';
	import ToolHeader from '$lib/components/ui/ToolHeader.svelte';
	import { onMount } from 'svelte';
	import { shortCpuName } from '$lib';
	import { load } from '$lib/cache';
	import { GetCPUDetail, GetHardwareSpecs, GetUptimeSeconds } from '$lib/wailsjs/go/main/App';

	let { systemInfo, onOpen }: { systemInfo: any; onOpen: (view: string) => void } = $props();

	type Detail = cpuinfo.Detail;

	const SECONDS = 60;
	let usage = $state<number[]>([]); // one reading a second, oldest first
	let speed = $state<number[]>([]); // MHz, same timing
	let now = $state<Detail | null>(null);
	let uptimeSec = $state(0);
	let cpu = $state<any>(null); // from the hardware inventory: cores, threads, socket
	// the hovered moment, as seconds before now; it stays put as new readings arrive
	let hoverAgo = $state<number | null>(null);
	let hover = $derived(
		hoverAgo !== null && usage.length - 1 - hoverAgo >= 0 ? usage.length - 1 - hoverAgo : null
	);

	let model = $derived(
		cpu?.name ? shortCpuName(cpu.name) : shortCpuName(systemInfo?.cpu_model ?? '')
	);
	let ghz = (mhz: number) => (mhz / 1000).toFixed(2);
	let uptimeText = $derived.by(() => {
		const d = Math.floor(uptimeSec / 86400);
		const h = Math.floor((uptimeSec % 86400) / 3600);
		const m = Math.floor((uptimeSec % 3600) / 60);
		return d ? `${d} d ${h} h` : h ? `${h} h ${m} min` : `${m} min`;
	});

	// the usage graph
	const W = 460;
	const H = 150;
	const x = (i: number, n: number) =>
		n <= 1 ? W : (i / (SECONDS - 1)) * W + (SECONDS - n) * (W / (SECONDS - 1));
	const y = (v: number) => H - (v / 100) * H;
	let line = $derived(
		usage
			.map((v, i) => `${i ? 'L' : 'M'}${x(i, usage.length).toFixed(1)},${y(v).toFixed(1)}`)
			.join('')
	);
	let area = $derived(
		usage.length > 1 ? `${line}L${W},${H}L${x(0, usage.length).toFixed(1)},${H}Z` : ''
	);

	// the speed sparkline
	let speedMax = $derived(Math.max(now?.base_mhz ?? 0, ...speed) * 1.1 || 1);
	let speedLine = $derived(
		speed
			.map(
				(v, i) =>
					`${i ? 'L' : 'M'}${((i / Math.max(1, speed.length - 1)) * 100).toFixed(1)},${(28 - (v / speedMax) * 26).toFixed(1)}`
			)
			.join('')
	);

	const coreTone = (v: number) =>
		v >= 90 ? 'fill-red-500' : v >= 70 ? 'fill-amber-500' : 'fill-accent';

	async function tick() {
		const d = (await GetCPUDetail().catch(() => null)) as Detail | null;
		if (!d) return;
		now = d;
		usage = [...usage, d.usage].slice(-SECONDS);
		if (d.current_mhz > 0) speed = [...speed, d.current_mhz].slice(-SECONDS);
	}

	function onMove(e: PointerEvent) {
		const r = (e.currentTarget as SVGSVGElement).getBoundingClientRect();
		const px = ((e.clientX - r.left) / r.width) * W;
		const n = usage.length;
		if (!n) return;
		let best = 0;
		for (let i = 0; i < n; i++) if (Math.abs(x(i, n) - px) < Math.abs(x(best, n) - px)) best = i;
		hoverAgo = n - 1 - best;
	}

	onMount(() => {
		tick();
		GetUptimeSeconds()
			.then((s) => (uptimeSec = s || 0))
			.catch(() => {});
		load('specs', GetHardwareSpecs)
			.then((s: any) => (cpu = s?.cpus?.[0] ?? null))
			.catch(() => {});
		const timer = setInterval(() => {
			tick();
			uptimeSec += 1;
		}, 1000);
		return () => clearInterval(timer);
	});
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<ToolHeader title="Processor">
		{#snippet sub()}
			{model || 'Live use, speed and activity'}
		{/snippet}
		{#snippet actions()}
			<div class="flex items-center gap-2 shrink-0">
				<button class="btn btn-secondary" onclick={() => onOpen('stress')}>Stress test</button>
				<button class="btn btn-primary" onclick={() => onOpen('slow')}>Why so slow?</button>
			</div>
		{/snippet}
	</ToolHeader>

	<div class="flex-1 min-h-0 grid grid-cols-[1fr_210px] gap-3">
		<!-- live usage, last minute -->
		<section class="min-h-0 flex flex-col gap-2">
			<div class="rounded-xl bg-ground px-3 pt-2.5 pb-2 flex flex-col gap-1.5">
				<div class="flex items-baseline justify-between h-8">
					<span class="text-xs text-muted">Use, last 60 seconds</span>
					<span class="flex items-baseline gap-1.5">
						<span class="text-[11px] text-muted w-14 text-right">
							{hover !== null ? (hoverAgo === 0 ? 'now' : `${hoverAgo} s ago`) : ''}
						</span>
						<!-- number and % together, in a fixed width so nothing shifts as it changes -->
						<span class="w-16 text-right"
							><span class="text-2xl font-semibold text-ink tabular-nums"
								>{(hover !== null ? usage[hover] : (now?.usage ?? 0)).toFixed(0)}</span
							><span class="text-ink">%</span></span
						>
					</span>
				</div>
				<svg
					viewBox="0 0 {W} {H}"
					preserveAspectRatio="none"
					class="w-full h-[130px] touch-none overflow-visible"
					role="img"
					aria-label="Processor use over the last minute, now {(now?.usage ?? 0).toFixed(
						0
					)} percent"
					onpointermove={onMove}
					onpointerleave={() => (hoverAgo = null)}
				>
					{#each [25, 50, 75] as g (g)}
						<line
							x1="0"
							x2={W}
							y1={y(g)}
							y2={y(g)}
							class="stroke-line"
							stroke-width="1"
							vector-effect="non-scaling-stroke"
						/>
					{/each}
					{#if area}
						<path d={area} fill="var(--accent)" opacity="0.12" />
						<path
							d={line}
							fill="none"
							stroke="var(--accent)"
							stroke-width="2"
							stroke-linejoin="round"
							vector-effect="non-scaling-stroke"
						/>
					{/if}
					{#if hover !== null}
						<line
							x1={x(hover, usage.length)}
							x2={x(hover, usage.length)}
							y1="0"
							y2={H}
							class="stroke-muted"
							stroke-width="1"
							stroke-dasharray="3 3"
							vector-effect="non-scaling-stroke"
						/>
					{/if}
				</svg>
				<div class="flex justify-between text-[10px] text-muted">
					<span>60 s ago</span><span>now</span>
				</div>
			</div>

			<!-- each logical processor -->
			<div class="rounded-xl bg-ground px-3 py-2 flex-1 min-h-0 flex flex-col gap-1">
				<span class="text-xs text-muted"
					>Each of the {now?.per_core.length ?? '…'} logical processors</span
				>
				<svg
					viewBox="0 0 {Math.max(1, now?.per_core.length ?? 1) * 20} 40"
					preserveAspectRatio="none"
					class="w-full flex-1 min-h-[34px]"
					aria-hidden="true"
				>
					{#each now?.per_core ?? [] as v, i (i)}
						<rect x={i * 20 + 3} y="0" width="14" height="40" rx="3" class="fill-track" />
						<rect
							x={i * 20 + 3}
							y={40 - (v / 100) * 40}
							width="14"
							height={(v / 100) * 40}
							rx="3"
							class="{coreTone(v)} transition-all duration-500"
						/>
					{/each}
				</svg>
			</div>
		</section>

		<!-- the numbers -->
		<section class="min-h-0 flex flex-col gap-2">
			<div class="rounded-xl bg-ground px-3 py-2">
				<div class="text-xs text-muted">Speed now</div>
				<div class="flex items-end justify-between gap-2">
					<span class="text-lg font-semibold text-ink tabular-nums">
						{now?.current_mhz
							? `${ghz(now.current_mhz)} GHz`
							: now?.base_mhz
								? `${ghz(now.base_mhz)} GHz`
								: '–'}
					</span>
					<svg viewBox="0 0 100 30" preserveAspectRatio="none" class="w-20 h-7" aria-hidden="true">
						{#if speed.length > 1}
							<path
								d={speedLine}
								fill="none"
								stroke="var(--accent)"
								stroke-width="1.75"
								vector-effect="non-scaling-stroke"
							/>
						{/if}
					</svg>
				</div>
				<div class="text-[11px] text-muted">
					{#if now?.current_mhz && now.base_mhz}
						Base {ghz(now.base_mhz)} GHz{now.current_mhz > now.base_mhz * 1.05 ? ' · boosting' : ''}
					{:else}
						Windows does not report the live speed here
					{/if}
				</div>
			</div>
			<div class="grid grid-cols-2 gap-2">
				{#each [{ label: 'Cores', value: cpu?.cores ? `${cpu.cores}` : systemInfo?.cpu_cores ? `${systemInfo.cpu_cores}` : '–' }, { label: 'Threads', value: cpu?.threads ? `${cpu.threads}` : now ? `${now.per_core.length}` : '–' }, { label: 'Programs', value: now && now.processes >= 0 ? now.processes.toLocaleString() : '–' }, { label: 'Running tasks', value: now && now.threads >= 0 ? now.threads.toLocaleString() : '–' }] as s (s.label)}
					<div class="rounded-xl bg-ground px-3 py-1.5">
						<div class="text-[11px] text-muted">{s.label}</div>
						<div class="font-semibold text-ink tabular-nums">{s.value}</div>
					</div>
				{/each}
			</div>
			<div class="rounded-xl bg-ground px-3 py-1.5">
				<div class="text-[11px] text-muted">Running since last restart</div>
				<div class="font-semibold tabular-nums {uptimeSec > 14 * 86400 ? 'text-warn' : 'text-ink'}">
					{uptimeSec ? uptimeText : '–'}
				</div>
			</div>
			{#if cpu?.socket}
				<div class="text-[11px] text-muted px-1">Socket {cpu.socket}</div>
			{/if}
		</section>
	</div>
</div>

<script lang="ts">
	import { onMount } from 'svelte';
	import { GetBatteryReport } from '$lib/wailsjs/go/main/App';
	import { load, recall } from '$lib/cache';
	import BusyArt from '../ui/BusyArt.svelte';

	type Battery = {
		name: string;
		chemistry: string;
		design_mwh: number;
		full_mwh: number;
		health_pct: number;
		cycles: number;
	};
	type Point = { date: string; full_mwh: number; design_mwh: number; health_pct: number };
	type Session = { start: string; minutes: number; used_pct: number; full_charge_minutes: number };
	type Report = {
		has_battery: boolean;
		batteries: Battery[];
		health_pct: number;
		runtime_now_min: number;
		runtime_new_min: number;
		history: Point[];
		sessions: Session[];
		tracked_since: string;
		loss_per_year: number;
		summary: string;
	};

	let report = $state<Report | null>(recall('battery-report') ?? null);
	let isLoading = $state(!recall('battery-report'));
	let error = $state('');
	let hover = $state<number | null>(null);

	async function refresh(force = false) {
		isLoading = force || !report;
		error = '';
		try {
			report = (await load('battery-report', GetBatteryReport, 10 * 60_000, force)) as Report;
		} catch (err) {
			error = `${err}`;
		} finally {
			isLoading = false;
		}
	}

	const hours = (min: number) => {
		if (!min) return 'n/a';
		const h = Math.floor(min / 60);
		const m = min % 60;
		return h ? `${h} h ${m.toString().padStart(2, '0')} min` : `${m} min`;
	};
	const wh = (mwh: number) => `${(mwh / 1000).toFixed(1)} Wh`;
	const dateText = (d: string) =>
		new Date(d).toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' });
	const dayText = (d: string) =>
		new Date(d).toLocaleString(undefined, {
			weekday: 'short',
			day: 'numeric',
			month: 'short',
			hour: 'numeric',
			minute: '2-digit'
		});

	// the health chart: one line, % of original capacity over time
	const W = 560;
	const H = 120;
	const PAD = { l: 34, r: 8, t: 8, b: 20 };
	let chart = $derived.by(() => {
		const h = report?.history ?? [];
		if (h.length < 2) return null;
		const times = h.map((p) => new Date(p.date).getTime());
		const t0 = times[0];
		const t1 = times[times.length - 1];
		const lowest = Math.min(...h.map((p) => p.health_pct));
		const yMin = Math.max(0, Math.floor((lowest - 5) / 10) * 10);
		const yMax = 100;
		const x = (t: number) => PAD.l + ((t - t0) / Math.max(1, t1 - t0)) * (W - PAD.l - PAD.r);
		const y = (v: number) =>
			PAD.t + (1 - (Math.min(v, yMax) - yMin) / (yMax - yMin)) * (H - PAD.t - PAD.b);
		const points = h.map((p, i) => ({ x: x(times[i]), y: y(p.health_pct), p }));
		const ticks: number[] = [];
		for (let v = yMin; v <= yMax; v += yMax - yMin > 40 ? 20 : 10) ticks.push(v);
		return {
			points,
			ticks,
			y,
			path: points.map((p, i) => `${i ? 'L' : 'M'}${p.x.toFixed(1)},${p.y.toFixed(1)}`).join('')
		};
	});

	function onMove(e: PointerEvent) {
		if (!chart) return;
		const svg = e.currentTarget as SVGSVGElement;
		const rect = svg.getBoundingClientRect();
		const px = ((e.clientX - rect.left) / rect.width) * W;
		let best = 0;
		chart.points.forEach((p, i) => {
			if (Math.abs(p.x - px) < Math.abs(chart!.points[best].x - px)) best = i;
		});
		hover = best;
	}

	onMount(() => {
		refresh();
	});
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<div class="flex justify-between items-center gap-3 pb-2 border-b border-line shrink-0">
		<div class="min-w-0">
			<div class="font-bold text-base text-ink">Battery</div>
			<div class="text-xs text-muted truncate">
				Real capacity, as the battery reports it to Windows{report?.tracked_since
					? `, recorded since ${dateText(report.tracked_since)}`
					: ''}
			</div>
		</div>
		<button class="btn btn-secondary shrink-0" disabled={isLoading} onclick={() => refresh(true)}>
			Refresh
		</button>
	</div>

	{#if isLoading}
		<div class="flex-1 flex items-center justify-center">
			<BusyArt kind="read" size={64} label="Reading Windows' battery report..." />
		</div>
	{:else if error}
		<div class="flex-1 flex items-center justify-center text-xs text-muted text-center px-8">
			{error}
		</div>
	{:else if report && !report.has_battery}
		<div class="flex-1 flex items-center justify-center text-muted text-center px-8">
			This PC has no battery, so there is nothing to report.
		</div>
	{:else if report}
		<div
			class="flex-1 min-h-0 overflow-y-auto overscroll-contain pb-5 [mask-image:linear-gradient(to_bottom,#000_calc(100%-20px),transparent)] pr-1.5 flex flex-col gap-3"
		>
			<!-- headline -->
			<div class="flex items-center gap-4 px-1">
				<div class="shrink-0 text-center">
					<div class="text-3xl font-bold text-ink leading-none">
						{Math.round(report.health_pct)}%
					</div>
					<div class="text-[11px] text-muted mt-1">of original capacity</div>
				</div>
				<p class="text-sm text-ink leading-relaxed">{report.summary}</p>
			</div>

			<div class="grid grid-cols-4 gap-2">
				{#each [{ label: 'Full charge now', value: wh(report.batteries.reduce((s, b) => s + b.full_mwh, 0)), sub: `was ${wh(report.batteries.reduce((s, b) => s + b.design_mwh, 0))} new` }, { label: 'Lasts about', value: hours(report.runtime_now_min), sub: `was ${hours(report.runtime_new_min)} new` }, { label: 'Charge cycles', value: report.batteries[0].cycles >= 0 ? `${report.batteries[0].cycles}` : 'Not reported', sub: report.batteries[0].cycles >= 0 ? 'counted by the battery' : 'this battery does not count them' }, { label: 'Wear per year', value: report.loss_per_year > 0 ? `${report.loss_per_year.toFixed(1)}%` : 'Too early', sub: report.loss_per_year > 0 ? 'of original capacity' : 'needs 3 months of history' }] as s (s.label)}
					<div class="rounded-xl bg-tile border border-line px-3 py-2">
						<div class="text-[10px] uppercase tracking-wider font-semibold text-muted">
							{s.label}
						</div>
						<div class="text-sm font-bold text-ink mt-0.5">{s.value}</div>
						<div class="text-[11px] text-muted">{s.sub}</div>
					</div>
				{/each}
			</div>

			{#if chart}
				<section class="rounded-xl bg-tile border border-line px-3 py-2">
					<div class="flex items-baseline justify-between">
						<h3 class="text-xs font-semibold text-ink">Capacity over time</h3>
						<span class="text-[11px] text-muted">
							{#if hover !== null}
								{dateText(chart.points[hover].p.date)} ·
								<span class="text-ink font-semibold"
									>{chart.points[hover].p.health_pct.toFixed(1)}%</span
								>
								({wh(chart.points[hover].p.full_mwh)})
							{:else}
								Point at the line for a week's reading
							{/if}
						</span>
					</div>
					<svg
						viewBox="0 0 {W} {H}"
						class="w-full h-auto mt-1 touch-none"
						role="img"
						aria-label="Battery capacity as a share of new, from {dateText(
							report.history[0].date
						)} ({report.history[0].health_pct}%) to {dateText(
							report.history[report.history.length - 1].date
						)} ({report.history[report.history.length - 1].health_pct}%)"
						onpointermove={onMove}
						onpointerleave={() => (hover = null)}
					>
						{#each chart.ticks as t (t)}
							<line
								x1={PAD.l}
								x2={W - PAD.r}
								y1={chart.y(t)}
								y2={chart.y(t)}
								class="stroke-line"
								stroke-width="1"
							/>
							<text
								x={PAD.l - 6}
								y={chart.y(t) + 3.5}
								text-anchor="end"
								class="fill-muted text-[10px]">{t}%</text
							>
						{/each}
						<text x={PAD.l} y={H - 4} class="fill-muted text-[10px]"
							>{dateText(report.history[0].date)}</text
						>
						<text x={W - PAD.r} y={H - 4} text-anchor="end" class="fill-muted text-[10px]">
							{dateText(report.history[report.history.length - 1].date)}
						</text>
						<path
							d={chart.path}
							fill="none"
							stroke="var(--accent)"
							stroke-width="2"
							stroke-linejoin="round"
							stroke-linecap="round"
						/>
						{#if hover !== null}
							{@const p = chart.points[hover]}
							<line
								x1={p.x}
								x2={p.x}
								y1={PAD.t}
								y2={H - PAD.b}
								class="stroke-muted"
								stroke-width="1"
								stroke-dasharray="3 3"
							/>
							<circle
								cx={p.x}
								cy={p.y}
								r="4"
								fill="var(--accent)"
								class="stroke-tile"
								stroke-width="2"
							/>
						{/if}
					</svg>
				</section>
			{/if}

			{#if report.sessions.length}
				<section class="flex flex-col gap-1">
					<h3 class="text-[10px] uppercase tracking-wider font-semibold text-muted px-1">
						Recent time on battery
					</h3>
					<table class="text-xs w-full">
						<thead class="sr-only">
							<tr
								><th>Started</th><th>Lasted</th><th>Charge used</th><th
									>A full charge at that pace</th
								></tr
							>
						</thead>
						<tbody>
							{#each report.sessions as s (s.start)}
								<tr class="border-b border-line/60 last:border-0">
									<td class="py-1 px-1 text-ink">{dayText(s.start)}</td>
									<td class="py-1 px-1 text-muted">{hours(s.minutes)}</td>
									<td class="py-1 px-1 text-muted">{s.used_pct.toFixed(0)}% used</td>
									<td class="py-1 px-1 text-right text-muted">
										a full charge would last <span class="text-ink font-semibold"
											>{hours(s.full_charge_minutes)}</span
										>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</section>
			{/if}

			<p class="text-[11px] text-muted px-1 pb-1">
				{report.batteries.map((b) => `${b.name} · ${b.chemistry}`).join(' · ')}
			</p>
		</div>
	{/if}
</div>

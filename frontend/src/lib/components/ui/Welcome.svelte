<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { fly } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { finishWelcome } from '$lib/onboarding.svelte';
	import { setTheme, theme, type ThemeMode } from '$lib/theme.svelte';
	import { AUDIENCES, loadAudience, saveAudience, toggleAudience, type Audience } from '$lib/tools';
	import Logo from './Logo.svelte';
	import Segmented from './Segmented.svelte';
	import StrokeIcon from './StrokeIcon.svelte';

	// Shown on the first start after installing, and from Settings. The first
	// page is the logo; the next three show what the app does, each with a short
	// drawing that plays once; the last lets people pick their theme and the
	// kinds of tools they want to see.
	const reduceMotion =
		typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;

	const LAST = 4;
	let step = $state(0);
	// which way the pages slide
	let direction = $state(1);
	let primary: HTMLButtonElement | undefined = $state();

	const CHECK = 'M5 12.5l4.5 4.5L19 7.5';

	const pages = [
		{
			eyebrow: 'Your PC, explained',
			title: 'Real numbers, in plain words',
			text: 'SysInfo Pro reads what Windows and your drives report and tells you what it means. Readings match Task Manager, and nothing is guessed.',
			points: [
				'A health score you can check',
				'Live processor, memory and storage',
				'Works without administrator rights'
			]
		},
		{
			eyebrow: 'When something is wrong',
			title: 'Find out why. Then fix it.',
			text: 'When the PC feels slow, the drive fills up or the internet stops working, one click shows what is behind it, the biggest cause first, and what to try.',
			points: ['Why so slow', 'Disk usage and duplicates', 'Network doctor']
		},
		{
			eyebrow: 'If you write code',
			title: 'Built for developers too',
			text: 'Find work that exists only on this PC, see which python or node really runs, and free gigabytes of old caches and build folders.',
			points: ['Git projects', 'Dev doctor', 'Ports and dev caches']
		}
	];

	// page 1: a live line, like the processor graph
	const SPARK =
		'M0 26 L18 24 L34 27 L50 18 L66 22 L82 12 L98 19 L114 15 L130 23 L146 9 L162 16 L178 13 L194 20 L210 11 L226 17 L242 8 L258 14 L274 10 L290 12';
	const RING_R = 35;
	const RING = 2 * Math.PI * RING_R;
	const BARS = [
		{ label: 'Processor', value: '23%', pct: 23 },
		{ label: 'Memory', value: '61%', pct: 61 },
		{ label: 'Storage', value: '89 GB free', pct: 55 }
	];

	// page 2: running programs as bars; the tallest is the culprit
	const PROGRAMS = [18, 26, 50, 22, 31, 14];

	// page 3: what the developer tools find
	const DEV_ROWS = [
		{
			icon: 'M6 3v12M18 9a3 3 0 1 0 0-6a3 3 0 0 0 0 6zM6 21a3 3 0 1 0 0-6a3 3 0 0 0 0 6zM18 9a9 9 0 0 1-9 9',
			name: 'payments-api',
			detail: '13 commits never pushed',
			pill: 'pill-fail',
			tag: 'Only here'
		},
		{
			icon: 'M4 17l6-5-6-5M12 19h8',
			name: 'python',
			detail: 'Opens the Microsoft Store',
			pill: 'pill-warn',
			tag: 'Fix PATH'
		},
		{
			icon: 'M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z',
			name: 'node_modules',
			detail: '4 old projects · 4.6 GB',
			pill: 'pill-info',
			tag: 'Free'
		}
	];

	const modes: { id: ThemeMode; label: string }[] = [
		{ id: 'system', label: 'System' },
		{ id: 'light', label: 'Light' },
		{ id: 'dark', label: 'Dark' },
		{ id: 'amoled', label: 'AMOLED' }
	];
	const systemIsDark =
		typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches;
	let audience = $state<Audience[]>(loadAudience());
	const AUDIENCE_HINT: Record<Audience, string> = {
		everyday: 'Health, cleanup, security',
		developer: 'Git, PATH, ports, caches',
		gaming: 'Processor, startup, stress test'
	};

	function pickAudience(id: Audience) {
		audience = toggleAudience(audience, id);
		saveAudience(audience);
	}

	// counts a number up from 0, in step with the ring filling
	function countUp(node: HTMLElement, to: number) {
		if (reduceMotion) {
			node.textContent = String(to);
			return;
		}
		const start = performance.now() + 250;
		const duration = 1100;
		let frame = 0;
		const draw = (now: number) => {
			const t = Math.min(1, Math.max(0, (now - start) / duration));
			node.textContent = String(Math.round(to * (1 - Math.pow(1 - t, 3))));
			if (t < 1) frame = requestAnimationFrame(draw);
		};
		frame = requestAnimationFrame(draw);
		return { destroy: () => cancelAnimationFrame(frame) };
	}

	// keyboard focus lands on the main button without drawing a focus ring
	function focusPrimary() {
		primary?.focus({ focusVisible: false } as FocusOptions);
	}

	async function go(to: number) {
		if (to < 0) return;
		if (to > LAST) {
			finishWelcome();
			return;
		}
		direction = to > step ? 1 : -1;
		step = to;
		await tick();
		focusPrimary();
	}

	function onKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') finishWelcome();
		else if (event.key === 'ArrowRight') go(step + 1);
		else if (event.key === 'ArrowLeft') go(step - 1);
	}

	onMount(focusPrimary);
</script>

<svelte:window onkeydown={onKeydown} />

<div
	class="fixed inset-0 z-[70] flex flex-col bg-ground text-ink text-xs select-none overflow-hidden"
	class:still={reduceMotion}
	role="dialog"
	aria-modal="true"
	aria-label="Welcome to SysInfo Pro"
>
	{#if step === 0}
		<div class="glow pointer-events-none absolute inset-0" aria-hidden="true"></div>
	{/if}

	<header class="drag-region relative flex items-center justify-between h-10 shrink-0 px-4 mt-3">
		<div class="min-w-0">
			{#if step > 0}
				<Logo size={24} />
			{/if}
		</div>
		<button class="ghost small" onclick={finishWelcome}>Skip</button>
	</header>

	<main class="relative flex-1 min-h-0">
		{#key step}
			<div
				class="absolute inset-0 px-10 pb-2"
				in:fly={{ x: 16 * direction, duration: reduceMotion ? 0 : 220, easing: cubicOut }}
			>
				{#if step === 0}
					<!-- the mark zooms out to its size, then the name slides out beside it -->
					<div class="h-full flex flex-col items-center justify-center gap-9 pb-6">
						<div class="flex items-center" role="img" aria-label="SysInfo Pro">
							<svg
								class="intro-mark"
								width="64"
								height="64"
								viewBox="0 0 24 24"
								fill="none"
								aria-hidden="true"
							>
								<rect x="1" y="1" width="22" height="22" rx="6" class="fill-accent" />
								<path
									d="M4.5 12.5h3.2l2.1-5 3.4 9.5 2.2-4.5h4.1"
									class="stroke-on-accent draw"
									style="--at: 650ms; --for: 600ms"
									pathLength="1"
									stroke-width="2"
									stroke-linecap="round"
									stroke-linejoin="round"
								/>
							</svg>
							<span class="intro-word" aria-hidden="true">
								<span
									class="inline-block pl-4 py-2 text-[44px] font-bold tracking-tight leading-none"
								>
									SysInfo <span class="font-medium text-muted">Pro</span>
								</span>
							</span>
						</div>
						<button
							bind:this={primary}
							class="primary rise"
							style="--at: 1650ms"
							onclick={() => go(1)}
						>
							Get started
						</button>
					</div>
				{:else if step <= pages.length}
					{@const page = pages[step - 1]}
					<div class="h-full grid grid-cols-[1fr_330px] items-center gap-10">
						<div class="flex flex-col gap-3 min-w-0">
							<span class="text-accent font-semibold uppercase tracking-wider text-[11px]">
								{page.eyebrow}
							</span>
							<h1 class="text-[26px] font-bold tracking-tight leading-[1.15]">{page.title}</h1>
							<p class="text-[13px] text-muted leading-relaxed">{page.text}</p>
							<ul class="flex flex-col gap-1.5 mt-1">
								{#each page.points as point, i (point)}
									<li
										class="rise flex items-center gap-2 text-[13px] font-medium"
										style="--at: {250 + i * 90}ms"
									>
										<span
											class="w-5 h-5 rounded-full bg-accent/15 text-accent grid place-items-center shrink-0"
										>
											<StrokeIcon d={CHECK} size={12} />
										</span>
										{point}
									</li>
								{/each}
							</ul>
						</div>

						<!-- a small picture of the app doing it, drawn once -->
						<div
							class="bg-tile rounded-2xl p-4 shadow-xl border border-line flex flex-col gap-3"
							aria-hidden="true"
						>
							{#if step === 1}
								<div class="flex items-center gap-4">
									<span class="relative w-[84px] h-[84px] text-ok shrink-0">
										<svg width="84" height="84" viewBox="0 0 84 84" fill="none" class="-rotate-90">
											<circle cx="42" cy="42" r={RING_R} stroke-width="8" class="stroke-track" />
											<circle
												class="ring"
												cx="42"
												cy="42"
												r={RING_R}
												stroke="currentColor"
												stroke-width="8"
												stroke-linecap="round"
												style="--to: {(RING * 87) / 100}px"
											/>
										</svg>
										<span
											class="absolute inset-0 grid place-items-center text-ink text-[22px] font-semibold tabular-nums"
											use:countUp={87}>0</span
										>
									</span>
									<span class="rise flex flex-col gap-0.5" style="--at: 900ms">
										<span class="text-[15px] font-semibold">Good</span>
										<span class="text-muted">Memory in use −6</span>
									</span>
								</div>
								<svg viewBox="0 0 290 34" class="w-full h-[34px] overflow-visible text-accent">
									<path
										d={SPARK}
										class="draw"
										style="--at: 300ms; --for: 1300ms"
										pathLength="1"
										fill="none"
										stroke="currentColor"
										stroke-width="2"
										stroke-linejoin="round"
										stroke-linecap="round"
									/>
									<circle cx="290" cy="12" r="3.5" class="pop fill-accent" style="--at: 1550ms" />
								</svg>
								{#each BARS as bar, i (bar.label)}
									<div class="flex flex-col gap-1">
										<span class="flex justify-between">
											<span class="text-muted">{bar.label}</span>
											<span class="font-semibold tabular-nums">{bar.value}</span>
										</span>
										<span class="block h-[5px] rounded-full bg-track">
											<span
												class="grow block h-[5px] rounded-full bg-accent"
												style="width: {bar.pct}%; --at: {500 + i * 150}ms"
											></span>
										</span>
									</div>
								{/each}
							{:else if step === 2}
								<span class="font-semibold text-[13px]">Why so slow</span>
								<!-- a magnifier sweeps the running programs and stops on the culprit -->
								<svg viewBox="0 0 298 64" class="w-full h-[64px] overflow-visible">
									<line x1="0" y1="62" x2="298" y2="62" class="stroke-line" stroke-width="1.5" />
									{#each PROGRAMS as h, i (i)}
										<rect
											x={12 + i * 48}
											y={60 - h}
											width="28"
											height={h}
											rx="4"
											class="bar {h === 50 ? 'culprit' : ''}"
											style="--at: {i * 70}ms"
										/>
									{/each}
									<g class="lens">
										<circle
											cx="0"
											cy="0"
											r="12"
											fill="none"
											class="stroke-ink"
											stroke-width="2.5"
										/>
										<path d="M9 9l8 8" class="stroke-ink" stroke-width="3" stroke-linecap="round" />
									</g>
								</svg>
								<span
									class="rise px-2.5 py-1.5 rounded-lg bg-fail/10 text-fail font-semibold"
									style="--at: 1500ms"
								>
									2 things are slowing the PC down
								</span>
								{#each [{ title: 'WSL is holding 7.8 GB of memory', pill: 'pill-fail', effect: 'Big effect', fix: 'Run wsl --shutdown when you are done' }, { title: 'Microsoft Defender is scanning', pill: 'pill-warn', effect: 'Some effect', fix: 'Let it finish; it takes 10 to 30 minutes' }] as row, i (row.title)}
									<div
										class="rise flex flex-col gap-1 rounded-xl border border-line p-2.5"
										style="--at: {1650 + i * 150}ms"
									>
										<span class="flex items-center justify-between gap-2">
											<span class="font-semibold truncate">{row.title}</span>
											<span
												class="{row.pill} shrink-0 px-2 py-0.5 rounded-md text-[11px] font-semibold"
												>{row.effect}</span
											>
										</span>
										<span class="text-muted"
											><span class="text-ink font-semibold">Try:</span> {row.fix}</span
										>
									</div>
								{/each}
							{:else}
								<!-- a branch whose last commits never left this PC -->
								<svg viewBox="0 0 298 58" class="w-full h-[58px] overflow-visible">
									<path
										d="M8 44H290"
										class="draw stroke-line"
										style="--at: 100ms; --for: 600ms"
										pathLength="1"
										stroke-width="2.5"
										stroke-linecap="round"
										fill="none"
									/>
									<path
										d="M120 44C150 44 150 14 180 14H284"
										class="draw stroke-fail"
										style="--at: 650ms; --for: 600ms"
										pathLength="1"
										stroke-width="2.5"
										stroke-linecap="round"
										fill="none"
									/>
									{#each [30, 75, 120] as x, i (x)}
										<circle
											cx={x}
											cy="44"
											r="5"
											class="pop fill-tile stroke-accent"
											stroke-width="2.5"
											style="--at: {250 + i * 120}ms"
										/>
									{/each}
									{#each [196, 236, 276] as x, i (x)}
										<circle
											cx={x}
											cy="14"
											r="5"
											class="pop fill-fail"
											style="--at: {1050 + i * 120}ms"
										/>
									{/each}
									<text
										x="196"
										y="36"
										class="rise fill-fail text-[10px] font-semibold"
										style="--at: 1400ms">not pushed</text
									>
								</svg>
								{#each DEV_ROWS as row, i (row.name)}
									<div
										class="rise flex items-center gap-2.5 rounded-xl border border-line p-2.5"
										style="--at: {1400 + i * 160}ms"
									>
										<svg
											width="20"
											height="20"
											viewBox="0 0 24 24"
											fill="none"
											class="text-muted shrink-0"
										>
											<path
												d={row.icon}
												class="draw"
												style="--at: {1500 + i * 160}ms; --for: 700ms"
												pathLength="1"
												stroke="currentColor"
												stroke-width="1.8"
												stroke-linecap="round"
												stroke-linejoin="round"
											/>
										</svg>
										<span class="flex flex-col min-w-0 flex-1">
											<span class="font-mono font-semibold truncate">{row.name}</span>
											<span class="text-muted truncate">{row.detail}</span>
										</span>
										<span
											class="{row.pill} shrink-0 px-2 py-0.5 rounded-md text-[11px] font-semibold"
											>{row.tag}</span
										>
									</div>
								{/each}
							{/if}
						</div>
					</div>
				{:else}
					<div class="h-full flex flex-col justify-center gap-5 max-w-[560px] mx-auto">
						<div class="flex flex-col gap-2">
							<span class="text-accent font-semibold uppercase tracking-wider text-[11px]">
								Make it yours
							</span>
							<h1 class="text-[26px] font-bold tracking-tight leading-[1.15]">
								Set it up your way
							</h1>
							<p class="text-[13px] text-muted">
								You can change both later, in Settings and in All tools.
							</p>
						</div>
						<div class="flex items-center justify-between gap-4">
							<div>
								<div class="font-semibold text-[13px]">Theme</div>
								<div class="text-muted">
									{theme.mode === 'system'
										? `Follows Windows, which is ${systemIsDark ? 'dark' : 'light'} right now`
										: 'AMOLED is pure black for OLED screens'}
								</div>
							</div>
							<Segmented
								options={modes}
								value={theme.mode}
								onchange={setTheme}
								label="Theme"
								size="sm"
							/>
						</div>
						<div class="flex flex-col gap-2">
							<div>
								<div class="font-semibold text-[13px]">Which tools do you want to see?</div>
								<div class="text-muted">Pick any; every tool stays one search away.</div>
							</div>
							<div class="grid grid-cols-3 gap-2" role="group" aria-label="Tools to show">
								{#each AUDIENCES as a (a.id)}
									{@const on = audience.includes(a.id)}
									<button
										aria-pressed={on}
										class="choice flex flex-col items-start gap-0.5 p-3 rounded-xl border text-left cursor-pointer transition-colors {on
											? 'bg-accent/15 border-accent/60'
											: 'bg-tile border-line hover:border-accent/40'}"
										onclick={() => pickAudience(a.id)}
									>
										<span class="flex items-center justify-between w-full">
											<span class="font-semibold text-[13px]">{a.label}</span>
											<span
												class="w-4 h-4 rounded-full grid place-items-center {on
													? 'bg-accent text-on-accent'
													: 'border border-line'}"
											>
												{#if on}<StrokeIcon d={CHECK} size={10} />{/if}
											</span>
										</span>
										<span class="text-muted">{AUDIENCE_HINT[a.id]}</span>
									</button>
								{/each}
							</div>
						</div>
					</div>
				{/if}
			</div>
		{/key}
	</main>

	{#if step > 0}
		<footer class="relative flex items-center justify-between h-14 shrink-0 px-10 pb-2">
			<div class="flex items-center gap-1.5" aria-label="Page {step} of {LAST}">
				{#each Array.from({ length: LAST }, (_, i) => i + 1) as dot (dot)}
					<button
						aria-label="Page {dot}"
						aria-current={dot === step ? 'step' : undefined}
						class="h-1.5 rounded-full cursor-pointer transition-all {dot === step
							? 'w-5 bg-accent'
							: 'w-1.5 bg-line hover:bg-muted'}"
						onclick={() => go(dot)}
					></button>
				{/each}
			</div>
			<div class="flex items-center gap-1">
				<button class="ghost" onclick={() => go(step - 1)}>Back</button>
				<button bind:this={primary} class="primary" onclick={() => go(step + 1)}>
					{step === LAST ? 'Start using SysInfo Pro' : 'Next'}
				</button>
			</div>
		</footer>
	{/if}
</div>

<style>
	/* buttons: a soft accent pill and a quiet text button, no outlines */
	.primary,
	.ghost {
		height: 36px;
		padding: 0 20px;
		border-radius: 999px;
		font-size: 13px;
		font-weight: 600;
		cursor: pointer;
		transition:
			background-color 0.15s,
			color 0.15s,
			filter 0.15s;
	}
	.primary {
		background: var(--accent);
		color: var(--on-accent);
		box-shadow: 0 8px 22px -10px color-mix(in srgb, var(--accent) 80%, transparent);
	}
	.primary:hover {
		filter: brightness(1.07);
	}
	.ghost {
		color: var(--muted);
		padding: 0 14px;
	}
	.ghost:hover {
		color: var(--ink);
		background: var(--line);
	}
	.ghost.small {
		height: 28px;
		font-size: 12px;
	}
	.primary:focus-visible,
	.ghost:focus-visible,
	.choice:focus-visible {
		outline: 2px solid color-mix(in srgb, var(--accent) 60%, transparent);
		outline-offset: 2px;
	}

	.glow {
		background: radial-gradient(
			420px 260px at 50% 46%,
			color-mix(in srgb, var(--accent) 16%, transparent),
			transparent 70%
		);
	}

	/* the mark arrives large and soft, and settles at its size */
	.intro-mark {
		animation: zoom-out 1000ms cubic-bezier(0.33, 1, 0.68, 1) both;
	}
	@keyframes zoom-out {
		from {
			transform: scale(3.4);
			opacity: 0;
			filter: blur(6px);
		}
		55% {
			opacity: 1;
			filter: blur(0);
		}
		to {
			transform: scale(1);
			opacity: 1;
		}
	}

	/* the name opens out from behind the mark, pushing it left */
	.intro-word {
		display: inline-block;
		max-width: 0;
		overflow: hidden;
		white-space: nowrap;
		animation: reveal 750ms 900ms cubic-bezier(0.16, 1, 0.3, 1) forwards;
	}
	@keyframes reveal {
		to {
			max-width: 380px;
		}
	}

	/* a line that draws itself; paths use pathLength="1" */
	.draw {
		stroke-dasharray: 1;
		stroke-dashoffset: 1;
		animation: draw var(--for, 700ms) var(--at, 0ms) cubic-bezier(0.33, 1, 0.68, 1) forwards;
	}
	@keyframes draw {
		to {
			stroke-dashoffset: 0;
		}
	}

	/* fades up into place */
	.rise {
		opacity: 0;
		animation: rise 450ms var(--at, 0ms) cubic-bezier(0.33, 1, 0.68, 1) forwards;
	}
	@keyframes rise {
		from {
			opacity: 0;
			transform: translateY(6px);
		}
		to {
			opacity: 1;
			transform: none;
		}
	}

	/* small things that appear with a slight overshoot */
	.pop {
		transform-box: fill-box;
		transform-origin: center;
		transform: scale(0);
		animation: pop 380ms var(--at, 0ms) cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
	}
	@keyframes pop {
		to {
			transform: scale(1);
		}
	}

	/* page 1: the score ring fills and the bars grow */
	.ring {
		stroke-dasharray: 0 999;
		animation: fill-ring 1100ms 250ms cubic-bezier(0.33, 1, 0.68, 1) forwards;
	}
	@keyframes fill-ring {
		to {
			stroke-dasharray: var(--to) 999;
		}
	}
	.grow {
		transform-origin: left;
		transform: scaleX(0);
		animation: grow 800ms var(--at, 0ms) cubic-bezier(0.33, 1, 0.68, 1) forwards;
	}
	@keyframes grow {
		to {
			transform: scaleX(1);
		}
	}

	/* page 2: programs rise, the magnifier sweeps, the culprit turns red */
	.bar {
		fill: var(--track);
		transform-box: fill-box;
		transform-origin: bottom;
		transform: scaleY(0);
		animation: bar-up 500ms var(--at, 0ms) cubic-bezier(0.33, 1, 0.68, 1) forwards;
	}
	@keyframes bar-up {
		to {
			transform: scaleY(1);
		}
	}
	.culprit {
		animation:
			bar-up 500ms var(--at, 0ms) cubic-bezier(0.33, 1, 0.68, 1) forwards,
			culprit 400ms 1250ms ease-out forwards;
	}
	@keyframes culprit {
		to {
			fill: var(--fail);
		}
	}
	.lens {
		opacity: 0;
		transform: translate(0px, 28px);
		animation: sweep 1300ms 350ms cubic-bezier(0.45, 0, 0.25, 1) forwards;
	}
	@keyframes sweep {
		0% {
			opacity: 0;
			transform: translate(10px, 28px);
		}
		12% {
			opacity: 1;
		}
		55% {
			transform: translate(280px, 28px);
		}
		100% {
			opacity: 1;
			transform: translate(122px, 22px);
		}
	}

	/* Windows' "reduce motion": everything is simply there */
	.still :is(.intro-mark, .intro-word, .draw, .rise, .pop, .ring, .grow, .bar, .culprit, .lens) {
		animation: none;
	}
	.still .draw {
		stroke-dashoffset: 0;
	}
	.still .intro-word {
		max-width: none;
	}
	.still .rise,
	.still .lens {
		opacity: 1;
	}
	.still :is(.pop, .grow, .bar) {
		transform: none;
	}
	.still .ring {
		stroke-dasharray: var(--to) 999;
	}
	.still .lens {
		transform: translate(122px, 22px);
	}
	.still .culprit {
		fill: var(--fail);
	}
</style>

<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { endTour } from '$lib/onboarding.svelte';

	// A short walk around the home screen: each step lights up one part and
	// says, in plain words, what it is and what clicking it does.
	type Step = { target: string; title: string; text: string };

	const STEPS: Step[] = [
		{
			target: '[data-tour="score"]',
			title: 'Health score',
			text: 'One number for how your PC is doing, worked out from real readings. Click it to see what lowers it and how to fix that.'
		},
		{
			target: '[data-tour="cpu"]',
			title: 'Processor',
			text: 'How hard your PC is working right now, the same reading as Task Manager. Click for details, or to ask why it is slow.'
		},
		{
			target: '[data-tour="memory"]',
			title: 'Memory',
			text: 'How much memory is in use. Click to see which programs use the most.'
		},
		{
			target: '[data-tour="storage"]',
			title: 'Storage',
			text: 'Space left on your main drive. Click to find out what is filling it.'
		},
		{
			target: '[data-tour="tools"]',
			title: 'Your tools',
			text: 'Shortcuts to the tools people use most. A number on a tool means it found something to look at.'
		},
		{
			target: '[data-tour="tools"] > :last-child',
			title: 'All tools',
			text: 'Every tool in one place, for everyday use, developers and gamers. Tip: press Ctrl+K anywhere and type what you need.'
		},
		{
			target: '[data-tour="widget"]',
			title: 'Widget',
			text: 'Shrink SysInfo Pro to a small window that stays on top while you work. Ctrl+Alt+W switches back and forth.'
		},
		{
			target: '[data-tour="share"]',
			title: 'Share',
			text: "Copy your PC's details or save a report, handy when you ask someone for help."
		},
		{
			target: '[data-tour="settings"]',
			title: 'Settings',
			text: 'Theme, alerts and where SysInfo Pro keeps its files. You can take this tour again from here.'
		}
	];

	const reduceMotion =
		typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
	const PAD = 6;
	const CARD_W = 280;
	const GAP = 12;

	// steps whose part is on screen (the Share button only shows on the home screen)
	let steps = $state<Step[]>([]);
	let index = $state(0);
	let hole = $state({ top: 0, left: 0, width: 0, height: 0 });
	let card = $state({ top: 0, left: 0 });
	let cardEl: HTMLDivElement | undefined = $state();
	let nextButton: HTMLButtonElement | undefined = $state();

	let step = $derived(steps[index]);
	let last = $derived(index === steps.length - 1);

	async function place() {
		const el = step && document.querySelector(step.target);
		if (!el) return;
		const r = el.getBoundingClientRect();
		hole = {
			top: r.top - PAD,
			left: r.left - PAD,
			width: r.width + PAD * 2,
			height: r.height + PAD * 2
		};
		await tick();
		const h = cardEl?.offsetHeight ?? 140;
		const vw = window.innerWidth;
		const vh = window.innerHeight;
		// below the part if it fits, else above, else beside it
		let top = hole.top + hole.height + GAP;
		let left = hole.left + hole.width / 2 - CARD_W / 2;
		if (top + h > vh - 8) top = hole.top - GAP - h;
		if (top < 8) {
			top = Math.min(Math.max(8, hole.top), vh - h - 8);
			left =
				hole.left + hole.width + GAP + CARD_W < vw
					? hole.left + hole.width + GAP
					: hole.left - GAP - CARD_W;
		}
		card = { top, left: Math.min(Math.max(8, left), vw - CARD_W - 8) };
		// focus for the keyboard, without drawing a focus ring
		nextButton?.focus({ focusVisible: false } as FocusOptions);
	}

	function go(to: number) {
		if (to < 0) return;
		if (to >= steps.length) {
			endTour();
			return;
		}
		index = to;
		place();
	}

	function onKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') endTour();
		else if (event.key === 'ArrowRight') go(index + 1);
		else if (event.key === 'ArrowLeft') go(index - 1);
	}

	onMount(() => {
		// wait a frame so the home screen has laid out
		requestAnimationFrame(() => {
			steps = STEPS.filter((s) => document.querySelector(s.target));
			if (!steps.length) {
				endTour();
				return;
			}
			index = 0;
			place();
		});
		window.addEventListener('resize', place);
		return () => window.removeEventListener('resize', place);
	});
</script>

<svelte:window onkeydown={onKeydown} />

{#if step}
	<div class="fixed inset-0 z-[65]" role="dialog" aria-modal="true" aria-labelledby="tour-title">
		<!-- everything but the lit part is dimmed, and clicks there do nothing -->
		<div class="absolute inset-0"></div>
		<div
			class="spot absolute rounded-2xl pointer-events-none"
			class:move={!reduceMotion}
			style="top: {hole.top}px; left: {hole.left}px; width: {hole.width}px; height: {hole.height}px"
		></div>

		<div
			bind:this={cardEl}
			class="absolute bg-tile text-ink border border-line rounded-2xl shadow-2xl p-4 flex flex-col gap-2 text-xs"
			class:move={!reduceMotion}
			style="top: {card.top}px; left: {card.left}px; width: {CARD_W}px"
		>
			<div class="flex items-center justify-between gap-2">
				<h2 id="tour-title" class="text-[14px] font-bold">{step.title}</h2>
				<span class="text-muted tabular-nums">{index + 1} of {steps.length}</span>
			</div>
			<p class="text-muted leading-relaxed">{step.text}</p>
			<div class="flex items-center justify-between gap-2 mt-1">
				<button class="ghost -ml-2" onclick={endTour}>
					{last ? 'Close' : 'End tour'}
				</button>
				<div class="flex items-center gap-2">
					{#if index > 0}
						<button class="ghost" onclick={() => go(index - 1)}>Back</button>
					{/if}
					<button bind:this={nextButton} class="primary" onclick={() => go(index + 1)}>
						{last ? 'Done' : 'Next'}
					</button>
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	/* the same soft buttons as the welcome screens */
	.primary,
	.ghost {
		height: 30px;
		padding: 0 16px;
		border-radius: 999px;
		font-size: 12px;
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
	}
	.primary:hover {
		filter: brightness(1.07);
	}
	.ghost {
		color: var(--muted);
		padding: 0 12px;
	}
	.ghost:hover {
		color: var(--ink);
		background: var(--line);
	}
	.primary:focus-visible,
	.ghost:focus-visible {
		outline: 2px solid color-mix(in srgb, var(--accent) 60%, transparent);
		outline-offset: 2px;
	}

	.spot {
		box-shadow: 0 0 0 9999px rgb(2 6 23 / 0.62);
		outline: 2px solid var(--accent);
	}
	.move {
		transition:
			top 240ms cubic-bezier(0.16, 1, 0.3, 1),
			left 240ms cubic-bezier(0.16, 1, 0.3, 1),
			width 240ms cubic-bezier(0.16, 1, 0.3, 1),
			height 240ms cubic-bezier(0.16, 1, 0.3, 1);
	}
</style>

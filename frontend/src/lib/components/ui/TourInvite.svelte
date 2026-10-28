<script lang="ts">
	import { onMount } from 'svelte';
	import { fade, scale } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { dismissInvite, startTour } from '$lib/onboarding.svelte';

	// Offered once, after the welcome screens: a short tour of the home screen.
	// The picture is a tiny home screen with the tour's light moving across it.
	const reduceMotion =
		typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
	const ms = reduceMotion ? 0 : 1;

	let takeButton: HTMLButtonElement | undefined = $state();

	function onKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') dismissInvite();
	}

	onMount(() => takeButton?.focus({ focusVisible: false } as FocusOptions));
</script>

<svelte:window onkeydown={onKeydown} />

<div
	class="fixed inset-0 z-[60] grid place-items-center bg-black/45"
	class:still={reduceMotion}
	transition:fade={{ duration: 160 * ms }}
>
	<div
		class="w-[340px] flex flex-col items-center text-center gap-3 px-6 pt-6 pb-5 rounded-2xl bg-tile text-ink border border-line shadow-2xl text-xs"
		role="dialog"
		aria-modal="true"
		aria-labelledby="invite-title"
		in:scale={{ start: 0.96, duration: 220 * ms, easing: cubicOut }}
	>
		<svg viewBox="0 0 132 76" class="w-[132px] h-[76px]" aria-hidden="true">
			<!-- the home screen: a tall score tile, four small tiles, a row of tools -->
			<rect x="2" y="2" width="36" height="50" rx="6" class="fill-ground" />
			<circle cx="20" cy="24" r="9" fill="none" class="stroke-ok" stroke-width="3" />
			{#each [[44, 2], [88, 2], [44, 28], [88, 28]] as [x, y] (`${x}${y}`)}
				<rect {x} {y} width="40" height="24" rx="6" class="fill-ground" />
			{/each}
			{#each [0, 1, 2, 3, 4] as i (i)}
				<rect x={2 + i * 26} y="58" width="22" height="16" rx="5" class="fill-ground" />
			{/each}
			<!-- the tour's light hops from the score to a gauge to the tools -->
			<rect class="spot" x="0" y="0" width="40" height="54" rx="8" fill="none" stroke-width="2" />
		</svg>

		<div class="flex flex-col gap-1">
			<h2 id="invite-title" class="text-[16px] font-bold">New here? Take a quick tour</h2>
			<p class="text-muted leading-relaxed">
				A one-minute walk around the home screen. You can always take it later from Settings.
			</p>
		</div>

		<div class="flex items-center gap-1 mt-1">
			<button class="ghost" onclick={dismissInvite}>Not now</button>
			<button bind:this={takeButton} class="primary" onclick={startTour}>Take the tour</button>
		</div>
	</div>
</div>

<style>
	.primary,
	.ghost {
		height: 34px;
		padding: 0 18px;
		border-radius: 999px;
		font-size: 12.5px;
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
	.primary:focus-visible,
	.ghost:focus-visible {
		outline: 2px solid color-mix(in srgb, var(--accent) 60%, transparent);
		outline-offset: 2px;
	}

	/* plays once: score, then a gauge, then the tools, and rests on the score */
	.spot {
		stroke: var(--accent);
		animation: hop 2600ms 300ms cubic-bezier(0.65, 0, 0.35, 1) both;
	}
	@keyframes hop {
		0% {
			opacity: 0;
			x: 0px;
			y: 0px;
			width: 40px;
			height: 54px;
		}
		10%,
		28% {
			opacity: 1;
			x: 0px;
			y: 0px;
			width: 40px;
			height: 54px;
		}
		40%,
		58% {
			x: 42px;
			y: 0px;
			width: 44px;
			height: 28px;
		}
		70%,
		86% {
			x: 0px;
			y: 56px;
			width: 128px;
			height: 20px;
		}
		100% {
			opacity: 1;
			x: 0px;
			y: 0px;
			width: 40px;
			height: 54px;
		}
	}
	.still .spot {
		animation: none;
	}
</style>

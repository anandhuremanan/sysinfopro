<script lang="ts">
	let {
		percent = 0,
		charging = false,
		width = 48
	}: { percent?: number; charging?: boolean; width?: number } = $props();

	const INNER = 34; // usable width inside the battery body, in viewBox units

	let level = $derived(Math.min(100, Math.max(0, percent || 0)));
	let levelColor = $derived(
		level <= 20 ? 'fill-red-500' : level <= 40 ? 'fill-amber-500' : 'fill-accent'
	);
</script>

<svg
	{width}
	height={width / 2}
	viewBox="0 0 48 24"
	fill="none"
	role="img"
	aria-label="Battery at {level.toFixed(0)}%{charging ? ', charging' : ''}"
	class="shrink-0 text-muted"
>
	<!-- body and terminal -->
	<rect x="1" y="3" width="40" height="18" rx="5" stroke="currentColor" stroke-width="2" />
	<rect x="43" y="8.5" width="3.5" height="7" rx="1.5" fill="currentColor" />

	<!-- charge level -->
	<rect
		x="4"
		y="6"
		width={(INNER * level) / 100}
		height="12"
		rx="2.5"
		class="level {levelColor}"
		class:charging
	/>

	{#if charging}
		<path
			d="M22.6 5.5l-5.2 7.3h4l-1.1 5.7 5.3-7.4h-4l1-5.6z"
			class="bolt fill-ink stroke-tile"
			stroke-width="1"
			stroke-linejoin="round"
		/>
	{/if}
</svg>

<style>
	.level {
		transition: width 700ms ease-out;
	}

	/* a slow breath while power is flowing in */
	.level.charging {
		animation: charge 1.8s ease-in-out infinite;
	}

	.bolt {
		animation: bolt 1.8s ease-in-out infinite;
		transform-origin: 21px 12px;
	}

	@keyframes charge {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.55;
		}
	}

	@keyframes bolt {
		0%,
		100% {
			transform: scale(1);
		}
		50% {
			transform: scale(1.12);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.level,
		.level.charging,
		.bolt {
			animation: none;
			transition: none;
		}
	}
</style>

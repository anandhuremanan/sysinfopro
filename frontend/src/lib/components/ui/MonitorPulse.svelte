<script lang="ts">
	// Takes its colour from the parent's text colour (e.g. the health score colour)
	let { size = 56 }: { size?: number } = $props();
</script>

<svg
	width={size}
	height={size}
	viewBox="0 0 48 48"
	fill="none"
	aria-hidden="true"
	class="shrink-0"
	stroke-linecap="round"
	stroke-linejoin="round"
>
	<!-- screen and stand -->
	<rect x="4" y="7" width="40" height="27" rx="5" class="stroke-muted" stroke-width="2.5" />
	<path d="M18 41h12M24 34v7" class="stroke-muted" stroke-width="2.5" />

	<!-- faint full trace with a bright segment running along it -->
	<path
		d="M9.5 21h7l3-7 5.5 13 3.5-6h10"
		stroke="currentColor"
		stroke-opacity="0.25"
		stroke-width="2.5"
	/>
	<path
		d="M9.5 21h7l3-7 5.5 13 3.5-6h10"
		stroke="currentColor"
		stroke-width="2.5"
		pathLength="100"
		class="trace"
	/>
</svg>

<style>
	.trace {
		stroke-dasharray: 28 72;
		animation: travel 2.2s linear infinite;
	}

	@keyframes travel {
		from {
			stroke-dashoffset: 100;
		}
		to {
			stroke-dashoffset: 0;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.trace {
			animation: none;
			stroke-dasharray: none;
		}
	}
</style>

<script lang="ts">
	// A drive stack; while active a scan line sweeps over it and the lights blink faster
	let { size = 96, active = false }: { size?: number; active?: boolean } = $props();
</script>

<svg
	width={size}
	height={size}
	viewBox="0 0 48 48"
	fill="none"
	aria-hidden="true"
	class="shrink-0"
	class:active
>
	<defs>
		<clipPath id="drive-scan-clip">
			<rect x="6" y="7" width="36" height="34" rx="5" />
		</clipPath>
	</defs>

	<!-- three drive bays -->
	{#each [8, 19.5, 31] as y, i}
		<rect x="7" {y} width="34" height="9" rx="3" class="fill-tile stroke-muted" stroke-width="2" />
		<path d="M12 {y + 4.5}h13" class="stroke-muted" stroke-width="2" stroke-linecap="round" />
		<circle cx="35" cy={y + 4.5} r="1.6" class="led fill-accent" style="animation-delay: {i * 0.4}s" />
	{/each}

	<!-- scan line, only drawn while active -->
	<g clip-path="url(#drive-scan-clip)">
		<rect x="6" y="-6" width="36" height="6" class="sweep fill-accent" />
	</g>
</svg>

<style>
	.led {
		animation: blink 2.4s ease-in-out infinite;
	}

	.active .led {
		animation-duration: 0.7s;
	}

	.sweep {
		opacity: 0;
	}

	.active .sweep {
		opacity: 0.28;
		animation: sweep 1.6s ease-in-out infinite;
	}

	@keyframes blink {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.25;
		}
	}

	@keyframes sweep {
		0% {
			transform: translateY(8px);
		}
		50% {
			transform: translateY(44px);
		}
		100% {
			transform: translateY(8px);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.led,
		.active .led,
		.active .sweep {
			animation: none;
		}
	}
</style>

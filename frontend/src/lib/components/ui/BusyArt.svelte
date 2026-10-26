<script lang="ts">
	// Small animated drawings for "working on it" states, in the accent colour.
	// With "reduce motion" on in Windows they stay still.
	let {
		kind = 'scan',
		size = 64,
		label = ''
	}: {
		kind?: 'scan' | 'search' | 'check' | 'measure' | 'read';
		size?: number;
		// what is happening, for screen readers and shown under the drawing
		label?: string;
	} = $props();

	// SVG's own animations ignore CSS, so "reduce motion" is checked here for those
	const reduceMotion =
		typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
</script>

<div class="busy flex flex-col items-center gap-2.5" role="status" aria-live="polite">
	<svg width={size} height={size} viewBox="0 0 64 64" fill="none" aria-hidden="true">
		{#if kind === 'scan'}
			<!-- a drive, with a beam sweeping across it -->
			<rect x="10" y="14" width="44" height="36" rx="7" class="soft" />
			<rect x="10" y="14" width="44" height="36" rx="7" class="line" />
			<circle cx="44" cy="40" r="2.5" class="fill blink" />
			<path d="M18 40h16" class="line" />
			<g class="sweep">
				<rect x="10" y="14" width="10" height="36" class="beam" />
			</g>
		{:else if kind === 'search'}
			<!-- folders, with a magnifier gliding over them -->
			<path d="M8 22a3 3 0 0 1 3-3h9l3 3h14a3 3 0 0 1 3 3v15a3 3 0 0 1-3 3H11a3 3 0 0 1-3-3z" class="soft" />
			<path d="M8 22a3 3 0 0 1 3-3h9l3 3h14a3 3 0 0 1 3 3v15a3 3 0 0 1-3 3H11a3 3 0 0 1-3-3z" class="line" />
			<g class="glide">
				<circle cx="40" cy="36" r="9" class="lens" />
				<circle cx="40" cy="36" r="9" class="line thick" />
				<path d="M47 43l8 8" class="line thick" />
			</g>
		{:else if kind === 'check'}
			<!-- a shield, with a line passing over it -->
			<path d="M32 8l20 7v14c0 12-8.5 22-20 27C20.5 51 12 41 12 29V15z" class="soft" />
			<path d="M32 8l20 7v14c0 12-8.5 22-20 27C20.5 51 12 41 12 29V15z" class="line" />
			<path d="M24 31l6 6 11-12" class="line thick tick" />
			<g class="pass">
				<path d="M12 20h40" class="line glow" />
			</g>
		{:else if kind === 'measure'}
			<!-- bars filling up one after another -->
			{#each [0, 1, 2, 3] as i (i)}
				<rect x={12 + i * 11} y="14" width="7" height="36" rx="3.5" class="soft" />
				<rect x={12 + i * 11} y="14" width="7" height="36" rx="3.5" class="fill grow" style="animation-delay: {i * 0.18}s" />
			{/each}
		{:else}
			<!-- a chip, with signals travelling along its pins -->
			<rect x="18" y="18" width="28" height="28" rx="6" class="soft" />
			<rect x="18" y="18" width="28" height="28" rx="6" class="line" />
			<rect x="26" y="26" width="12" height="12" rx="2.5" class="fill blink" />
			{#each [24, 32, 40] as p, i (p)}
				<path d="M{p} 18V9M{p} 46v9M18 {p}H9M46 {p}h9" class="line" />
				{#if !reduceMotion}
				<circle r="2" class="fill">
					<animateMotion dur="1.4s" begin="{i * 0.3}s" repeatCount="indefinite" path="M{p} 9V18" />
				</circle>
				<circle r="2" class="fill">
					<animateMotion dur="1.4s" begin="{i * 0.3 + 0.15}s" repeatCount="indefinite" path="M55 {p}H46" />
				</circle>
				{/if}
			{/each}
		{/if}
	</svg>
	{#if label}
		<div class="text-xs text-muted text-center">{label}</div>
	{/if}
</div>

<style>
	.line {
		stroke: var(--accent);
		stroke-width: 2.2;
		stroke-linecap: round;
		stroke-linejoin: round;
	}
	.thick {
		stroke-width: 3;
	}
	.soft {
		fill: color-mix(in srgb, var(--accent) 14%, transparent);
	}
	.fill {
		fill: var(--accent);
	}
	.lens {
		fill: color-mix(in srgb, var(--accent) 22%, transparent);
	}
	.beam {
		fill: color-mix(in srgb, var(--accent) 35%, transparent);
	}
	.glow {
		stroke-width: 2.5;
		opacity: 0.8;
	}

	.sweep {
		animation: sweep 1.6s ease-in-out infinite alternate;
	}
	.glide {
		animation: glide 2.4s ease-in-out infinite;
	}
	.pass {
		animation: pass 1.8s ease-in-out infinite;
	}
	.tick {
		stroke-dasharray: 30;
		animation: tick 1.8s ease-in-out infinite;
	}
	.grow {
		transform-box: fill-box;
		transform-origin: bottom;
		animation: grow 1.2s ease-in-out infinite alternate;
	}
	.blink {
		animation: blink 1s ease-in-out infinite;
	}

	@keyframes sweep {
		from { transform: translateX(0); }
		to { transform: translateX(34px); }
	}
	@keyframes glide {
		0%, 100% { transform: translate(0, 0); }
		33% { transform: translate(-18px, -8px); }
		66% { transform: translate(-8px, 4px); }
	}
	@keyframes pass {
		from { transform: translateY(0); opacity: 0; }
		20% { opacity: 1; }
		80% { opacity: 1; }
		to { transform: translateY(30px); opacity: 0; }
	}
	@keyframes tick {
		0%, 40% { stroke-dashoffset: 30; }
		70%, 100% { stroke-dashoffset: 0; }
	}
	@keyframes grow {
		from { transform: scaleY(0.2); }
		to { transform: scaleY(1); }
	}
	@keyframes blink {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.35; }
	}

	@media (prefers-reduced-motion: reduce) {
		.sweep, .glide, .pass, .tick, .grow, .blink {
			animation: none;
		}
		.tick { stroke-dashoffset: 0; }
	}
</style>

<script lang="ts">
	import BusyArt from './BusyArt.svelte';

	// The middle of a screen while a long background search runs
	let {
		label,
		percent = null,
		current = ''
	}: {
		label: string; // what is happening, e.g. "Listing files: 12,400 so far"
		percent?: number | null; // a progress bar when known
		current?: string; // the folder or file being looked at
	} = $props();
</script>

<div class="flex-1 flex flex-col items-center justify-center gap-3 text-center">
	<BusyArt kind="search" size={64} />
	<div class="text-xs font-semibold text-ink">{label}</div>
	{#if percent !== null}
		<div
			class="w-64 h-1.5 rounded-full bg-track overflow-hidden"
			role="progressbar"
			aria-valuemin={0}
			aria-valuemax={100}
			aria-valuenow={Math.round(percent)}
		>
			<div
				class="h-full bg-accent rounded-full transition-[width] duration-200 motion-reduce:transition-none"
				style="width: {Math.max(0, Math.min(100, percent))}%"
			></div>
		</div>
	{/if}
	<div class="text-[11px] text-muted max-w-sm truncate" title={current}>{current}</div>
	<div class="text-[11px] text-muted">You can use other tools meanwhile; it keeps going.</div>
</div>

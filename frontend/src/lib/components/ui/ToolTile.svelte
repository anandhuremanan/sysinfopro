<script lang="ts">
	import StrokeIcon from './StrokeIcon.svelte';

	let {
		name,
		hint,
		icon,
		count = 0,
		compact = false,
		onclick
	}: {
		name: string;
		hint: string;
		icon: string;
		count?: number;
		compact?: boolean;
		onclick: () => void;
	} = $props();
</script>

{#snippet badge()}
	{#if count > 0}
		<span
			class="shrink-0 min-w-[18px] h-[18px] px-[5px] rounded-full bg-amber-200 text-amber-900 text-[11px] font-bold leading-[18px] text-center"
		>
			{count}
		</span>
	{/if}
{/snippet}

{#if compact}
	<!-- Stacked tile for the six-across home row -->
	<button
		class="bg-tile text-ink rounded-[14px] px-3 py-[9px] text-left flex flex-col justify-between cursor-pointer min-w-0 transition-transform active:scale-[0.98] hover:ring-1 hover:ring-line"
		{onclick}
	>
		<span class="flex items-start justify-between">
			<span
				class="w-6 h-6 rounded-[7px] bg-accent text-on-accent flex items-center justify-center"
			>
				<StrokeIcon d={icon} size={14} />
			</span>
			{@render badge()}
		</span>
		<span class="flex flex-col leading-[1.3] min-w-0">
			<span class="font-semibold truncate">{name}</span>
			<span class="text-muted truncate">{hint}</span>
		</span>
	</button>
{:else}
	<button
		class="h-[52px] bg-tile text-ink rounded-xl px-3 text-left flex items-center gap-2.5 cursor-pointer min-w-0 transition-transform active:scale-[0.98] hover:ring-1 hover:ring-line"
		{onclick}
	>
		<span
			class="shrink-0 w-[30px] h-[30px] rounded-[9px] bg-accent text-on-accent flex items-center justify-center"
		>
			<StrokeIcon d={icon} />
		</span>
		<span class="flex-1 min-w-0 flex flex-col">
			<span class="font-semibold truncate">{name}</span>
			<span class="text-muted truncate">{hint}</span>
		</span>
		{@render badge()}
	</button>
{/if}

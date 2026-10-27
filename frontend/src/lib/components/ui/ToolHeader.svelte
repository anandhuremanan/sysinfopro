<script lang="ts">
	import type { Snippet } from 'svelte';

	// The top of every tool screen: its name, one line about it, and its actions
	let {
		title,
		subtitle = '',
		sub,
		actions
	}: {
		title: string;
		subtitle?: string;
		// a subtitle with live parts, instead of plain text
		sub?: Snippet;
		actions?: Snippet;
	} = $props();
</script>

<div class="flex justify-between items-center gap-3 pb-2 border-b border-line shrink-0">
	<div class="min-w-0">
		<div class="font-bold text-base text-ink">{title}</div>
		{#if sub}
			<div class="text-xs text-muted truncate">{@render sub()}</div>
		{:else if subtitle}
			<div class="text-xs text-muted truncate">{subtitle}</div>
		{/if}
	</div>
	{#if actions}
		<div class="flex items-center gap-2 shrink-0">{@render actions()}</div>
	{/if}
</div>

<script lang="ts">
	import { matchTool, tools, type ToolStatus } from '$lib/tools';
	import StrokeIcon from '../ui/StrokeIcon.svelte';

	let {
		toolStatus,
		onOpen
	}: {
		toolStatus: Record<string, ToolStatus>;
		onOpen: (view: string) => void;
	} = $props();

	let searchTerm = $state('');

	// groups in display order, each with its own shade of the accent colour
	const GROUPS = [
		{ name: 'System', tone: 100 },
		{ name: 'Clean up', tone: 82 },
		{ name: 'Developer', tone: 64 },
		{ name: 'Network', tone: 48 }
	];
	const toneOf = (group: string) => GROUPS.find((g) => g.name === group)?.tone ?? 100;
	// --tone-mix is white for light accents and black for dark ones, so a lighter
	// shade keeps the icon on it readable
	const tint = (group: string) => `color-mix(in srgb, var(--accent) ${toneOf(group)}%, var(--tone-mix))`;

	let shown = $derived(
		GROUPS.flatMap((g) => tools.filter((t) => t.group === g.name)).filter((t) => matchTool(t, searchTerm) > 0)
	);
</script>

<div class="h-full flex flex-col gap-2.5 min-h-0">
	<div class="flex items-center justify-between gap-3 shrink-0">
		<div class="flex items-center gap-3.5 min-w-0 text-[11px] text-muted">
			<span class="text-xs">{tools.length} tools</span>
			{#each GROUPS as group (group.name)}
				<span class="flex items-center gap-1.5 whitespace-nowrap">
					<i class="w-2 h-2 rounded-[3px]" style="background: {tint(group.name)}"></i>
					{group.name}
				</span>
			{/each}
		</div>
		<input
			type="search"
			aria-label="Search tools"
			placeholder="Search tools (Ctrl+K)"
			bind:value={searchTerm}
			class="text-xs w-48 h-7 px-2.5 py-0 rounded-lg border border-line bg-tile text-ink placeholder:text-muted focus:outline-none focus:ring-1 focus:ring-accent focus:border-accent shrink-0"
		/>
	</div>

	<div class="flex-1 min-h-0 overflow-y-auto pr-1">
		{#if shown.length === 0}
			<div class="h-full flex items-center justify-center text-muted">No matching tools found.</div>
		{:else}
			<div class="grid grid-cols-6 gap-2 auto-rows-[100px]">
				{#each shown as tool (tool.id)}
					{@const count = toolStatus[tool.id]?.count ?? 0}
					<button
						class="relative min-w-0 rounded-2xl bg-tile px-2.5 py-2 flex flex-col items-center justify-center gap-1 text-center cursor-pointer transition-[background-color,transform] duration-150 hover:bg-line/60 active:scale-[0.97]"
						onclick={() => onOpen(tool.id)}
					>
						<span
							class="w-[34px] h-[34px] mb-0.5 rounded-[10px] flex items-center justify-center text-on-accent"
							style="background: {tint(tool.group)}"
						>
							<StrokeIcon d={tool.icon} size={18} />
						</span>
						<span class="font-semibold text-ink leading-tight max-w-full truncate">{tool.name}</span>
						<span class="text-[11px] text-muted leading-tight max-w-full truncate">
							{toolStatus[tool.id]?.hint ?? tool.hint}
						</span>
						{#if count > 0}
							<span
								class="absolute top-2 right-2 min-w-[18px] h-[18px] px-[5px] rounded-full bg-amber-200 text-amber-900 text-[11px] font-bold leading-[18px] text-center"
							>
								{count}
							</span>
						{/if}
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>

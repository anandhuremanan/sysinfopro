<script lang="ts">
	import { matchTool, tools, type Tool, type ToolStatus } from '$lib/tools';
	import ToolTile from '../ui/ToolTile.svelte';

	let {
		toolStatus,
		onOpen
	}: {
		toolStatus: Record<string, ToolStatus>;
		onOpen: (view: string) => void;
	} = $props();

	let searchTerm = $state('');

	let groups = $derived.by(() => {
		const byGroup = new Map<string, Tool[]>();
		for (const tool of tools) {
			if (matchTool(tool, searchTerm) === 0) continue;
			byGroup.set(tool.group, [...(byGroup.get(tool.group) ?? []), tool]);
		}
		return [...byGroup.entries()].map(([name, items]) => ({ name, items }));
	});
</script>

<div class="h-full flex flex-col gap-3 min-h-0">
	<div class="flex items-center justify-between shrink-0">
		<span class="text-muted">{tools.length} tools</span>
		<input
			type="search"
			aria-label="Search tools"
			placeholder="Search tools (Ctrl+K)"
			bind:value={searchTerm}
			class="text-xs w-48 h-7 px-2.5 py-0 rounded-lg border border-line bg-tile text-ink placeholder:text-muted focus:outline-none focus:ring-1 focus:ring-accent focus:border-accent"
		/>
	</div>

	<div class="flex-1 min-h-0 overflow-y-auto flex flex-col gap-3 pr-1">
		{#each groups as group (group.name)}
			<section class="flex flex-col gap-1.5">
				<h2 class="text-[11px] font-semibold tracking-wide uppercase text-muted">{group.name}</h2>
				<div class="grid grid-cols-4 gap-2.5">
					{#each group.items as tool (tool.id)}
						<ToolTile
							name={tool.name}
							hint={toolStatus[tool.id]?.hint ?? tool.hint}
							icon={tool.icon}
							count={toolStatus[tool.id]?.count ?? 0}
							onclick={() => onOpen(tool.id)}
						/>
					{/each}
				</div>
			</section>
		{:else}
			<div class="flex-1 flex items-center justify-center text-muted">No matching tools found.</div>
		{/each}
	</div>
</div>

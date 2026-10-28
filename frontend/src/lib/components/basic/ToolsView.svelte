<script lang="ts">
	import {
		AUDIENCES,
		audienceOf,
		loadAudience,
		matchTool,
		saveAudience,
		toggleAudience,
		tools,
		type Audience,
		type ToolStatus
	} from '$lib/tools';

	let {
		toolStatus,
		onOpen
	}: {
		toolStatus: Record<string, ToolStatus>;
		onOpen: (view: string) => void;
	} = $props();

	let searchTerm = $state('');

	// groups in display order; the first card of each group names it
	const GROUPS = ['System', 'Clean up', 'Developer', 'Network'];

	// which kinds of tools to show; remembered on this PC. All on = every tool.
	let audience = $state<Audience[]>(loadAudience());

	function toggle(id: Audience) {
		audience = toggleAudience(audience, id);
		saveAudience(audience);
	}

	let ordered = $derived(GROUPS.flatMap((g) => tools.filter((t) => t.group === g)));
	// typing searches every tool, so a filtered-out tool is never unreachable
	let searching = $derived(searchTerm.trim().length > 0);
	let shown = $derived(
		searching
			? ordered.filter((t) => matchTool(t, searchTerm) > 0)
			: ordered.filter((t) => audienceOf[t.id].some((a) => audience.includes(a)))
	);
</script>

<div class="h-full flex flex-col gap-2.5 min-h-0">
	<div class="flex items-center justify-between gap-3 shrink-0">
		<div class="flex items-center gap-1.5 min-w-0" role="group" aria-label="Show tools for">
			<span class="text-xs text-muted mr-1">Show tools for</span>
			{#each AUDIENCES as a (a.id)}
				{@const on = audience.includes(a.id)}
				<button
					class="h-6 px-2.5 rounded-full text-[11px] font-semibold border cursor-pointer transition-colors {on
						? 'bg-accent/15 border-accent/50 text-ink'
						: 'bg-transparent border-line text-muted hover:text-ink'}"
					aria-pressed={on}
					onclick={() => toggle(a.id)}
				>
					{a.label}
				</button>
			{/each}
			<span class="text-[11px] text-muted ml-1.5">
				{searching ? `${shown.length} found in all tools` : `${shown.length} of ${tools.length}`}
			</span>
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
			<div class="grid grid-cols-6 gap-2 auto-rows-[76px] p-0.5">
				{#each shown as tool, i (tool.id)}
					{@const count = toolStatus[tool.id]?.count ?? 0}
					<button
						class="card relative min-w-0 overflow-hidden rounded-[14px] px-2.5 py-2 flex flex-col justify-end text-left cursor-pointer"
						onclick={() => onOpen(tool.id)}
					>
						<!-- the tool's icon, large and faint; brighter on hover -->
						<svg
							class="mark absolute -top-1.5 -right-2 text-accent pointer-events-none"
							width="52"
							height="52"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="1.6"
							stroke-linecap="round"
							stroke-linejoin="round"
							aria-hidden="true"
						>
							<path class="base" d={tool.icon} />
						</svg>
						{#if i === 0 || shown[i - 1].group !== tool.group}
							<span
								class="absolute top-2 left-2.5 text-[9.5px] uppercase tracking-wider font-semibold text-accent"
							>
								{tool.group}
							</span>
						{/if}
						{#if count > 0}
							<span
								class="absolute top-1.5 right-2 z-10 min-w-[18px] h-[18px] px-[5px] rounded-full bg-amber-200 text-amber-900 text-[11px] font-bold leading-[18px] text-center"
								aria-label="{count} to look at"
							>
								{count}
							</span>
						{/if}
						<span class="relative text-[12.5px] font-semibold text-ink leading-tight line-clamp-2"
							>{tool.name}</span
						>
						<span class="relative text-[11px] text-muted leading-tight truncate">
							{toolStatus[tool.id]?.hint ?? tool.hint}
						</span>
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>

<style>
	.card {
		background: linear-gradient(
			150deg,
			color-mix(in srgb, var(--accent) 14%, var(--tile)) 0%,
			var(--tile) 70%
		);
		border: 1px solid color-mix(in srgb, var(--accent) 16%, transparent);
	}
	/* hover: a stronger border and a brighter icon, nothing moving */
	.card:hover {
		border-color: color-mix(in srgb, var(--accent) 50%, transparent);
	}
	.card:focus-visible {
		outline: 2px solid var(--accent);
		outline-offset: 2px;
	}
	.mark .base {
		opacity: 0.18;
	}
	.card:hover .mark .base,
	.card:focus-visible .mark .base {
		opacity: 0.32;
	}
</style>

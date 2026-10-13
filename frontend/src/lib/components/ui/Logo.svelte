<script lang="ts">
	let { size = 28, showWordmark = true }: { size?: number; showWordmark?: boolean } = $props();
</script>

<div class="flex items-center gap-2.5 text-ink" role="img" aria-label="SysInfo Pro">
	<!-- Pulse mark: keep in step with build/appicon.png and static/favicon.png -->
	<svg width={size} height={size} viewBox="0 0 24 24" fill="none" aria-hidden="true">
		<rect x="1" y="1" width="22" height="22" rx="6" class="fill-accent" />
		<path
			d="M4.5 12.5h3.2l2.1-5 3.4 9.5 2.2-4.5h4.1"
			class="stroke-on-accent"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
		/>
	</svg>
	{#if showWordmark}
		<span class="text-[19px] font-bold tracking-tight leading-none whitespace-nowrap">
			SysInfo <span class="font-medium text-muted">Pro</span>
		</span>
	{/if}
</div>

<script lang="ts">
	import ToolHeader from '$lib/components/ui/ToolHeader.svelte';
	import { Status } from '$lib/status.svelte';
	import BusyArt from '../ui/BusyArt.svelte';
	import { onMount } from 'svelte';
	import { buildSpecSections, exportSystemReport, specsToText } from '$lib/utils/reportExporter';
	import { load, recall } from '$lib/cache';
	import { GetDrives, GetHardwareSpecs, GetSecurityReport } from '$lib/wailsjs/go/main/App';

	let { systemInfo } = $props();

	// hardware does not change while the app is open, so one read serves the session
	let specs: any = $state(recall('specs'));
	let isLoading = $state(false);
	let isExporting = $state(false);
	let loadError = $state<string | undefined>(undefined);
	const status = new Status(5000);

	let sections = $derived(buildSpecSections(specs));

	async function loadSpecs() {
		isLoading = !specs;
		loadError = undefined;
		try {
			specs = await load('specs', GetHardwareSpecs);
		} catch (err) {
			console.error('Failed to get hardware specs:', err);
			loadError = `${err}`;
		} finally {
			isLoading = false;
		}
	}

	async function copySpecs() {
		try {
			await navigator.clipboard.writeText(specsToText(specs));
			status.show('Specs copied to clipboard');
		} catch (err) {
			status.show(`Failed to copy: ${err}`);
		}
	}

	async function exportReport() {
		isExporting = true;
		try {
			// the report also carries drives and the security check; either may be unavailable
			const [drives, security] = await Promise.all([
				GetDrives().catch(() => []),
				load('security', GetSecurityReport, 5 * 60_000).catch(() => undefined)
			]);
			const savedPath = await exportSystemReport({ systemInfo, specs, security, drives });
			status.show(savedPath ? `Report saved: ${savedPath}` : 'Failed to save the report.');
		} finally {
			isExporting = false;
		}
	}

	// load once when the screen opens
	onMount(() => {
		loadSpecs();
	});
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<!-- Header Bar -->
	<ToolHeader title="Specs">
		{#snippet sub()}
			{specs
				? `${specs.system.manufacturer} ${specs.system.model}`
				: 'Full inventory of this machine'}
		{/snippet}
		{#snippet actions()}
			<div class="flex items-center gap-2 shrink-0">
				<button class="btn btn-secondary" onclick={copySpecs} disabled={!specs}>Copy</button>
				<button class="btn btn-primary" onclick={exportReport} disabled={!specs || isExporting}>
					{isExporting ? 'Building report...' : 'Export report'}
				</button>
			</div>
		{/snippet}
	</ToolHeader>

	{#if status.text}
		<div class="text-xs text-ok font-semibold px-1 shrink-0 truncate" title={status.text}>
			{status.text}
		</div>
	{/if}

	{#if isLoading}
		<div class="flex-1 flex items-center justify-center">
			<BusyArt kind="read" size={64} label="Reading hardware inventory..." />
		</div>
	{:else if loadError}
		<div class="flex-1 flex flex-col items-center justify-center gap-2 text-center">
			<div class="text-xs text-red-500 max-w-md">{loadError}</div>
			<button class="btn btn-secondary" onclick={loadSpecs}>Try again</button>
		</div>
	{:else}
		<div
			class="flex-1 min-h-0 overflow-y-auto overscroll-contain pb-5 [mask-image:linear-gradient(to_bottom,#000_calc(100%-20px),transparent)] pr-1.5"
		>
			<div class="grid grid-cols-2 gap-2.5">
				{#each sections as section (section.title)}
					<section class="bg-ground border border-line rounded-xl px-3 py-2.5 min-w-0">
						<h2 class="text-[10px] uppercase tracking-wider font-semibold text-muted pb-1">
							{section.title}
						</h2>
						{#each section.rows as [label, value]}
							<div class="flex justify-between gap-3 py-1 border-t border-line first:border-t-0">
								<span class="text-muted shrink-0">{label}</span>
								<span class="font-semibold text-ink text-right min-w-0 break-words select-text">
									{value}
								</span>
							</div>
						{/each}
					</section>
				{/each}
			</div>
		</div>
	{/if}
</div>

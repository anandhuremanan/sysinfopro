<script lang="ts">
	import {
		CheckDevTools,
		DiagnoseSlowness,
		GetBatteryReport,
		GetDriveHealth,
		GetDrives,
		GetHardwareSpecs,
		GetPrivateTerms,
		GetSecurityReport,
		RevealInExplorer,
		RunNetworkDoctor
	} from '$lib/wailsjs/go/main/App';
	import { load, remember } from '$lib/cache';
	import { exportSystemReport, type ReportInput } from '$lib/utils/reportExporter';
	import Modal from './Modal.svelte';

	let { showModal = $bindable(false), systemInfo }: { showModal?: boolean; systemInfo: any } =
		$props();

	// each part of the report, and how to get it (cached results are reused)
	const PARTS = [
		{ id: 'security', label: 'Security check', hint: 'Antivirus, firewall, encryption, updates' },
		{ id: 'driveHealth', label: 'Drives and their health', hint: 'Space, wear and temperature' },
		{ id: 'battery', label: 'Battery', hint: 'Real capacity and wear' },
		{ id: 'network', label: 'Network doctor', hint: 'Where the connection breaks, if it does' },
		{ id: 'devTools', label: 'Developer tools', hint: 'Python, Node, Java, Git and others' },
		{ id: 'slow', label: 'Slowness check', hint: 'Takes a few seconds' }
	] as const;
	type PartId = (typeof PARTS)[number]['id'];

	let chosen = $state<Record<PartId, boolean>>({
		security: true,
		driveHealth: true,
		battery: true,
		network: true,
		devTools: false,
		slow: true
	});
	let hidePersonal = $state(true);
	let isSaving = $state(false);
	let step = $state('');
	let savedPath = $state('');
	let error = $state('');

	async function save() {
		isSaving = true;
		savedPath = '';
		error = '';
		step = 'Gathering the details...';
		const want = (id: PartId) => chosen[id];
		const quiet = <T,>(p: Promise<T>) => p.catch(() => undefined);
		try {
			const [specs, security, drives, driveHealth, battery, network, devTools, slow, terms] =
				await Promise.all([
					quiet(load('specs', GetHardwareSpecs)),
					want('security') ? quiet(load('security', GetSecurityReport, 5 * 60_000)) : undefined,
					want('driveHealth') ? quiet(load('drives', GetDrives, 60_000)) : undefined,
					want('driveHealth') ? quiet(load('drive-health', GetDriveHealth, 5 * 60_000)) : undefined,
					want('battery')
						? quiet(load('battery-report', GetBatteryReport, 10 * 60_000))
						: undefined,
					want('network')
						? quiet(load('netdoctor', () => RunNetworkDoctor(''), 60_000))
						: undefined,
					want('devTools') ? quiet(load('devtools', CheckDevTools, 5 * 60_000)) : undefined,
					want('slow')
						? quiet(
								DiagnoseSlowness().then((r) => {
									remember('slownow', { at: Date.now(), report: r });
									return r;
								})
							)
						: undefined,
					hidePersonal ? quiet(GetPrivateTerms()) : undefined
				]);
			step = 'Saving...';
			const input: ReportInput = {
				systemInfo,
				specs,
				security,
				drives: drives as any[],
				driveHealth: driveHealth as any[],
				battery,
				network,
				devTools: devTools as any[],
				slow,
				hidePersonal,
				privateTerms: (terms as string[]) ?? []
			};
			savedPath = await exportSystemReport(input);
			if (!savedPath) error = 'The report could not be saved.';
		} catch (err) {
			error = `${err}`;
		} finally {
			isSaving = false;
			step = '';
		}
	}

	$effect(() => {
		if (showModal) {
			savedPath = '';
			error = '';
		}
	});
</script>

<Modal
	bind:showModal
	width="max-w-lg"
	showClose={!isSaving}
	title="Save a full report"
	description="One page to send to IT, a friend or a forum. It opens in any browser. Hardware details are always included."
>
	<div class="grid grid-cols-2 gap-x-2 gap-y-0.5">
		{#each PARTS as part (part.id)}
			<label class="flex items-center gap-2.5 px-1 py-1 rounded-lg cursor-pointer hover:bg-ground">
				<input
					type="checkbox"
					class="rounded border-line bg-tile checked:bg-accent text-accent focus:ring-accent"
					bind:checked={chosen[part.id]}
					disabled={isSaving}
				/>
				<span class="flex flex-col leading-tight">
					<span class="text-ink font-semibold">{part.label}</span>
					<span class="text-[11px] text-muted">{part.hint}</span>
				</span>
			</label>
		{/each}
		<label
			class="col-span-2 flex items-start gap-2.5 px-1 py-2 mt-1.5 rounded-lg cursor-pointer bg-ground"
		>
			<input
				type="checkbox"
				class="mt-0.5 rounded border-line bg-tile checked:bg-accent text-accent focus:ring-accent"
				bind:checked={hidePersonal}
				disabled={isSaving}
			/>
			<span class="flex flex-col leading-tight">
				<span class="text-ink font-semibold">Leave out personal details</span>
				<span class="text-[11px] text-muted">
					Serial numbers, the PC's name, your user name, network addresses and emails. Keep this on
					for anything posted in public.
				</span>
			</span>
		</label>
		{#if savedPath}
			<p class="text-xs font-semibold col-span-2 text-ok px-1 pt-2 break-all">
				Saved to {savedPath}
			</p>
		{:else if error}
			<p class="text-xs font-semibold col-span-2 text-fail px-1 pt-2">
				{error}
			</p>
		{:else if step}
			<p class="col-span-2 text-xs text-muted px-1 pt-2">{step}</p>
		{/if}
	</div>
	{#snippet actions()}
		{#if savedPath}
			<button class="btn btn-secondary" onclick={() => RevealInExplorer(savedPath)}
				>Show in folder</button
			>
			<button class="btn btn-primary" onclick={() => (showModal = false)}>Done</button>
		{:else}
			<button class="btn btn-secondary" disabled={isSaving} onclick={() => (showModal = false)}
				>Cancel</button
			>
			<button class="btn btn-primary" disabled={isSaving} onclick={save}>
				{isSaving ? 'Working...' : 'Save report'}
			</button>
		{/if}
	{/snippet}
</Modal>

<script lang="ts">
	import { getSystemHealthScore } from '$lib';
	import { GetBatteryDetails } from '$lib/wailsjs/go/main/App';
	import BatteryDetails from './BatteryDetails.svelte';
	import DiskCleaner from './DiskCleaner.svelte';
	import EnvPathInspector from './EnvPathInspector.svelte';
	import PortInspector from './PortInspector.svelte';
	import SoftwareChecker from './SoftwareChecker.svelte';
	import StartupManager from './StartupManager.svelte';
	import StressBenchmark from './StressBenchmark.svelte';
	import SystemStat from './SystemStat.svelte';

	let { systemInfo } = $props();

	let batteryDetails: any = $state(undefined);
	let isLoadingBattery = $state(false);
	let overallScore = $state(0);
	let activeTab = $state<'health' | 'ports' | 'stress' | 'startup' | 'env' | 'software'>('health');

	async function getBatteryDetials() {
		try {
			isLoadingBattery = true;
			const battery = await GetBatteryDetails();
			if (battery) {
				batteryDetails = JSON.parse(battery);
			} else {
				batteryDetails = null;
			}
		} catch (error) {
			console.error('Failed to get battery details:', error);
		} finally {
			setTimeout(() => {
				isLoadingBattery = false;
			}, 500);
		}
	}

	async function init() {
		await getBatteryDetials();
		const systemMetrics = {
			battery: {
				designedCapacity: batteryDetails?.design_capacity,
				currentCapacity: batteryDetails?.current_capacity
			},
			disk: { total: systemInfo.total_disk_gb, free: systemInfo.free_disk_gb },
			ram: { total: systemInfo.total_ram_mb, available: systemInfo.free_ram_mb }
		};
		let score = getSystemHealthScore(systemMetrics);
		overallScore = score;
	}

	$effect(() => {
		init();
	});
</script>

<div class="flex flex-col h-full min-h-0 space-y-2">
	<!-- Tab Bar Navigation with Instant Active Scale Feedback -->
	<div
		class="flex gap-2 items-center pb-1 text-xs shrink-0 overflow-x-auto"
	>
		<button
			class="px-3 py-1 rounded-lg cursor-pointer transition-transform active:scale-95 font-medium shrink-0 {activeTab ===
			'health'
				? 'bg-blue-500 text-white shadow-xs'
				: 'bg-slate-300/60 dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-400 dark:hover:bg-slate-700'}"
			onclick={() => (activeTab = 'health')}
		>
			Storage & Health
		</button>
		<button
			class="px-3 py-1 rounded-lg cursor-pointer transition-transform active:scale-95 font-medium shrink-0 {activeTab ===
			'ports'
				? 'bg-blue-500 text-white shadow-xs'
				: 'bg-slate-300/60 dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-400 dark:hover:bg-slate-700'}"
			onclick={() => (activeTab = 'ports')}
		>
			Ports
		</button>
		<button
			class="px-3 py-1 rounded-lg cursor-pointer transition-transform active:scale-95 font-medium shrink-0 {activeTab ===
			'stress'
				? 'bg-blue-500 text-white shadow-xs'
				: 'bg-slate-300/60 dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-400 dark:hover:bg-slate-700'}"
			onclick={() => (activeTab = 'stress')}
		>
			10s Stress Test
		</button>
		<button
			class="px-3 py-1 rounded-lg cursor-pointer transition-transform active:scale-95 font-medium shrink-0 {activeTab ===
			'startup'
				? 'bg-blue-500 text-white shadow-xs'
				: 'bg-slate-300/60 dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-400 dark:hover:bg-slate-700'}"
			onclick={() => (activeTab = 'startup')}
		>
			Startup
		</button>
		<button
			class="px-3 py-1 rounded-lg cursor-pointer transition-transform active:scale-95 font-medium shrink-0 {activeTab ===
			'env'
				? 'bg-blue-500 text-white shadow-xs'
				: 'bg-slate-300/60 dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-400 dark:hover:bg-slate-700'}"
			onclick={() => (activeTab = 'env')}
		>
			PATH Debug
		</button>
		<button
			class="px-3 py-1 rounded-lg cursor-pointer transition-transform active:scale-95 font-medium shrink-0 {activeTab ===
			'software'
				? 'bg-blue-500 text-white shadow-xs'
				: 'bg-slate-300/60 dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-400 dark:hover:bg-slate-700'}"
			onclick={() => (activeTab = 'software')}
		>
			Software Health
		</button>
	</div>

	<!-- Tab Content Area -->
	<div class="flex-1 min-h-0">
		{#if activeTab === 'health'}
			<div class="grid grid-cols-2 gap-3 h-full py-0.5">
				<div class="grid grid-rows-2 gap-3 h-full">
					<BatteryDetails {batteryDetails} {isLoadingBattery} {getBatteryDetials} />
					<SystemStat {overallScore} />
				</div>
				<div class="h-full">
					<DiskCleaner />
				</div>
			</div>
		{:else if activeTab === 'ports'}
			<PortInspector />
		{:else if activeTab === 'stress'}
			<StressBenchmark />
		{:else if activeTab === 'startup'}
			<StartupManager />
		{:else if activeTab === 'env'}
			<EnvPathInspector />
		{:else if activeTab === 'software'}
			<SoftwareChecker />
		{/if}
	</div>
</div>




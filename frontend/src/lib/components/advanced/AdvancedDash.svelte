<script lang="ts">
	import { getSystemHealthScore } from '$lib';
	import { GetBatteryDetails } from '$lib/wailsjs/go/main/App';
	import BatteryDetails from './BatteryDetails.svelte';
	import DevCacheCleaner from './DevCacheCleaner.svelte';
	import DiskAnalyzer from './DiskAnalyzer.svelte';
	import DiskCleaner from './DiskCleaner.svelte';
	import EnvPathInspector from './EnvPathInspector.svelte';
	import FileLocks from './FileLocks.svelte';
	import GrowthWatch from './GrowthWatch.svelte';
	import HardwareSpecs from './HardwareSpecs.svelte';
	import PortInspector from './PortInspector.svelte';
	import SecurityCheck from './SecurityCheck.svelte';
	import SoftwareChecker from './SoftwareChecker.svelte';
	import StartupManager from './StartupManager.svelte';
	import StressBenchmark from './StressBenchmark.svelte';
	import VirtualDisks from './VirtualDisks.svelte';
	import SystemStat from './SystemStat.svelte';
	import WhatChanged from './WhatChanged.svelte';

	let {
		systemInfo,
		tool = 'health',
		drive
	}: {
		systemInfo: any;
		// drive for Disk usage to open on
		drive?: string;
		tool?:
			| 'specs'
			| 'security'
			| 'devcache'
			| 'changes'
			| 'locks'
			| 'growth'
			| 'health'
			| 'space'
			| 'ports'
			| 'stress'
			| 'startup'
			| 'env'
			| 'software'
			| 'wsl';
	} = $props();

	let batteryDetails: any = $state(undefined);
	let isLoadingBattery = $state(false);
	let overallScore = $state(0);

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
			isLoadingBattery = false;
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

<div class="flex flex-col h-full min-h-0">
	<!-- Tool Content Area -->
	<div class="flex-1 min-h-0">
		{#if tool === 'health'}
			<div class="grid grid-cols-2 gap-3 h-full py-0.5">
				<div class="grid grid-rows-2 gap-3 h-full">
					<BatteryDetails {batteryDetails} {isLoadingBattery} {getBatteryDetials} />
					<SystemStat {overallScore} />
				</div>
				<div class="h-full min-h-0">
					<DiskCleaner />
				</div>
			</div>
		{:else if tool === 'specs'}
			<HardwareSpecs {systemInfo} />
		{:else if tool === 'security'}
			<SecurityCheck />
		{:else if tool === 'devcache'}
			<DevCacheCleaner />
		{:else if tool === 'changes'}
			<WhatChanged />
		{:else if tool === 'locks'}
			<FileLocks />
		{:else if tool === 'growth'}
			<GrowthWatch />
		{:else if tool === 'space'}
			<DiskAnalyzer {drive} />
		{:else if tool === 'ports'}
			<PortInspector />
		{:else if tool === 'stress'}
			<StressBenchmark />
		{:else if tool === 'startup'}
			<StartupManager />
		{:else if tool === 'env'}
			<EnvPathInspector />
		{:else if tool === 'software'}
			<SoftwareChecker />
		{:else if tool === 'wsl'}
			<VirtualDisks />
		{/if}
	</div>
</div>

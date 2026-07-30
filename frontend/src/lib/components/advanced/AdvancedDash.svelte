<script lang="ts">
	import { getSystemHealthScore } from '$lib';
	import { GetBatteryDetails } from '$lib/wailsjs/go/main/App';
	import BatteryDetails from './BatteryDetails.svelte';
	import DiskCleaner from './DiskCleaner.svelte';
	import SystemStat from './SystemStat.svelte';

	let { systemInfo } = $props();

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

<div class="grid grid-cols-2 gap-3 h-full py-0.5">
	<!-- Left column with 2 strictly equal fixed-height rows -->
	<div class="grid grid-rows-2 gap-3 h-full">
		<BatteryDetails {batteryDetails} {isLoadingBattery} {getBatteryDetials} />
		<SystemStat {overallScore} />
	</div>
	<!-- Right column matching exact full height -->
	<div class="h-full">
		<DiskCleaner />
	</div>
</div>

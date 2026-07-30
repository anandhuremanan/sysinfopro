<script lang="ts">
	import Card from '../ui/Card.svelte';
	import RefreshButton from '../ui/RefreshButton.svelte';

	let { isLoadingBattery, batteryDetails, getBatteryDetials } = $props();
</script>

<Card isLoading={isLoadingBattery}>
	<div class="h-full flex flex-col justify-between">
		<!-- Header Row with unified baseline height -->
		<div class="h-6 flex justify-between items-center w-full">
			<div class="flex items-center gap-2">
				<span class="font-bold text-xs text-slate-800 dark:text-slate-100">Battery Details</span>
				{#if batteryDetails?.health_status}
					<span
						class="text-[10px] px-1.5 py-0.5 rounded font-semibold {batteryDetails.health_status ===
						'Excellent'
							? 'bg-green-500/20 text-green-400 border border-green-500/30'
							: 'bg-amber-500/20 text-amber-400 border border-amber-500/30'}"
					>
						{batteryDetails.health_status}
					</span>
				{/if}
			</div>
			<div class="shrink-0">
				<RefreshButton onclick={getBatteryDetials} showText={false} />
			</div>
		</div>

		<!-- Main Content -->
		{#if isLoadingBattery}
			<div class="flex-1 flex items-center justify-center text-xs text-slate-400">Loading Battery Details...</div>
		{:else}
			<div class="flex justify-center items-center flex-1 w-full gap-3 py-1">
				<img src="/battery.png" alt="battery" class="w-14 h-14 shrink-0 object-contain" />
				<div class="flex-1 text-xs space-y-1">
					{#if batteryDetails}
						<div class="grid grid-cols-2 gap-x-3 gap-y-1">
							<p>Percentage: <span class="font-bold">{batteryDetails.percentage.toFixed(0)}%</span></p>
							<p>State: <span class="font-bold capitalize">{batteryDetails.state}</span></p>
							<p>Design: <span class="font-bold">{batteryDetails.design_capacity} mWh</span></p>
							<p>Current: <span class="font-bold">{batteryDetails.current_capacity} mWh</span></p>
							{#if batteryDetails.degradation_pct !== undefined}
								<p>Degradation: <span class="font-bold text-amber-400">{batteryDetails.degradation_pct}%</span></p>
							{/if}
							{#if batteryDetails.cycle_estimate !== undefined}
								<p>Est. Cycles: <span class="font-bold">{batteryDetails.cycle_estimate}</span></p>
							{/if}
						</div>
					{:else}
						<div class="font-semibold text-slate-300">No Battery Detected</div>
						<p class="text-gray-400 text-[11px]">
							Battery details are only available for laptop hardware.
						</p>
					{/if}
				</div>
			</div>
		{/if}
	</div>
</Card>

<script lang="ts">
	import { checkInternetConnection, copyStat, mbToGb } from '$lib';
	import { exportSpecCardPNG } from '$lib/utils/cardExporter';
	import AdvancedDash from '../advanced/AdvancedDash.svelte';
	import CpuStats from './CpuStats.svelte';
	import ProcessView from './ProcessView.svelte';
	import RamStats from './RamStats.svelte';
	import SpeedAnimation from './SpeedAnimation.svelte';

	let {
		systemInfo,
		showSpeedTest = $bindable(),
		showAdvancedDashboard = $bindable(),
		showTopProcesses = $bindable()
	} = $props();

	let logMessage = $state<string | undefined>(undefined);

	async function getInternetSpeed() {
		const isConnected = await checkInternetConnection();

		if (!isConnected) {
			logMessage = 'No internet connection detected!';
			setTimeout(() => {
				logMessage = undefined;
			}, 2000);
			return;
		}

		showSpeedTest = !showSpeedTest;
	}

	async function copyStatHandler() {
		logMessage = await copyStat(systemInfo);

		setTimeout(() => {
			logMessage = undefined;
		}, 2000);
	}

	async function exportSpecCardHandler() {
		const savedPath = await exportSpecCardPNG(systemInfo);
		if (savedPath) {
			logMessage = `Spec Card saved: ${savedPath}`;
		} else {
			logMessage = 'Failed to export spec card.';
		}
		setTimeout(() => {
			logMessage = undefined;
		}, 4000);
	}
</script>

<div
	class="bg-slate-200 dark:bg-slate-900 dark:text-white p-4 rounded-2xl flex flex-col justify-between w-[95vw] h-[78vh] shrink-0 relative"
>
	{#if showSpeedTest}
		<div class="h-full flex items-center justify-center">
			<SpeedAnimation />
		</div>
	{:else if showAdvancedDashboard && systemInfo}
		<div class="h-full">
			<AdvancedDash {systemInfo} />
		</div>
	{:else if showTopProcesses}
		<div class="h-full">
			<ProcessView />
		</div>
	{:else}
		<div class="flex flex-col justify-between h-full gap-3">
			<!-- Header Bar: Minimal OS Hero Badge & System Status -->
			<div
				class="flex justify-between items-center pb-2.5 border-b border-slate-300 dark:border-slate-800"
			>
				<div class="flex items-center gap-3">
					{#if systemInfo.host_os == 'windows'}
						<img src="/windows.png" alt="windows" class="w-8 h-8 object-contain shrink-0" />
					{:else if systemInfo.host_os == 'darwin'}
						<img src="/mac.png" alt="macos" class="w-8 h-8 object-contain shrink-0" />
					{:else}
						<img src="/pc.png" alt="system" class="w-8 h-8 object-contain shrink-0" />
					{/if}
					<div>
						<div class="font-bold text-sm leading-tight text-slate-800 dark:text-slate-100">
							{systemInfo.host_platform || 'Desktop System'}
						</div>
						<div class="text-xs text-gray-500 dark:text-slate-400">
							{systemInfo.host_platform_version} &bull; {systemInfo.logged_in_user}
						</div>
					</div>
				</div>

				<div
					class="text-xs px-2.5 py-1 rounded-full bg-green-500/15 text-green-500 font-semibold border border-green-500/30 flex items-center gap-1.5"
				>
					<span class="w-2 h-2 rounded-full bg-green-500 animate-pulse"></span>
					System Active
				</div>
			</div>

			<!-- Live Gauges Cards (CPU & RAM Side-by-Side) -->
			<div class="grid grid-cols-2 gap-4 my-1">
				<div
					class="bg-slate-300/40 dark:bg-slate-800/60 p-3.5 rounded-xl border border-slate-300 dark:border-slate-700/60 flex flex-col justify-center"
				>
					<CpuStats />
				</div>

				<div
					class="bg-slate-300/40 dark:bg-slate-800/60 p-3.5 rounded-xl border border-slate-300 dark:border-slate-700/60 flex flex-col justify-center"
				>
					<RamStats />
				</div>
			</div>

			<!-- Specifications Grid (3 Columns) -->
			<div
				class="grid grid-cols-3 gap-4 bg-slate-300/20 dark:bg-slate-800/30 p-3.5 rounded-xl border border-slate-300/50 dark:border-slate-700/40 text-xs"
			>
				<!-- Col 1: CPU -->
				<div class="space-y-1">
					<div
						class="font-bold text-gray-500 dark:text-slate-400 uppercase tracking-wider text-[10px]"
					>
						Processor
					</div>
					<div class="truncate" title={systemInfo.cpu_model}>
						Model: <span class="font-bold text-slate-800 dark:text-slate-100"
							>{systemInfo.cpu_model}</span
						>
					</div>
					<div>
						Vendor: <span class="font-bold text-slate-800 dark:text-slate-100"
							>{systemInfo.cpu_vendor}</span
						>
					</div>
					<div>
						Cores: <span class="font-bold text-slate-800 dark:text-slate-100"
							>{systemInfo.cpu_cores} Cores</span
						>
					</div>
				</div>

				<!-- Col 2: Storage & Memory -->
				<div class="space-y-1">
					<div
						class="font-bold text-gray-500 dark:text-slate-400 uppercase tracking-wider text-[10px]"
					>
						Memory & Storage
					</div>
					<div>
						RAM: <span class="font-bold text-slate-800 dark:text-slate-100"
							>{mbToGb(systemInfo.total_ram_mb)} GB Installed</span
						>
					</div>
					<div>
						Disk Type: <span class="font-bold text-slate-800 dark:text-slate-100"
							>{systemInfo.disk_type}</span
						>
					</div>
					<div>
						Storage: <span class="font-bold text-slate-800 dark:text-slate-100"
							>{systemInfo.free_disk_gb} GB free / {systemInfo.total_disk_gb} GB</span
						>
					</div>
				</div>

				<!-- Col 3: System & Network -->
				<div class="space-y-1">
					<div
						class="font-bold text-gray-500 dark:text-slate-400 uppercase tracking-wider text-[10px]"
					>
						Network & User
					</div>
					<div>
						Network: <span class="font-bold text-slate-800 dark:text-slate-100 capitalize"
							>{systemInfo.network_status}</span
						>
					</div>
					<div>
						User: <span class="font-bold text-slate-800 dark:text-slate-100"
							>{systemInfo.logged_in_user}</span
						>
					</div>
					<div>
						Platform: <span class="font-bold text-slate-800 dark:text-slate-100"
							>{systemInfo.host_platform}</span
						>
					</div>
				</div>
			</div>

			<!-- Floating Toast Badge (Prevents button displacement) -->
			{#if logMessage}
				<div
					class="absolute bottom-14 left-6 z-30 bg-slate-900/95 dark:bg-slate-800/95 text-green-400 border border-green-500/40 px-3.5 py-2 rounded-xl shadow-2xl backdrop-blur-md text-xs font-semibold flex items-center gap-2 max-w-[420px]"
					title={logMessage}
				>
					<span class="w-2 h-2 rounded-full bg-green-500 animate-pulse shrink-0"></span>
					<span class="truncate">{logMessage}</span>
				</div>
			{/if}

			<!-- Bottom Action Bar (Fixed Right Alignment) -->
			<div class="flex justify-end items-center pt-1 w-full gap-2">
				<button
					class="dark:bg-zinc-800 bg-zinc-300 px-3 py-1.5 text-xs cursor-pointer rounded-lg hover:bg-zinc-400 dark:hover:bg-zinc-700 font-medium transition-transform active:scale-95 border border-slate-300 dark:border-slate-700/60 shadow-xs shrink-0"
					onclick={copyStatHandler}
				>
					Copy Stat
				</button>
				<button
					class="dark:bg-zinc-800 bg-zinc-300 px-3 py-1.5 text-xs cursor-pointer rounded-lg hover:bg-zinc-400 dark:hover:bg-zinc-700 font-medium transition-transform active:scale-95 border border-slate-300 dark:border-slate-700/60 shadow-xs text-sky-400 font-semibold shrink-0"
					onclick={exportSpecCardHandler}
				>
					Export Card
				</button>
				<button
					class="dark:bg-zinc-800 bg-zinc-300 px-3 py-1.5 text-xs cursor-pointer rounded-lg hover:bg-zinc-400 dark:hover:bg-zinc-700 font-medium transition-transform active:scale-95 border border-slate-300 dark:border-slate-700/60 shadow-xs shrink-0"
					onclick={() => (showTopProcesses = true)}
				>
					Top Processes
				</button>
				<button
					class="dark:bg-zinc-800 bg-zinc-300 px-3 py-1.5 text-xs cursor-pointer rounded-lg hover:bg-zinc-400 dark:hover:bg-zinc-700 font-medium transition-transform active:scale-95 border border-slate-300 dark:border-slate-700/60 shadow-xs shrink-0"
					onclick={getInternetSpeed}
				>
					Speed Test
				</button>
				<button
					class="dark:bg-zinc-800 bg-zinc-300 px-3 py-1.5 text-xs cursor-pointer rounded-lg hover:bg-zinc-400 dark:hover:bg-zinc-700 font-medium transition-transform active:scale-95 border border-slate-300 dark:border-slate-700/60 shadow-xs shrink-0"
					onclick={() => (showAdvancedDashboard = !showAdvancedDashboard)}
				>
					Advanced Metrics
				</button>
			</div>
		</div>
	{/if}
</div>

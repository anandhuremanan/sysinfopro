<script lang="ts">
	import Credits from '$lib/components/ui/Credits.svelte';
	import Logo from '$lib/components/ui/Logo.svelte';
	import Modal from '$lib/components/ui/Modal.svelte';
	import RefreshButton from '$lib/components/ui/RefreshButton.svelte';
	import SysInfoCard from '$lib/components/basic/SysInfoCard.svelte';
	import { GetSysInfo } from '$lib/wailsjs/go/main/App';
	import { Quit } from '$lib/wailsjs/runtime/runtime';

	let systemInfo = $state(undefined);
	let error = $state('');
	let showSpeedTest = $state(false);
	let showAdvancedDashboard = $state(false);
	let showTopProcesses = $state(false);
	let showQuitConfirm = $state(false);
	let isLoading = $state(false);

	function toggleBack() {
		if (showSpeedTest) {
			showSpeedTest = false;
		} else if (showAdvancedDashboard) {
			showAdvancedDashboard = false;
		} else if (showTopProcesses) {
			showTopProcesses = false;
		}
	}

	function toggleQuitModal() {
		showQuitConfirm = !showQuitConfirm;
	}

	async function getSysInfo() {
		isLoading = true;
		try {
			const result = await GetSysInfo();
			if (result) {
				systemInfo = JSON.parse(result);
			}
		} catch (err) {
			error = `Failed to get system information: ${err}`;
		} finally {
			setTimeout(() => {
				isLoading = false;
			}, 500);
		}
	}

	$effect(() => {
		getSysInfo();
	});
</script>

{#if isLoading}
	<div
		class="min-h-screen flex flex-col items-center justify-center dark:bg-slate-800 bg-slate-300"
	>
		<div class="flex flex-col items-center justify-center dark:text-white">
			<div class="flex"><Logo />™</div>
			<div class="text-xs">Loading...</div>
		</div>
	</div>
{:else}
	<section
		class="h-screen w-screen flex flex-col justify-between items-center p-4 dark:bg-slate-800 bg-slate-300 overflow-hidden select-none"
	>
		<!-- Fixed Height Header Bar -->
		<header class="flex justify-between items-center w-[95vw] h-10 shrink-0">
			<div class="flex items-center dark:text-white">
				<Logo />™
			</div>
			<div class="flex space-x-6 items-center h-full">
				{#if !showSpeedTest && !showAdvancedDashboard && !showTopProcesses}
					<RefreshButton onclick={getSysInfo} />
				{/if}

				{#if showAdvancedDashboard || showSpeedTest || showTopProcesses}
					<button
						class="dark:text-slate-400 cursor-pointer text-sm font-medium hover:text-slate-200 transition-colors"
						onclick={toggleBack}
					>
						Back
					</button>
				{/if}
				<button
					class="dark:text-slate-400 cursor-pointer text-sm font-medium hover:text-slate-200 transition-colors"
					onclick={toggleQuitModal}
				>
					Close
				</button>
			</div>
		</header>

		<!-- Main Card Container -->
		<main class="flex-1 flex items-center justify-center w-full min-h-0 my-2">
			{#if systemInfo}
				<SysInfoCard
					{systemInfo}
					bind:showSpeedTest
					bind:showAdvancedDashboard
					bind:showTopProcesses
				/>
			{/if}
		</main>

		<!-- Right-Aligned Footer Bar -->
		<footer class="w-[95vw] h-6 shrink-0 flex items-center justify-end">
			<Credits />
		</footer>

		<!-- Quit Confirmation Modal -->
		<Modal showModal={showQuitConfirm} showClose={false}>
			<div class="flex flex-col justify-center items-center text-sm gap-2">
				<div>Are You Sure to Quit?</div>
				<div class="flex gap-2 mt-2">
					<button class="px-2 py-1 bg-red-500 rounded-lg text-white cursor-pointer" onclick={Quit}>Yes</button>
					<button class="px-2 py-1 border border-slate-400 rounded-lg cursor-pointer" onclick={toggleQuitModal}>No</button>
				</div>
			</div>
		</Modal>
	</section>
{/if}

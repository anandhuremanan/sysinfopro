<script lang="ts">
	import { checkInternetConnection, copyStat, mbToGb, shortCpuName } from '$lib';
	import AdvancedDash from '$lib/components/advanced/AdvancedDash.svelte';
	import HomeDash from '$lib/components/basic/HomeDash.svelte';
	import ProcessView from '$lib/components/basic/ProcessView.svelte';
	import SpeedAnimation from '$lib/components/basic/SpeedAnimation.svelte';
	import ToolsView from '$lib/components/basic/ToolsView.svelte';
	import Logo from '$lib/components/ui/Logo.svelte';
	import Modal from '$lib/components/ui/Modal.svelte';
	import SettingsModal from '$lib/components/ui/SettingsModal.svelte';
	import StrokeIcon from '$lib/components/ui/StrokeIcon.svelte';
	import { getTool, tools, type ToolId, type ToolStatus } from '$lib/tools';
	import { exportSpecCardPNG } from '$lib/utils/cardExporter';
	import {
		GetActiveSockets,
		GetEnvPathAnalysis,
		GetStartupItems,
		GetSysInfo
	} from '$lib/wailsjs/go/main/App';
	import { Quit } from '$lib/wailsjs/runtime/runtime';

	type View = 'home' | 'processes' | 'tools' | ToolId;

	const SETTINGS_ICON =
		'M4 6h9M17 6h3M17 6a2 2 0 1 1-4 0a2 2 0 0 1 4 0zM4 12h3M11 12h9M11 12a2 2 0 1 1-4 0a2 2 0 0 1 4 0zM4 18h11M19 18h1M19 18a2 2 0 1 1-4 0a2 2 0 0 1 4 0z';

	let systemInfo: any = $state(undefined);
	let error = $state('');
	let view = $state<View>('home');
	// Where Back returns to: a tool opened from the tools list goes back to the list
	let parentView = $state<'home' | 'tools'>('home');
	let showQuitConfirm = $state(false);
	let showSettings = $state(false);
	let isLoading = $state(false);
	let logMessage = $state<string | undefined>(undefined);
	let toolStatus = $state<Record<string, ToolStatus>>({});

	let viewTitle = $derived(
		view === 'processes'
			? 'Processes'
			: view === 'tools'
				? 'All tools'
				: view === 'home'
					? ''
					: getTool(view).name
	);

	function showLog(message: string, duration = 2000) {
		logMessage = message;
		setTimeout(() => {
			logMessage = undefined;
		}, duration);
	}

	function openView(target: string) {
		if (target === 'speed') {
			openSpeedTest();
			return;
		}
		parentView = view === 'tools' ? 'tools' : 'home';
		view = target as View;
	}

	async function openSpeedTest() {
		const isConnected = await checkInternetConnection();

		if (!isConnected) {
			showLog('No internet connection detected!');
			return;
		}

		parentView = view === 'tools' ? 'tools' : 'home';
		view = 'speed';
	}

	function goBack() {
		view = view === 'tools' ? 'home' : parentView;
		parentView = 'home';
	}

	function toggleQuitModal() {
		showQuitConfirm = !showQuitConfirm;
	}

	async function copyStatHandler() {
		showLog(await copyStat(systemInfo));
	}

	async function exportSpecCardHandler() {
		const savedPath = await exportSpecCardPNG(systemInfo);
		showLog(savedPath ? `Spec Card saved: ${savedPath}` : 'Failed to export spec card.', 4000);
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

	// Cheap local checks only; Software and Speed test run when opened
	function loadToolStatus() {
		GetActiveSockets()
			.then((sockets) => {
				if (sockets) toolStatus.ports = { hint: `${sockets.length} listening` };
			})
			.catch((err) => console.error('Failed to get sockets:', err));

		GetStartupItems()
			.then((items) => {
				if (items) toolStatus.startup = { hint: `${items.length} apps` };
			})
			.catch((err) => console.error('Failed to get startup items:', err));

		GetEnvPathAnalysis()
			.then((analysis) => {
				if (!analysis) return;
				toolStatus.env =
					analysis.invalid_entries > 0
						? { hint: 'Broken entries', count: analysis.invalid_entries }
						: { hint: `${analysis.total_entries} entries` };
			})
			.catch((err) => console.error('Failed to analyse PATH:', err));
	}

	$effect(() => {
		getSysInfo();
		loadToolStatus();
	});
</script>

{#if isLoading}
	<div class="min-h-screen flex flex-col items-center justify-center bg-ground text-ink">
		<div class="flex flex-col items-center justify-center">
			<div class="flex"><Logo />™</div>
			<div class="text-xs">Loading...</div>
		</div>
	</div>
{:else}
	<section
		class="relative h-screen w-screen flex flex-col gap-3 px-4 pt-3 pb-3.5 bg-ground text-ink text-xs overflow-hidden select-none"
	>
		<!-- Fixed Height Header Bar -->
		<header class="flex justify-between items-center gap-3 h-10 shrink-0">
			<div class="flex items-center gap-2.5 min-w-0">
				<div class="flex items-center shrink-0">
					<Logo />™
				</div>
				{#if view === 'home'}
					{#if systemInfo}
						<span
							class="text-muted truncate"
							title="{systemInfo.host_platform} {systemInfo.host_platform_version} · {systemInfo.cpu_model} ({systemInfo.cpu_cores} cores) · {mbToGb(
								systemInfo.total_ram_mb
							)} GB RAM"
						>
							{(systemInfo.host_platform || 'Desktop System').replace(/^Microsoft /, '')} · {mbToGb(
								systemInfo.total_ram_mb
							)} GB · {shortCpuName(systemInfo.cpu_model)}
						</span>
					{/if}
				{:else}
					<nav aria-label="Breadcrumb" class="flex items-center gap-1.5 text-[13px] min-w-0">
						<button
							class="h-7 pl-1 pr-2 rounded-lg flex items-center gap-0.5 text-muted font-medium cursor-pointer hover:text-ink hover:bg-line transition-colors"
							onclick={goBack}
						>
							<StrokeIcon d="M15 6l-6 6 6 6" size={14} />
							{view === 'tools' || parentView === 'home' ? 'Home' : 'Tools'}
						</button>
						<span class="text-muted">/</span>
						<span aria-current="page" class="font-bold truncate">{viewTitle}</span>
					</nav>
				{/if}
			</div>

			<div class="flex items-center gap-1 shrink-0">
				{#if view === 'home'}
					<button
						class="h-7 px-2.5 rounded-lg text-muted font-medium cursor-pointer hover:text-ink hover:bg-line transition-colors"
						onclick={getSysInfo}
					>
						Refresh
					</button>
					<button
						class="h-7 px-2.5 rounded-lg text-muted font-medium cursor-pointer hover:text-ink hover:bg-line transition-colors"
						onclick={copyStatHandler}
					>
						Copy stats
					</button>
					<button
						class="h-7 px-3 rounded-lg bg-ink text-tile font-semibold cursor-pointer transition-transform active:scale-95"
						onclick={exportSpecCardHandler}
					>
						Export card
					</button>
				{/if}
				<button
					aria-label="Settings"
					title="Settings"
					class="w-7 h-7 ml-1 rounded-lg flex items-center justify-center text-muted cursor-pointer hover:text-ink hover:bg-line transition-colors"
					onclick={() => (showSettings = true)}
				>
					<StrokeIcon d={SETTINGS_ICON} />
				</button>
				<button
					aria-label="Close"
					title="Close"
					class="w-7 h-7 rounded-lg flex items-center justify-center text-muted cursor-pointer hover:text-ink hover:bg-line transition-colors"
					onclick={toggleQuitModal}
				>
					<StrokeIcon d="M6 6l12 12M18 6L6 18" />
				</button>
			</div>
		</header>

		<!-- Main Content -->
		<main class="flex-1 min-h-0">
			{#if error && !systemInfo}
				<div class="h-full flex items-center justify-center text-red-500">{error}</div>
			{:else if systemInfo}
				{#if view === 'home'}
					<HomeDash {systemInfo} {toolStatus} onOpen={openView} onSpeedTest={openSpeedTest} />
				{:else if view === 'tools'}
					<ToolsView {toolStatus} onOpen={openView} />
				{:else}
					<div class="h-full bg-tile rounded-2xl p-4">
						{#if view === 'speed'}
							<div class="h-full flex items-center justify-center">
								<SpeedAnimation />
							</div>
						{:else if view === 'processes'}
							<ProcessView />
						{:else}
							<AdvancedDash {systemInfo} tool={view} />
						{/if}
					</div>
				{/if}
			{/if}
		</main>

		<!-- Floating Toast Badge -->
		{#if logMessage}
			<div
				class="absolute bottom-5 left-6 z-30 bg-slate-900/95 text-green-400 border border-green-500/40 px-3.5 py-2 rounded-xl shadow-2xl backdrop-blur-md text-xs font-semibold flex items-center gap-2 max-w-[420px]"
				title={logMessage}
			>
				<span class="w-2 h-2 rounded-full bg-green-500 animate-pulse shrink-0"></span>
				<span class="truncate">{logMessage}</span>
			</div>
		{/if}

		<SettingsModal bind:showModal={showSettings} />

		<!-- Quit Confirmation Modal -->
		<Modal
			showModal={showQuitConfirm}
			showClose={false}
			width="max-w-xs"
			title="Quit SysInfo Pro?"
			description="Any scan or test still running will stop."
		>
			{#snippet actions()}
				<button class="btn btn-secondary" onclick={toggleQuitModal}>Cancel</button>
				<button class="btn btn-danger" onclick={Quit}>Quit</button>
			{/snippet}
		</Modal>
	</section>
{/if}

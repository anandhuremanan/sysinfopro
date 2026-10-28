<script lang="ts">
	import { onMount } from 'svelte';
	import { checkInternetConnection, copyStat, mbToGb, shortCpuName } from '$lib';
	import HomeDash from '$lib/components/basic/HomeDash.svelte';
	import ToolsView from '$lib/components/basic/ToolsView.svelte';
	import WidgetView from '$lib/components/basic/WidgetView.svelte';
	import Logo from '$lib/components/ui/Logo.svelte';
	import Modal from '$lib/components/ui/Modal.svelte';
	import AlertBanner from '$lib/components/ui/AlertBanner.svelte';
	import QuickSearch from '$lib/components/ui/QuickSearch.svelte';
	import RatingPrompt from '$lib/components/ui/RatingPrompt.svelte';
	import SettingsModal from '$lib/components/ui/SettingsModal.svelte';
	import ReportModal from '$lib/components/ui/ReportModal.svelte';
	import StrokeIcon from '$lib/components/ui/StrokeIcon.svelte';
	import Tour from '$lib/components/ui/Tour.svelte';
	import TourInvite from '$lib/components/ui/TourInvite.svelte';
	import Welcome from '$lib/components/ui/Welcome.svelte';
	import { loadOnboarding, onboarding } from '$lib/onboarding.svelte';
	import { listenForAlerts, type DriveAlert } from '$lib/alerts.svelte';
	import { type ToolStatus } from '$lib/tools';
	import { isView, popView, pushView, screens, viewTitle, type View } from '$lib/screens';
	import { Status } from '$lib/status.svelte';
	import { exportSpecCardPNG } from '$lib/utils/cardExporter';
	import { enterWidget, initWidget, toggleWidget, widget } from '$lib/widget.svelte';
	import {
		GetDriveHealth,
		GetEnvPathAnalysis,
		GetHardwareSpecs,
		GetSecurityReport,
		GetStartupItems,
		GetSysInfo
	} from '$lib/wailsjs/go/main/App';
	import { EventsOn, Quit, WindowMinimise } from '$lib/wailsjs/runtime/runtime';
	import { prefetch, remember } from '$lib/cache';
	import { setActiveView, tasks } from '$lib/tasks.svelte';
	import { cubicOut } from 'svelte/easing';
	import { fly } from 'svelte/transition';

	// screens ease in over a sixth of a second, or appear at once with reduced motion
	const enterMs =
		typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
			? 0
			: 160;

	const WIDGET_ICON = 'M3 5h18v14H3zM12 12h7v5h-7z';
	const SHARE_ICON = 'M12 3v12M7 8l5-5 5 5M5 14v5a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-5';
	const SETTINGS_ICON =
		'M4 6h9M17 6h3M17 6a2 2 0 1 1-4 0a2 2 0 0 1 4 0zM4 12h3M11 12h9M11 12a2 2 0 1 1-4 0a2 2 0 0 1 4 0zM4 18h11M19 18h1M19 18a2 2 0 1 1-4 0a2 2 0 0 1 4 0z';

	let systemInfo: any = $state(undefined);
	let error = $state('');
	// the screens visited, most recent last; Back steps down it
	let stack = $state<View[]>(['home']);
	let view = $derived(stack[stack.length - 1]);
	let previous = $derived(stack.length > 1 ? stack[stack.length - 2] : 'home');
	let showQuitConfirm = $state(false);
	let showSettings = $state(false);
	let isLoading = $state(false);
	const toast = new Status(2000);
	let toolStatus = $state<Record<string, ToolStatus>>({});
	// the drive Disk usage opens on when reached from an alert; cleared once used
	let usageDrive = $state<string | undefined>(undefined);
	let showQuickSearch = $state(false);

	// Ctrl+K opens quick search from anywhere in the full window
	function onWindowKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape' && showShare) showShare = false;
		if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k' && !widget.active) {
			event.preventDefault();
			showQuickSearch = !showQuickSearch;
		}
	}

	function quickAction(id: 'settings' | 'widget' | 'processes' | 'home') {
		if (id === 'settings') showSettings = true;
		else if (id === 'widget') enterWidget();
		else openView(id);
	}

	function openView(target: string) {
		if (target === 'speed') {
			openSpeedTest();
			return;
		}
		if (isView(target)) stack = pushView(stack, target);
	}

	async function openSpeedTest() {
		if (!(await checkInternetConnection())) {
			toast.show('No internet connection detected!');
			return;
		}
		stack = pushView(stack, 'speed');
	}

	function goBack() {
		stack = popView(stack);
	}

	function toggleQuitModal() {
		showQuitConfirm = !showQuitConfirm;
	}

	// Share menu: copying and saving the PC's specs, out of the way until needed
	let showShare = $state(false);
	let showReport = $state(false);

	function openReport() {
		showShare = false;
		showReport = true;
	}

	async function copyStatHandler() {
		showShare = false;
		toast.show(await copyStat(systemInfo));
	}

	async function exportSpecCardHandler() {
		showShare = false;
		const savedPath = await exportSpecCardPNG(systemInfo);
		toast.show(savedPath ? `Spec Card saved: ${savedPath}` : 'Failed to export spec card.', 4000);
	}

	async function getSysInfo() {
		isLoading = true;
		try {
			const result = await GetSysInfo();
			if (result) {
				systemInfo = result;
				remember('sysinfo', systemInfo);
			}
		} catch (err) {
			error = `Failed to get system information: ${err}`;
		} finally {
			isLoading = false;
		}
	}

	// Cheap local checks only; Software and Speed test run when opened
	function loadToolStatus() {
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

	onMount(() => {
		loadOnboarding();
		getSysInfo();
		loadToolStatus();
		initWidget();
		listenForAlerts();
		const offHotkey = EventsOn('widget-hotkey', toggleWidget);
		// fetch the slower screens' data once the window has settled, so they open at once
		const timer = setTimeout(() => {
			prefetch('specs', GetHardwareSpecs);
			prefetch('security', GetSecurityReport);
			prefetch('drive-health', GetDriveHealth);
		}, 3000);
		return () => {
			clearTimeout(timer);
			offHotkey();
		};
	});

	// the tour walks around the home screen, so it always starts there
	$effect(() => {
		if (onboarding.tour && view !== 'home') stack = ['home'];
	});

	// jobs still running elsewhere show as chips in the header
	$effect(() => {
		setActiveView(view);
	});
	let backgroundTasks = $derived(tasks.running.filter((t) => t.view !== view));

	function openForAlert(alert: DriveAlert) {
		if (alert.kind === 'task' && alert.view) {
			openView(alert.view);
			return;
		}
		if (alert.kind === 'reminder') {
			openView('devcache');
			return;
		}
		usageDrive = alert.drive;
		openView('space');
	}
</script>

<svelte:window onkeydown={onWindowKeydown} />

{#if widget.active}
	<WidgetView />
{:else if isLoading && !systemInfo}
	<div class="min-h-screen flex flex-col items-center justify-center bg-ground text-ink">
		<div class="flex flex-col items-center justify-center">
			<Logo size={36} />
			<div class="text-xs">Loading...</div>
		</div>
	</div>
{:else}
	<section
		class="relative h-screen w-screen flex flex-col gap-3 px-4 pt-3 pb-3.5 bg-ground text-ink text-xs overflow-hidden select-none"
	>
		<!-- Fixed Height Header Bar -->
		<header class="drag-region flex justify-between items-center gap-3 h-10 shrink-0">
			<div class="flex items-center gap-2.5 min-w-0">
				<div class="flex items-center shrink-0">
					<Logo />
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
							{viewTitle(previous)}
						</button>
						<span class="text-muted">/</span>
						<span aria-current="page" class="font-bold truncate">{viewTitle(view)}</span>
					</nav>
				{/if}
			</div>

			<div class="flex items-center gap-1 shrink-0">
				{#each backgroundTasks as task (task.id)}
					<button
						class="h-7 max-w-[220px] pl-2 pr-2.5 mr-1 rounded-full flex items-center gap-2 bg-tile border border-line text-ink cursor-pointer hover:border-accent transition-colors"
						title="Running in the background. Click to see it."
						onclick={() => openView(task.view)}
					>
						<span class="relative shrink-0 w-3.5 h-3.5">
							<span class="absolute inset-0 rounded-full border-2 border-accent/30"></span>
							<span
								class="absolute inset-0 rounded-full border-2 border-accent border-t-transparent motion-safe:animate-spin"
							></span>
						</span>
						<span class="font-semibold truncate">{task.label}</span>
						{#if task.detail}
							<span class="text-muted truncate tabular-nums">{task.detail}</span>
						{/if}
					</button>
				{/each}
				{#if view === 'home' && systemInfo}
					<div class="relative">
						<button
							aria-label="Share"
							data-tour="share"
							aria-haspopup="menu"
							aria-expanded={showShare}
							title="Copy or save this PC's specs"
							class="w-7 h-7 rounded-lg flex items-center justify-center cursor-pointer transition-colors {showShare
								? 'text-ink bg-line'
								: 'text-muted hover:text-ink hover:bg-line'}"
							onclick={() => (showShare = !showShare)}
						>
							<StrokeIcon d={SHARE_ICON} />
						</button>
						{#if showShare}
							<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
							<div class="fixed inset-0 z-40" onclick={() => (showShare = false)}></div>
							<div
								role="menu"
								class="absolute right-0 top-9 z-50 w-56 p-1 rounded-xl bg-tile border border-line shadow-2xl"
							>
								{#each [{ label: 'Copy stats', hint: 'System details as text', run: copyStatHandler }, { label: 'Save spec card', hint: 'A shareable image, to Downloads', run: exportSpecCardHandler }, { label: 'Save full report', hint: 'Specs, health and checks, to Downloads', run: openReport }] as item (item.label)}
									<button
										role="menuitem"
										class="w-full flex flex-col items-start px-2.5 py-1.5 rounded-lg text-left cursor-pointer hover:bg-ground"
										onclick={item.run}
									>
										<span class="font-semibold text-ink">{item.label}</span>
										<span class="text-[11px] text-muted">{item.hint}</span>
									</button>
								{/each}
							</div>
						{/if}
					</div>
				{/if}
				<button
					aria-label="Shrink to widget"
					data-tour="widget"
					title="Shrink to a small always-visible widget"
					class="w-7 h-7 rounded-lg flex items-center justify-center text-muted cursor-pointer hover:text-ink hover:bg-line transition-colors"
					onclick={enterWidget}
				>
					<StrokeIcon d={WIDGET_ICON} />
				</button>
				<button
					aria-label="Settings"
					data-tour="settings"
					title="Settings"
					class="w-7 h-7 rounded-lg flex items-center justify-center text-muted cursor-pointer hover:text-ink hover:bg-line transition-colors"
					onclick={() => (showSettings = true)}
				>
					<StrokeIcon d={SETTINGS_ICON} />
				</button>
				<button
					aria-label="Minimise"
					title="Minimise"
					class="w-7 h-7 rounded-lg flex items-center justify-center text-muted cursor-pointer hover:text-ink hover:bg-line transition-colors"
					onclick={WindowMinimise}
				>
					<StrokeIcon d="M6 12h12" />
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
				<div class="h-full flex items-center justify-center text-fail">{error}</div>
			{:else if systemInfo}
				<!-- the new screen eases in; the old one goes at once, so a click never waits -->
				{#key view}
					<div class="h-full" in:fly={{ y: 6, duration: enterMs, easing: cubicOut }}>
						{#if view === 'home'}
							<HomeDash {systemInfo} {toolStatus} onOpen={openView} onSpeedTest={openSpeedTest} />
						{:else if view === 'tools'}
							<ToolsView {toolStatus} onOpen={openView} />
						{:else}
							{@const Screen = screens[view]}
							<div class="h-full bg-tile rounded-2xl p-4">
								<Screen
									{systemInfo}
									onOpen={openView}
									drive={view === 'space' ? usageDrive : undefined}
									onDriveUsed={() => (usageDrive = undefined)}
								/>
							</div>
						{/if}
					</div>
				{/key}
			{/if}
		</main>

		<!-- Floating Toast Badge -->
		{#if toast.text}
			<div
				class="absolute bottom-5 left-6 z-30 bg-tile text-ink border border-ok/40 px-3.5 py-2 rounded-xl shadow-2xl text-xs font-semibold flex items-center gap-2 max-w-[420px]"
				title={toast.text}
				role="status"
			>
				<span class="w-2 h-2 rounded-full bg-ok shrink-0"></span>
				<span class="truncate">{toast.text}</span>
			</div>
		{/if}

		<AlertBanner onOpen={openForAlert} />

		<QuickSearch bind:open={showQuickSearch} onOpen={openView} onAction={quickAction} />

		<RatingPrompt />

		<SettingsModal bind:showModal={showSettings} />
		<ReportModal bind:showModal={showReport} {systemInfo} />

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

{#if !widget.active}
	{#if onboarding.welcome}
		<Welcome />
	{:else if onboarding.tour && view === 'home' && systemInfo}
		<Tour />
	{:else if onboarding.invite && view === 'home' && systemInfo}
		<!-- after the welcome screens, a first-time user is offered the tour once -->
		<TourInvite />
	{/if}
{/if}

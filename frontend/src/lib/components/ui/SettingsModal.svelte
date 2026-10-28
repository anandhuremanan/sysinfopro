<script lang="ts">
	import { ACCENT_PRESETS, setAccent, setTheme, theme, type ThemeMode } from '$lib/theme.svelte';
	import { enterWidget, toggleWidgetMetric, widget, widgetOptions } from '$lib/widget.svelte';
	import {
		CancelDataFolderMove,
		ChooseDataFolder,
		GetAlertSettings,
		GetDataFolder,
		GetDiagnosticsText,
		GetHotkeyCombos,
		GetHotkeySettings,
		GetNotificationStatus,
		GetReminderSettings,
		RestartApp,
		SendTestAlert,
		SetAlertSettings,
		SetHotkeySettings,
		SetReminderEnabled,
		UseDefaultDataFolder
	} from '$lib/wailsjs/go/main/App';
	import type { appdata } from '$lib/wailsjs/go/models';
	import { formatBytes } from '$lib';
	import {
		CARD_STYLES,
		generateSpecCardPNG,
		getCardStyle,
		setCardStyle,
		type CardStyle
	} from '$lib/utils/cardExporter';
	import { fade } from 'svelte/transition';
	// with Windows' "reduce motion" on, tabs switch without the fade
	const reduceMotion =
		typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
	import Logo from './Logo.svelte';
	import { showWelcome, startTour } from '$lib/onboarding.svelte';
	import { BrowserOpenURL, ClipboardSetText } from '$lib/wailsjs/runtime/runtime';

	// version, Windows build and recent log lines, for a bug report
	let diagnosticsCopied = $state(false);
	async function copyDiagnostics() {
		try {
			await ClipboardSetText(await GetDiagnosticsText());
			diagnosticsCopied = true;
			setTimeout(() => (diagnosticsCopied = false), 2000);
		} catch (err) {
			console.error('Could not copy diagnostics:', err);
		}
	}
	import Segmented from './Segmented.svelte';
	import Modal from './Modal.svelte';
	import Switch from './Switch.svelte';

	let { showModal = $bindable(false) } = $props();

	type Tab = 'general' | 'card' | 'widget' | 'alerts' | 'about';
	type AlertSettings = { enabled: boolean; min_free_gb: number; drop_gb: number };
	type HotkeySettings = { enabled: boolean; combo: string; error?: string };

	const tabs: { id: Tab; label: string }[] = [
		{ id: 'general', label: 'General' },
		{ id: 'card', label: 'Card' },
		{ id: 'widget', label: 'Widget' },
		{ id: 'alerts', label: 'Alerts' },
		{ id: 'about', label: 'About' }
	];

	const LINKS = [
		{ label: 'Website', url: 'https://sysinfopro.imanandhu.in' },
		{ label: 'Source code', url: 'https://github.com/anandhuremanan/sysinfopro' },
		{ label: 'Report a bug', url: 'https://github.com/anandhuremanan/sysinfopro/issues' },
		{ label: 'Privacy', url: 'https://sysinfopro.imanandhu.in/privacy.html' }
	];
	const modes: { id: ThemeMode; label: string }[] = [
		{ id: 'system', label: 'System' },
		{ id: 'light', label: 'Light' },
		{ id: 'dark', label: 'Dark' },
		{ id: 'amoled', label: 'AMOLED' }
	];
	// a colour picked with the custom picker, as opposed to one of the presets
	let isCustomAccent = $derived(
		!!theme.accent && !ACCENT_PRESETS.some((p) => p.id === theme.accent)
	);
	const LOW_LIMITS = [2, 5, 10, 20];
	const DROP_LIMITS = [0, 1, 2, 5];

	const NOTIFICATION_NOTE: Record<string, string> = {
		enabled: 'Alerts appear as Windows notifications and inside SysInfo Pro.',
		off_for_app:
			'Windows notifications are off for SysInfo Pro (Windows Settings > System > Notifications), so alerts appear inside SysInfo Pro only.',
		off: 'Windows notifications are turned off on this PC, so alerts appear inside SysInfo Pro only.',
		blocked_by_policy:
			'Your organisation has turned off Windows notifications on this PC, so alerts appear inside SysInfo Pro and its widget only.',
		unknown: 'Alerts appear inside SysInfo Pro, and as Windows notifications where allowed.'
	};

	let tab = $state<Tab>('general');

	// the share card's look, previewed as it will be saved
	let cardStyle = $state<CardStyle>(getCardStyle());
	let cardPreview = $state('');
	let previewToken = 0;

	$effect(() => {
		if (!showModal || tab !== 'card') return;
		// redraw when the style, theme or accent changes
		const style = cardStyle;
		theme.mode;
		theme.accent;
		const token = ++previewToken;
		generateSpecCardPNG(undefined, style).then((url) => {
			if (token === previewToken) cardPreview = url;
		});
	});

	function pickCardStyle(style: CardStyle) {
		cardStyle = style;
		setCardStyle(style);
	}
	let alertSettings = $state<AlertSettings | null>(null);
	let notificationStatus = $state<string>('');
	let hotkeySettings = $state<HotkeySettings | null>(null);
	let reminder = $state<{ enabled: boolean; projects_root: string } | null>(null);
	let hotkeyCombos = $state<string[]>([]);

	async function loadHotkey() {
		try {
			[hotkeySettings, hotkeyCombos] = await Promise.all([GetHotkeySettings(), GetHotkeyCombos()]);
		} catch (err) {
			console.error('Failed to read shortcut settings:', err);
		}
	}

	async function updateHotkey(enabled: boolean, combo: string) {
		try {
			hotkeySettings = await SetHotkeySettings(enabled, combo);
		} catch (err) {
			console.error('Failed to save shortcut settings:', err);
		}
	}

	async function loadAlerts() {
		try {
			[alertSettings, reminder] = await Promise.all([GetAlertSettings(), GetReminderSettings()]);
		} catch (err) {
			console.error('Failed to read alert settings:', err);
		}
		notificationStatus = await GetNotificationStatus().catch(() => 'unknown');
	}

	async function updateAlerts(change: Partial<AlertSettings>) {
		if (!alertSettings) return;
		alertSettings = { ...alertSettings, ...change };
		try {
			await SetAlertSettings(alertSettings);
		} catch (err) {
			console.error('Failed to save alert settings:', err);
		}
	}

	// where the app keeps its files; a move happens at the next start
	let dataFolder = $state<appdata.Status | null>(null);
	let dataFolderError = $state('');
	let dataFolderBusy = $state(false);

	async function loadDataFolder() {
		try {
			dataFolder = await GetDataFolder();
		} catch (err) {
			console.error('Failed to read the data folder:', err);
		}
	}

	async function changeDataFolder(action: () => Promise<appdata.Status>) {
		dataFolderBusy = true;
		dataFolderError = '';
		try {
			dataFolder = await action();
		} catch (err) {
			dataFolderError = String(err);
			await loadDataFolder();
		} finally {
			dataFolderBusy = false;
		}
	}

	async function restart() {
		dataFolderBusy = true;
		try {
			await RestartApp();
		} catch (err) {
			dataFolderError = String(err);
			dataFolderBusy = false;
		}
	}

	$effect(() => {
		if (showModal && tab === 'general' && !dataFolder) loadDataFolder();
		if (showModal && tab === 'alerts' && !alertSettings) loadAlerts();
		if (showModal && tab === 'widget' && !hotkeySettings) loadHotkey();
	});
</script>

{#snippet chips(
	options: any[],
	selected: any,
	label: (value: any) => string,
	onpick: (value: any) => void,
	disabled = false,
	group = ''
)}
	<Segmented
		options={options.map((o) => ({ id: o, label: label(o) }))}
		value={selected}
		onchange={onpick}
		{disabled}
		label={group}
		size="sm"
	/>
{/snippet}

<Modal bind:showModal title="Settings">
	<div class="flex flex-col gap-5">
		<div class="self-start">
			<Segmented
				options={tabs}
				value={tab}
				onchange={(t) => (tab = t)}
				label="Settings sections"
				tabs
			/>
		</div>

		{#key tab}
			<div class="flex flex-col gap-5" in:fade={{ duration: reduceMotion ? 0 : 160 }}>
				{#if tab === 'general'}
					<div class="flex items-center justify-between gap-4">
						<div>
							<div class="font-semibold">Theme</div>
							<div class="text-muted">AMOLED is pure black for OLED screens</div>
						</div>
						{@render chips(
							modes.map((m) => m.id),
							theme.mode,
							(id) => modes.find((m) => m.id === id)?.label ?? id,
							setTheme,
							false,
							'Theme'
						)}
					</div>

					<div class="flex items-center justify-between gap-4">
						<div>
							<div class="font-semibold">Accent</div>
							<div class="text-muted">Buttons, bars and highlights</div>
						</div>
						<div role="radiogroup" aria-label="Accent colour" class="flex items-center gap-1.5">
							{#each ACCENT_PRESETS as preset (preset.id)}
								<button
									role="radio"
									aria-checked={theme.accent === preset.id}
									aria-label={preset.label}
									title={preset.label}
									class="w-6 h-6 rounded-full cursor-pointer ring-offset-2 ring-offset-tile transition-shadow {theme.accent ===
									preset.id
										? 'ring-2 ring-ink'
										: 'hover:ring-2 hover:ring-line'}"
									style="background: {preset.color}"
									onclick={() => setAccent(preset.id)}
								></button>
							{/each}
							<label
								title="Pick any colour"
								class="relative w-6 h-6 rounded-full cursor-pointer ring-offset-2 ring-offset-tile overflow-hidden {isCustomAccent
									? 'ring-2 ring-ink'
									: 'hover:ring-2 hover:ring-line'}"
								style="background: {isCustomAccent
									? theme.accent
									: 'conic-gradient(#ff3b3b, #ffd400, #39ff14, #00e5ff, #a855f7, #ff2bd6, #ff3b3b)'}"
							>
								<span class="sr-only">Custom colour</span>
								<input
									type="color"
									class="absolute inset-0 opacity-0 cursor-pointer"
									value={theme.accent || '#2dd4bf'}
									oninput={(event) => setAccent(event.currentTarget.value)}
								/>
							</label>
						</div>
					</div>

					<div class="flex flex-col gap-2">
						<div class="flex items-center justify-between gap-4">
							<div class="min-w-0">
								<div class="font-semibold">Data folder</div>
								<div class="text-muted">
									Settings, history and logs{dataFolder
										? ` · ${formatBytes(dataFolder.size_bytes)}`
										: ''}
								</div>
							</div>
							<div class="flex items-center gap-2 shrink-0">
								{#if dataFolder && !dataFolder.is_default && !dataFolder.pending}
									<button
										class="btn btn-secondary"
										disabled={dataFolderBusy}
										onclick={() => changeDataFolder(UseDefaultDataFolder)}
									>
										Use default
									</button>
								{/if}
								<button
									class="btn btn-secondary"
									disabled={dataFolderBusy || !dataFolder}
									onclick={() => changeDataFolder(ChooseDataFolder)}
								>
									Move...
								</button>
							</div>
						</div>
						{#if dataFolder}
							<div class="font-mono text-xs text-muted break-all" title={dataFolder.dir}>
								{dataFolder.dir}
							</div>
						{/if}
						{#if dataFolder?.pending}
							<div
								class="flex items-center justify-between gap-3 rounded-xl bg-accent/10 px-3 py-2"
							>
								<span class="leading-snug min-w-0">
									Moves to <span class="font-mono text-xs break-all">{dataFolder.pending_to}</span>
									when SysInfo Pro restarts.
								</span>
								<div class="flex items-center gap-2 shrink-0">
									<button
										class="btn btn-secondary"
										disabled={dataFolderBusy}
										onclick={() => changeDataFolder(CancelDataFolderMove)}
									>
										Cancel
									</button>
									<button class="btn btn-primary" disabled={dataFolderBusy} onclick={restart}>
										Restart now
									</button>
								</div>
							</div>
						{/if}
						{#if dataFolderError}
							<p class="text-fail leading-snug" role="alert">{dataFolderError}</p>
						{:else if dataFolder?.notice}
							<p class="leading-snug {dataFolder.notice_is_error ? 'text-warn' : 'text-muted'}">
								{dataFolder.notice}
							</p>
						{/if}
					</div>

					<div class="flex items-center justify-between gap-4">
						<div>
							<div class="font-semibold">Getting started</div>
							<div class="text-muted">The welcome screens, or a short tour of the home screen</div>
						</div>
						<div class="flex items-center gap-2 shrink-0">
							<button
								class="btn btn-secondary"
								onclick={() => {
									showModal = false;
									showWelcome();
								}}
							>
								Welcome screens
							</button>
							<button
								class="btn btn-secondary"
								onclick={() => {
									showModal = false;
									startTour();
								}}
							>
								Take the tour
							</button>
						</div>
					</div>
				{:else if tab === 'card'}
					<div class="flex flex-col gap-2.5">
						<div class="flex items-center justify-between gap-4">
							<div class="min-w-0">
								<div class="font-semibold">Share card</div>
								<div class="text-muted leading-snug">
									{CARD_STYLES.find((s) => s.id === cardStyle)?.hint}
								</div>
							</div>
							{@render chips(
								CARD_STYLES.map((s) => s.id),
								cardStyle,
								(id) => CARD_STYLES.find((s) => s.id === id)?.label ?? id,
								pickCardStyle,
								false,
								'Share card style'
							)}
						</div>
						<!-- a faint outline, so a card the same colour as this window still shows its edges -->
						<div class="rounded-xl overflow-hidden bg-ground aspect-[16/9] ring-1 ring-line">
							{#if cardPreview}
								<img
									src={cardPreview}
									alt="Preview of the share card"
									class="w-full h-full object-cover"
								/>
							{/if}
						</div>
					</div>
				{:else if tab === 'widget'}
					<div class="flex items-center justify-between gap-4">
						<div>
							<div class="font-semibold">Widget</div>
							<div class="text-muted">A small window that stays on top of every app</div>
						</div>
						<button
							class="btn btn-primary"
							onclick={() => {
								showModal = false;
								enterWidget();
							}}
						>
							Open widget
						</button>
					</div>
					<div class="flex flex-col gap-2">
						<span class="font-semibold">Show</span>
						<div
							class="flex flex-wrap gap-1.5"
							role="group"
							aria-label="Readings shown in the widget"
						>
							{#each widgetOptions() as metric (metric.id)}
								<button
									aria-pressed={widget.metrics.includes(metric.id)}
									class="h-7 px-2.5 rounded-lg border cursor-pointer font-medium transition-colors {widget.metrics.includes(
										metric.id
									)
										? 'border-accent bg-accent text-on-accent'
										: 'border-line text-muted hover:text-ink'}"
									onclick={() => toggleWidgetMetric(metric.id)}
								>
									{metric.label}
								</button>
							{/each}
						</div>
					</div>
					{#if hotkeySettings}
						<div class="flex flex-col gap-2">
							<div class="flex items-center justify-between gap-4">
								<div>
									<div class="font-semibold">Keyboard shortcut</div>
									<div class="text-muted">Switch to the widget and back from any app</div>
								</div>
								<Switch
									label="Keyboard shortcut"
									checked={hotkeySettings.enabled}
									onchange={(enabled) => updateHotkey(enabled, hotkeySettings!.combo)}
								/>
							</div>
							<div class="flex items-center justify-between gap-4">
								{#if hotkeySettings.error}
									<span class="text-warn leading-snug">
										{hotkeySettings.error}. Pick another.
									</span>
								{:else}
									<span class={hotkeySettings.enabled ? 'text-muted' : 'text-muted opacity-60'}>
										{hotkeySettings.enabled ? 'Works while SysInfo Pro is open' : 'Off'}
									</span>
								{/if}
								{@render chips(
									hotkeyCombos,
									hotkeySettings.combo,
									(combo) => combo,
									(combo) => updateHotkey(hotkeySettings!.enabled, combo),
									!hotkeySettings.enabled,
									'Shortcut'
								)}
							</div>
						</div>
					{/if}
				{:else if tab === 'about'}
					<div class="flex items-center justify-between gap-4">
						<div class="flex flex-col gap-1.5">
							<Logo size={32} />
							<span class="text-muted">
								Version {__APP_VERSION__} · by Probably Fine Labs
							</span>
						</div>
						<div class="flex flex-col items-center gap-1 shrink-0 text-muted text-center">
							<img src="/Open_Source_Initiative.svg" alt="" class="w-9 h-9" />
							<span>Open source, MIT License</span>
						</div>
					</div>

					<p class="leading-relaxed">
						SysInfo Pro explains, in plain words, why a Windows PC is slow, what is filling its
						drives and what is wrong with a developer setup, and helps fix it. Every reading is what
						Windows and the drives report, and every tool works without administrator rights.
					</p>

					<div class="flex flex-wrap gap-2">
						{#each LINKS as link (link.url)}
							<button class="btn btn-secondary" onclick={() => BrowserOpenURL(link.url)}>
								{link.label}
							</button>
						{/each}
					</div>

					<div class="flex items-center justify-between gap-4">
						<div>
							<div class="font-semibold">Diagnostics</div>
							<div class="text-muted">
								Version, Windows build and recent log lines, to paste into a bug report
							</div>
						</div>
						<button class="btn btn-secondary shrink-0" onclick={copyDiagnostics}>
							{diagnosticsCopied ? 'Copied' : 'Copy diagnostics'}
						</button>
					</div>

					<div class="text-muted pt-3 border-t border-line">
						Made with Go, Wails, Svelte and Tailwind · © {new Date().getFullYear()} Probably Fine Labs
					</div>
				{:else if !alertSettings}
					<div class="h-32 flex items-center justify-center text-muted motion-safe:animate-pulse">
						Loading...
					</div>
				{:else}
					<div class="flex items-center justify-between gap-4">
						<div>
							<div class="font-semibold">Low-space alerts</div>
							<div class="text-muted">Checks every drive twice a minute while the app is open</div>
						</div>
						<Switch
							label="Low-space alerts"
							checked={alertSettings.enabled}
							onchange={(enabled) => updateAlerts({ enabled })}
						/>
					</div>

					<div class="flex flex-col gap-3">
						<div class="flex items-center justify-between gap-4">
							<span class={alertSettings.enabled ? '' : 'text-muted'}
								>When free space drops below</span
							>
							{@render chips(
								LOW_LIMITS,
								alertSettings.min_free_gb,
								(gb) => `${gb} GB`,
								(gb) => updateAlerts({ min_free_gb: gb }),
								!alertSettings.enabled,
								'Low-space limit'
							)}
						</div>
						<div class="flex items-center justify-between gap-4">
							<span class={alertSettings.enabled ? '' : 'text-muted'}
								>When a drive loses, in 10 minutes</span
							>
							{@render chips(
								DROP_LIMITS,
								alertSettings.drop_gb,
								(gb) => (gb === 0 ? 'Off' : `${gb} GB`),
								(gb) => updateAlerts({ drop_gb: gb }),
								!alertSettings.enabled,
								'Fast-drop limit'
							)}
						</div>
					</div>

					{#if reminder}
						<div class="flex items-center justify-between gap-4">
							<div class="min-w-0">
								<div class="font-semibold">Monthly cleanup reminder</div>
								<div class="text-muted leading-snug">
									Package caches, temporary files and projects unused for 6 months in
									<span class="text-ink" title={reminder.projects_root}>
										{reminder.projects_root || 'the folder you search in Dev caches'}
									</span>
								</div>
							</div>
							<Switch
								label="Monthly cleanup reminder"
								checked={reminder.enabled}
								onchange={async (enabled) => {
									reminder = await SetReminderEnabled(enabled);
								}}
							/>
						</div>
					{/if}

					<div class="flex items-start justify-between gap-4">
						<p
							class="leading-snug {notificationStatus === 'enabled' || !notificationStatus
								? 'text-muted'
								: 'text-warn'}"
						>
							{notificationStatus
								? NOTIFICATION_NOTE[notificationStatus]
								: 'Checking notifications...'}
						</p>
						<button
							class="btn btn-secondary shrink-0"
							onclick={() => {
								showModal = false;
								SendTestAlert();
							}}
						>
							Send a test
						</button>
					</div>
				{/if}
			</div>
		{/key}
	</div>
</Modal>

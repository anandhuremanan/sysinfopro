<script lang="ts">
	import {
		ACCENT_PRESETS,
		setAccent,
		setTheme,
		theme,
		type ThemeMode
	} from '$lib/theme.svelte';
	import { enterWidget, toggleWidgetMetric, widget, widgetOptions } from '$lib/widget.svelte';
	import {
		GetAlertSettings,
		GetHotkeyCombos,
		GetHotkeySettings,
		GetNotificationStatus,
		GetReminderSettings,
		SendTestAlert,
		SetAlertSettings,
		SetHotkeySettings,
		SetReminderEnabled
	} from '$lib/wailsjs/go/main/App';
	import Credits from './Credits.svelte';
	import Modal from './Modal.svelte';
	import Switch from './Switch.svelte';

	let { showModal = $bindable(false) } = $props();

	type Tab = 'general' | 'widget' | 'alerts';
	type AlertSettings = { enabled: boolean; min_free_gb: number; drop_gb: number };
	type HotkeySettings = { enabled: boolean; combo: string; error?: string };

	const tabs: { id: Tab; label: string }[] = [
		{ id: 'general', label: 'General' },
		{ id: 'widget', label: 'Widget' },
		{ id: 'alerts', label: 'Alerts' }
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

	$effect(() => {
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
	<div role="radiogroup" aria-label={group} class="flex gap-0.5 p-[3px] rounded-[10px] bg-line">
		{#each options as option (option)}
			<button
				role="radio"
				aria-checked={selected === option}
				{disabled}
				class="h-6 px-2.5 rounded-lg cursor-pointer font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed {selected ===
				option
					? 'bg-tile text-ink font-semibold shadow-xs'
					: 'text-muted hover:text-ink'}"
				onclick={() => onpick(option)}
			>
				{label(option)}
			</button>
		{/each}
	</div>
{/snippet}

<Modal bind:showModal title="Settings">
	<div class="flex flex-col gap-4">
		<div role="tablist" aria-label="Settings sections" class="flex gap-1 -mt-1 border-b border-line">
			{#each tabs as t (t.id)}
				<button
					role="tab"
					aria-selected={tab === t.id}
					class="h-8 px-3 -mb-px border-b-2 cursor-pointer font-semibold transition-colors {tab ===
					t.id
						? 'border-accent text-ink'
						: 'border-transparent text-muted hover:text-ink'}"
					onclick={() => (tab = t.id)}
				>
					{t.label}
				</button>
			{/each}
		</div>

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

			<div class="flex items-center justify-between gap-4 pt-4 border-t border-line">
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

			<div class="flex items-center justify-between gap-4 pt-4 border-t border-line">
				<div>
					<div class="font-semibold">SysInfo Pro</div>
					<div class="text-muted">About, credits and source code</div>
				</div>
				<Credits />
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
			<div class="flex flex-col gap-2 pt-4 border-t border-line">
				<span class="font-semibold">Show</span>
				<div class="flex flex-wrap gap-1.5" role="group" aria-label="Readings shown in the widget">
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
				<div class="flex flex-col gap-2 pt-4 border-t border-line">
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
							<span class="text-amber-700 dark:text-amber-400 leading-snug">
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
		{:else if !alertSettings}
			<div class="h-32 flex items-center justify-center text-muted animate-pulse">Loading...</div>
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

			<div class="flex flex-col gap-3 pt-4 border-t border-line">
				<div class="flex items-center justify-between gap-4">
					<span class={alertSettings.enabled ? '' : 'text-muted'}>When free space drops below</span>
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
					<span class={alertSettings.enabled ? '' : 'text-muted'}>When a drive loses, in 10 minutes</span>
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
				<div class="flex items-center justify-between gap-4 pt-4 border-t border-line">
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

			<div class="flex items-start justify-between gap-4 pt-4 border-t border-line">
				<p
					class="leading-snug {notificationStatus === 'enabled' || !notificationStatus
						? 'text-muted'
						: 'text-amber-700 dark:text-amber-400'}"
				>
					{notificationStatus ? NOTIFICATION_NOTE[notificationStatus] : 'Checking notifications...'}
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
</Modal>

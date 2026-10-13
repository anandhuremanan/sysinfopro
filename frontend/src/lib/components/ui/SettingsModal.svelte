<script lang="ts">
	import { setTheme, theme, type ThemeMode } from '$lib/theme.svelte';
	import { enterWidget, toggleWidgetMetric, widget, WIDGET_METRICS } from '$lib/widget.svelte';
	import Credits from './Credits.svelte';
	import Modal from './Modal.svelte';

	let { showModal = $bindable(false) } = $props();

	const modes: { id: ThemeMode; label: string }[] = [
		{ id: 'system', label: 'System' },
		{ id: 'light', label: 'Light' },
		{ id: 'dark', label: 'Dark' }
	];
</script>

<Modal bind:showModal title="Settings">
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between gap-4">
			<div>
				<div class="font-semibold">Theme</div>
				<div class="text-muted">System follows your OS appearance</div>
			</div>
			<div role="radiogroup" aria-label="Theme" class="flex gap-0.5 p-[3px] rounded-[10px] bg-line">
				{#each modes as mode (mode.id)}
					<button
						role="radio"
						aria-checked={theme.mode === mode.id}
						class="h-7 px-3 rounded-lg cursor-pointer font-medium transition-colors {theme.mode ===
						mode.id
							? 'bg-tile text-ink font-semibold shadow-xs'
							: 'text-muted hover:text-ink'}"
						onclick={() => setTheme(mode.id)}
					>
						{mode.label}
					</button>
				{/each}
			</div>
		</div>

		<div class="flex flex-col gap-2.5 pt-4 border-t border-line">
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
			<div class="flex items-center gap-3">
				<span class="text-muted">Show</span>
				<div class="flex gap-1.5" role="group" aria-label="Readings shown in the widget">
					{#each WIDGET_METRICS as metric (metric.id)}
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
		</div>

		<div class="flex items-center justify-between gap-4 pt-4 border-t border-line">
			<div>
				<div class="font-semibold">SysInfo Pro</div>
				<div class="text-muted">About, credits and source code</div>
			</div>
			<Credits />
		</div>
	</div>
</Modal>

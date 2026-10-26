<script lang="ts">
	import { dismissAlert, driveAlerts, type DriveAlert } from '$lib/alerts.svelte';
	import StrokeIcon from './StrokeIcon.svelte';

	// open the tool that helps with this alert
	let { onOpen }: { onOpen?: (alert: DriveAlert) => void } = $props();

	let alert = $derived(driveAlerts.current);
	// a reminder or test is news, not a warning
	let calm = $derived(alert?.kind !== 'low' && alert?.kind !== 'dropping');
</script>

{#if alert}
	<div
		role="alert"
		class="absolute top-14 right-4 z-40 w-[300px] flex gap-2.5 p-3 rounded-xl bg-tile border shadow-2xl {calm
			? 'border-accent/40'
			: 'border-red-500/40'}"
	>
		<span
			class="shrink-0 w-7 h-7 rounded-lg flex items-center justify-center {calm
				? 'bg-accent/15 text-accent'
				: 'bg-red-500/15 text-red-600 dark:text-red-400'}"
		>
			<StrokeIcon
				d={calm
					? alert.kind === 'task'
						? 'M5 12.5l4.5 4.5L19 7.5'
						: 'M12 3l9 5-9 5-9-5 9-5zM3 13l9 5 9-5M3 17l9 5 9-5'
					: 'M12 8v5M12 16.5v.5M10.3 3.9 2.6 17.5A2 2 0 0 0 4.3 20.5h15.4a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z'}
				size={15}
			/>
		</span>
		<div class="flex-1 min-w-0">
			<div class="font-semibold text-ink">{alert.title}</div>
			<div class="text-muted leading-snug mt-0.5">{alert.body}</div>
			{#if alert.kind !== 'test' && onOpen}
				<button
					class="mt-2 text-accent font-semibold cursor-pointer hover:underline"
					onclick={() => {
						onOpen(alert);
						dismissAlert();
					}}
				>
					{alert.kind === 'reminder'
						? 'Open Dev caches'
						: alert.kind === 'task'
							? 'Show results'
							: `See what is using ${alert.drive}`}
				</button>
			{/if}
		</div>
		<button
			aria-label="Dismiss"
			class="shrink-0 w-6 h-6 -mr-1 -mt-1 rounded-md flex items-center justify-center text-muted cursor-pointer hover:text-ink hover:bg-line"
			onclick={dismissAlert}
		>
			<StrokeIcon d="M6 6l12 12M18 6L6 18" size={12} />
		</button>
	</div>
{/if}

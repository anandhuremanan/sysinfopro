<script lang="ts">
	import type { Snippet } from 'svelte';
	import StrokeIcon from './StrokeIcon.svelte';

	let {
		showModal = $bindable(false),
		title = '',
		description = '',
		showClose = true,
		width = 'max-w-md',
		children,
		actions
	}: {
		showModal?: boolean;
		title?: string;
		description?: string;
		showClose?: boolean;
		// Tailwind max-width class for the dialog
		width?: string;
		children?: Snippet;
		// Footer buttons, right-aligned
		actions?: Snippet;
	} = $props();

	function close() {
		if (showClose) showModal = false;
	}

	function onKeydown(event: KeyboardEvent) {
		if (showModal && event.key === 'Escape') close();
	}
</script>

<svelte:window onkeydown={onKeydown} />

{#if showModal}
	<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
	<div
		class="fixed z-50 inset-0 flex items-center justify-center p-4 bg-black/50 backdrop-blur-xs"
		onclick={(event) => {
			if (event.target === event.currentTarget) close();
		}}
	>
		<div
			role="dialog"
			aria-modal="true"
			aria-label={title || 'Dialog'}
			class="w-full {width} max-h-full flex flex-col bg-tile text-ink text-xs rounded-2xl border border-line shadow-2xl"
		>
			{#if title || showClose}
				<div class="flex items-start justify-between gap-3 pl-5 pr-3 pt-4 shrink-0">
					<div class="min-w-0 pt-0.5">
						{#if title}
							<div class="text-sm font-bold">{title}</div>
						{/if}
						{#if description}
							<div class="text-muted mt-0.5">{description}</div>
						{/if}
					</div>
					{#if showClose}
						<button
							aria-label="Close"
							class="shrink-0 w-7 h-7 rounded-lg flex items-center justify-center text-muted cursor-pointer hover:text-ink hover:bg-line transition-colors"
							onclick={close}
						>
							<StrokeIcon d="M6 6l12 12M18 6L6 18" size={14} />
						</button>
					{/if}
				</div>
			{/if}

			{#if children}
				<div class="px-5 py-4 min-h-0 overflow-y-auto">{@render children()}</div>
			{:else}
				<div class="h-4 shrink-0"></div>
			{/if}

			{#if actions}
				<div class="flex justify-end gap-2 px-5 py-3 border-t border-line shrink-0">
					{@render actions()}
				</div>
			{/if}
		</div>
	</div>
{/if}

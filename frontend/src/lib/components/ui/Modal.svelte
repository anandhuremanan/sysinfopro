<script lang="ts" module>
	// open dialogs, newest last: only the top one answers Escape
	const openStack: symbol[] = [];
	let nextId = 0;
</script>

<script lang="ts">
	import { tick, type Snippet } from 'svelte';
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

	const me = Symbol('modal');
	const uid = `modal-${nextId++}`;
	let dialog = $state<HTMLDivElement | undefined>(undefined);
	// where focus was before the dialog opened, to return it on close
	let returnTo: HTMLElement | null = null;

	const FOCUSABLE =
		'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

	function close() {
		if (showClose) showModal = false;
	}

	// opening: remember focus, join the stack, focus the first control (or the dialog);
	// closing: leave the stack and put focus back
	$effect(() => {
		if (!showModal) return;
		returnTo = document.activeElement as HTMLElement | null;
		openStack.push(me);
		tick().then(() => {
			const first = dialog?.querySelector<HTMLElement>(FOCUSABLE);
			(first ?? dialog)?.focus();
		});
		return () => {
			const at = openStack.indexOf(me);
			if (at >= 0) openStack.splice(at, 1);
			returnTo?.focus?.();
		};
	});

	function onKeydown(event: KeyboardEvent) {
		if (!showModal || openStack[openStack.length - 1] !== me) return;
		if (event.key === 'Escape') {
			event.stopPropagation();
			close();
			return;
		}
		// keep Tab inside the dialog
		if (event.key === 'Tab' && dialog) {
			const items = [...dialog.querySelectorAll<HTMLElement>(FOCUSABLE)];
			if (items.length === 0) {
				event.preventDefault();
				dialog.focus();
				return;
			}
			const first = items[0];
			const last = items[items.length - 1];
			if (event.shiftKey && document.activeElement === first) {
				event.preventDefault();
				last.focus();
			} else if (!event.shiftKey && document.activeElement === last) {
				event.preventDefault();
				first.focus();
			}
		}
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
			bind:this={dialog}
			role="dialog"
			aria-modal="true"
			tabindex="-1"
			aria-labelledby={title ? `${uid}-title` : undefined}
			aria-describedby={description ? `${uid}-desc` : undefined}
			aria-label={title ? undefined : 'Dialog'}
			class="w-full {width} max-h-full flex flex-col bg-tile text-ink text-xs rounded-2xl border border-line shadow-2xl outline-none"
		>
			{#if title || showClose}
				<div class="flex items-start justify-between gap-3 pl-5 pr-3 pt-4 shrink-0">
					<div class="min-w-0 pt-0.5">
						{#if title}
							<div id="{uid}-title" class="text-sm font-bold">{title}</div>
						{/if}
						{#if description}
							<div id="{uid}-desc" class="text-muted mt-0.5">{description}</div>
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
				<div class="flex justify-end gap-2 px-5 pt-1 pb-4 shrink-0">
					{@render actions()}
				</div>
			{/if}
		</div>
	</div>
{/if}

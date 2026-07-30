<script lang="ts">
	import Icons from './icons.svelte';

	let { showModal = $bindable(false), children, showClose = true } = $props();
</script>

{#if showModal}
	<div
		aria-labelledby="modal-title"
		role="dialog"
		aria-modal="true"
		class="fixed z-50 overflow-y-auto inset-0 flex items-center justify-center bg-slate-900/60 backdrop-blur-xs transition-opacity"
	>
		<div class="bg-zinc-200 m-4 w-full max-w-lg rounded-2xl dark:bg-zinc-800 dark:text-white p-5 relative shadow-2xl border border-slate-300 dark:border-slate-700">
			{#if showClose}
				<button
					class="absolute top-4 right-4 text-xs cursor-pointer text-slate-400 hover:text-slate-200 transition-colors"
					onclick={() => (showModal = false)}
				>
					<Icons iconName="x" />
				</button>
			{/if}
			<div>{@render children()}</div>
		</div>
	</div>
{/if}

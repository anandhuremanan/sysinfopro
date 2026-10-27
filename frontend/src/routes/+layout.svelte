<script lang="ts">
	import { onMount } from 'svelte';
	import { initTheme } from '$lib/theme.svelte';
	import { SetWindowVisible } from '$lib/wailsjs/go/main/App';
	import '../app.css';

	let { children } = $props();

	$effect(() => initTheme());

	// Live readings slow right down while the window is minimised or hidden, to
	// save power; the widget is a visible window, so it keeps updating
	onMount(() => {
		const report = () => SetWindowVisible(!document.hidden).catch(() => {});
		report();
		document.addEventListener('visibilitychange', report);
		return () => document.removeEventListener('visibilitychange', report);
	});
</script>

<main class="flex flex-col min-h-screen bg-ground">
	{@render children()}
</main>

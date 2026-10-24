<script lang="ts">
	import { initTheme } from '$lib/theme.svelte';
	import '../app.css';

	let { children } = $props();

	$effect(() => initTheme());
</script>

<main class="flex flex-col min-h-screen bg-ground">
	{@render children()}
</main>

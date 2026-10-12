<script lang="ts">
	let { children, isLoading = false } = $props();
</script>

<div
	class="relative text-xs {isLoading &&
		'animate-pulse'} justify-center bg-ground border border-line rounded-2xl w-full p-4 h-full"
>
	{@render children()}
</div>

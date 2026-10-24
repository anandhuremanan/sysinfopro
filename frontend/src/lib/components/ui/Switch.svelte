<script lang="ts">
	let {
		checked = false,
		label,
		disabled = false,
		onchange
	}: {
		checked?: boolean;
		// read by screen readers; the visible label sits next to the switch
		label: string;
		disabled?: boolean;
		onchange: (checked: boolean) => void;
	} = $props();
</script>

<button
	role="switch"
	aria-checked={checked}
	aria-label={label}
	{disabled}
	class="relative shrink-0 w-9 h-5 rounded-full cursor-pointer transition-colors disabled:opacity-50 disabled:cursor-not-allowed {checked
		? 'bg-accent'
		: 'bg-line'}"
	onclick={() => onchange(!checked)}
>
	<span
		class="absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-tile shadow-sm transition-transform {checked
			? 'translate-x-4'
			: ''}"
	></span>
</button>

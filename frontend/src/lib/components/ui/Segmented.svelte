<script lang="ts" generics="T extends string | number">
	import { tick } from 'svelte';

	// A row of options with a highlight that slides to the chosen one
	let {
		options,
		value,
		onchange,
		label,
		disabled = false,
		tabs = false,
		size = 'md'
	}: {
		options: { id: T; label: string }[];
		value: T;
		onchange: (id: T) => void;
		label: string;
		disabled?: boolean;
		// tabs switch sections; otherwise this is a choice of values
		tabs?: boolean;
		size?: 'sm' | 'md';
	} = $props();

	let buttons: HTMLButtonElement[] = $state([]);
	let thumb = $state({ left: 0, width: 0 });
	// no slide on first appearance, only when the choice changes
	let ready = $state(false);

	async function place() {
		await tick();
		const i = options.findIndex((o) => o.id === value);
		const el = buttons[i];
		if (el) thumb = { left: el.offsetLeft, width: el.offsetWidth };
	}

	$effect(() => {
		value;
		options;
		place().then(() => requestAnimationFrame(() => (ready = true)));
	});
</script>

<div
	role={tabs ? 'tablist' : 'radiogroup'}
	aria-label={label}
	class="relative flex gap-0.5 p-[3px] rounded-[10px] bg-line {disabled ? 'opacity-50' : ''}"
>
	<span
		aria-hidden="true"
		class="absolute top-[3px] bottom-[3px] rounded-lg bg-tile shadow-xs {ready
			? 'transition-[left,width] duration-200 ease-out motion-reduce:transition-none'
			: ''}"
		style="left: {thumb.left}px; width: {thumb.width}px"
	></span>
	{#each options as option, i (option.id)}
		<button
			bind:this={buttons[i]}
			role={tabs ? 'tab' : 'radio'}
			aria-selected={tabs ? value === option.id : undefined}
			aria-checked={tabs ? undefined : value === option.id}
			{disabled}
			class="relative z-10 rounded-lg cursor-pointer font-medium transition-colors duration-200 disabled:cursor-not-allowed {size === 'sm'
				? 'h-6 px-2.5'
				: 'h-7 px-3'} {value === option.id ? 'text-ink font-semibold' : 'text-muted hover:text-ink'}"
			onclick={() => onchange(option.id)}
		>
			{option.label}
		</button>
	{/each}
</div>

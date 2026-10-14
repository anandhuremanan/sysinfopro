<script lang="ts">
	import Modal from './Modal.svelte';
	import { BrowserOpenURL } from '$lib/wailsjs/runtime/runtime';

	let showModal = $state(false);
	let modalTitle: string | undefined = $state(undefined);
	let today = new Date();
	let year = today.getFullYear();

	function setModal(title = '') {
		modalTitle = title;
		showModal = !showModal;
	}
</script>

<div class="text-xs text-muted flex space-x-4">
	<button class="hover:text-ink cursor-pointer" onclick={() => setModal('About')}>
		About</button
	>
	<button class="hover:text-ink cursor-pointer" onclick={() => setModal('Credits')}
		>Credits</button
	>
	<button
		class="hover:text-ink cursor-pointer"
		onclick={() => {
			BrowserOpenURL('https://github.com/ananduremanan/sysinfopro');
		}}>Source</button
	>
</div>

<Modal bind:showModal title={modalTitle}>
	{#if modalTitle === 'About'}
		<div class="flex flex-col gap-3 leading-relaxed">
			<p>
				SysInfo Pro is a lightweight system diagnostics tool for Windows, designed to give you
				instant insights into your machine's performance and hardware configuration. Whether you're
				a developer, IT professional, or just a curious user, SysInfo Pro provides real-time data in
				a clean, readable format — with zero bloat.
			</p>
			<div>
				<div class="text-muted">Report a bug</div>
				<button
					class="text-accent font-medium hover:underline cursor-pointer"
					onclick={() => {
						BrowserOpenURL('https://github.com/ananduremanan/sysinfopro/issues');
					}}>https://github.com/ananduremanan/sysinfopro/issues</button
				>
			</div>
			<div class="text-muted pt-3 border-t border-line">
				Sysinfo Pro © {year} - Present · sysinfopro.imanandhu.in
			</div>
		</div>
	{:else if modalTitle === 'Credits'}
		<div class="flex justify-between items-center gap-4">
			<div class="flex flex-col gap-1">
				<p>SysInfo Pro version 3.0.0</p>
				<p>Author: Probably Fine Labs</p>
				<p>Made with Wails, Go, Svelte and Tailwind</p>
				<p class="mt-3 text-muted">sysinfopro.imanandhu.in</p>
			</div>
			<div class="flex flex-col gap-2 items-center justify-center shrink-0">
				<img src="/Open_Source_Initiative.svg" alt="oss" class="w-20 h-20" />
				<div class="text-center text-muted">We love and Support <br /> Open Source</div>
			</div>
		</div>
	{/if}
</Modal>

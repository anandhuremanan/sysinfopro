<script lang="ts">
	import Modal from '$lib/components/ui/Modal.svelte';
	import { initTheme } from '$lib/theme.svelte';
	import { CheckUpdate } from '$lib/wailsjs/go/main/App';
	import { BrowserOpenURL } from '$lib/wailsjs/runtime/runtime';
	import '../app.css';

	let { children } = $props();

	const url =
		'https://github.com/ananduremanan/sysinfopro/releases/download/v-beta-5/sysinfopro.exe';

	let showUpdate = $state(false);
	let versionName = $state('');

	async function checkUpdate() {
		try {
			const result = await CheckUpdate();
			if (result) {
				console.log(result);
				showUpdate = JSON.parse(result).is_update_available;
			}
		} catch (error) {
			console.error('Error checking for updates:', error);
		}
	}

	$effect(() => {
		checkUpdate();
	});

	$effect(() => initTheme());
</script>

<main class="flex flex-col min-h-screen bg-ground">
	{@render children()}

	<Modal
		showModal={showUpdate}
		showClose={false}
		width="max-w-sm"
		title="Update available"
		description="A new version of SysInfo Pro is ready to download."
	>
		{#snippet actions()}
			<button
				class="btn btn-secondary"
				onclick={() => {
					showUpdate = false;
				}}
			>
				Not now
			</button>
			<button
				class="btn btn-primary"
				onclick={() => {
					BrowserOpenURL('https://sysinfopro.netlify.app');
				}}
			>
				Download
			</button>
		{/snippet}
	</Modal>
</main>

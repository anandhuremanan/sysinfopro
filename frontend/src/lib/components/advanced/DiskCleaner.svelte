<script lang="ts">
	import { CheckCleanerPermissions, GetDriveHealthDetails, ScanSafeCleanableFiles } from '$lib/wailsjs/go/main/App';
	import Card from '../ui/Card.svelte';
	import PieDisk from './PieDisk.svelte';

	let isScanningFiles = $state(false);
	let files: any = $state(undefined);
	let driveHealth: any = $state(null);

	async function checkPermission() {
		try {
			const permission = await CheckCleanerPermissions();
			if (permission) {
				console.log('Permission granted:', permission);
			}
		} catch (error) {
			console.error(error);
		}
	}

	async function loadDriveHealth() {
		try {
			// @ts-ignore
			const result = await GetDriveHealthDetails();
			if (result && result.length > 0) {
				driveHealth = result[0];
			}
		} catch (err) {
			console.error('Error fetching drive health:', err);
		}
	}

	async function scanFiles() {
		isScanningFiles = true;
		try {
			const fileResult: any = await ScanSafeCleanableFiles();
			if (fileResult) {
				files = fileResult;
			}
		} catch (error) {
			console.error('Error scanning files:', error);
		} finally {
			isScanningFiles = false;
		}
	}

	async function analyseStorageSpace() {
		await checkPermission();
		await scanFiles();
	}

	function resetScan() {
		files = undefined;
	}

	$effect(() => {
		loadDriveHealth();
	});
</script>

<Card isLoading={false}>
	<div class="flex flex-col h-full justify-between">
		<!-- Header Row with unified baseline height -->
		<div class="h-6 flex justify-between items-center w-full">
			<div class="flex items-center gap-2">
				<span class="font-bold text-xs leading-tight text-ink">
					Disk Cleaner & Storage
				</span>
				{#if driveHealth}
					<span
						class="text-[10px] px-2 py-0.5 rounded font-semibold border {driveHealth.health_status ===
						'Good' || driveHealth.health_status === 'Healthy'
							? 'bg-green-500/20 text-green-700 dark:text-green-400 border-green-500/30'
							: 'bg-amber-500/20 text-amber-700 dark:text-amber-400 border-amber-500/30'}"
					>
						{driveHealth.health_status} ({driveHealth.estimated_life_pct}%)
					</span>
				{/if}
			</div>

			{#if files}
				<button
					class="text-[10px] bg-tile hover:bg-line px-2.5 py-1 rounded-lg text-ink font-medium transition-colors cursor-pointer shrink-0 border border-line"
					onclick={resetScan}
				>
					Rescan
				</button>
			{/if}
		</div>

		<!-- Main Card Content -->
		{#if isScanningFiles}
			<!-- Scanning State -->
			<div class="flex-1 flex flex-col items-center justify-center space-y-3 py-8">
				<div class="w-10 h-10 border-3 border-accent border-t-transparent rounded-full animate-spin"></div>
				<div class="text-xs text-ink font-medium animate-pulse">
					Scanning temporary & cleanable files...
				</div>
				<div class="text-[11px] text-muted">Inspecting system logs, caches & temp directories</div>
			</div>
		{:else if files}
			<!-- Pie Chart Results State -->
			<div class="flex-1 flex flex-col justify-center py-2">
				<PieDisk {files} />
			</div>
		{:else}
			<!-- Initial Ready State -->
			<div class="flex-1 flex flex-col justify-between py-2 space-y-3">
				<!-- Hero graphic & Storage Diagnostic summary with button directly below Drive Life -->
				<div class="flex items-center gap-4 my-auto">
					<img src="/server-cleaning.png" alt="disk-cleaner" class="w-24 h-24 shrink-0 object-contain" />
					<div class="flex-1 text-xs space-y-2">
						<div class="font-bold text-ink">Storage Optimization</div>
						<div class="space-y-1 text-muted">
							<div class="flex justify-between">
								<span>Target Caches:</span>
								<span class="font-bold text-ink">OS Temp & Caches</span>
							</div>
							<div class="flex justify-between">
								<span>Cleaning Type:</span>
								<span class="font-bold text-ink">Safe System Purge</span>
							</div>
							<div class="flex justify-between">
								<span>Drive Life:</span>
								<span class="font-bold text-ink">
									{driveHealth ? `${driveHealth.estimated_life_pct}% (~${driveHealth.estimated_life_years || 4.7} yrs)` : 'Good'}
								</span>
							</div>
						</div>

						<!-- Scan Button directly below Drive Life -->
						<div class="pt-1">
							<button
								class="w-full py-1.5 text-xs bg-accent hover:opacity-90 text-on-accent rounded-lg cursor-pointer transition-colors font-medium shadow-xs"
								onclick={analyseStorageSpace}
							>
								Analyse Storage Space
							</button>
						</div>
					</div>
				</div>

				<!-- Description text at bottom -->
				<div class="pt-2 border-t border-line">
					<p class="text-[11px] text-muted">
						Safely clear temporary files, system logs, and browser caches to recover disk space.
					</p>
				</div>
			</div>
		{/if}
	</div>
</Card>

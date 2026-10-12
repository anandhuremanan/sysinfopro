<script lang="ts">
	import { bytesToGb } from '$lib';
	import { PurgeJunkFiles } from '$lib/wailsjs/go/main/App';
	import Modal from '../ui/Modal.svelte';

	let showConfirmation = $state(false);
	let isPurging = $state(false);
	let purgeSuccessMessage = $state<string | null>(null);

	interface Files {
		[key: string]: any[];
	}

	interface FileData {
		Files: Files;
		TotalSize: number;
	}

	interface PieProps {
		files: FileData;
	}

	let { files = { Files: { '': [] }, TotalSize: 0 } }: PieProps = $props();

	// Calculate sizes per category
	let chartData: any = $state([]);
	let colors = $state(['#4F46E5', '#10B981', '#F59E0B', '#EF4444', '#8B5CF6', '#EC4899']);

	function prepareChartData() {
		chartData = [];

		if (files && files.Files) {
			Object.entries(files.Files).forEach(([key, value], index) => {
				const itemCount = value.length;
				const totalItems = Object.values(files.Files).reduce((sum, arr) => sum + arr.length, 0);
				const estimatedSize = files.TotalSize * (itemCount / totalItems);

				chartData.push({
					name: key,
					value: estimatedSize,
					color: colors[index % colors.length],
					count: itemCount
				});
			});
		}
	}

	$effect(() => {
		prepareChartData();
		drawChart();
	});

	function drawChart() {
		const canvas: any = document.getElementById('pieChart');
		if (!canvas) return;

		const ctx = canvas.getContext('2d');
		const centerX = canvas.width / 2;
		const centerY = canvas.height / 2;
		const radius = Math.min(centerX, centerY) - 10;

		let total = chartData.reduce((sum: any, item: any) => sum + item.value, 0);
		let startAngle = 0;

		ctx.clearRect(0, 0, canvas.width, canvas.height);

		chartData.forEach((item: any) => {
			const sliceAngle = total > 0 ? (2 * Math.PI * item.value) / total : 0;

			ctx.beginPath();
			ctx.moveTo(centerX, centerY);
			ctx.arc(centerX, centerY, radius, startAngle, startAngle + sliceAngle);
			ctx.closePath();

			ctx.fillStyle = item.color;
			ctx.fill();

			startAngle += sliceAngle;
		});
	}

	async function handleExecutePurge() {
		showConfirmation = false;
		isPurging = true;
		try {
			// @ts-ignore
			const result = await PurgeJunkFiles();
			if (result && result.formattedSize) {
				purgeSuccessMessage = `Successfully purged ${result.cleanedCount || 'junk'} files (${result.formattedSize} freed)!`;
			} else {
				purgeSuccessMessage = 'Junk cleanup completed successfully!';
			}
		} catch (err) {
			console.error('Failed to purge files:', err);
			purgeSuccessMessage = `Cleanup error: ${err}`;
		} finally {
			isPurging = false;
		}
	}
</script>

{#if purgeSuccessMessage}
	<div class="p-4 rounded-xl bg-green-500/20 border border-green-500/40 text-center space-y-2">
		<div class="text-sm font-bold text-green-700 dark:text-green-400">Disk Cleanup Complete</div>
		<div class="text-xs text-ink">{purgeSuccessMessage}</div>
		<button
			class="mt-2 text-xs bg-tile hover:bg-line border border-line px-3 py-1 rounded-md cursor-pointer text-ink font-medium transition-colors"
			onclick={() => (purgeSuccessMessage = null)}
		>
			Close Summary
		</button>
	</div>
{:else if isPurging}
	<div class="flex flex-col items-center justify-center py-6 space-y-2">
		<div class="w-8 h-8 border-2 border-green-500 border-t-transparent rounded-full animate-spin"></div>
		<div class="text-xs text-ink font-semibold animate-pulse">Purging temporary files & browser caches...</div>
	</div>
{:else}
	<div class="grid grid-cols-2 gap-2">
		<div class="relative w-full flex justify-center items-center">
			<canvas id="pieChart" width="100" height="100" class="mx-auto"></canvas>
		</div>

		<div class="mt-2 w-full">
			<div class="flex flex-col text-xs gap-1 justify-between">
				{#each chartData.slice(-3) as item}
					<div class="flex items-center">
						<div class="w-2 h-2 rounded-full mr-2 shrink-0" style="background-color: {item.color}"></div>
						<span class="flex-1 truncate" title={item.name}>{item.name}</span>
						<span class="ml-2 font-medium shrink-0">{bytesToGb(item.value)} GB</span>
					</div>
				{/each}
				<div class="flex flex-col justify-between mt-2 border-line font-bold self-end">
					<span>Total {bytesToGb(files.TotalSize)} GB</span>
					<button
						class="bg-accent hover:opacity-90 px-3 py-1.5 rounded-xl mt-2 text-on-accent text-xs cursor-pointer font-medium transition-colors"
						onclick={() => (showConfirmation = true)}
					>
						Free Space
					</button>
				</div>
			</div>
		</div>

		<Modal
			bind:showModal={showConfirmation}
			width="max-w-sm"
			title="Confirm Disk Cleanup"
			description="This action cannot be reversed."
		>
			<p class="text-muted leading-relaxed">
				This will permanently delete temporary system files, log dumps, and browser cache files.
			</p>

			{#snippet actions()}
				<button class="btn btn-secondary" onclick={() => (showConfirmation = false)}>Cancel</button>
				<button class="btn btn-danger" onclick={handleExecutePurge}>Clear Space Now</button>
			{/snippet}
		</Modal>
	</div>
{/if}

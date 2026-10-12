<script lang="ts">
	import { checkInternetConnection } from '$lib';
	import { GetInterNetSpeed } from '$lib/wailsjs/go/main/App';
	import { EventsOn } from '$lib/wailsjs/runtime/runtime';
	import { onDestroy } from 'svelte';

	type Phase = 'idle' | 'latency' | 'download' | 'upload' | 'complete';

	type SpeedProgress = {
		phase: 'latency' | 'download' | 'upload';
		mbps: number;
		progress: number;
	};

	const GAUGE_STEPS = [10, 25, 50, 100, 250, 500, 1000, 2500, 10000];

	let speed = $state(0);
	let isRunning = $state(false);
	let progress = $state(0);
	let phase = $state<Phase>('idle');
	let gaugeMax = $state(GAUGE_STEPS[0]);
	let speedTestResult: any = $state(undefined);

	let logMessage = $state<string | undefined>(undefined);

	// Bumped on every start/cancel so a cancelled run's late result is ignored
	let runId = 0;

	function showLog(message: string) {
		logMessage = message;
		setTimeout(() => {
			logMessage = undefined;
		}, 3000);
	}

	async function getInternetSpeed() {
		const isConnected = await checkInternetConnection();

		if (!isConnected) {
			showLog('No internet connection detected!');
			return false;
		}

		return true;
	}

	// Smallest gauge scale that leaves headroom above the current reading
	function gaugeScaleFor(mbps: number) {
		return GAUGE_STEPS.find((step) => step >= mbps * 1.15) ?? GAUGE_STEPS[GAUGE_STEPS.length - 1];
	}

	// Live readings measured by the backend while the test runs
	const unsubscribe = EventsOn('speedtest-progress', (data: SpeedProgress) => {
		if (!isRunning || !data) return;

		if (data.phase !== phase) {
			phase = data.phase;
			gaugeMax = GAUGE_STEPS[0];
		}
		speed = data.mbps || 0;
		progress = data.progress || 0;
		gaugeMax = Math.max(gaugeMax, gaugeScaleFor(speed));
	});

	async function startTest() {
		if (isRunning) return;

		const isConnected = await getInternetSpeed();
		if (!isConnected) return;

		const id = ++runId;
		isRunning = true;
		phase = 'latency';
		progress = 0;
		speed = 0;
		gaugeMax = GAUGE_STEPS[0];
		speedTestResult = undefined;

		try {
			const res = await GetInterNetSpeed();
			if (id !== runId) return;
			speedTestResult = JSON.parse(res);
			phase = 'complete';
		} catch (err) {
			if (id !== runId) return;
			console.error('Error fetching internet speed: ', err);
			showLog(`Speed test failed: ${err}`);
			phase = 'idle';
		} finally {
			if (id === runId) {
				isRunning = false;
				speed = 0;
				progress = 0;
			}
		}
	}

	function resetTest() {
		runId++;
		isRunning = false;
		phase = 'idle';
		speed = 0;
		progress = 0;
	}

	function getNeedleRotation(speed: number) {
		const angle = -120 + (Math.min(speed, gaugeMax) / gaugeMax) * 240;
		return `rotate(${angle}deg)`;
	}

	function phaseError(code: string | undefined) {
		if (code === 'rate_limited') {
			return "Cloudflare is limiting speed tests from this network right now. Limits usually clear within an hour.";
		}
		return code ? 'This part of the test could not reach the server.' : undefined;
	}

	let resultNote = $derived(
		phaseError(speedTestResult?.download_error) ?? phaseError(speedTestResult?.upload_error)
	);

	function gaugeLabel(fraction: number) {
		const value = gaugeMax * fraction;
		return Number.isInteger(value) ? value : value.toFixed(1);
	}

	onDestroy(() => {
		runId++;
		unsubscribe();
	});
</script>

{#if phase === 'complete' && speedTestResult}
	<div class="flex flex-col items-center justify-center gap-6 mt-8">
		<!-- Download & Upload Speeds -->
		<div class="flex gap-12 items-center">
			<div class="flex flex-col items-center">
				<div class="font-bold text-5xl text-ink font-mono tracking-tight">
					{#if speedTestResult.download_error}
						<span class="text-muted">&ndash;</span>
					{:else}
						{speedTestResult.download_speed.toFixed(1)}<span
							class="text-sm font-sans text-muted ml-1">Mbps</span
						>
					{/if}
				</div>
				<div class="text-xs text-muted mt-1 uppercase font-semibold tracking-wider text-[10px]">
					Download Speed
				</div>
			</div>
			<div class="h-10 w-[1px] bg-line"></div>
			<div class="flex flex-col items-center">
				<div class="font-bold text-5xl text-ink font-mono tracking-tight">
					{#if speedTestResult.upload_error}
						<span class="text-muted">&ndash;</span>
					{:else}
						{speedTestResult.upload_speed.toFixed(1)}<span
							class="text-sm font-sans text-muted ml-1">Mbps</span
						>
					{/if}
				</div>
				<div class="text-xs text-muted mt-1 uppercase font-semibold tracking-wider text-[10px]">
					Upload Speed
				</div>
			</div>
		</div>

		<!-- Latency & Connection Diagnostics (Spacious, un-congested formatting) -->
		<div
			class="grid grid-cols-3 gap-6 bg-ground px-8 py-3.5 rounded-2xl border border-line text-xs w-full max-w-[480px] shadow-xs"
		>
			<div class="flex flex-col items-center">
				<span class="text-muted text-[10px] uppercase font-bold tracking-wider whitespace-nowrap"
					>Latency</span
				>
				<span class="font-bold text-sm font-mono text-ink mt-0.5 whitespace-nowrap">
					{speedTestResult.ping_latency_ms} ms
				</span>
			</div>
			<div class="flex flex-col items-center">
				<span class="text-muted text-[10px] uppercase font-bold tracking-wider whitespace-nowrap"
					>Jitter</span
				>
				<span class="font-bold text-sm font-mono text-ink mt-0.5 whitespace-nowrap">
					{speedTestResult.jitter_ms} ms
				</span>
			</div>
			<div class="flex flex-col items-center">
				<span class="text-muted text-[10px] uppercase font-bold tracking-wider whitespace-nowrap"
					>Failed pings</span
				>
				<span
					class="font-bold text-sm font-mono mt-0.5 whitespace-nowrap {speedTestResult.packet_loss_pct >
					0
						? 'text-amber-700 dark:text-amber-400'
						: 'text-green-700 dark:text-green-400'}"
				>
					{speedTestResult.packet_loss_pct}%
				</span>
			</div>
		</div>

		{#if resultNote}
			<div class="text-xs font-semibold text-amber-700 dark:text-amber-400 text-center max-w-[480px]">
				{resultNote}
			</div>
		{:else}
			<div class="text-[11px] text-muted">Measured against Cloudflare's nearest server</div>
		{/if}

		<button
			onclick={startTest}
			class="px-5 py-2 text-xs bg-accent hover:opacity-90 text-on-accent font-medium rounded-xl cursor-pointer transition-colors shadow-xs"
		>
			Test Again
		</button>
	</div>
{:else}
	<div class="grid grid-cols-2">
		<div class="flex flex-col items-center justify-center">
			<div class="relative w-64 h-64 mx-auto mb-4">
				<div class="absolute inset-0">
					{#each Array(11) as _, i}
						<div
							class="absolute w-1 h-4 bg-muted"
							style="left: calc(50% - 0.5px); top: 8px; transform: rotate({-120 +
								i * 24}deg); transform-origin: center 120px;"
						></div>
					{/each}
				</div>

				<div class="absolute text-xs text-muted" style="left: 15%; top: 70%;">0</div>
				<div class="absolute text-xs text-muted" style="left: 25%; top: 30%;">
					{gaugeLabel(0.25)}
				</div>
				<div
					class="absolute text-xs text-muted"
					style="left: 50%; top: 15%; transform: translateX(-50%);"
				>
					{gaugeLabel(0.5)}
				</div>
				<div class="absolute text-xs text-muted" style="right: 25%; top: 30%;">
					{gaugeLabel(0.75)}
				</div>
				<div class="absolute text-xs text-muted" style="right: 15%; top: 70%;">
					{gaugeLabel(1)}
				</div>

				<div
					class="absolute left-1/2 top-2 w-1 h-32 bg-accent rounded-full -mt-2 transition-transform duration-200 ease-out"
					style="transform: {getNeedleRotation(speed)}; transform-origin: center bottom;"
				></div>

				<div class="absolute left-1/2 top-1/2 w-6 h-6 rounded-full bg-ink -ml-3 -mt-3"></div>

				<div class="absolute bottom-8 left-0 right-0 text-center">
					<div class="text-3xl font-bold text-ink tabular-nums">
						{speed.toFixed(1)}
					</div>
					<div class="text-sm text-muted">
						{phase === 'upload' ? 'Mbps Upload' : 'Mbps Download'}
					</div>
				</div>
			</div>

			<div class="text-center text-muted mb-2 text-sm">
				{#if phase === 'idle'}
					Ready to test your connection speed
				{:else if phase === 'latency'}
					Measuring latency...
				{:else if phase === 'download'}
					Testing download speed...
				{:else if phase === 'upload'}
					Testing upload speed...
				{/if}
			</div>

			<div class="w-40 h-1 rounded-full bg-track {isRunning ? '' : 'invisible'}">
				<div
					class="h-1 rounded-full bg-accent transition-all duration-200 ease-linear"
					style="width: {progress}%"
				></div>
			</div>
		</div>

		<div class="flex flex-col justify-center items-end gap-4">
			<div class="text-xs text-justify">
				Test your internet speed in less than 30 seconds. The test typically uses between 40 to 100
				MB of data, though faster connections may use more. <br /> <br /> To perform the test, you'll
				be connected to Cloudflare. Your IP address will be shared with and processed by them according
				to their privacy policy. Cloudflare manages the test and makes all results publicly available
				to support internet research. These results include your IP address and speed data, but no personal
				information about you as a user is included.
			</div>

			<div>
				{#if phase === 'idle'}
					<div>
						<button
							onclick={startTest}
							class="px-4 text-sm py-2 bg-accent text-on-accent font-medium rounded-lg cursor-pointer"
						>
							Start Test
						</button>
					</div>
				{:else}
					<button
						onclick={resetTest}
						class="px-4 py-2 bg-ground border border-line text-ink font-medium rounded-lg text-sm cursor-pointer"
					>
						Cancel
					</button>
				{/if}
			</div>
			<div class="text-xs text-red-500">{logMessage}</div>
		</div>
	</div>
{/if}

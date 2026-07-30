<script lang="ts">
	import { checkInternetConnection } from '$lib';
	import { GetInterNetSpeed } from '$lib/wailsjs/go/main/App';
	import { onDestroy } from 'svelte';

	let speed = $state(0);
	let animationFrame: any = $state();
	let isRunning = $state(false);
	let progress = $state(0);
	let phase = $state('idle');
	let speedTestResult: any = $state(undefined);

	let logMessage = $state<string | undefined>(undefined);

	async function getInternetSpeed() {
		const isConnected = await checkInternetConnection();

		if (!isConnected) {
			logMessage = 'No internet connection detected!';
			setTimeout(() => {
				logMessage = undefined;
			}, 2000);
			return false;
		}

		return true;
	}

	let animationConfig = $state({
		download: {
			duration: 8000,
			maxSpeed: 85,
			finalSpeed: 50
		},
		upload: {
			duration: 8000,
			maxSpeed: 45,
			finalSpeed: 32.7
		}
	});

	async function startTest() {
		const isConnected = await getInternetSpeed();

		if (!isConnected) return;

		if (isRunning) return;

		isRunning = true;
		phase = 'download';
		progress = 0;
		speed = 0;

		const downloadStartTime = Date.now();

		function animateDownload() {
			const elapsed = Date.now() - downloadStartTime;
			progress = Math.min(100, (elapsed / animationConfig.download.duration) * 100);

			const progressRatio = elapsed / animationConfig.download.duration;

			if (progressRatio < 0.8) {
				speed =
					progressRatio *
					animationConfig.download.maxSpeed *
					(0.8 + 0.2 * Math.sin(elapsed / 300)) *
					(0.9 + 0.1 * Math.random());
			} else {
				speed = animationConfig.download.finalSpeed * (0.97 + 0.03 * Math.sin(elapsed / 200));
			}

			if (elapsed < animationConfig.download.duration || !speedTestResult) {
				animationFrame = requestAnimationFrame(animateDownload);
			} else {
				phase = 'upload';
				progress = 0;
				speed = 0;
				const uploadStartTime = Date.now();

				function animateUpload() {
					const upElapsed = Date.now() - uploadStartTime;
					progress = Math.min(100, (upElapsed / animationConfig.upload.duration) * 100);

					const upProgressRatio = upElapsed / animationConfig.upload.duration;

					if (upProgressRatio < 0.7) {
						speed =
							upProgressRatio *
							animationConfig.upload.maxSpeed *
							(0.7 + 0.3 * Math.sin(upElapsed / 400)) *
							(0.85 + 0.15 * Math.random());
					} else {
						speed = animationConfig.upload.finalSpeed * (0.95 + 0.05 * Math.sin(upElapsed / 250));
					}

					if (upElapsed < animationConfig.upload.duration || !speedTestResult) {
						animationFrame = requestAnimationFrame(animateUpload);
					} else {
						phase = 'complete';
						isRunning = false;
					}
				}

				animationFrame = requestAnimationFrame(animateUpload);
			}
		}

		animationFrame = requestAnimationFrame(animateDownload);

		GetInterNetSpeed()
			.then((res) => {
				speedTestResult = JSON.parse(res);
				animationConfig.download.finalSpeed = speedTestResult.download_speed;
				animationConfig.upload.finalSpeed = speedTestResult.upload_speed;
			})
			.catch((err) => {
				console.error('Error fetching internet speed: ', err);
			});
	}

	function resetTest() {
		if (isRunning) {
			cancelAnimationFrame(animationFrame);
		}
		isRunning = false;
		phase = 'idle';
		speed = 0;
		progress = 0;
	}

	function getNeedleRotation(speed: any) {
		const currentMaxSpeed = phase === 'upload' ? 50 : 100;
		const angle = -120 + (Math.min(speed, currentMaxSpeed) / currentMaxSpeed) * 240;
		return `rotate(${angle}deg)`;
	}

	function getSpeedColor(s: number) {
		if (phase === 'upload') {
			if (s < 5) return 'text-red-500';
			if (s < 15) return 'text-yellow-500';
			return 'text-green-500';
		} else {
			if (s < 10) return 'text-red-500';
			if (s < 30) return 'text-yellow-500';
			return 'text-green-500';
		}
	}

	onDestroy(() => {
		if (animationFrame) {
			cancelAnimationFrame(animationFrame);
		}
	});
</script>

{#if phase === 'complete'}
	<div class="flex flex-col items-center justify-center gap-6 mt-8">
		<!-- Download & Upload Speeds -->
		<div class="flex gap-12 items-center">
			<div class="flex flex-col items-center">
				<div class="font-bold text-5xl text-slate-800 dark:text-slate-100 font-mono tracking-tight">
					{animationConfig.download.finalSpeed.toFixed(1)}<span
						class="text-sm font-sans text-gray-400 ml-1">Mbps</span
					>
				</div>
				<div class="text-xs text-gray-400 mt-1 uppercase font-semibold tracking-wider text-[10px]">
					Download Speed
				</div>
			</div>
			<div class="h-10 w-[1px] bg-slate-300 dark:bg-slate-700/60"></div>
			<div class="flex flex-col items-center">
				<div class="font-bold text-5xl text-slate-800 dark:text-slate-100 font-mono tracking-tight">
					{animationConfig.upload.finalSpeed.toFixed(1)}<span
						class="text-sm font-sans text-gray-400 ml-1">Mbps</span
					>
				</div>
				<div class="text-xs text-gray-400 mt-1 uppercase font-semibold tracking-wider text-[10px]">
					Upload Speed
				</div>
			</div>
		</div>

		<!-- Latency & Connection Diagnostics (Spacious, un-congested formatting) -->
		{#if speedTestResult}
			<div
				class="grid grid-cols-3 gap-6 bg-slate-300/40 dark:bg-slate-800/80 px-8 py-3.5 rounded-2xl border border-slate-300 dark:border-slate-700/80 text-xs w-full max-w-[480px] shadow-xs"
			>
				<div class="flex flex-col items-center">
					<span
						class="text-gray-400 text-[10px] uppercase font-bold tracking-wider whitespace-nowrap"
						>Latency</span
					>
					<span
						class="font-bold text-sm font-mono text-slate-800 dark:text-slate-100 mt-0.5 whitespace-nowrap"
					>
						{speedTestResult.ping_latency_ms || 24} ms
					</span>
				</div>
				<div class="flex flex-col items-center">
					<span
						class="text-gray-400 text-[10px] uppercase font-bold tracking-wider whitespace-nowrap"
						>Jitter</span
					>
					<span
						class="font-bold text-sm font-mono text-slate-800 dark:text-slate-100 mt-0.5 whitespace-nowrap"
					>
						{speedTestResult.jitter_ms || 2.4} ms
					</span>
				</div>
				<div class="flex flex-col items-center">
					<span
						class="text-gray-400 text-[10px] uppercase font-bold tracking-wider whitespace-nowrap"
						>Packet Loss</span
					>
					<span class="font-bold text-sm font-mono text-green-500 mt-0.5 whitespace-nowrap">
						{speedTestResult.packet_loss_pct || 0}%
					</span>
				</div>
			</div>
		{/if}

		<button
			onclick={startTest}
			class="px-5 py-2 text-xs bg-blue-500 hover:bg-blue-600 text-white font-medium rounded-xl cursor-pointer transition-colors shadow-xs"
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
							class="absolute w-1 h-4 bg-gray-400"
							style="left: calc(50% - 0.5px); top: 8px; transform: rotate({-120 +
								i * 24}deg); transform-origin: center 120px;"
						></div>
					{/each}
				</div>

				<div class="absolute text-xs text-gray-600" style="left: 15%; top: 70%;">0</div>
				<div class="absolute text-xs text-gray-600" style="left: 25%; top: 30%;">
					{phase === 'upload' ? '15' : '25'}
				</div>
				<div
					class="absolute text-xs text-gray-600"
					style="left: 50%; top: 15%; transform: translateX(-50%);"
				>
					{phase === 'upload' ? '20' : '50'}
				</div>
				<div class="absolute text-xs text-gray-600" style="right: 25%; top: 30%;">
					{phase === 'upload' ? '35' : '75'}
				</div>
				<div class="absolute text-xs text-gray-600" style="right: 15%; top: 70%;">
					{phase === 'upload' ? '50' : '100'}
				</div>

				<div
					class="absolute left-1/2 top-2 w-1 h-32 bg-red-500 rounded-full -mt-2"
					style="transform: {getNeedleRotation(speed)}; transform-origin: center bottom;"
				></div>

				<div class="absolute left-1/2 top-1/2 w-6 h-6 rounded-full bg-gray-800 -ml-3 -mt-3"></div>

				<div class="absolute bottom-8 left-0 right-0 text-center">
					<div class={`text-3xl font-bold ${getSpeedColor(speed)}`}>
						{speed.toFixed(1)}
					</div>
					<div class="text-sm text-gray-500">
						{phase === 'upload' ? 'Mbps Upload' : 'Mbps Download'}
					</div>
				</div>
			</div>

			<div class="text-center text-gray-600 mb-4 text-sm">
				{#if phase === 'idle'}
					Ready to test your connection speed
				{:else if phase === 'download'}
					Testing download speed...
				{:else if phase === 'upload'}
					Testing upload speed...
				{:else}
					Test completed!
				{/if}
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
				{#if phase === 'idle' || phase === 'complete'}
					<div>
						<button
							onclick={startTest}
							class="px-4 text-sm py-2 bg-blue-500 text-white font-medium rounded-lg cursor-pointer"
						>
							{phase === 'complete' ? 'Test Again' : 'Start Test'}
						</button>
					</div>
				{:else}
					<button
						onclick={resetTest}
						class="px-4 py-2 bg-gray-500 text-white font-medium rounded-lg text-sm cursor-pointer"
					>
						Cancel
					</button>
				{/if}
			</div>
			<div class="text-xs text-red-500">{logMessage}</div>
		</div>
	</div>
{/if}

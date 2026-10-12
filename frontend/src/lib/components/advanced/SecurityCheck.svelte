<script lang="ts">
	import { GetSecurityReport } from '$lib/wailsjs/go/main/App';

	type Check = {
		id: string;
		title: string;
		status: 'pass' | 'warn' | 'fail' | 'unknown';
		detail: string;
		advice: string;
	};

	type Report = {
		checks: Check[];
		passed: number;
		warned: number;
		failed: number;
		unknown: number;
	};

	const STATUS: Record<Check['status'], { label: string; pill: string; dot: string }> = {
		pass: {
			label: 'OK',
			pill: 'bg-green-500/15 text-green-700 dark:text-green-400 border-green-500/30',
			dot: 'bg-green-500'
		},
		warn: {
			label: 'Review',
			pill: 'bg-amber-500/20 text-amber-700 dark:text-amber-400 border-amber-500/30',
			dot: 'bg-amber-500'
		},
		fail: {
			label: 'Problem',
			pill: 'bg-red-500/15 text-red-600 dark:text-red-400 border-red-500/30',
			dot: 'bg-red-500'
		},
		unknown: { label: 'Unknown', pill: 'bg-ground text-muted border-line', dot: 'bg-muted' }
	};

	// Problems first, then things to review
	const ORDER: Check['status'][] = ['fail', 'warn', 'unknown', 'pass'];

	let report = $state<Report | undefined>(undefined);
	let isLoading = $state(false);
	let loadError = $state<string | undefined>(undefined);

	let sortedChecks = $derived(
		report
			? [...report.checks].sort((a, b) => ORDER.indexOf(a.status) - ORDER.indexOf(b.status))
			: []
	);

	async function loadReport() {
		isLoading = true;
		loadError = undefined;
		try {
			report = (await GetSecurityReport()) as Report;
		} catch (err) {
			console.error('Failed to run security check:', err);
			loadError = `${err}`;
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		loadReport();
	});
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<!-- Header Bar -->
	<div class="flex justify-between items-center gap-3 pb-2 border-b border-line shrink-0">
		<div class="min-w-0">
			<div class="font-bold text-base text-ink">Security Check</div>
			<div class="text-xs text-muted truncate">
				Read-only check of this PC's protection settings. Nothing is changed.
			</div>
		</div>

		<div class="flex items-center gap-3 shrink-0">
			{#if report && !isLoading}
				<span class="font-semibold text-ink">
					{report.passed} of {report.checks.length} passed
				</span>
			{/if}
			<button class="btn btn-secondary" onclick={loadReport} disabled={isLoading}>
				{isLoading ? 'Checking...' : 'Check again'}
			</button>
		</div>
	</div>

	{#if isLoading}
		<div class="flex-1 flex items-center justify-center text-xs text-muted animate-pulse">
			Checking antivirus, firewall, encryption and updates...
		</div>
	{:else if loadError}
		<div class="flex-1 flex flex-col items-center justify-center gap-2 text-center">
			<div class="text-xs text-red-500 max-w-md">{loadError}</div>
			<button class="btn btn-secondary" onclick={loadReport}>Try again</button>
		</div>
	{:else if report}
		<div class="flex-1 min-h-0 overflow-y-auto overscroll-contain pr-1.5">
			{#each sortedChecks as check (check.id)}
				<div class="flex items-center gap-3 py-2 border-b border-line">
					<span class="w-2 h-2 rounded-full shrink-0 {STATUS[check.status].dot}"></span>
					<span class="w-40 shrink-0 font-semibold text-ink">{check.title}</span>
					<span class="flex-1 min-w-0">
						<span class="block text-ink">{check.detail}</span>
						{#if check.advice}
							<span class="block text-muted">{check.advice}</span>
						{/if}
					</span>
					<span
						class="shrink-0 w-16 text-center text-[10px] uppercase font-bold px-2 py-0.5 rounded-md border {STATUS[
							check.status
						].pill}"
					>
						{STATUS[check.status].label}
					</span>
				</div>
			{/each}
		</div>

		{#if report.unknown > 0}
			<div class="text-[11px] text-muted pt-1 shrink-0">
				"Unknown" means Windows did not report that setting to a standard account.
			</div>
		{/if}
	{/if}
</div>

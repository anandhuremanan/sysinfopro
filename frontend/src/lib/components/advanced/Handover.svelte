<script lang="ts">
	import ToolHeader from '$lib/components/ui/ToolHeader.svelte';
	import { REVEAL_ICON } from '$lib/path';
	import { onMount } from 'svelte';
	import { formatBytes } from '$lib';
	import { CheckHandover, RevealInExplorer } from '$lib/wailsjs/go/main/App';
	import { load, recall } from '$lib/cache';
	import BusyArt from '../ui/BusyArt.svelte';
	import StrokeIcon from '../ui/StrokeIcon.svelte';

	type Item = {
		id: string;
		group: string;
		title: string;
		status: 'todo' | 'ok' | 'info';
		detail: string;
		how: string;
	};
	type Folder = {
		label: string;
		path: string;
		kind: 'personal' | 'other';
		bytes: number;
		files: number;
		partial: boolean;
	};
	type Report = { items: Item[]; folders: Folder[]; todo: number; duration_ms: number };

	const GROUPS = ['Accounts', 'Your files', 'Secrets', 'Device'];
	const MARK = {
		todo: {
			d: 'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zM12 8v5M12 16h.01',
			tone: 'text-warn',
			pill: 'bg-amber-500/15 text-warn',
			label: 'To do'
		},
		ok: {
			d: 'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zM8 12l3 3 5-6',
			tone: 'text-ok',
			pill: 'bg-ground text-muted',
			label: 'Fine'
		},
		info: {
			d: 'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zM12 11v5M12 8h.01',
			tone: 'text-muted',
			pill: 'bg-ground text-muted',
			label: 'Note'
		}
	};

	let report = $state<Report | null>(recall('handover') ?? null);
	let isLoading = $state(!recall('handover'));
	let showFolders = $state(false);

	let groups = $derived(
		GROUPS.map((g) => ({
			name: g,
			items: (report?.items ?? []).filter((i) => i.group === g)
		})).filter((g) => g.items.length)
	);
	let folders = $derived((report?.folders ?? []).filter((f) => f.bytes > 0));

	async function refresh(force = false) {
		isLoading = force || !report;
		try {
			report = (await load('handover', CheckHandover, 5 * 60_000, force)) as Report;
		} catch (err) {
			console.error('Hand-over check failed:', err);
		} finally {
			isLoading = false;
		}
	}

	onMount(() => {
		refresh();
	});
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<ToolHeader title="Hand over">
		{#snippet sub()}
			{#if report && !isLoading}
				{report.todo
					? `${report.todo} ${report.todo === 1 ? 'thing' : 'things'} to deal with before this PC changes hands`
					: 'Nothing left to deal with before this PC changes hands'}
			{:else}
				What to deal with before you sell, give away or return this PC
			{/if}
		{/snippet}
		{#snippet actions()}
			<button class="btn btn-secondary shrink-0" disabled={isLoading} onclick={() => refresh(true)}>
				Check again
			</button>
		{/snippet}
	</ToolHeader>

	{#if isLoading}
		<div class="flex-1 flex items-center justify-center">
			<BusyArt kind="check" size={64} label="Looking at accounts, files and saved passwords..." />
		</div>
	{:else if report}
		<div
			class="flex-1 min-h-0 overflow-y-auto overscroll-contain pb-5 [mask-image:linear-gradient(to_bottom,#000_calc(100%-20px),transparent)] pr-1.5 flex flex-col gap-3"
		>
			<p class="text-[11px] text-muted px-1">
				Only looked at, nothing is changed. Passwords and keys are never read.
			</p>
			{#each groups as g (g.name)}
				<section class="flex flex-col gap-1.5">
					<h3 class="text-[10px] uppercase tracking-wider font-semibold text-muted px-1">
						{g.name}
					</h3>
					{#each g.items as item (item.id)}
						<div
							class="rounded-xl bg-tile border px-3 py-2 {item.status === 'todo'
								? 'border-amber-500/40'
								: 'border-line'}"
						>
							<div class="flex items-center gap-2">
								<span class="{MARK[item.status].tone} shrink-0"
									><StrokeIcon d={MARK[item.status].d} size={16} /></span
								>
								<span class="text-xs font-semibold text-ink flex-1 min-w-0">{item.title}</span>
								<span
									class="text-[11px] font-semibold px-1.5 py-0.5 rounded-md shrink-0 {MARK[
										item.status
									].pill}"
								>
									{MARK[item.status].label}
								</span>
							</div>
							{#if item.detail}<p class="text-xs text-muted mt-0.5 leading-relaxed">
									{item.detail}
								</p>{/if}
							{#if item.how}
								<p class="text-xs text-ink mt-1 leading-relaxed">
									<span class="font-semibold">How:</span>
									{item.how}
								</p>
							{/if}
							{#if (item.id === 'personal-files' || item.id === 'other-folders') && folders.length}
								<button
									class="mt-1 text-[11px] font-semibold text-accent cursor-pointer hover:underline"
									onclick={() => (showFolders = !showFolders)}
								>
									{showFolders ? 'Hide folders' : 'Show folders'}
								</button>
								{#if showFolders}
									<ul class="mt-1 flex flex-col gap-0.5">
										{#each folders.filter( (f) => (item.id === 'personal-files' ? f.kind === 'personal' : f.kind === 'other') ) as f (f.path)}
											<li class="flex items-center gap-2 text-xs min-w-0">
												<span class="text-ink truncate" title={f.path}>{f.label}</span>
												<span class="ml-auto font-mono text-[11px] text-muted shrink-0">
													{f.partial ? 'at least ' : ''}{formatBytes(f.bytes)}
												</span>
												<button
													class="text-muted hover:text-ink cursor-pointer shrink-0"
													title="Show in File Explorer"
													aria-label="Show {f.label} in File Explorer"
													onclick={() => RevealInExplorer(f.path)}
												>
													<StrokeIcon d={REVEAL_ICON} size={13} />
												</button>
											</li>
										{/each}
									</ul>
								{/if}
							{/if}
						</div>
					{/each}
				</section>
			{/each}
		</div>
	{/if}
</div>

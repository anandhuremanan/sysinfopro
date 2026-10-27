<script lang="ts">
	import { onMount } from 'svelte';
	import { CheckDevTools, RevealInExplorer } from '$lib/wailsjs/go/main/App';
	import { load, recall } from '$lib/cache';
	import BusyArt from '../ui/BusyArt.svelte';
	import StrokeIcon from '../ui/StrokeIcon.svelte';

	type Copy = { path: string; scope: string; version: string; active: boolean; note: string };
	type Issue = { level: 'warn' | 'fail'; text: string; fix: string };
	type Tool = {
		id: string;
		name: string;
		command: string;
		status: 'ok' | 'warn' | 'fail' | 'missing';
		version: string;
		copies: Copy[];
		issues: Issue[];
		vars: { key: string; value: string }[];
	};

	const REVEAL_ICON = 'M14 4h6v6M20 4l-9 9M18 14v4a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4';
	const MARK = {
		ok: 'M5 12l5 5 9-10',
		warn: 'M12 7v6M12 17h.01',
		fail: 'M7 7l10 10M17 7L7 17',
		missing: 'M7 12h10'
	};
	const TONE = {
		ok: 'text-green-700 dark:text-green-400',
		warn: 'text-amber-700 dark:text-amber-400',
		fail: 'text-red-700 dark:text-red-400',
		missing: 'text-muted'
	};

	let tools = $state<Tool[]>(recall('devtools') ?? []);
	let isLoading = $state(!recall('devtools'));
	let selectedId = $state('');

	let installed = $derived(tools.filter((t) => t.status !== 'missing'));
	let missing = $derived(tools.filter((t) => t.status === 'missing'));
	let problemCount = $derived(
		tools.filter((t) => t.status === 'warn' || t.status === 'fail').length
	);
	// open on the first tool with a problem
	let selected = $derived(
		tools.find((t) => t.id === selectedId) ??
			installed.find((t) => t.status === 'fail') ??
			installed.find((t) => t.status === 'warn') ??
			installed[0]
	);

	async function refresh(force = false) {
		isLoading = force || tools.length === 0;
		try {
			tools = ((await load('devtools', CheckDevTools, 5 * 60_000, force)) || []) as Tool[];
		} catch (err) {
			console.error('Dev tool check failed:', err);
		} finally {
			isLoading = false;
		}
	}

	onMount(() => {
		refresh();
	});
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<div class="flex justify-between items-center gap-3 pb-2 border-b border-line shrink-0">
		<div class="min-w-0">
			<div class="font-bold text-base text-ink">Dev doctor</div>
			<div class="text-xs text-muted truncate">
				{#if isLoading}
					Which copy of each tool runs when you type its name in a new terminal
				{:else if problemCount}
					{problemCount}
					{problemCount === 1 ? 'tool needs' : 'tools need'} a look · {installed.length} found
				{:else}
					All {installed.length} tools found run the way you would expect
				{/if}
			</div>
		</div>
		<button class="btn btn-secondary shrink-0" disabled={isLoading} onclick={() => refresh(true)}>
			Check again
		</button>
	</div>

	{#if isLoading}
		<div class="flex-1 flex items-center justify-center">
			<BusyArt kind="check" size={64} label="Asking each tool its version..." />
		</div>
	{:else if installed.length === 0}
		<div class="flex-1 flex items-center justify-center text-muted text-center px-8">
			None of the common developer tools are on PATH.
		</div>
	{:else}
		<div class="flex-1 min-h-0 grid grid-cols-[200px_1fr] gap-3">
			<!-- the tools -->
			<div
				class="min-h-0 overflow-y-auto overscroll-contain pb-5 [mask-image:linear-gradient(to_bottom,#000_calc(100%-20px),transparent)] flex flex-col gap-1 pr-1"
			>
				{#each installed as tool (tool.id)}
					<button
						class="flex items-center gap-2 px-2.5 py-2 rounded-xl text-left cursor-pointer border transition-colors {selected?.id ===
						tool.id
							? 'border-line bg-ground'
							: 'border-transparent hover:bg-tile'}"
						onclick={() => (selectedId = tool.id)}
					>
						<span class={TONE[tool.status]}><StrokeIcon d={MARK[tool.status]} size={14} /></span>
						<span class="flex-1 min-w-0">
							<span class="block text-xs font-semibold text-ink truncate">{tool.name}</span>
							<span class="block text-[11px] text-muted font-mono truncate">
								{tool.version || 'no answer'}{tool.copies.length > 1
									? ` · ${tool.copies.length} copies`
									: ''}
							</span>
						</span>
					</button>
				{/each}
				{#if missing.length}
					<p class="text-[11px] text-muted px-2.5 pt-2 leading-relaxed">
						Not found: {missing.map((t) => t.name).join(', ')}
					</p>
				{/if}
			</div>

			<!-- the selected tool -->
			{#if selected}
				{#key selected.id}
					<div
						class="min-h-0 overflow-y-auto overscroll-contain pb-5 [mask-image:linear-gradient(to_bottom,#000_calc(100%-20px),transparent)] flex flex-col gap-2 pr-1.5"
					>
						<div class="flex items-baseline gap-2">
							<span class="font-bold text-ink">{selected.name}</span>
							<span class="text-xs text-muted">
								typing <span class="font-mono text-ink">{selected.command}</span> runs
								{selected.version ? `version ${selected.version}` : 'the first copy below'}
							</span>
						</div>

						{#each selected.issues as issue, i (i)}
							<div
								class="rounded-xl px-3 py-2 text-xs leading-relaxed {issue.level === 'fail'
									? 'bg-red-500/10'
									: 'bg-amber-500/10'}"
							>
								<p class="font-semibold {TONE[issue.level]}">{issue.text}</p>
								{#if issue.fix}<p class="text-ink mt-0.5">
										<span class="font-semibold">Try:</span>
										{issue.fix}
									</p>{/if}
							</div>
						{/each}

						<section class="flex flex-col gap-1">
							<h3 class="text-[10px] uppercase tracking-wider font-semibold text-muted">
								{selected.copies.length === 1
									? 'Installed at'
									: 'Copies, in the order Windows looks'}
							</h3>
							{#each selected.copies as copy, i (copy.path)}
								<div
									class="flex items-center gap-2.5 rounded-xl border px-3 py-2 {copy.active
										? 'border-ink/30 bg-tile'
										: 'border-line bg-tile opacity-80'}"
								>
									<span class="text-[11px] font-mono text-muted w-4 shrink-0">{i + 1}</span>
									<span class="flex-1 min-w-0">
										<!-- cut long paths from the left: the end (folder and file) is what matters -->
										<span
											class="block text-xs font-mono text-ink truncate [direction:rtl] text-left"
											title={copy.path}><bdi>{copy.path}</bdi></span
										>
										<span class="block text-[11px] text-muted">
											{copy.version ? `Version ${copy.version}` : 'Did not say its version'}
											· {copy.scope === 'user'
												? 'your PATH'
												: copy.scope === 'system'
													? 'system PATH'
													: 'PATH'}
											{#if copy.note}· {copy.note}{/if}
										</span>
									</span>
									{#if copy.active}
										<span
											class="text-[11px] font-semibold px-1.5 py-0.5 rounded-md bg-ground text-ink shrink-0"
											>Runs</span
										>
									{/if}
									<button
										class="text-muted hover:text-ink cursor-pointer shrink-0"
										title="Show in File Explorer"
										aria-label="Show in File Explorer"
										onclick={() => RevealInExplorer(copy.path)}
									>
										<StrokeIcon d={REVEAL_ICON} size={14} />
									</button>
								</div>
							{/each}
						</section>

						{#if selected.vars.length}
							<section class="flex flex-col gap-1">
								<h3 class="text-[10px] uppercase tracking-wider font-semibold text-muted">
									Settings it reads
								</h3>
								<dl
									class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 text-xs rounded-xl bg-tile border border-line px-3 py-2"
								>
									{#each selected.vars as v (v.key)}
										<dt class="font-mono text-muted">{v.key}</dt>
										<dd class="font-mono text-ink truncate" title={v.value}>{v.value}</dd>
									{/each}
								</dl>
							</section>
						{/if}

						{#if selected.issues.length === 0}
							<p class="text-xs text-muted">
								Nothing to fix: {selected.copies.length > 1
									? 'the copies are the same version, so it does not matter which runs.'
									: 'one copy, and it answers.'}
							</p>
						{/if}
					</div>
				{/key}
			{/if}
		</div>
	{/if}
</div>

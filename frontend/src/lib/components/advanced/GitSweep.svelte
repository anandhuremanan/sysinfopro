<script lang="ts">
	import { onMount } from 'svelte';
	import {
		GetDrives,
		GetDuplicateFolders,
		PickFolder,
		RevealInExplorer
	} from '$lib/wailsjs/go/main/App';
	import { ClipboardSetText } from '$lib/wailsjs/runtime/runtime';
	import { cancelGitSweep, gitSweep, sweepRepos, type GitRepo } from '$lib/gitSweep.svelte';
	import BusyArt from '../ui/BusyArt.svelte';
	import Segmented from '../ui/Segmented.svelte';
	import StrokeIcon from '../ui/StrokeIcon.svelte';

	type Drive = { path: string };
	type Place = { label: string; path: string };
	type Filter = 'at-risk' | 'changes' | 'all';

	const REVEAL_ICON = 'M14 4h6v6M20 4l-9 9M18 14v4a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4';
	const PAGE = 60;

	let drives = $state<Drive[]>([]);
	let places = $state<Place[]>([]);
	let filter = $state<Filter>('at-risk');
	let opened = $state('');
	let copied = $state('');
	let shown = $state(PAGE);

	let result = $derived(gitSweep.result);
	let repos = $derived(
		(result?.repos ?? []).filter((r) =>
			filter === 'all'
				? true
				: filter === 'at-risk'
					? r.status === 'at-risk'
					: r.status === 'changes'
		)
	);
	let progress = $derived(gitSweep.progress);

	const folderName = (path: string) =>
		path
			.replace(/[\\/]$/, '')
			.split(/[\\/]/)
			.pop() || path;
	// where a project is, inside the folder searched
	function placeOf(path: string) {
		const root = (result?.root ?? '').replace(/[\\/]+$/, '');
		const parent = path.slice(0, path.lastIndexOf('\\'));
		if (root && parent.toLowerCase().startsWith(root.toLowerCase())) {
			return parent.slice(root.length).replace(/^[\\/]+/, '') || folderName(root) || root;
		}
		return parent;
	}
	function ago(ms: number) {
		if (!ms) return 'no commits';
		const days = Math.floor((Date.now() - ms) / 86_400_000);
		if (days < 1) return 'today';
		if (days < 2) return 'yesterday';
		if (days < 45) return `${days} days ago`;
		if (days < 600) return `${Math.round(days / 30)} months ago`;
		return `${(days / 365).toFixed(1)} years ago`;
	}
	const plural = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`;

	// the facts about a project, most serious first
	function badges(r: GitRepo) {
		const out: { text: string; tone: 'risk' | 'warn' | 'muted' }[] = [];
		if (r.secret_in_url) out.push({ text: 'password in remote address', tone: 'risk' });
		if (r.no_remote && !r.empty) out.push({ text: 'no remote: only on this PC', tone: 'risk' });
		else if (r.local_only)
			out.push({
				text: plural(r.local_only, 'commit not pushed', 'commits not pushed'),
				tone: 'risk'
			});
		if (r.stashes) out.push({ text: plural(r.stashes, 'stash', 'stashes'), tone: 'risk' });
		if (r.conflicts) out.push({ text: plural(r.conflicts, 'conflict', 'conflicts'), tone: 'warn' });
		if (r.changed)
			out.push({ text: plural(r.changed, 'changed file', 'changed files'), tone: 'warn' });
		if (r.untracked) out.push({ text: plural(r.untracked, 'new file', 'new files'), tone: 'warn' });
		if (r.behind) out.push({ text: `${r.behind} behind`, tone: 'muted' });
		return out;
	}
	const TONE = {
		risk: 'bg-red-500/10 text-red-700 dark:text-red-400',
		warn: 'bg-amber-500/10 text-amber-700 dark:text-amber-400',
		muted: 'bg-track text-muted'
	};

	function start(root: string) {
		filter = 'at-risk';
		opened = '';
		shown = PAGE;
		sweepRepos(root);
	}
	async function chooseFolder() {
		const path = await PickFolder('Choose a folder to search for git projects').catch(() => '');
		if (path) start(path);
	}
	async function copy(path: string) {
		await ClipboardSetText(path);
		copied = path;
		setTimeout(() => copied === path && (copied = ''), 1500);
	}

	onMount(() => {
		GetDrives()
			.then((d) => (drives = (d || []) as Drive[]))
			.catch(() => {});
		GetDuplicateFolders()
			.then(
				(p: Place[]) =>
					(places = (p || []).filter((x) => x.label === 'User folder' || x.label === 'Documents'))
			)
			.catch(() => {});
	});
</script>

<div class="flex flex-col h-full min-h-0 gap-2 p-1">
	<div class="flex justify-between items-center gap-3 pb-2 border-b border-line shrink-0">
		<div class="min-w-0">
			<div class="font-bold text-base text-ink">Git projects</div>
			<div class="text-xs text-muted truncate">
				{#if gitSweep.isSearching}
					Searching {gitSweep.root}
				{:else if result && !result.git_missing}
					{#if result.at_risk}
						<span class="text-red-700 dark:text-red-400 font-semibold"
							>{plural(result.at_risk, 'project has', 'projects have')} work only on this PC</span
						>
						· {result.repos.length} found in {folderName(result.root)}
					{:else}
						All {result.repos.length} projects in {folderName(result.root)} are pushed
					{/if}
				{:else}
					Work that exists only on this PC: not committed, not pushed, or no remote at all
				{/if}
			</div>
		</div>
		{#if gitSweep.isSearching}
			<button class="btn btn-secondary shrink-0" onclick={cancelGitSweep}>Stop</button>
		{:else if result}
			<button
				class="text-[11px] text-muted font-medium cursor-pointer hover:text-ink shrink-0"
				onclick={() => (gitSweep.result = null)}
			>
				New search
			</button>
		{/if}
	</div>

	{#if gitSweep.isSearching}
		<div class="flex-1 flex flex-col items-center justify-center gap-3 text-center">
			<BusyArt kind="search" size={64} />
			<div class="text-xs font-semibold text-ink">
				{#if progress?.phase === 'checking'}
					Checking project {progress.done} of {progress.found}...
				{:else}
					Looking for projects{progress ? `: ${progress.found} so far` : '...'}
				{/if}
			</div>
			{#if progress?.phase === 'checking' && progress.found}
				<div class="w-64 h-1.5 rounded-full bg-track overflow-hidden">
					<div
						class="h-full bg-accent rounded-full transition-[width] duration-200"
						style="width: {(progress.done / progress.found) * 100}%"
					></div>
				</div>
			{/if}
			<div class="text-[11px] text-muted max-w-sm truncate" title={progress?.current}>
				{progress?.current ?? ''}
			</div>
			<div class="text-[11px] text-muted">You can use other tools meanwhile; it keeps going.</div>
		</div>
	{:else if result?.git_missing}
		<div class="flex-1 flex flex-col items-center justify-center gap-1 text-center px-8">
			<div class="text-sm font-semibold text-ink">Git is not installed</div>
			<div class="text-xs text-muted">
				This tool asks git about each project. Install it from git-scm.com and try again.
			</div>
		</div>
	{:else if !result}
		<div class="flex-1 min-h-0 overflow-y-auto flex flex-col gap-4 pt-1">
			{#if gitSweep.error}<p class="text-xs text-red-700 dark:text-red-400">
					{gitSweep.error}
				</p>{/if}
			<section class="flex flex-col gap-2">
				<h3 class="text-[10px] uppercase tracking-wider font-semibold text-muted">Search</h3>
				<div class="flex flex-wrap gap-2">
					{#each places as place (place.path)}
						<button class="btn btn-secondary" title={place.path} onclick={() => start(place.path)}
							>{place.label}</button
						>
					{/each}
					{#each drives as drive (drive.path)}
						{@const letter = drive.path.replace(/[\\/]$/, '')}
						<button class="btn btn-secondary" onclick={() => start(letter + '\\')}
							>All of {letter}</button
						>
					{/each}
					<button class="btn btn-primary" onclick={chooseFolder}>Choose a folder...</button>
				</div>
			</section>
			<p class="text-[11px] text-muted leading-relaxed max-w-2xl">
				Every project found is asked, read-only, what it has that no remote has: commits never
				pushed, stashes, changes not committed, and projects never pushed anywhere. Nothing is
				fetched or changed. Before you reset or hand over a PC, this is the work you would lose.
			</p>
		</div>
	{:else if result.repos.length === 0}
		<div class="flex-1 flex items-center justify-center text-muted text-center px-8">
			No git projects in {result.root}.
		</div>
	{:else}
		<div class="flex items-center gap-3 shrink-0 px-1">
			<Segmented
				options={[
					{ id: 'at-risk', label: `Only on this PC · ${result.at_risk}` },
					{ id: 'changes', label: `Uncommitted · ${result.with_changes}` },
					{ id: 'all', label: `All · ${result.repos.length}` }
				]}
				value={filter}
				onchange={(v) => {
					filter = v as Filter;
					shown = PAGE;
				}}
				label="Which projects to show"
				size="sm"
			/>
		</div>

		<div class="flex-1 min-h-0 overflow-y-auto overscroll-contain pr-1.5 flex flex-col gap-1.5">
			{#each repos.slice(0, shown) as r (r.path)}
				{@const open = opened === r.path}
				<div
					class="rounded-xl bg-tile border px-3 py-2 {r.status === 'at-risk'
						? 'border-red-500/30'
						: 'border-line'}"
				>
					<div class="flex items-center gap-2 min-w-0">
						<button
							class="flex-1 min-w-0 text-left cursor-pointer"
							aria-expanded={open}
							onclick={() => (opened = open ? '' : r.path)}
						>
							<span class="flex items-baseline gap-2 min-w-0">
								<span class="text-xs font-semibold text-ink truncate">{r.name}</span>
								<span class="text-[11px] text-muted truncate">{placeOf(r.path)}</span>
							</span>
						</button>
						<span class="text-[11px] text-muted shrink-0"
							>{r.branch ? `${r.branch} · ` : ''}{ago(r.last_commit)}</span
						>
						<button
							class="text-muted hover:text-ink cursor-pointer shrink-0"
							title="Show in File Explorer"
							aria-label="Show {r.name} in File Explorer"
							onclick={() => RevealInExplorer(r.path)}
						>
							<StrokeIcon d={REVEAL_ICON} size={13} />
						</button>
					</div>
					{#if badges(r).length || r.error}
						<div class="flex flex-wrap gap-1 mt-1.5">
							{#each badges(r) as b (b.text)}
								<span class="text-[11px] font-semibold px-1.5 py-0.5 rounded-md {TONE[b.tone]}"
									>{b.text}</span
								>
							{/each}
							{#if r.error}<span
									class="text-[11px] px-1.5 py-0.5 rounded-md bg-track text-muted"
									title={r.error}>git could not read it</span
								>{/if}
						</div>
					{/if}
					{#if open}
						<div class="mt-2 text-xs flex flex-col gap-1">
							<div class="flex items-center gap-2 min-w-0">
								<span class="text-muted shrink-0">Folder</span>
								<span class="font-mono text-[11px] text-ink truncate" title={r.path}>{r.path}</span>
								<button
									class="text-[11px] font-semibold text-accent cursor-pointer hover:underline shrink-0"
									onclick={() => copy(r.path)}
								>
									{copied === r.path ? 'Copied' : 'Copy'}
								</button>
							</div>
							<div class="flex items-center gap-2 min-w-0">
								<span class="text-muted shrink-0">Remote</span>
								<span class="font-mono text-[11px] text-ink truncate">{r.remote || 'none'}</span>
							</div>
							{#if r.branches.length}
								<div class="text-muted">Branches with commits no remote has:</div>
								<ul class="flex flex-col gap-0.5 pl-2">
									{#each r.branches as b (b.name)}
										<li class="flex gap-2">
											<span class="font-mono text-[11px] text-ink">{b.name}</span>
											<span class="text-[11px] text-muted">
												{plural(b.local_only, 'commit', 'commits')}{b.upstream
													? ''
													: ' · never pushed'}
											</span>
										</li>
									{/each}
								</ul>
							{/if}
							{#if r.secret_in_url}
								<p class="text-red-700 dark:text-red-400 leading-relaxed">
									The remote's address has a password or token written in it, so anyone who copies
									this folder gets it too. Revoke the token, then sign in with Git Credential
									Manager instead.
								</p>
							{/if}
						</div>
					{/if}
				</div>
			{:else}
				<div class="text-xs text-muted px-1 py-6 text-center">
					{filter === 'at-risk'
						? 'Every commit is on a remote. Nothing here exists only on this PC.'
						: 'No uncommitted changes.'}
				</div>
			{/each}
			{#if repos.length > shown}
				<button class="btn btn-secondary self-center my-1" onclick={() => (shown += PAGE)}>
					Show more ({repos.length - shown} left)
				</button>
			{/if}
		</div>
	{/if}
</div>

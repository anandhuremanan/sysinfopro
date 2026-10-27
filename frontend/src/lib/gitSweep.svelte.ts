import { endTask, startTask, updateTask } from '$lib/tasks.svelte';
import { CancelGitSweep, FindGitRepos } from '$lib/wailsjs/go/main/App';
import { EventsOn } from '$lib/wailsjs/runtime/runtime';

export type GitBranch = { name: string; local_only: number; upstream: boolean };
export type GitRepo = {
	path: string;
	name: string;
	branch: string;
	changed: number;
	untracked: number;
	conflicts: number;
	local_only: number;
	branches: GitBranch[];
	behind: number;
	stashes: number;
	remote: string;
	no_remote: boolean;
	secret_in_url: boolean;
	empty: boolean;
	last_commit: number;
	status: 'at-risk' | 'changes' | 'clean' | 'unreadable';
	error: string;
};
export type GitResult = {
	root: string;
	repos: GitRepo[];
	git_missing: boolean;
	at_risk: number;
	with_changes: number;
	unreadable: number;
	duration_ms: number;
};
type Progress = { phase: 'finding' | 'checking'; found: number; done: number; current: string };

// The project search lives here rather than in its screen, so it keeps going
// (and its result stays) when you move to another screen.
export const gitSweep = $state<{
	root: string;
	result: GitResult | null;
	isSearching: boolean;
	progress: Progress | null;
	error: string;
}>({ root: '', result: null, isSearching: false, progress: null, error: '' });

const TASK = 'git-sweep';
let searchId = 0;
let listening = false;

const folderName = (path: string) =>
	path
		.replace(/[\\/]$/, '')
		.split(/[\\/]/)
		.pop() || path;

function listen() {
	if (listening) return;
	listening = true;
	EventsOn('gitsweep-progress', (p: Progress) => {
		if (!gitSweep.isSearching) return;
		gitSweep.progress = p;
		updateTask(
			TASK,
			p.phase === 'finding' ? `${p.found} found` : `checking ${p.done} of ${p.found}`
		);
	});
}

export async function sweepRepos(root: string) {
	listen();
	const id = ++searchId;
	gitSweep.root = root;
	gitSweep.result = null;
	gitSweep.progress = null;
	gitSweep.error = '';
	gitSweep.isSearching = true;
	startTask({ id: TASK, label: `Git projects in ${folderName(root)}`, view: 'gitsweep' });

	try {
		const result = (await FindGitRepos(root)) as GitResult;
		if (id !== searchId) return;
		gitSweep.result = result;
		endTask(TASK, {
			title: 'Git project search finished',
			body: result.at_risk
				? `${result.at_risk} of ${result.repos.length} projects have work that exists only on this PC.`
				: `${result.repos.length} projects found; all work is pushed.`
		});
	} catch (err) {
		if (id !== searchId) return;
		gitSweep.error = `${err}`.includes('canceled') ? '' : `${err}`;
		endTask(TASK);
	} finally {
		if (id === searchId) gitSweep.isSearching = false;
	}
}

export function cancelGitSweep() {
	if (!gitSweep.isSearching) return;
	searchId++;
	gitSweep.isSearching = false;
	CancelGitSweep();
	endTask(TASK);
}

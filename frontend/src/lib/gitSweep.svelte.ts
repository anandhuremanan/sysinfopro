import { createJob, errorText } from '$lib/job';
import { folderName } from '$lib/path';
import { CancelGitSweep, FindGitRepos } from '$lib/wailsjs/go/main/App';
import type { gitsweep } from '$lib/wailsjs/go/models';

// the shapes come from the Go structs, so a renamed field breaks the build, not the screen
export type GitBranch = gitsweep.Branch;
export type GitRepo = gitsweep.Repo;
export type GitResult = gitsweep.Result;
// progress events are not part of a binding, so this one is written by hand
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

const job = createJob<Progress>({
	id: 'git-sweep',
	view: 'gitsweep',
	event: 'gitsweep-progress',
	isRunning: () => gitSweep.isSearching,
	onProgress: (p) => (gitSweep.progress = p),
	detail: (p) => (p.phase === 'finding' ? `${p.found} found` : `checking ${p.done} of ${p.found}`),
	cancelBackend: CancelGitSweep
});

export async function sweepRepos(root: string) {
	gitSweep.root = root;
	gitSweep.result = null;
	gitSweep.progress = null;
	gitSweep.error = '';
	gitSweep.isSearching = true;
	const id = job.start(`Git projects in ${folderName(root)}`);

	try {
		const result = await FindGitRepos(root);
		if (!job.isCurrent(id)) return;
		gitSweep.result = result;
		job.finish(id, {
			title: 'Git project search finished',
			body: result.at_risk
				? `${result.at_risk} of ${result.repos.length} projects have work that exists only on this PC.`
				: `${result.repos.length} projects found; all work is pushed.`
		});
	} catch (err) {
		if (!job.isCurrent(id)) return;
		gitSweep.error = errorText(err);
		job.finish(id);
	} finally {
		if (job.isCurrent(id)) gitSweep.isSearching = false;
	}
}

export function cancelGitSweep() {
	if (!gitSweep.isSearching) return;
	gitSweep.isSearching = false;
	job.cancel();
}

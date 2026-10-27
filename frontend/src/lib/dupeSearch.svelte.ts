import { formatBytes } from '$lib';
import { createJob, errorText } from '$lib/job';
import { folderName } from '$lib/path';
import { CancelDuplicateSearch, FindDuplicates } from '$lib/wailsjs/go/main/App';
import { dupes } from '$lib/wailsjs/go/models';

// the shapes come from the Go structs, so a renamed field breaks the build, not the screen
export type DupeFile = dupes.File;
export type DupeGroup = dupes.Group;
export type DupeResult = dupes.Result;
// progress events are not part of a binding, so this one is written by hand
type Progress = {
	phase: 'listing' | 'comparing';
	files: number;
	done: number;
	total: number;
	current: string;
};

// The duplicate search lives here rather than in its screen, so it keeps going
// (and its result stays) when you move to another screen.
export const dupeSearch = $state<{
	root: string;
	result: DupeResult | null;
	isSearching: boolean;
	progress: Progress | null;
	error: string;
}>({ root: '', result: null, isSearching: false, progress: null, error: '' });

const job = createJob<Progress>({
	id: 'dupe-search',
	view: 'dupes',
	event: 'dupes-progress',
	isRunning: () => dupeSearch.isSearching,
	onProgress: (p) => (dupeSearch.progress = p),
	detail: (p) =>
		p.phase === 'listing'
			? `${p.files.toLocaleString()} files`
			: `comparing ${p.total ? Math.round((p.done / p.total) * 100) : 0}%`,
	cancelBackend: CancelDuplicateSearch
});

export async function searchDupes(root: string, minSize: number) {
	dupeSearch.root = root;
	dupeSearch.result = null;
	dupeSearch.progress = null;
	dupeSearch.error = '';
	dupeSearch.isSearching = true;
	const id = job.start(`Duplicates in ${folderName(root)}`);

	try {
		const result = await FindDuplicates(root, minSize);
		if (!job.isCurrent(id)) return;
		dupeSearch.result = result;
		job.finish(id, {
			title: 'Duplicate search finished',
			body: result.groups.length
				? `${formatBytes(result.wasted)} in extra copies in ${folderName(root)}.`
				: `No duplicates in ${folderName(root)}.`
		});
	} catch (err) {
		if (!job.isCurrent(id)) return;
		dupeSearch.error = errorText(err);
		job.finish(id);
	} finally {
		if (job.isCurrent(id)) dupeSearch.isSearching = false;
	}
}

export function cancelDupeSearch() {
	if (!dupeSearch.isSearching) return;
	dupeSearch.isSearching = false;
	job.cancel();
}

// take removed files out of the shown result; a group with one file left is no longer a duplicate
export function dropRemoved(paths: Set<string>) {
	const r = dupeSearch.result;
	if (!r) return;
	const groups: DupeGroup[] = [];
	for (const g of r.groups) {
		const files = g.files.filter((f) => !paths.has(f.path));
		if (files.length > 1)
			groups.push(new dupes.Group({ ...g, files, wasted: g.size * (files.length - 1) }));
	}
	dupeSearch.result = new dupes.Result({
		...r,
		groups,
		wasted: groups.reduce((sum, g) => sum + g.wasted, 0)
	});
}

import { formatBytes } from '$lib';
import { endTask, startTask, updateTask } from '$lib/tasks.svelte';
import { CancelDuplicateSearch, FindDuplicates } from '$lib/wailsjs/go/main/App';
import { EventsOn } from '$lib/wailsjs/runtime/runtime';

export type DupeFile = { path: string; modified: number };
export type DupeGroup = {
	id: string;
	size: number;
	files: DupeFile[];
	wasted: number;
	risky: boolean;
};
export type DupeResult = {
	root: string;
	groups: DupeGroup[];
	files: number;
	wasted: number;
	cloud_skipped: number;
	unreadable: number;
	truncated: boolean;
	duration_ms: number;
};
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

const TASK = 'dupe-search';
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
	EventsOn('dupes-progress', (p: Progress) => {
		if (!dupeSearch.isSearching) return;
		dupeSearch.progress = p;
		updateTask(
			TASK,
			p.phase === 'listing'
				? `${p.files.toLocaleString()} files`
				: `comparing ${p.total ? Math.round((p.done / p.total) * 100) : 0}%`
		);
	});
}

export async function searchDupes(root: string, minSize: number) {
	listen();
	const id = ++searchId;
	dupeSearch.root = root;
	dupeSearch.result = null;
	dupeSearch.progress = null;
	dupeSearch.error = '';
	dupeSearch.isSearching = true;
	startTask({ id: TASK, label: `Duplicates in ${folderName(root)}`, view: 'dupes' });

	try {
		const result = (await FindDuplicates(root, minSize)) as DupeResult;
		if (id !== searchId) return;
		dupeSearch.result = result;
		endTask(TASK, {
			title: 'Duplicate search finished',
			body: result.groups.length
				? `${formatBytes(result.wasted)} in extra copies in ${folderName(root)}.`
				: `No duplicates in ${folderName(root)}.`
		});
	} catch (err) {
		if (id !== searchId) return;
		dupeSearch.error = `${err}`.includes('canceled') ? '' : `${err}`;
		endTask(TASK);
	} finally {
		if (id === searchId) dupeSearch.isSearching = false;
	}
}

export function cancelDupeSearch() {
	if (!dupeSearch.isSearching) return;
	searchId++;
	dupeSearch.isSearching = false;
	CancelDuplicateSearch();
	endTask(TASK);
}

// take removed files out of the shown result; a group with one file left is no longer a duplicate
export function dropRemoved(paths: Set<string>) {
	const r = dupeSearch.result;
	if (!r) return;
	const groups: DupeGroup[] = [];
	for (const g of r.groups) {
		const files = g.files.filter((f) => !paths.has(f.path));
		if (files.length > 1) groups.push({ ...g, files, wasted: g.size * (files.length - 1) });
	}
	dupeSearch.result = { ...r, groups, wasted: groups.reduce((sum, g) => sum + g.wasted, 0) };
}

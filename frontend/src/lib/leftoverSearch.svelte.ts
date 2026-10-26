import { endTask, startTask, updateTask } from '$lib/tasks.svelte';
import { CancelLeftoverSearch, FindLeftovers } from '$lib/wailsjs/go/main/App';
import { EventsOn } from '$lib/wailsjs/runtime/runtime';

export type LeftoverItem = {
	id: string;
	name: string;
	kind: 'cache' | 'leftover';
	label: string;
	folder: string;
	path: string;
	size_bytes: number;
	files: number;
	note: string;
	deletable: boolean;
	last_used: string;
};

// The Dev caches project search lives here rather than in its screen, so it keeps
// going (and its result stays) when you move to another screen.
export const leftoverSearch = $state<{
	root: string;
	items: LeftoverItem[];
	isSearching: boolean;
	current: string; // folder being searched right now
	error: string;
}>({ root: '', items: [], isSearching: false, current: '', error: '' });

const TASK = 'leftover-search';
let searchId = 0;
let listening = false;

function listen() {
	if (listening) return;
	listening = true;
	EventsOn('devcache-progress', (current: string) => {
		if (!leftoverSearch.isSearching || !current) return;
		leftoverSearch.current = current;
		updateTask(TASK, current.split(/[\\/]/).filter(Boolean).pop() ?? '');
	});
}

const folderName = (path: string) => path.replace(/[\\/]$/, '').split(/[\\/]/).pop() || path;

export async function searchLeftovers(root: string) {
	listen();
	const id = ++searchId;
	leftoverSearch.root = root;
	leftoverSearch.current = root;
	leftoverSearch.items = [];
	leftoverSearch.error = '';
	leftoverSearch.isSearching = true;
	startTask({ id: TASK, label: `Searching ${folderName(root)}`, view: 'devcache' });

	try {
		const found = ((await FindLeftovers(root)) || []) as LeftoverItem[];
		if (id !== searchId) return;
		leftoverSearch.items = found;
		const total = found.reduce((sum, item) => sum + item.size_bytes, 0);
		endTask(TASK, {
			title: 'Project search finished',
			body: found.length
				? `${found.length} leftover ${found.length === 1 ? 'folder' : 'folders'} in ${folderName(root)}, ${(total / 1024 ** 3).toFixed(1)} GB in all.`
				: `No leftovers found in ${folderName(root)}.`
		});
	} catch (err) {
		if (id !== searchId) return;
		leftoverSearch.error = `${err}`;
		endTask(TASK);
	} finally {
		if (id === searchId) leftoverSearch.isSearching = false;
	}
}

export function cancelLeftoverSearch() {
	if (!leftoverSearch.isSearching) return;
	searchId++;
	leftoverSearch.isSearching = false;
	CancelLeftoverSearch();
	endTask(TASK);
}

// drop folders that were deleted from the shown result
export function removeLeftovers(paths: Set<string>) {
	leftoverSearch.items = leftoverSearch.items.filter((item) => !paths.has(item.path));
}

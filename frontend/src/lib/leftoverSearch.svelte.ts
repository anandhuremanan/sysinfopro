import { createJob } from '$lib/job';
import { folderName } from '$lib/path';
import { CancelLeftoverSearch, FindLeftovers } from '$lib/wailsjs/go/main/App';
import type { devcache } from '$lib/wailsjs/go/models';

// the shape comes from the Go struct, so a renamed field breaks the build, not the screen
export type LeftoverItem = devcache.Item;

// The Dev caches project search lives here rather than in its screen, so it keeps
// going (and its result stays) when you move to another screen.
export const leftoverSearch = $state<{
	root: string;
	items: LeftoverItem[];
	isSearching: boolean;
	current: string; // folder being searched right now
	error: string;
}>({ root: '', items: [], isSearching: false, current: '', error: '' });

const job = createJob<string>({
	id: 'leftover-search',
	view: 'devcache',
	event: 'devcache-progress',
	isRunning: () => leftoverSearch.isSearching,
	onProgress: (current) => (leftoverSearch.current = current),
	detail: (current) => folderName(current),
	cancelBackend: CancelLeftoverSearch
});

export async function searchLeftovers(root: string) {
	leftoverSearch.root = root;
	leftoverSearch.current = root;
	leftoverSearch.items = [];
	leftoverSearch.error = '';
	leftoverSearch.isSearching = true;
	const id = job.start(`Searching ${folderName(root)}`);

	try {
		const found = (await FindLeftovers(root)) || [];
		if (!job.isCurrent(id)) return;
		leftoverSearch.items = found;
		const total = found.reduce((sum, item) => sum + item.size_bytes, 0);
		job.finish(id, {
			title: 'Project search finished',
			body: found.length
				? `${found.length} leftover ${found.length === 1 ? 'folder' : 'folders'} in ${folderName(root)}, ${(total / 1024 ** 3).toFixed(1)} GB in all.`
				: `No leftovers found in ${folderName(root)}.`
		});
	} catch (err) {
		if (!job.isCurrent(id)) return;
		leftoverSearch.error = `${err}`;
		job.finish(id);
	} finally {
		if (job.isCurrent(id)) leftoverSearch.isSearching = false;
	}
}

export function cancelLeftoverSearch() {
	if (!leftoverSearch.isSearching) return;
	leftoverSearch.isSearching = false;
	job.cancel();
}

// drop folders that were deleted from the shown result
export function removeLeftovers(paths: Set<string>) {
	leftoverSearch.items = leftoverSearch.items.filter((item) => !paths.has(item.path));
}

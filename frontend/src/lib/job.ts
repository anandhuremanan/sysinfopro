import { endTask, startTask, updateTask } from '$lib/tasks.svelte';
import { EventsOn } from '$lib/wailsjs/runtime/runtime';

// The common part of a long background job (a disk analysis, a duplicate or
// project search): a header chip while it runs, progress events from the backend,
// a banner when it ends, and a run number so a cancelled or replaced run's late
// answer is ignored. Each job keeps its own result state; this keeps the plumbing.
export function createJob<P>(options: {
	id: string; // task id, e.g. 'dupe-search'
	view: string; // the screen that shows the result
	event?: string; // backend progress event
	isRunning: () => boolean; // progress is ignored once the job has stopped
	onProgress?: (progress: P) => void;
	detail?: (progress: P) => string; // live text on the header chip
	cancelBackend: () => void;
}) {
	let run = 0;
	let listening = false;

	function listen() {
		if (listening || !options.event) return;
		listening = true;
		EventsOn(options.event, (progress: P) => {
			if (!options.isRunning() || !progress) return;
			options.onProgress?.(progress);
			if (options.detail) updateTask(options.id, options.detail(progress));
		});
	}

	return {
		// starts a run and returns its number
		start(label: string): number {
			listen();
			startTask({ id: options.id, label, view: options.view });
			return ++run;
		},
		// true while this run has not been cancelled or replaced
		isCurrent(id: number): boolean {
			return id === run;
		},
		// ends the run; with a result, a banner announces it when its screen is not showing
		finish(id: number, result?: { title: string; body: string }): boolean {
			if (id !== run) return false;
			endTask(options.id, result);
			return true;
		},
		cancel() {
			run++;
			options.cancelBackend();
			endTask(options.id);
		}
	};
}

// a cancelled run reports "context canceled"; that is not an error to show
export const errorText = (err: unknown) => (`${err}`.includes('canceled') ? '' : `${err}`);

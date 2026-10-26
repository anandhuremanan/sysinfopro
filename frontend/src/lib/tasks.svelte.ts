import { driveAlerts } from '$lib/alerts.svelte';

// Long jobs (a disk analysis, a project search) keep running when you leave their
// screen. The header shows each as a chip, and a banner says when one finishes.
export type Task = {
	id: string;
	label: string; // "Analysing C:"
	detail: string; // live progress, e.g. "12,400 files"
	view: string; // the screen that shows the result
};

export const tasks = $state<{ running: Task[]; view: string }>({ running: [], view: 'home' });

// the page reports which screen is showing, so finished jobs know whether to announce themselves
export function setActiveView(view: string) {
	tasks.view = view;
}

export function startTask(task: Omit<Task, 'detail'>) {
	tasks.running = [...tasks.running.filter((t) => t.id !== task.id), { ...task, detail: '' }];
}

export function updateTask(id: string, detail: string) {
	const task = tasks.running.find((t) => t.id === id);
	if (task) task.detail = detail;
}

// Ends a task. With a result, and when its screen is not showing, a banner offers
// to open it; if you are already looking at it, the result is simply there.
export function endTask(id: string, result?: { title: string; body: string }) {
	const task = tasks.running.find((t) => t.id === id);
	tasks.running = tasks.running.filter((t) => t.id !== id);
	if (task && result && tasks.view !== task.view) {
		driveAlerts.current = {
			kind: 'task',
			drive: '',
			title: result.title,
			body: result.body,
			view: task.view,
			at: Date.now(),
			free_bytes: 0
		};
	}
}

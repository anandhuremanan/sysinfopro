import type { storage } from '$lib/wailsjs/go/models';
import { formatBytes } from '$lib';
import { endTask, startTask, updateTask } from '$lib/tasks.svelte';
import { AnalyzeDiskUsage, CancelDiskScan } from '$lib/wailsjs/go/main/App';
import { EventsOn } from '$lib/wailsjs/runtime/runtime';

export type UsageEntry = storage.UsageEntry;

export type FolderUsage = storage.FolderUsage;

export type ScanProgress = { files: number; bytes: number; current: string };

// Disk usage analysis lives here rather than in its screen, so a scan keeps going
// (and its result is kept) when you move to another screen.
export const diskScan = $state<{
	drivePath: string; // selected drive, "C:"
	currentPath: string; // folder being shown or scanned
	usage: FolderUsage | undefined;
	isScanning: boolean;
	progress: ScanProgress | undefined;
	error: string;
}>({
	drivePath: '',
	currentPath: '',
	usage: undefined,
	isScanning: false,
	progress: undefined,
	error: ''
});

const TASK = 'disk-scan';
// finished scans, so going back up a level is instant
const cache = new Map<string, FolderUsage>();
// bumped on every scan or cancel, so a superseded scan's result is ignored
let scanId = 0;
let listening = false;

function listen() {
	if (listening) return;
	listening = true;
	EventsOn('disk-scan-progress', (data: ScanProgress) => {
		if (!diskScan.isScanning || !data) return;
		diskScan.progress = data;
		updateTask(TASK, `${data.files.toLocaleString()} files`);
	});
}

// show a folder: from the cache if it was scanned before, otherwise empty until scanned
export function showPath(drivePath: string, path: string) {
	diskScan.drivePath = drivePath;
	diskScan.currentPath = path;
	diskScan.usage = cache.get(path);
}

export async function scanPath(path: string, force = false) {
	listen();
	cancelScan();
	diskScan.currentPath = path;
	diskScan.error = '';

	const cached = cache.get(path);
	if (cached && !force) {
		diskScan.usage = cached;
		return;
	}

	const id = ++scanId;
	diskScan.isScanning = true;
	diskScan.progress = undefined;
	diskScan.usage = undefined;
	const where = path.replace(/[\\/]$/, '');
	startTask({
		id: TASK,
		label: `Analysing ${where.length > 18 ? '…' + where.slice(-16) : where}`,
		view: 'space'
	});

	try {
		const result = (await AnalyzeDiskUsage(path)) as FolderUsage;
		if (id !== scanId || result.cancelled) return;
		cache.set(path, result);
		if (diskScan.currentPath === path) diskScan.usage = result;
		endTask(TASK, {
			title: 'Disk analysis finished',
			body: `${formatBytes(result.total_bytes)} in ${result.files.toLocaleString()} files on ${where}`
		});
	} catch (err) {
		if (id !== scanId) return;
		diskScan.error = `${err}`;
		endTask(TASK);
	} finally {
		if (id === scanId) diskScan.isScanning = false;
	}
}

export function cancelScan() {
	if (!diskScan.isScanning) return;
	scanId++;
	diskScan.isScanning = false;
	CancelDiskScan();
	endTask(TASK);
}

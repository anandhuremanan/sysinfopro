import { GetDrives, SetWidgetMode } from '$lib/wailsjs/go/main/App';
import {
	WindowCenter,
	WindowSetAlwaysOnTop,
	WindowSetSize
} from '$lib/wailsjs/runtime/runtime';

// 'cpu' | 'memory' | 'battery', or a drive as 'drive:C:'
export type WidgetMetric = string;

const DRIVE_PREFIX = 'drive:';
const STORAGE_KEY = 'sysinfopro-widget';

// Keep in step with Width/Height in main.go
const FULL_SIZE = { width: 768, height: 460 };
const WIDGET_WIDTH = 212;
const WIDGET_HEADER = 26;
const WIDGET_ROW = 26;
const WIDGET_PADDING = 8;

export const widget = $state<{
	active: boolean;
	metrics: WidgetMetric[];
	// drive letters found on this PC, e.g. ['C:', 'D:']
	drives: string[];
}>({
	active: false,
	metrics: ['cpu', 'memory', 'battery'],
	drives: []
});

export const driveMetric = (path: string) => DRIVE_PREFIX + path;
export const isDriveMetric = (id: WidgetMetric) => id.startsWith(DRIVE_PREFIX);
export const drivePath = (id: WidgetMetric) => id.slice(DRIVE_PREFIX.length);

// Everything the widget can show, in display order: CPU, memory, each drive, battery
export function widgetOptions(): { id: WidgetMetric; label: string }[] {
	return [
		{ id: 'cpu', label: 'CPU' },
		{ id: 'memory', label: 'Memory' },
		...widget.drives.map((path) => ({ id: driveMetric(path), label: path })),
		{ id: 'battery', label: 'Battery' }
	];
}

function inDisplayOrder(metrics: WidgetMetric[]) {
	const known = widgetOptions().map((option) => option.id);
	// a saved drive that is not plugged in right now is kept, after the known ones
	return [...known.filter((id) => metrics.includes(id)), ...metrics.filter((id) => !known.includes(id))];
}

function widgetHeight() {
	return WIDGET_HEADER + widget.metrics.length * WIDGET_ROW + WIDGET_PADDING;
}

function save() {
	try {
		localStorage.setItem(STORAGE_KEY, JSON.stringify({ metrics: widget.metrics }));
	} catch {
		// storage unavailable, the choice still applies for this session
	}
}

// Call once on startup: finds the drives and restores what the user chose to watch
export async function initWidget() {
	try {
		const drives = (await GetDrives()) || [];
		widget.drives = drives.map((d) => d.path.replace(/[\\/]$/, ''));
	} catch {
		widget.drives = [];
	}

	let saved: WidgetMetric[] | undefined;
	try {
		saved = JSON.parse(localStorage.getItem(STORAGE_KEY) || 'null')?.metrics;
	} catch {
		// unreadable settings, fall through to the defaults
	}

	if (Array.isArray(saved) && saved.length > 0) {
		// an older version stored one 'disk' entry for the system drive
		const migrated = saved.map((id) =>
			id === 'disk' && widget.drives[0] ? driveMetric(widget.drives[0]) : id
		);
		const valid = migrated.filter(
			(id) => id === 'cpu' || id === 'memory' || id === 'battery' || isDriveMetric(id)
		);
		if (valid.length > 0) {
			widget.metrics = inDisplayOrder([...new Set(valid)]);
			return;
		}
	}

	// default: CPU, memory, the system drive and battery
	widget.metrics = inDisplayOrder([
		'cpu',
		'memory',
		...(widget.drives[0] ? [driveMetric(widget.drives[0])] : []),
		'battery'
	]);
}

export function enterWidget() {
	widget.active = true;
	WindowSetSize(WIDGET_WIDTH, widgetHeight());
	// the widget always floats above other windows
	WindowSetAlwaysOnTop(true);
	// and stays visible through "Show desktop" (Win+D)
	SetWidgetMode(true);
}

export function exitWidget() {
	widget.active = false;
	SetWidgetMode(false);
	WindowSetAlwaysOnTop(false);
	WindowSetSize(FULL_SIZE.width, FULL_SIZE.height);
	// The widget may sit at a screen edge where the full window would not fit, so re-centre.
	// Centring uses the window's current size, so it has to wait for the resize to land.
	setTimeout(WindowCenter, 120);
}

export function toggleWidgetMetric(id: WidgetMetric) {
	if (widget.metrics.includes(id)) {
		// the widget always shows at least one thing
		if (widget.metrics.length === 1) return;
		widget.metrics = widget.metrics.filter((m) => m !== id);
	} else {
		widget.metrics = inDisplayOrder([...widget.metrics, id]);
	}
	save();
	if (widget.active) WindowSetSize(WIDGET_WIDTH, widgetHeight());
}

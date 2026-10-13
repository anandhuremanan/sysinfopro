import {
	WindowCenter,
	WindowSetAlwaysOnTop,
	WindowSetSize
} from '$lib/wailsjs/runtime/runtime';

export type WidgetMetric = 'cpu' | 'memory' | 'disk' | 'battery';

export const WIDGET_METRICS: { id: WidgetMetric; label: string }[] = [
	{ id: 'cpu', label: 'CPU' },
	{ id: 'memory', label: 'Memory' },
	{ id: 'disk', label: 'Disk' },
	{ id: 'battery', label: 'Battery' }
];

const STORAGE_KEY = 'sysinfopro-widget';

// Keep in step with Width/Height in main.go
const FULL_SIZE = { width: 768, height: 460 };
const WIDGET_WIDTH = 196;
const WIDGET_HEADER = 26;
const WIDGET_ROW = 26;
const WIDGET_PADDING = 8;

export const widget = $state<{ active: boolean; metrics: WidgetMetric[] }>({
	active: false,
	metrics: WIDGET_METRICS.map((m) => m.id)
});

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

// Call once on startup to restore what the user chose to watch
export function initWidget() {
	try {
		const saved = JSON.parse(localStorage.getItem(STORAGE_KEY) || 'null');
		if (!saved) return;
		const known = WIDGET_METRICS.map((m) => m.id);
		const metrics = (saved.metrics ?? []).filter((id: WidgetMetric) => known.includes(id));
		if (metrics.length > 0) widget.metrics = metrics;
	} catch {
		// unreadable settings, keep the defaults
	}
}

export function enterWidget() {
	widget.active = true;
	WindowSetSize(WIDGET_WIDTH, widgetHeight());
	// the widget always floats above other windows
	WindowSetAlwaysOnTop(true);
}

export function exitWidget() {
	widget.active = false;
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
		// keep the display order fixed regardless of the order they were ticked
		widget.metrics = WIDGET_METRICS.map((m) => m.id).filter(
			(m) => m === id || widget.metrics.includes(m)
		);
	}
	save();
	if (widget.active) WindowSetSize(WIDGET_WIDTH, widgetHeight());
}

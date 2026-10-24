import { EventsOn } from '$lib/wailsjs/runtime/runtime';

export type DriveAlert = {
	drive: string;
	kind: 'low' | 'dropping' | 'reminder' | 'test';
	title: string;
	body: string;
	free_bytes: number;
	at: number;
};

// a drive stays marked in the widget this long after its last alert
const MARK_FOR_MS = 15 * 60 * 1000;

// Alerts arrive from the drive watcher in Go. They are also shown as Windows
// notifications, but kept here too because some PCs block notifications.
export const driveAlerts = $state<{
	// the alert shown in the banner, until dismissed
	current: DriveAlert | null;
	// latest alert per drive, for marking drives in the widget
	byDrive: Record<string, DriveAlert>;
}>({ current: null, byDrive: {} });

let listening = false;

export function listenForAlerts() {
	if (listening) return;
	listening = true;
	EventsOn('drive-alert', (alert: DriveAlert) => {
		driveAlerts.current = alert;
		if (alert.kind === 'low' || alert.kind === 'dropping') driveAlerts.byDrive[alert.drive] = alert;
	});
}

export function dismissAlert() {
	driveAlerts.current = null;
}

// whether a drive had an alert recently enough to stay marked
export function isDriveAlerted(drive: string, now = Date.now()) {
	const alert = driveAlerts.byDrive[drive];
	return !!alert && now - alert.at < MARK_FOR_MS;
}

import { CheckInternetConnection } from './wailsjs/go/main/App';

export async function checkInternetConnection() {
	const result = await CheckInternetConnection();
	if (result) {
		return true;
	} else {
		return false;
	}
}

export function mbToGb(mb: number) {
	if (!mb) return '0.00';
	return (mb / 1024).toFixed(2);
}

// Bar colour for a "how full is it" percentage (memory, storage)
export function usageBarColor(percent: number): string {
	if (percent < 80) return 'bg-accent';
	if (percent < 90) return 'bg-amber-500';
	return 'bg-red-500';
}

// "12th Gen Intel(R) Core(TM) i7-1260P CPU @ 2.10GHz" -> "Intel Core i7-1260P"
export function shortCpuName(model: string): string {
	if (!model) return '';
	return model
		.replace(/\((R|TM)\)/gi, '')
		.replace(/\b\d+(st|nd|rd|th) Gen\b/i, '')
		.replace(/\b(CPU|Processor)\b/gi, '')
		.replace(/@.*$/, '')
		.replace(/\s+/g, ' ')
		.trim();
}

export function formatBytes(bytes: number): string {
	if (!bytes || bytes < 0) return '0 B';
	const units = ['B', 'KB', 'MB', 'GB', 'TB'];
	const i = Math.min(units.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)));
	const value = bytes / Math.pow(1024, i);
	return `${value >= 100 || i === 0 ? value.toFixed(0) : value.toFixed(1)} ${units[i]}`;
}

export function bytesToGb(kb: number) {
	if (!kb) return '0.00';
	return (kb / 1024 / 1024 / 1024).toFixed(2);
}

export async function copyStat(systemInfo: any): Promise<string> {
	const stats = JSON.stringify(systemInfo, null, 2);

	try {
		await navigator.clipboard.writeText(stats);
		return 'Stats copied to clipboard!';
	} catch (err) {
		return `Failed to copy: ${err}`;
	}
}

export function getSystemHealthScore(metrics: any): number {
	if (!metrics) return 0;
	const {
		battery,
		disk,
		ram
	} = metrics;

	let totalScore = 0;
	let factorsCount = 0;

	if (ram && ram.total > 0) {
		const ramHealth = (ram.available / ram.total) * 100;
		totalScore += ramHealth;
		factorsCount++;
	}

	if (disk && disk.total > 0) {
		const diskHealth = (disk.free / disk.total) * 100;
		totalScore += diskHealth;
		factorsCount++;
	}

	if (battery && battery.designedCapacity > 0 && battery.currentCapacity > 0) {
		const batteryHealth = Math.min(100, (battery.currentCapacity / battery.designedCapacity) * 100);
		totalScore += batteryHealth;
		factorsCount++;
	}

	if (factorsCount === 0) return 0;
	return Math.round((totalScore / factorsCount) * 100) / 100;
}

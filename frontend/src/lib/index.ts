import { CheckInternetConnection, GetPrivateTerms } from './wailsjs/go/main/App';
import { makeRedactor } from './redact';

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
	// a normal amount is not a status, so it stays neutral; colour only past a threshold
	if (percent < 80) return 'bg-ink/35';
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

// "charging" but not "discharging" / "not charging"
export function isBatteryCharging(state: string | undefined): boolean {
	const s = (state || '').toLowerCase();
	return s.includes('charging') && !s.includes('dis') && !s.includes('not');
}

export function formatBytes(bytes: number): string {
	if (!bytes || bytes < 0) return '0 B';
	const units = ['B', 'KB', 'MB', 'GB', 'TB'];
	let i = Math.min(units.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)));
	let value = bytes / Math.pow(1024, i);
	// 1000 to 1023 of a unit reads better as the next one up ("1.0 GB", not "1001 MB")
	if (value >= 999.5 && i < units.length - 1) {
		i++;
		value /= 1024;
	}
	return `${value >= 100 || i === 0 ? value.toFixed(0) : value.toFixed(1)} ${units[i]}`;
}

// Copies the system details as text, with the PC's name, the user name and
// addresses hidden, since copied stats usually end up pasted somewhere public
export async function copyStat(systemInfo: any): Promise<string> {
	const terms = ((await GetPrivateTerms().catch(() => [])) ?? []) as string[];
	const stats = makeRedactor(terms)(JSON.stringify(systemInfo, null, 2));

	try {
		await navigator.clipboard.writeText(stats);
		return 'Stats copied to clipboard (personal details hidden)';
	} catch (err) {
		return `Failed to copy: ${err}`;
	}
}

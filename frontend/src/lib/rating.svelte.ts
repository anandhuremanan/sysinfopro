import { IsStoreApp } from '$lib/wailsjs/go/main/App';
import { BrowserOpenURL } from '$lib/wailsjs/runtime/runtime';

// Ask for a Store rating once, after SysInfo Pro has freed at least this much in
// total. Only the Store version asks; either answer ends it for good.
const ASK_AFTER_BYTES = 1024 ** 3;
const STORAGE_KEY = 'sysinfopro-rating';
const REVIEW_URL = 'ms-windows-store://review/?ProductId=9PD2G09HCXK3';

type Saved = { freed: number; asked: boolean };

export const rating = $state({ show: false, freedBytes: 0 });

function load(): Saved {
	try {
		const saved = JSON.parse(localStorage.getItem(STORAGE_KEY) || 'null');
		if (saved && typeof saved.freed === 'number') return saved;
	} catch {
		// unreadable or blocked storage: start fresh
	}
	return { freed: 0, asked: false };
}

function save(saved: Saved) {
	try {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(saved));
	} catch {
		// the prompt may then appear again later, which is harmless
	}
}

// Call after a cleanup with the bytes it freed
export async function recordFreed(bytes: number) {
	if (!(bytes > 0)) return;
	const saved = load();
	saved.freed += bytes;
	save(saved);
	if (saved.asked || saved.freed < ASK_AFTER_BYTES) return;
	if (!(await IsStoreApp().catch(() => false))) return;
	rating.freedBytes = saved.freed;
	rating.show = true;
}

export function answerRating(rate: boolean) {
	const saved = load();
	saved.asked = true;
	save(saved);
	rating.show = false;
	if (rate) BrowserOpenURL(REVIEW_URL);
}

// Results of slow queries, kept while the app is open, so reopening a screen shows
// the last result at once instead of waiting again.

type Entry = { value?: unknown; at: number; pending?: Promise<unknown> };

const entries = new Map<string, Entry>();

// the last stored value, or undefined
export function recall<T>(key: string): T | undefined {
	return entries.get(key)?.value as T | undefined;
}

export function remember<T>(key: string, value: T) {
	entries.set(key, { value, at: Date.now() });
}

// Returns the stored value if it is younger than maxAgeMs, otherwise fetches a new
// one. Callers asking at the same time share one fetch. force always fetches.
export async function load<T>(
	key: string,
	fetch: () => Promise<T>,
	maxAgeMs = Infinity,
	force = false
): Promise<T> {
	const entry = entries.get(key);
	if (entry?.pending) return entry.pending as Promise<T>;
	if (!force && entry && entry.value !== undefined && Date.now() - entry.at < maxAgeMs) {
		return entry.value as T;
	}
	const pending = fetch().then(
		(value) => {
			entries.set(key, { value, at: Date.now() });
			return value;
		},
		(err) => {
			// keep the old value, if any, and let the caller see the error
			if (entry?.value !== undefined) entries.set(key, { value: entry.value, at: entry.at });
			else entries.delete(key);
			throw err;
		}
	);
	entries.set(key, { value: entry?.value, at: entry?.at ?? 0, pending });
	return pending;
}

// fetch in the background ahead of time; failures are ignored
export function prefetch<T>(key: string, fetch: () => Promise<T>) {
	load(key, fetch).catch(() => {});
}

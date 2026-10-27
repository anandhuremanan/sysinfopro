import { describe, expect, it, vi } from 'vitest';
import { load, recall, remember } from './cache';

describe('cache', () => {
	it('shares one request between callers asking at the same time', async () => {
		const fetch = vi.fn(async () => {
			await new Promise((r) => setTimeout(r, 10));
			return 42;
		});
		const [a, b] = await Promise.all([load('shared', fetch), load('shared', fetch)]);
		expect(a).toBe(42);
		expect(b).toBe(42);
		expect(fetch).toHaveBeenCalledTimes(1);
	});

	it('reuses a fresh value and fetches again once it is too old', async () => {
		const fetch = vi.fn(async () => Date.now());
		await load('aged', fetch, 60_000);
		await load('aged', fetch, 60_000);
		expect(fetch).toHaveBeenCalledTimes(1);
		await load('aged', fetch, 0);
		expect(fetch).toHaveBeenCalledTimes(2);
	});

	it('force always fetches', async () => {
		const fetch = vi.fn(async () => 1);
		await load('forced', fetch);
		await load('forced', fetch, Infinity, true);
		expect(fetch).toHaveBeenCalledTimes(2);
	});

	it('keeps the last good value when a refresh fails', async () => {
		remember('kept', 'old');
		await expect(load('kept', async () => Promise.reject(new Error('offline')), 0)).rejects.toThrow(
			'offline'
		);
		expect(recall('kept')).toBe('old');
	});
});

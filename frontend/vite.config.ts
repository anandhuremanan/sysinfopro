import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { readFileSync } from 'node:fs';
import { defineConfig } from 'vitest/config';

// the app's version lives in one place, wails.json, and is built into the frontend
const wails = JSON.parse(readFileSync(new URL('../wails.json', import.meta.url), 'utf8'));
const appVersion: string = wails?.info?.productVersion ?? '0.0.0';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	define: {
		__APP_VERSION__: JSON.stringify(appVersion)
	},
	test: {
		include: ['src/**/*.test.ts'],
		environment: 'node'
	}
});

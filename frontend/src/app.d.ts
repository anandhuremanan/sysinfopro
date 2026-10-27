// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
declare global {
	// the version from wails.json, built in by vite.config.ts
	const __APP_VERSION__: string;
	interface Window {
		go: any;
		runtime: any;
		[key: string]: any;
	}
	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}
}

export {};

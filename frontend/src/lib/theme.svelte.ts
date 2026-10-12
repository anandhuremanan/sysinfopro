export type ThemeMode = 'system' | 'light' | 'dark';

const STORAGE_KEY = 'sysinfopro-theme';

export const theme = $state<{ mode: ThemeMode }>({ mode: 'system' });

function readSavedMode(): ThemeMode {
	try {
		const saved = localStorage.getItem(STORAGE_KEY);
		if (saved === 'light' || saved === 'dark' || saved === 'system') return saved;
	} catch {
		// storage unavailable, fall back to system
	}
	return 'system';
}

function applyTheme(mode: ThemeMode) {
	const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
	const dark = mode === 'dark' || (mode === 'system' && prefersDark);
	document.documentElement.classList.toggle('dark', dark);
}

export function setTheme(mode: ThemeMode) {
	theme.mode = mode;
	try {
		localStorage.setItem(STORAGE_KEY, mode);
	} catch {
		// storage unavailable, theme still applies for this session
	}
	applyTheme(mode);
}

// Call once on startup; returns a cleanup for the OS theme listener
export function initTheme() {
	theme.mode = readSavedMode();
	applyTheme(theme.mode);

	const media = window.matchMedia('(prefers-color-scheme: dark)');
	const onChange = () => {
		if (theme.mode === 'system') applyTheme('system');
	};
	media.addEventListener('change', onChange);
	return () => media.removeEventListener('change', onChange);
}

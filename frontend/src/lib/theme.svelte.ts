export type ThemeMode = 'system' | 'light' | 'dark' | 'amoled';

const STORAGE_KEY = 'sysinfopro-theme';
const ACCENT_KEY = 'sysinfopro-accent';
// accent colours already worked out for light and dark backgrounds, read by the
// pre-paint script in app.html so the first frame has the right colours
const ACCENT_VARS_KEY = 'sysinfopro-accent-vars';

export type AccentPreset = { id: string; label: string; color: string };

// '' is the built-in teal. The rest are bright on purpose: they suit a dark or
// AMOLED theme and a gaming setup, and are toned down automatically in light mode.
export const ACCENT_PRESETS: AccentPreset[] = [
	{ id: '', label: 'Teal', color: '#2dd4bf' },
	{ id: '#39ff14', label: 'Neon green', color: '#39ff14' },
	{ id: '#00e5ff', label: 'Cyan', color: '#00e5ff' },
	{ id: '#a855f7', label: 'Purple', color: '#a855f7' },
	{ id: '#ff2bd6', label: 'Magenta', color: '#ff2bd6' },
	{ id: '#ff3b3b', label: 'Red', color: '#ff3b3b' },
	{ id: '#ff8a00', label: 'Orange', color: '#ff8a00' },
	{ id: '#ffd400', label: 'Gold', color: '#ffd400' }
];

export const theme = $state<{ mode: ThemeMode; accent: string }>({ mode: 'system', accent: '' });

// the backgrounds accents must stay readable on
const TILE = { light: '#ffffff', dark: '#0f172a' };

function readSaved() {
	let mode: ThemeMode = 'system';
	let accent = '';
	try {
		const saved = localStorage.getItem(STORAGE_KEY);
		if (saved === 'light' || saved === 'dark' || saved === 'system' || saved === 'amoled') mode = saved;
		const savedAccent = localStorage.getItem(ACCENT_KEY) || '';
		if (/^#[0-9a-f]{6}$/i.test(savedAccent)) accent = savedAccent.toLowerCase();
	} catch {
		// storage unavailable: defaults
	}
	return { mode, accent };
}

// --- colour maths ---

function toRgb(hex: string): [number, number, number] {
	const n = parseInt(hex.slice(1), 16);
	return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
}

function toHex([r, g, b]: number[]) {
	return '#' + [r, g, b].map((v) => Math.round(Math.min(255, Math.max(0, v))).toString(16).padStart(2, '0')).join('');
}

function luminance(hex: string) {
	const [r, g, b] = toRgb(hex).map((v) => {
		const c = v / 255;
		return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
	});
	return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

export function contrast(a: string, b: string) {
	const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
	return (hi + 0.05) / (lo + 0.05);
}

function mix(hex: string, toward: string, amount: number) {
	const a = toRgb(hex);
	const b = toRgb(toward);
	return toHex(a.map((v, i) => v + (b[i] - v) * amount));
}

// Shift the colour toward black or white until it reads well on the background
// (4.5:1, the WCAG level for normal text), keeping its hue
function readableOn(color: string, background: string) {
	const toward = luminance(background) > 0.5 ? '#000000' : '#ffffff';
	let result = color;
	for (let step = 1; step <= 20 && contrast(result, background) < 4.5; step++) {
		result = mix(color, toward, step * 0.05);
	}
	return result;
}

// black or white text on the accent, whichever is clearer
function textOn(color: string) {
	return contrast(color, '#000000') >= contrast(color, '#ffffff') ? '#000000' : '#ffffff';
}

function accentVars(accent: string) {
	if (!accent) return null;
	const light = readableOn(accent, TILE.light);
	const dark = readableOn(accent, TILE.dark);
	return {
		light: { accent: light, on: textOn(light) },
		dark: { accent: dark, on: textOn(dark) }
	};
}

// --- applying ---

function applyTheme(mode: ThemeMode, accent: string) {
	const root = document.documentElement;
	const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
	const dark = mode === 'dark' || mode === 'amoled' || (mode === 'system' && prefersDark);
	root.classList.toggle('dark', dark);
	root.classList.toggle('amoled', mode === 'amoled');

	const vars = accentVars(accent);
	if (vars) {
		const v = dark ? vars.dark : vars.light;
		root.style.setProperty('--accent', v.accent);
		root.style.setProperty('--on-accent', v.on);
	} else {
		root.style.removeProperty('--accent');
		root.style.removeProperty('--on-accent');
	}
}

function save() {
	try {
		localStorage.setItem(STORAGE_KEY, theme.mode);
		localStorage.setItem(ACCENT_KEY, theme.accent);
		const vars = accentVars(theme.accent);
		if (vars) localStorage.setItem(ACCENT_VARS_KEY, JSON.stringify(vars));
		else localStorage.removeItem(ACCENT_VARS_KEY);
	} catch {
		// storage unavailable, the theme still applies for this session
	}
}

export function setTheme(mode: ThemeMode) {
	theme.mode = mode;
	save();
	applyTheme(theme.mode, theme.accent);
}

// accent: a #rrggbb colour, or '' for the built-in teal
export function setAccent(accent: string) {
	theme.accent = /^#[0-9a-f]{6}$/i.test(accent) ? accent.toLowerCase() : '';
	save();
	applyTheme(theme.mode, theme.accent);
}

// Call once on startup; returns a cleanup for the OS theme listener
export function initTheme() {
	const saved = readSaved();
	theme.mode = saved.mode;
	theme.accent = saved.accent;
	applyTheme(theme.mode, theme.accent);

	const media = window.matchMedia('(prefers-color-scheme: dark)');
	const onChange = () => {
		if (theme.mode === 'system') applyTheme('system', theme.accent);
	};
	media.addEventListener('change', onChange);
	return () => media.removeEventListener('change', onChange);
}

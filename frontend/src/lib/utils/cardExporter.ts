import { formatBytes, shortCpuName } from '$lib';
import { load, recall } from '$lib/cache';
import { GetHardwareSpecs, SaveSpecCardPNG } from '$lib/wailsjs/go/main/App';

// A shareable "rig card": the hardware worth showing off, drawn like the bento
// tiles in the app. Drawn at 2x so the PNG stays sharp when shared.
const WIDTH = 800;
const HEIGHT = 450;
const SCALE = 2;
const FONT = '"Segoe UI Variable Display", "Segoe UI", system-ui, sans-serif';

type Palette = {
	ground: string;
	tile: string;
	ink: string;
	muted: string;
	track: string;
	accent: string;
	onAccent: string;
};

// The card follows the theme and accent colour the app is showing
function readPalette(): Palette {
	const css = getComputedStyle(document.documentElement);
	const token = (name: string, fallback: string) => css.getPropertyValue(name).trim() || fallback;
	return {
		ground: token('--ground', '#eceef1'),
		tile: token('--tile', '#ffffff'),
		ink: token('--ink', '#15171a'),
		muted: token('--muted', '#5e6570'),
		track: token('--track', '#e6e9ed'),
		accent: token('--accent', '#0f766e'),
		onAccent: token('--on-accent', '#ffffff')
	};
}

function roundRect(ctx: CanvasRenderingContext2D, x: number, y: number, w: number, h: number, r: number) {
	ctx.beginPath();
	ctx.moveTo(x + r, y);
	ctx.arcTo(x + w, y, x + w, y + h, r);
	ctx.arcTo(x + w, y + h, x, y + h, r);
	ctx.arcTo(x, y + h, x, y, r);
	ctx.arcTo(x, y, x + w, y, r);
	ctx.closePath();
}

// Shortens text with an ellipsis until it fits maxWidth in the current font
function fitText(ctx: CanvasRenderingContext2D, text: string, maxWidth: number) {
	if (ctx.measureText(text).width <= maxWidth) return text;
	let cut = text;
	while (cut.length > 1 && ctx.measureText(cut + '…').width > maxWidth) cut = cut.slice(0, -1);
	return cut.trimEnd() + '…';
}

// The largest font size (down to min) at which text fits; long names shrink before they get cut
function fitFont(ctx: CanvasRenderingContext2D, text: string, maxWidth: number, weight: number, size: number, min: number) {
	for (let s = size; s >= min; s--) {
		ctx.font = `${weight} ${s}px ${FONT}`;
		if (ctx.measureText(text).width <= maxWidth) return text;
	}
	return fitText(ctx, text, maxWidth);
}

function withAlpha(ctx: CanvasRenderingContext2D, alpha: number, draw: () => void) {
	ctx.save();
	ctx.globalAlpha = alpha;
	draw();
	ctx.restore();
}

// ---------- the facts ----------

type Rig = {
	name: string;
	cpu: string;
	cpuDetail: string;
	cores: number;
	gpu: string;
	gpuDetail: string;
	// a separate graphics card, as opposed to graphics built into the processor
	gpuDiscrete: boolean;
	memory: string;
	memoryDetail: string;
	modules: number;
	storage: string;
	storageDetail: string;
	display: string;
	displayDetail: string;
	os: string;
};

const DISCRETE = /nvidia|geforce|quadro|rtx|gtx|radeon (rx|pro)|arc a\d/i;

// a discrete card beats integrated graphics, and anything beats the basic display driver
function pickGPU(gpus: any[]) {
	const real = (gpus || []).filter((g) => g?.name && !/basic display|remote/i.test(g.name));
	const discrete = real.filter((g) => DISCRETE.test(g.name));
	return [...(discrete.length ? discrete : real)].sort((a, b) => (b.vram_bytes || 0) - (a.vram_bytes || 0))[0];
}

function rigName(specs: any): string {
	const maker = String(specs?.system?.manufacturer || '')
		.replace(/\b(inc|corp(oration)?|co|ltd|limited|technology)\.?\b/gi, '')
		.replace(/[.,\s]+$/, '')
		.trim();
	const model = String(specs?.system?.model || '').trim();
	const placeholder = /to be filled|system product name|default string|o\.?e\.?m/i;
	if (model && !placeholder.test(model)) {
		return maker && !placeholder.test(maker) && !model.toLowerCase().startsWith(maker.toLowerCase())
			? `${maker} ${model}`
			: model;
	}
	const board = String(specs?.board?.product || '').trim();
	return board && !placeholder.test(board) ? board : 'My PC';
}

const trimSize = (bytes: number) => formatBytes(bytes).replace(/\.0 /, ' ');

function describeRig(systemInfo: any, specs: any): Rig {
	const cpu = specs?.cpus?.[0];
	const cores = Number(cpu?.cores || systemInfo.cpu_cores) || 0;
	const threads = Number(cpu?.threads) || 0;

	const gpu = pickGPU(specs?.gpus);
	const modules: any[] = (specs?.memory || []).filter((m: any) => m?.size_bytes > 0);
	const memBytes = modules.reduce((sum, m) => sum + m.size_bytes, 0) || (systemInfo.total_ram_mb || 0) * 1024 * 1024;
	const speed = Math.max(0, ...modules.map((m) => Number(m.speed_mhz) || 0));
	const sameSize = new Set(modules.map((m) => m.size_bytes)).size === 1;

	const disks: any[] = (specs?.disks || []).filter((d: any) => d?.size_bytes > 0);
	const diskBytes = disks.reduce((sum, d) => sum + d.size_bytes, 0) || systemInfo.total_disk_bytes || 0;
	const diskKind = (d: any) => {
		const text = `${d?.interface || ''} ${d?.media || ''} ${d?.model || ''}`;
		if (/nvme/i.test(text)) return 'NVMe SSD';
		if (/ssd|solid/i.test(text)) return 'SSD';
		if (/hdd|hard|unspecified/i.test(text) && !systemInfo.disk_type) return 'HDD';
		return systemInfo.disk_type || 'Drive';
	};

	const osName = String(specs?.os?.name || systemInfo.host_platform || 'Windows').replace(/^Microsoft /, '');
	const osVersion = systemInfo.host_platform_version || specs?.os?.version || '';
	return {
		name: rigName(specs),
		// AMD names end in "8-Core Processor"; the core count is shown underneath anyway
		cpu:
			shortCpuName(cpu?.name || systemInfo.cpu_model)
				.replace(/\s+\d+-Core\b/i, '')
				.replace(/\s+(Processor|CPU)\s*$/i, '')
				.trim() || 'Processor',
		cpuDetail:
			[cores ? `${cores} cores` : '', threads ? `${threads} threads` : ''].filter(Boolean).join(' · ') ||
			systemInfo.cpu_vendor ||
			'',
		cores,
		gpu: String(gpu?.name || 'Graphics').replace(/\((R|TM)\)/gi, '').replace(/\s+/g, ' ').trim(),
		gpuDiscrete: DISCRETE.test(gpu?.name || ''),
		// built-in graphics borrow system memory rather than having their own
		gpuDetail: DISCRETE.test(gpu?.name || '')
			? gpu?.vram_bytes
				? `${trimSize(gpu.vram_bytes)} video memory`
				: 'Graphics card'
			: gpu?.vram_bytes
				? `Built-in · ${trimSize(gpu.vram_bytes)} shared memory`
				: 'Built-in graphics',
		memory: trimSize(memBytes),
		memoryDetail:
			[
				modules.length ? (sameSize ? `${modules.length} × ${trimSize(modules[0].size_bytes)}` : `${modules.length} modules`) : '',
				speed ? `${speed} MHz` : ''
			]
				.filter(Boolean)
				.join(' · ') || 'Installed',
		modules: Math.max(1, Math.min(4, modules.length || 2)),
		storage: trimSize(diskBytes),
		storageDetail: disks.length > 1 ? `${disks.length} drives · ${diskKind(disks[0])}` : diskKind(disks[0]),
		display: gpu?.width && gpu?.height ? `${gpu.width} × ${gpu.height}` : 'Display',
		displayDetail: gpu?.refresh_hz ? `${gpu.refresh_hz} Hz` : '',
		os: osVersion ? `${osName} · ${osVersion}` : osName
	};
}

// ---------- drawings ----------

function background(ctx: CanvasRenderingContext2D, p: Palette) {
	ctx.fillStyle = p.ground;
	ctx.fillRect(0, 0, WIDTH, HEIGHT);
	for (const [x, y, radius, alpha] of [
		[WIDTH - 40, -60, 460, 0.28],
		[-60, HEIGHT + 40, 380, 0.14]
	]) {
		const glow = ctx.createRadialGradient(x, y, 0, x, y, radius);
		glow.addColorStop(0, p.accent);
		glow.addColorStop(1, 'transparent');
		withAlpha(ctx, alpha, () => {
			ctx.fillStyle = glow;
			ctx.fillRect(0, 0, WIDTH, HEIGHT);
		});
	}
	withAlpha(ctx, 0.06, () => {
		ctx.fillStyle = p.ink;
		for (let x = 14; x < WIDTH; x += 18) for (let y = 14; y < HEIGHT; y += 18) ctx.fillRect(x, y, 1.2, 1.2);
	});
}

function tile(ctx: CanvasRenderingContext2D, p: Palette, x: number, y: number, w: number, h: number) {
	ctx.fillStyle = p.tile;
	roundRect(ctx, x, y, w, h, 16);
	ctx.fill();
	withAlpha(ctx, 0.08, () => {
		ctx.strokeStyle = p.ink;
		ctx.lineWidth = 1;
		roundRect(ctx, x + 0.5, y + 0.5, w - 1, h - 1, 16);
		ctx.stroke();
	});
}

// label in accent capitals, value large, detail muted; textWidth keeps clear of the drawing
function tileText(
	ctx: CanvasRenderingContext2D,
	p: Palette,
	x: number,
	y: number,
	h: number,
	label: string,
	value: string,
	detail: string,
	valueSize: number,
	textWidth: number
) {
	const pad = 20;
	ctx.fillStyle = p.accent;
	ctx.font = `700 11px ${FONT}`;
	ctx.fillText(label.toUpperCase(), x + pad, y + pad + 10);

	ctx.fillStyle = p.ink;
	const v = fitFont(ctx, value, textWidth, 700, valueSize, 14);
	ctx.fillText(v, x + pad, y + h - pad - 26);

	ctx.fillStyle = p.muted;
	ctx.font = `500 13px ${FONT}`;
	ctx.fillText(fitText(ctx, detail, textWidth), x + pad, y + h - pad);
}

// a processor with one square per core (up to 16), or with a label in place of the cores
function chip(ctx: CanvasRenderingContext2D, p: Palette, cx: number, cy: number, cores: number, label?: string) {
	const size = 92;
	const x = cx - size / 2;
	const y = cy - size / 2;
	ctx.save();
	ctx.strokeStyle = p.accent;
	ctx.lineCap = 'round';
	ctx.lineWidth = 2.5;
	for (let i = 0; i < 6; i++) {
		const t = x + 13 + i * 13.2;
		const u = y + 13 + i * 13.2;
		ctx.beginPath();
		ctx.moveTo(t, y - 10);
		ctx.lineTo(t, y - 3);
		ctx.moveTo(t, y + size + 3);
		ctx.lineTo(t, y + size + 10);
		ctx.moveTo(x - 10, u);
		ctx.lineTo(x - 3, u);
		ctx.moveTo(x + size + 3, u);
		ctx.lineTo(x + size + 10, u);
		ctx.stroke();
	}
	withAlpha(ctx, 0.14, () => {
		ctx.fillStyle = p.accent;
		roundRect(ctx, x, y, size, size, 14);
		ctx.fill();
	});
	ctx.lineWidth = 2;
	roundRect(ctx, x, y, size, size, 14);
	ctx.stroke();

	if (label) {
		ctx.fillStyle = p.accent;
		ctx.font = `800 22px ${FONT}`;
		ctx.textAlign = 'center';
		ctx.fillText(label, cx, cy + 8);
		ctx.textAlign = 'left';
		ctx.restore();
		return;
	}

	const shown = Math.max(1, Math.min(16, cores || 4));
	const cols = Math.ceil(Math.sqrt(shown));
	const rows = Math.ceil(shown / cols);
	const cell = 13;
	const gap = 4;
	const gw = cols * cell + (cols - 1) * gap;
	const gh = rows * cell + (rows - 1) * gap;
	ctx.fillStyle = p.accent;
	for (let i = 0; i < shown; i++) {
		roundRect(ctx, cx - gw / 2 + (i % cols) * (cell + gap), cy - gh / 2 + Math.floor(i / cols) * (cell + gap), cell, cell, 3.5);
		ctx.fill();
	}
	ctx.restore();
}

// a graphics card: board, two fans, bracket and edge connector
function graphicsCard(ctx: CanvasRenderingContext2D, p: Palette, cx: number, cy: number) {
	const w = 150;
	const h = 64;
	const x = cx - w / 2;
	const y = cy - h / 2;
	ctx.save();
	ctx.strokeStyle = p.accent;
	ctx.lineWidth = 2;
	ctx.lineCap = 'round';
	withAlpha(ctx, 0.14, () => {
		ctx.fillStyle = p.accent;
		roundRect(ctx, x, y, w, h, 12);
		ctx.fill();
	});
	roundRect(ctx, x, y, w, h, 12);
	ctx.stroke();
	ctx.beginPath();
	ctx.moveTo(x - 8, y - 6);
	ctx.lineTo(x - 8, y + h + 10);
	ctx.moveTo(x + 30, y + h);
	ctx.lineTo(x + 30, y + h + 7);
	ctx.lineTo(x + 100, y + h + 7);
	ctx.lineTo(x + 100, y + h);
	ctx.stroke();
	for (const fx of [x + 40, x + 108]) {
		ctx.beginPath();
		ctx.arc(fx, cy, 23, 0, Math.PI * 2);
		ctx.stroke();
		for (let b = 0; b < 7; b++) {
			const a = (b / 7) * Math.PI * 2;
			ctx.beginPath();
			ctx.moveTo(fx + Math.cos(a) * 6, cy + Math.sin(a) * 6);
			ctx.quadraticCurveTo(
				fx + Math.cos(a + 0.5) * 15,
				cy + Math.sin(a + 0.5) * 15,
				fx + Math.cos(a + 0.9) * 20,
				cy + Math.sin(a + 0.9) * 20
			);
			ctx.stroke();
		}
		ctx.fillStyle = p.accent;
		ctx.beginPath();
		ctx.arc(fx, cy, 5, 0, Math.PI * 2);
		ctx.fill();
	}
	ctx.restore();
}

// memory sticks, one per installed module (up to 4)
function memorySticks(ctx: CanvasRenderingContext2D, p: Palette, right: number, cy: number, count: number) {
	ctx.save();
	ctx.strokeStyle = p.accent;
	ctx.fillStyle = p.accent;
	ctx.lineWidth = 1.6;
	const w = 12;
	const h = 54;
	for (let i = 0; i < count; i++) {
		const x = right - (count - i) * (w + 7) + 7;
		const y = cy - h / 2;
		withAlpha(ctx, 0.16, () => {
			roundRect(ctx, x, y, w, h, 3);
			ctx.fill();
		});
		roundRect(ctx, x, y, w, h, 3);
		ctx.stroke();
		for (let c = 0; c < 4; c++) {
			roundRect(ctx, x + 3, y + 5 + c * 11.5, w - 6, 7.5, 1.5);
			ctx.fill();
		}
	}
	ctx.restore();
}

// an M.2 drive
function drive(ctx: CanvasRenderingContext2D, p: Palette, right: number, cy: number) {
	const w = 74;
	const h = 24;
	const x = right - w;
	const y = cy - h / 2;
	ctx.save();
	ctx.strokeStyle = p.accent;
	ctx.fillStyle = p.accent;
	ctx.lineWidth = 1.6;
	withAlpha(ctx, 0.16, () => {
		roundRect(ctx, x, y, w, h, 4);
		ctx.fill();
	});
	roundRect(ctx, x, y, w, h, 4);
	ctx.stroke();
	roundRect(ctx, x + 10, y + 5, 22, h - 10, 2);
	ctx.fill();
	roundRect(ctx, x + 38, y + 5, 16, h - 10, 2);
	ctx.fill();
	ctx.beginPath();
	ctx.arc(x + w - 8, cy, 3, 0, Math.PI * 2);
	ctx.stroke();
	ctx.restore();
}

// a monitor showing the refresh rate on screen
function monitor(ctx: CanvasRenderingContext2D, p: Palette, right: number, cy: number, hz: string) {
	const w = 66;
	const h = 40;
	const x = right - w;
	const y = cy - h / 2 - 6;
	ctx.save();
	ctx.strokeStyle = p.accent;
	ctx.lineWidth = 2;
	ctx.lineCap = 'round';
	withAlpha(ctx, 0.16, () => {
		ctx.fillStyle = p.accent;
		roundRect(ctx, x, y, w, h, 6);
		ctx.fill();
	});
	roundRect(ctx, x, y, w, h, 6);
	ctx.stroke();
	ctx.beginPath();
	ctx.moveTo(x + w / 2, y + h);
	ctx.lineTo(x + w / 2, y + h + 8);
	ctx.moveTo(x + w / 2 - 14, y + h + 9);
	ctx.lineTo(x + w / 2 + 14, y + h + 9);
	ctx.stroke();
	if (hz) {
		ctx.fillStyle = p.accent;
		ctx.font = `800 14px ${FONT}`;
		ctx.textAlign = 'center';
		ctx.fillText(hz.replace(' ', ''), x + w / 2, y + h / 2 + 5);
		ctx.textAlign = 'left';
	}
	ctx.restore();
}

// the Pulse mark, same geometry as Logo.svelte on a 24-unit grid
function logo(ctx: CanvasRenderingContext2D, p: Palette, x: number, y: number, size: number) {
	const unit = size / 24;
	ctx.fillStyle = p.accent;
	roundRect(ctx, x + unit, y + unit, 22 * unit, 22 * unit, 6 * unit);
	ctx.fill();
	ctx.strokeStyle = p.onAccent;
	ctx.lineWidth = 2 * unit;
	ctx.lineCap = 'round';
	ctx.lineJoin = 'round';
	ctx.beginPath();
	[
		[4.5, 12.5],
		[7.7, 12.5],
		[9.8, 7.5],
		[13.2, 17],
		[15.4, 12.5],
		[19.5, 12.5]
	].forEach(([px, py], i) => {
		if (i === 0) ctx.moveTo(x + px * unit, y + py * unit);
		else ctx.lineTo(x + px * unit, y + py * unit);
	});
	ctx.stroke();
}

// ---------- styles ----------

export type CardStyle = 'rig' | 'terminal' | 'neon' | 'minimal';

export const CARD_STYLES: { id: CardStyle; label: string; hint: string }[] = [
	{ id: 'rig', label: 'Rig', hint: 'Bento tiles with drawings of each part' },
	{ id: 'terminal', label: 'Terminal', hint: 'A neofetch-style printout' },
	{ id: 'neon', label: 'Neon', hint: 'Glowing outlines for RGB setups' },
	{ id: 'minimal', label: 'Minimal', hint: 'A clean list of the essentials' }
];

const STYLE_KEY = 'sysinfopro-card-style';

export function getCardStyle(): CardStyle {
	try {
		const saved = localStorage.getItem(STYLE_KEY);
		if (CARD_STYLES.some((s) => s.id === saved)) return saved as CardStyle;
	} catch {
		// storage unavailable: the default
	}
	return 'rig';
}

export function setCardStyle(style: CardStyle) {
	try {
		localStorage.setItem(STYLE_KEY, style);
	} catch {
		// the choice then lasts for this session only
	}
}

const MONO = '"Cascadia Mono", "Cascadia Code", Consolas, "Courier New", monospace';
const CREDIT = 'sysinfopro.imanandhu.in';

// Rig: the bento with a drawing for each part
function drawRig(ctx: CanvasRenderingContext2D, p: Palette, rig: Rig) {
	background(ctx, p);
	const m = 28;
	const gap = 12;

	ctx.fillStyle = p.muted;
	ctx.font = `700 11px ${FONT}`;
	ctx.fillText('MY SETUP', m, 40);
	ctx.fillStyle = p.ink;
	ctx.fillText(fitFont(ctx, rig.name, WIDTH - m * 2 - 170, 800, 30, 18), m, 72);

	logo(ctx, p, WIDTH - m - 128, 44, 24);
	ctx.fillStyle = p.ink;
	ctx.font = `700 15px ${FONT}`;
	ctx.fillText('SysInfo Pro', WIDTH - m - 98, 62);

	const heroY = 96;
	const heroH = 168;
	const heroW = (WIDTH - m * 2 - gap) / 2;
	const cpuX = m;
	const gpuX = m + heroW + gap;

	tile(ctx, p, cpuX, heroY, heroW, heroH);
	chip(ctx, p, cpuX + heroW - 78, heroY + heroH / 2, rig.cores);
	tileText(ctx, p, cpuX, heroY, heroH, 'Processor', rig.cpu, rig.cpuDetail, 26, heroW - 170);

	tile(ctx, p, gpuX, heroY, heroW, heroH);
	if (rig.gpuDiscrete) {
		graphicsCard(ctx, p, gpuX + heroW - 104, heroY + 60);
		tileText(ctx, p, gpuX, heroY, heroH, 'Graphics', rig.gpu, rig.gpuDetail, 24, heroW - 40);
	} else {
		chip(ctx, p, gpuX + heroW - 78, heroY + heroH / 2, 0, 'GPU');
		tileText(ctx, p, gpuX, heroY, heroH, 'Graphics', rig.gpu, rig.gpuDetail, 24, heroW - 170);
	}

	const rowY = heroY + heroH + gap;
	const rowH = 112;
	const cellW = (WIDTH - m * 2 - gap * 2) / 3;
	const cells = [m, m + cellW + gap, m + (cellW + gap) * 2];

	tile(ctx, p, cells[0], rowY, cellW, rowH);
	memorySticks(ctx, p, cells[0] + cellW - 18, rowY + 40, rig.modules);
	tileText(ctx, p, cells[0], rowY, rowH, 'Memory', rig.memory, rig.memoryDetail, 26, cellW - 40);

	tile(ctx, p, cells[1], rowY, cellW, rowH);
	drive(ctx, p, cells[1] + cellW - 18, rowY + 32);
	tileText(ctx, p, cells[1], rowY, rowH, 'Storage', rig.storage, rig.storageDetail, 26, cellW - 40);

	tile(ctx, p, cells[2], rowY, cellW, rowH);
	monitor(ctx, p, cells[2] + cellW - 18, rowY + 38, rig.displayDetail);
	tileText(ctx, p, cells[2], rowY, rowH, 'Display', rig.display, rig.displayDetail ? `${rig.displayDetail} refresh rate` : 'Main display', 24, cellW - 40);

	ctx.fillStyle = p.muted;
	ctx.font = `500 12px ${FONT}`;
	ctx.fillText(rig.os, m, HEIGHT - 24);
	ctx.fillText(CREDIT, WIDTH - m - ctx.measureText(CREDIT).width, HEIGHT - 24);
}

// The logo drawn small, then read back cell by cell: '█' where the tile is, ' ' for the
// background and the pulse line.
function logoAscii(cols = 22, rows = 11): string[] {
	const size = 120;
	const canvas = document.createElement('canvas');
	canvas.width = size;
	canvas.height = size;
	const ctx = canvas.getContext('2d');
	if (!ctx) return [];
	const unit = size / 24;
	ctx.fillStyle = '#000';
	roundRect(ctx, unit, unit, 22 * unit, 22 * unit, 6 * unit);
	ctx.fill();
	ctx.globalCompositeOperation = 'destination-out';
	ctx.lineWidth = 1.5 * unit;
	ctx.lineCap = 'round';
	ctx.lineJoin = 'round';
	ctx.beginPath();
	[[4.5, 12.5], [7.7, 12.5], [9.8, 7.5], [13.2, 17], [15.4, 12.5], [19.5, 12.5]].forEach(([x, y], i) => {
		if (i === 0) ctx.moveTo(x * unit, y * unit);
		else ctx.lineTo(x * unit, y * unit);
	});
	ctx.stroke();
	const pixels = ctx.getImageData(0, 0, size, size).data;
	const lines: string[] = [];
	for (let r = 0; r < rows; r++) {
		let line = '';
		for (let col = 0; col < cols; col++) {
			const x = Math.floor(((col + 0.5) / cols) * size);
			const y = Math.floor(((r + 0.5) / rows) * size);
			line += pixels[(y * size + x) * 4 + 3] > 128 ? '█' : ' ';
		}
		lines.push(line);
	}
	return lines;
}

// Cards that are always dark use the bright form of the accent: in light mode the
// app darkens the accent so text stays readable on white, which would dull a glow
function brightAccent(p: Palette): string {
	try {
		const vars = JSON.parse(localStorage.getItem('sysinfopro-accent-vars') || 'null');
		if (vars?.dark?.accent) return vars.dark.accent;
	} catch {
		// storage unavailable
	}
	return document.documentElement.classList.contains('dark') ? p.accent : '#2dd4bf';
}

// Terminal: the way people post their specs with neofetch; always a dark console
function drawTerminal(ctx: CanvasRenderingContext2D, p: Palette, rig: Rig) {
	const accent = brightAccent(p);
	const bg = '#0c0c0e';
	const text = '#e7e7ea';
	const dim = '#8a8f98';
	ctx.fillStyle = bg;
	ctx.fillRect(0, 0, WIDTH, HEIGHT);

	// window chrome
	ctx.fillStyle = '#17171b';
	ctx.fillRect(0, 0, WIDTH, 34);
	['#ff5f57', '#febc2e', '#28c840'].forEach((c, i) => {
		ctx.fillStyle = c;
		ctx.beginPath();
		ctx.arc(22 + i * 20, 17, 6, 0, Math.PI * 2);
		ctx.fill();
	});
	ctx.fillStyle = dim;
	ctx.font = `500 12px ${MONO}`;
	ctx.textAlign = 'center';
	ctx.fillText('sysinfo — specs', WIDTH / 2, 21);
	ctx.textAlign = 'left';

	// the Pulse mark as pixel blocks, sampled from the real logo so it always matches
	ctx.fillStyle = accent;
	const cell = 10;
	logoAscii(24, 24).forEach((line, r) => {
		[...line].forEach((ch, col) => {
			if (ch === '█') ctx.fillRect(40 + col * cell, 84 + r * cell, cell - 1.5, cell - 1.5);
		});
	});

	// the specs, as key: value lines
	const x = 300;
	let y = 84;
	ctx.font = `700 17px ${MONO}`;
	ctx.fillStyle = accent;
	ctx.fillText(fitText(ctx, `you@${rig.name.toLowerCase().replace(/\s+/g, '-')}`, WIDTH - x - 30), x, y);
	y += 12;
	ctx.fillStyle = dim;
	ctx.fillText('─'.repeat(24), x, y + 6);
	y += 30;

	const rows: [string, string][] = [
		['OS', rig.os],
		['Host', rig.name],
		['CPU', rig.cores ? `${rig.cpu} (${rig.cores})` : rig.cpu],
		['GPU', rig.gpu],
		['Memory', rig.memoryDetail === 'Installed' ? rig.memory : `${rig.memory} · ${rig.memoryDetail}`],
		['Disk', `${rig.storage} · ${rig.storageDetail}`],
		['Display', rig.displayDetail ? `${rig.display} @ ${rig.displayDetail}` : rig.display]
	];
	ctx.font = `500 15px ${MONO}`;
	for (const [key, value] of rows) {
		ctx.fillStyle = accent;
		ctx.fillText(`${key}:`, x, y);
		ctx.fillStyle = text;
		ctx.fillText(fitText(ctx, value, WIDTH - x - 120), x + 90, y);
		y += 27;
	}

	// the colour blocks neofetch ends with
	y += 4;
	['#1e1e24', '#ff5f57', '#28c840', '#febc2e', '#3b82f6', '#a855f7', '#22d3ee', '#e7e7ea'].forEach((c, i) => {
		ctx.fillStyle = c;
		ctx.fillRect(x + i * 30, y, 30, 18);
	});

	ctx.fillStyle = dim;
	ctx.font = `500 12px ${MONO}`;
	ctx.fillText(`$ made with SysInfo Pro · ${CREDIT}`, 36, HEIGHT - 22);
}

// Neon: dark, with glowing outlines and a synthwave grid on the floor
function drawNeon(ctx: CanvasRenderingContext2D, base: Palette, rig: Rig) {
	const p = { ...base, accent: brightAccent(base), onAccent: '#000000' };
	const bg = '#05050a';
	ctx.fillStyle = bg;
	ctx.fillRect(0, 0, WIDTH, HEIGHT);

	// floor grid, fading toward the horizon
	const horizon = 300;
	ctx.save();
	ctx.strokeStyle = p.accent;
	ctx.lineWidth = 1;
	for (let i = 0; i <= 10; i++) {
		const t = i / 10;
		const yy = horizon + (HEIGHT - horizon) * t * t;
		ctx.globalAlpha = 0.08 + 0.3 * t;
		ctx.beginPath();
		ctx.moveTo(0, yy);
		ctx.lineTo(WIDTH, yy);
		ctx.stroke();
	}
	for (let i = -12; i <= 12; i++) {
		ctx.globalAlpha = 0.18;
		ctx.beginPath();
		ctx.moveTo(WIDTH / 2 + i * 14, horizon);
		ctx.lineTo(WIDTH / 2 + i * 110, HEIGHT);
		ctx.stroke();
	}
	ctx.restore();
	const fade = ctx.createLinearGradient(0, horizon - 20, 0, horizon + 60);
	fade.addColorStop(0, bg);
	fade.addColorStop(1, 'transparent');
	ctx.fillStyle = fade;
	ctx.fillRect(0, horizon - 20, WIDTH, 80);

	const glow = (blur: number, draw: () => void) => {
		ctx.save();
		ctx.shadowColor = p.accent;
		ctx.shadowBlur = blur;
		draw();
		ctx.restore();
	};

	// title
	glow(18, () => {
		ctx.fillStyle = '#ffffff';
		ctx.font = `800 30px ${FONT}`;
		ctx.fillText(fitFont(ctx, rig.name.toUpperCase(), WIDTH - 220, 800, 30, 18), 32, 66);
	});
	logo(ctx, p, WIDTH - 158, 42, 24);
	ctx.fillStyle = '#ffffff';
	ctx.font = `700 15px ${FONT}`;
	ctx.fillText('SysInfo Pro', WIDTH - 128, 60);

	// panels: outlines that glow
	const panel = (x: number, y: number, w: number, h: number, label: string, value: string, detail: string, size: number) => {
		glow(16, () => {
			ctx.strokeStyle = p.accent;
			ctx.lineWidth = 1.6;
			roundRect(ctx, x, y, w, h, 14);
			ctx.stroke();
		});
		withAlpha(ctx, 0.07, () => {
			ctx.fillStyle = p.accent;
			roundRect(ctx, x, y, w, h, 14);
			ctx.fill();
		});
		ctx.fillStyle = p.accent;
		ctx.font = `800 11px ${FONT}`;
		ctx.fillText(label, x + 18, y + 26);
		glow(10, () => {
			ctx.fillStyle = '#ffffff';
			ctx.fillText(fitFont(ctx, value, w - 36, 800, size, 14), x + 18, y + h - 40);
		});
		ctx.fillStyle = '#b8bcc6';
		ctx.font = `500 13px ${FONT}`;
		ctx.fillText(fitText(ctx, detail, w - 36), x + 18, y + h - 18);
	};

	const m = 32;
	const gap = 14;
	const w2 = (WIDTH - m * 2 - gap) / 2;
	panel(m, 96, w2, 118, 'CPU', rig.cpu, rig.cpuDetail, 26);
	panel(m + w2 + gap, 96, w2, 118, 'GPU', rig.gpu, rig.gpuDetail, 24);
	const w3 = (WIDTH - m * 2 - gap * 2) / 3;
	panel(m, 228, w3, 104, 'RAM', rig.memory, rig.memoryDetail, 26);
	panel(m + w3 + gap, 228, w3, 104, 'STORAGE', rig.storage, rig.storageDetail, 26);
	panel(m + (w3 + gap) * 2, 228, w3, 104, 'DISPLAY', rig.display, rig.displayDetail ? `${rig.displayDetail}` : 'Main display', 24);

	ctx.fillStyle = '#b8bcc6';
	ctx.font = `500 12px ${FONT}`;
	ctx.fillText(rig.os, m, HEIGHT - 22);
	ctx.fillText(CREDIT, WIDTH - m - ctx.measureText(CREDIT).width, HEIGHT - 22);
}

// Minimal: the theme's own colours, type only, nothing drawn
function drawMinimal(ctx: CanvasRenderingContext2D, p: Palette, rig: Rig) {
	ctx.fillStyle = p.tile;
	ctx.fillRect(0, 0, WIDTH, HEIGHT);
	ctx.fillStyle = p.accent;
	ctx.fillRect(0, 0, 6, HEIGHT);

	const m = 52;
	ctx.fillStyle = p.muted;
	ctx.font = `600 13px ${FONT}`;
	ctx.fillText('Specifications', m, 62);
	ctx.fillStyle = p.ink;
	ctx.fillText(fitFont(ctx, rig.name, WIDTH - m * 2, 700, 34, 20), m, 104);

	const rows: [string, string][] = [
		['Processor', rig.cores ? `${rig.cpu} · ${rig.cpuDetail}` : rig.cpu],
		['Graphics', rig.gpu],
		['Memory', rig.memoryDetail === 'Installed' ? rig.memory : `${rig.memory} · ${rig.memoryDetail}`],
		['Storage', `${rig.storage} · ${rig.storageDetail}`],
		['Display', rig.displayDetail ? `${rig.display} · ${rig.displayDetail}` : rig.display],
		['System', rig.os]
	];
	let y = 158;
	for (const [label, value] of rows) {
		ctx.fillStyle = p.muted;
		ctx.font = `500 14px ${FONT}`;
		ctx.fillText(label, m, y);
		ctx.fillStyle = p.ink;
		ctx.font = `600 16px ${FONT}`;
		ctx.fillText(fitText(ctx, value, WIDTH - m * 2 - 150), m + 150, y);
		y += 40;
	}

	logo(ctx, p, m, HEIGHT - 50, 18);
	ctx.fillStyle = p.muted;
	ctx.font = `500 12px ${FONT}`;
	ctx.fillText('SysInfo Pro', m + 26, HEIGHT - 36);
	ctx.fillText(CREDIT, WIDTH - m - ctx.measureText(CREDIT).width, HEIGHT - 36);
}

// ---------- the card ----------

export async function generateSpecCardPNG(systemInfo?: any, style: CardStyle = getCardStyle()): Promise<string> {
	// the Specs screen's data, usually already fetched in the background
	const specs = await load('specs', GetHardwareSpecs).catch(() => undefined);
	const rig = describeRig(systemInfo || recall('sysinfo') || {}, specs);

	const canvas = document.createElement('canvas');
	canvas.width = WIDTH * SCALE;
	canvas.height = HEIGHT * SCALE;
	const ctx = canvas.getContext('2d');
	if (!ctx) return '';
	ctx.scale(SCALE, SCALE);
	const p = readPalette();

	if (style === 'terminal') drawTerminal(ctx, p, rig);
	else if (style === 'neon') drawNeon(ctx, p, rig);
	else if (style === 'minimal') drawMinimal(ctx, p, rig);
	else drawRig(ctx, p, rig);

	return canvas.toDataURL('image/png');
}

export async function exportSpecCardPNG(systemInfo?: any): Promise<string> {
	const dataUrl = await generateSpecCardPNG(systemInfo);
	try {
		return await SaveSpecCardPNG(dataUrl);
	} catch (err) {
		console.error('Failed to save spec card PNG:', err);
		return '';
	}
}

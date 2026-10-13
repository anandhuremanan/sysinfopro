import { formatBytes, shortCpuName } from '$lib';
import { SaveSpecCardPNG } from '$lib/wailsjs/go/main/App';

const WIDTH = 800;
const HEIGHT = 450;
// Drawn at 2x so the PNG stays sharp when shared
const SCALE = 2;
const FONT = '"Segoe UI", system-ui, -apple-system, sans-serif';

type Palette = {
	ground: string;
	tile: string;
	ink: string;
	muted: string;
	track: string;
	accent: string;
	onAccent: string;
};

// The card follows the theme the app is showing
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
	while (cut.length > 1 && ctx.measureText(cut + '…').width > maxWidth) {
		cut = cut.slice(0, -1);
	}
	return cut.trimEnd() + '…';
}

type Tile = {
	x: number;
	y: number;
	w: number;
	h: number;
	label: string;
	value: string;
	detail: string;
	// 0-100 draws a usage bar under the value
	barPct?: number;
};

function drawTile(ctx: CanvasRenderingContext2D, p: Palette, tile: Tile) {
	const pad = 18;
	const inner = tile.w - pad * 2;

	ctx.fillStyle = p.tile;
	roundRect(ctx, tile.x, tile.y, tile.w, tile.h, 14);
	ctx.fill();

	ctx.fillStyle = p.muted;
	ctx.font = `500 13px ${FONT}`;
	ctx.fillText(tile.label, tile.x + pad, tile.y + pad + 12);

	ctx.fillStyle = p.ink;
	ctx.font = `600 24px ${FONT}`;
	ctx.fillText(fitText(ctx, tile.value, inner), tile.x + pad, tile.y + tile.h / 2 + 12);

	const detailY = tile.y + tile.h - pad;
	if (tile.barPct !== undefined) {
		const barY = tile.y + tile.h - pad - 26;
		ctx.fillStyle = p.track;
		roundRect(ctx, tile.x + pad, barY, inner, 5, 2.5);
		ctx.fill();
		ctx.fillStyle = p.accent;
		roundRect(ctx, tile.x + pad, barY, Math.max(5, (inner * tile.barPct) / 100), 5, 2.5);
		ctx.fill();
	}

	ctx.fillStyle = p.muted;
	ctx.font = `400 13px ${FONT}`;
	ctx.fillText(fitText(ctx, tile.detail, inner), tile.x + pad, detailY);
}

export async function generateSpecCardPNG(systemInfo: any): Promise<string> {
	const canvas = document.createElement('canvas');
	canvas.width = WIDTH * SCALE;
	canvas.height = HEIGHT * SCALE;
	const ctx = canvas.getContext('2d');
	if (!ctx) return '';
	ctx.scale(SCALE, SCALE);

	const p = readPalette();
	const margin = 28;
	const gap = 12;

	ctx.fillStyle = p.ground;
	ctx.fillRect(0, 0, WIDTH, HEIGHT);

	// Header: Pulse mark (same geometry as Logo.svelte, on a 24-unit grid) and wordmark
	const markSize = 30;
	const markY = 28;
	const unit = markSize / 24;
	ctx.fillStyle = p.accent;
	roundRect(ctx, margin + unit, markY + unit, 22 * unit, 22 * unit, 6 * unit);
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
	].forEach(([x, y], i) => {
		const px = margin + x * unit;
		const py = markY + y * unit;
		if (i === 0) ctx.moveTo(px, py);
		else ctx.lineTo(px, py);
	});
	ctx.stroke();

	const brandX = margin + markSize + 10;
	ctx.fillStyle = p.ink;
	ctx.font = `700 22px ${FONT}`;
	ctx.fillText('SysInfo Pro', brandX, 50);
	const brandWidth = ctx.measureText('SysInfo Pro').width;

	ctx.fillStyle = p.muted;
	ctx.font = `400 14px ${FONT}`;
	ctx.fillText('System specifications', brandX + brandWidth + 12, 50);

	const platform = (systemInfo.host_platform || 'Desktop System').replace(/^Microsoft /, '');
	const chip = platform;
	ctx.font = `600 13px ${FONT}`;
	const chipWidth = ctx.measureText(chip).width + 24;
	ctx.fillStyle = p.accent;
	roundRect(ctx, WIDTH - margin - chipWidth, 30, chipWidth, 28, 9);
	ctx.fill();
	ctx.fillStyle = p.onAccent;
	ctx.fillText(chip, WIDTH - margin - chipWidth + 12, 49);

	// Bento grid: 4 columns, two rows
	const colW = (WIDTH - margin * 2 - gap * 3) / 4;
	const col = (i: number) => margin + i * (colW + gap);
	const span = (n: number) => colW * n + gap * (n - 1);
	const row1Y = 80;
	const row1H = 150;
	const row2Y = row1Y + row1H + gap;
	const row2H = 124;

	const ramGB = systemInfo.total_ram_mb ? (systemInfo.total_ram_mb / 1024).toFixed(1) : '0';
	const GB = 1024 * 1024 * 1024;
	const totalDisk = systemInfo.total_disk_bytes ?? (systemInfo.total_disk_gb || 0) * GB;
	const freeDisk = systemInfo.free_disk_bytes ?? (systemInfo.free_disk_gb || 0) * GB;
	const diskUsedPct = totalDisk > 0 ? Math.min(100, Math.max(0, 100 - (freeDisk / totalDisk) * 100)) : 0;

	const tiles: Tile[] = [
		{
			x: col(0),
			y: row1Y,
			w: span(2),
			h: row1H,
			label: 'Processor',
			value: shortCpuName(systemInfo.cpu_model) || 'Unknown',
			detail: `${systemInfo.cpu_vendor || 'Unknown vendor'} · ${systemInfo.cpu_cores || 0} cores`
		},
		{
			x: col(2),
			y: row1Y,
			w: span(1),
			h: row1H,
			label: 'Memory',
			value: `${ramGB} GB`,
			detail: 'Installed RAM'
		},
		{
			x: col(3),
			y: row1Y,
			w: span(1),
			h: row1H,
			label: 'Storage',
			value: formatBytes(totalDisk),
			detail: `${formatBytes(freeDisk)} free · ${systemInfo.disk_type || 'Disk'}`,
			barPct: diskUsedPct
		},
		{
			x: col(0),
			y: row2Y,
			w: span(2),
			h: row2H,
			label: 'System',
			value: platform,
			detail: `Version ${systemInfo.host_platform_version || 'unknown'} · ${systemInfo.host_os || ''}`
		},
		{
			x: col(2),
			y: row2Y,
			w: span(1),
			h: row2H,
			label: 'Network',
			value: (systemInfo.network_status || 'Unknown').replace(/^./, (c: string) => c.toUpperCase()),
			detail: 'At time of export'
		},
		{
			x: col(3),
			y: row2Y,
			w: span(1),
			h: row2H,
			label: 'User',
			value: systemInfo.logged_in_user || 'Unknown',
			detail: 'Signed-in account'
		}
	];
	tiles.forEach((tile) => drawTile(ctx, p, tile));

	// Footer
	ctx.fillStyle = p.muted;
	ctx.font = `400 12px ${FONT}`;
	ctx.fillText('Generated by SysInfo Pro', margin, HEIGHT - 26);

	const date = new Date().toLocaleDateString(undefined, {
		year: 'numeric',
		month: 'short',
		day: 'numeric'
	});
	ctx.fillText(date, WIDTH - margin - ctx.measureText(date).width, HEIGHT - 26);

	const dataUrl = canvas.toDataURL('image/png');
	return dataUrl;
}

export async function exportSpecCardPNG(systemInfo: any): Promise<string> {
	const dataUrl = await generateSpecCardPNG(systemInfo);
	try {
		// @ts-ignore
		const savedPath = await SaveSpecCardPNG(dataUrl);
		return savedPath;
	} catch (err) {
		console.error('Failed to save spec card PNG:', err);
		return '';
	}
}

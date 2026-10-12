import { formatBytes, mbToGb } from '$lib';
import { SaveSystemReport } from '$lib/wailsjs/go/main/App';

type Row = [label: string, value: string];
type Section = { title: string; rows: Row[] };

export type ReportInput = {
	systemInfo: any;
	specs?: any;
	security?: any;
	drives?: any[];
};

function escapeHtml(value: unknown): string {
	return String(value ?? '')
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;')
		.replace(/"/g, '&quot;');
}

// Sections shared by the on-screen specs view and the exported report
export function buildSpecSections(specs: any): Section[] {
	if (!specs) return [];
	const sections: Section[] = [];
	const add = (title: string, rows: Row[]) => {
		const filled = rows.filter(([, value]) => value && value.trim() !== '');
		if (filled.length > 0) sections.push({ title, rows: filled });
	};

	add('System', [
		['Manufacturer', specs.system.manufacturer],
		['Model', specs.system.model],
		['Type', specs.system.type],
		['Serial number', specs.bios.serial]
	]);

	specs.cpus.forEach((cpu: any, i: number) => {
		add(specs.cpus.length > 1 ? `Processor ${i + 1}` : 'Processor', [
			['Model', cpu.name],
			['Cores', `${cpu.cores} cores, ${cpu.threads} threads`],
			['Max speed', cpu.max_mhz ? `${(cpu.max_mhz / 1000).toFixed(2)} GHz` : ''],
			['Socket', cpu.socket]
		]);
	});

	specs.gpus.forEach((gpu: any, i: number) => {
		add(specs.gpus.length > 1 ? `Graphics ${i + 1}` : 'Graphics', [
			['Adapter', gpu.name],
			// Windows reports this as a 32-bit value, so cards over 4 GB read low
			['Video memory', gpu.vram_bytes > 0 ? formatBytes(gpu.vram_bytes) : ''],
			['Display', gpu.width ? `${gpu.width} × ${gpu.height} at ${gpu.refresh_hz} Hz` : ''],
			['Driver', gpu.driver]
		]);
	});

	const totalRam = specs.memory.reduce((sum: number, m: any) => sum + m.size_bytes, 0);
	add('Memory', [
		['Installed', totalRam > 0 ? formatBytes(totalRam) : ''],
		[
			'Slots',
			specs.memory_slots > 0 ? `${specs.memory.length} of ${specs.memory_slots} used` : ''
		],
		...specs.memory.map(
			(m: any): Row => [
				m.slot || 'Module',
				[formatBytes(m.size_bytes), m.speed_mhz ? `${m.speed_mhz} MHz` : '', m.manufacturer, m.part]
					.filter(Boolean)
					.join(' · ')
			]
		)
	]);

	add(
		'Storage devices',
		specs.disks.map(
			(d: any, i: number): Row => [
				`Disk ${i + 1}`,
				[d.model, formatBytes(d.size_bytes), d.interface].filter(Boolean).join(' · ')
			]
		)
	);

	add('Firmware and board', [
		['BIOS', [specs.bios.vendor, specs.bios.version].filter(Boolean).join(' ')],
		['BIOS date', specs.bios.date],
		['Motherboard', [specs.board.manufacturer, specs.board.product].filter(Boolean).join(' ')]
	]);

	add('Operating system', [
		['Edition', specs.os.name],
		['Version', [specs.os.version, specs.os.arch].filter(Boolean).join(' · ')],
		['Installed', specs.os.installed],
		['Last started', specs.os.last_boot]
	]);

	return sections;
}

export function specsToText(specs: any): string {
	return buildSpecSections(specs)
		.map(
			(section) =>
				`${section.title}\n${section.rows.map(([label, value]) => `  ${label}: ${value}`).join('\n')}`
		)
		.join('\n\n');
}

const STATUS_LABEL: Record<string, string> = {
	pass: 'OK',
	warn: 'Review',
	fail: 'Problem',
	unknown: 'Unknown'
};

export function generateReportHTML({ systemInfo, specs, security, drives }: ReportInput): string {
	const sections = buildSpecSections(specs);

	if (sections.length === 0 && systemInfo) {
		// hardware inventory unavailable: fall back to the basic overview
		sections.push({
			title: 'System',
			rows: [
				['Platform', `${systemInfo.host_platform} ${systemInfo.host_platform_version}`],
				['Processor', `${systemInfo.cpu_model} (${systemInfo.cpu_cores} cores)`],
				['Memory', `${mbToGb(systemInfo.total_ram_mb)} GB`]
			]
		});
	}

	const table = (rows: Row[]) =>
		`<table>${rows
			.map(([label, value]) => `<tr><th>${escapeHtml(label)}</th><td>${escapeHtml(value)}</td></tr>`)
			.join('')}</table>`;

	const specHtml = sections
		.map((s) => `<section><h2>${escapeHtml(s.title)}</h2>${table(s.rows)}</section>`)
		.join('');

	const driveHtml =
		drives && drives.length > 0
			? `<section><h2>Drives</h2>${table(
					drives.map((d): Row => [
						d.path,
						`${formatBytes(d.free_bytes)} free of ${formatBytes(d.total_bytes)} (${d.used_pct.toFixed(0)}% used) · ${d.fs_type}`
					])
				)}</section>`
			: '';

	const securityHtml = security
		? `<section class="wide"><h2>Security check: ${security.passed} of ${security.checks.length} passed</h2><table>${security.checks
				.map(
					(c: any) =>
						`<tr><th>${escapeHtml(c.title)}</th><td><span class="status ${escapeHtml(c.status)}">${STATUS_LABEL[c.status] ?? escapeHtml(c.status)}</span> ${escapeHtml(c.detail)}${c.advice ? `<div class="advice">${escapeHtml(c.advice)}</div>` : ''}</td></tr>`
				)
				.join('')}</table></section>`
		: '';

	const title = specs?.system?.model
		? `${specs.system.manufacturer} ${specs.system.model}`
		: systemInfo?.host_platform || 'System';
	const generated = new Date().toLocaleString();

	return `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>System report: ${escapeHtml(title)}</title>
<style>
:root { --ground: #eceef1; --tile: #fff; --ink: #15171a; --muted: #5e6570; --line: #dde0e5; --accent: #0f766e; }
@media (prefers-color-scheme: dark) {
	:root { --ground: #020617; --tile: #0f172a; --ink: #f1f5f9; --muted: #94a3b8; --line: #1e293b; --accent: #2dd4bf; }
}
* { box-sizing: border-box; }
body { margin: 0; padding: 32px 16px; background: var(--ground); color: var(--ink); font: 14px/1.5 "Segoe UI", system-ui, sans-serif; }
main { max-width: 980px; margin: 0 auto; }
header { display: flex; justify-content: space-between; align-items: baseline; flex-wrap: wrap; gap: 8px; margin-bottom: 20px; }
h1 { margin: 0; font-size: 22px; }
header p { margin: 0; color: var(--muted); }
.grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(340px, 1fr)); gap: 12px; }
section { background: var(--tile); border-radius: 14px; padding: 16px 18px; min-width: 0; }
section.wide { grid-column: 1 / -1; }
h2 { margin: 0 0 8px; font-size: 12px; text-transform: uppercase; letter-spacing: .05em; color: var(--muted); }
table { width: 100%; border-collapse: collapse; }
th, td { text-align: left; vertical-align: top; padding: 6px 0; border-top: 1px solid var(--line); overflow-wrap: anywhere; }
tr:first-child th, tr:first-child td { border-top: 0; }
th { width: 38%; font-weight: 400; color: var(--muted); padding-right: 12px; }
td { font-weight: 600; }
.status { display: inline-block; min-width: 64px; margin-right: 8px; padding: 0 8px; border-radius: 6px; font-size: 12px; text-align: center; border: 1px solid var(--line); }
.status.pass { color: var(--accent); }
.status.warn { color: #b45309; }
.status.fail { color: #dc2626; }
.advice { font-weight: 400; color: var(--muted); }
footer { margin-top: 20px; color: var(--muted); font-size: 12px; }
@media print { body { background: #fff; padding: 0; } section { border: 1px solid var(--line); break-inside: avoid; } }
</style>
</head>
<body>
<main>
<header><h1>${escapeHtml(title)}</h1><p>System report · ${escapeHtml(generated)}</p></header>
<div class="grid">${specHtml}${driveHtml}${securityHtml}</div>
<footer>Generated by SysInfo Pro. This report contains your device serial number; remove it before sharing publicly.</footer>
</main>
</body>
</html>`;
}

export async function exportSystemReport(input: ReportInput): Promise<string> {
	try {
		return await SaveSystemReport(generateReportHTML(input));
	} catch (err) {
		console.error('Failed to save system report:', err);
		return '';
	}
}

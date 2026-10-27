import { formatBytes, mbToGb } from '$lib';
import { SaveSystemReport } from '$lib/wailsjs/go/main/App';
import { makeRedactor, type Redactor } from '$lib/redact';

export { makeRedactor };

type Row = [label: string, value: string];
type Section = { title: string; rows: Row[] };

export type ReportInput = {
	systemInfo: any;
	specs?: any;
	security?: any;
	drives?: any[];
	driveHealth?: any[];
	battery?: any;
	network?: any;
	devTools?: any[];
	slow?: any;
	// leave out serial numbers, the PC's name, the user name, addresses and emails
	hidePersonal?: boolean;
	privateTerms?: string[];
};

type Esc = (value: unknown) => string;
const keepAll: Redactor = (t) => t;

export function escapeHtml(value: unknown): string {
	return String(value ?? '')
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;')
		.replace(/"/g, '&quot;');
}

// every value in the report passes through this: redacted first, then escaped
const escaper =
	(redact: Redactor): Esc =>
	(value) =>
		escapeHtml(redact(String(value ?? '')));

// "Intel(R) Core(TM) i5" -> "Intel Core i5"
export const tidyName = (name: string) =>
	String(name ?? '')
		.replace(/\((R|TM|tm|C)\)/g, '')
		.replace(/\s{2,}/g, ' ')
		.trim();

// Windows reports video memory as a 32-bit number, so a card with more than
// 4 GB reads as about 4 GB; built-in graphics report only the slice set aside
// for them, and borrow more system memory as needed
const VRAM_CAP = 4 * 1024 ** 3 - 1024 ** 2;
const DISCRETE_GPU = /nvidia|geforce|quadro|rtx|gtx|radeon (rx|pro)|arc a\d/i;
export function videoMemory(gpu: any): string {
	const v = Number(gpu?.vram_bytes) || 0;
	if (v <= 0) return '';
	if (!DISCRETE_GPU.test(gpu?.name || ''))
		return `${formatBytes(v)} set aside, borrows more as needed`;
	return v >= VRAM_CAP ? '4 GB or more (Windows cannot report higher)' : formatBytes(v);
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
			['Model', tidyName(cpu.name)],
			['Cores', `${cpu.cores} cores, ${cpu.threads} threads`],
			// Windows' "max clock speed" is the base speed, not the boost speed
			['Base speed', cpu.max_mhz ? `${(cpu.max_mhz / 1000).toFixed(2)} GHz` : ''],
			['Socket', cpu.socket]
		]);
	});

	specs.gpus.forEach((gpu: any, i: number) => {
		add(specs.gpus.length > 1 ? `Graphics ${i + 1}` : 'Graphics', [
			['Adapter', tidyName(gpu.name)],
			['Video memory', videoMemory(gpu)],
			['Display', gpu.width ? `${gpu.width} × ${gpu.height} at ${gpu.refresh_hz} Hz` : ''],
			['Driver', gpu.driver]
		]);
	});

	const totalRam = specs.memory.reduce((sum: number, m: any) => sum + m.size_bytes, 0);
	add('Memory', [
		['Installed', totalRam > 0 ? formatBytes(totalRam) : ''],
		['Slots', specs.memory_slots > 0 ? `${specs.memory.length} of ${specs.memory_slots} used` : ''],
		...specs.memory.map(
			(m: any, i: number): Row => [
				// Windows names slots like "Controller0-ChannelA-DIMM0"; a number is clearer
				specs.memory.length > 1 ? `Slot ${i + 1}` : 'Module',
				// Windows' memory "speed" is in transfers a second (DDR4-3200 = 3200 MT/s)
				[
					formatBytes(m.size_bytes),
					m.speed_mhz ? `${m.speed_mhz} MT/s` : '',
					m.manufacturer,
					m.part
				]
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

export function generateReportHTML(input: ReportInput): string {
	const redact = input.hidePersonal
		? makeRedactor([...(input.privateTerms ?? []), input.specs?.bios?.serial ?? ''])
		: keepAll;
	return writeReport(input, escaper(redact));
}

// a row whose value carries an OK / Review / Problem label
const statusCell = (esc: Esc, status: string, text: string, advice = '') =>
	`<span class="status ${esc(status)}">${STATUS_LABEL[status] ?? esc(status)}</span> ${esc(text)}${advice ? `<div class="advice">${esc(advice)}</div>` : ''}`;
const statusTable = (esc: Esc, rows: [label: string, cell: string][]) =>
	`<table>${rows.map(([label, cell]) => `<tr><th>${esc(label)}</th><td>${cell}</td></tr>`).join('')}</table>`;
const LEVEL_STATUS: Record<string, string> = {
	ok: 'pass',
	warn: 'warn',
	fail: 'fail',
	skip: 'unknown',
	high: 'fail',
	medium: 'warn',
	info: 'unknown',
	missing: 'unknown'
};

function extraSections(
	{ driveHealth, battery, network, devTools, slow }: ReportInput,
	esc: Esc
): string {
	const out: string[] = [];
	const minutes = (m: number) => (m ? `${Math.floor(m / 60)} h ${m % 60} min` : 'n/a');

	if (driveHealth?.length) {
		out.push(
			`<section><h2>Drive health</h2>${statusTable(
				esc,
				driveHealth.map((d: any) => [
					`${d.drive} ${d.model ?? ''}`.trim(),
					statusCell(
						esc,
						d.status === 'Healthy'
							? 'pass'
							: d.status === 'Unknown'
								? 'unknown'
								: d.status === 'Warning'
									? 'warn'
									: 'fail',
						[
							d.type,
							d.life_left_pct >= 0 ? `${d.life_left_pct}% of rated life left` : '',
							d.nvme?.temperature_c ? `${d.nvme.temperature_c} °C` : ''
						]
							.filter(Boolean)
							.join(' · '),
						d.reason
					)
				])
			)}</section>`
		);
	}

	if (battery?.has_battery) {
		const b = battery.batteries[0];
		out.push(
			`<section><h2>Battery</h2><table>${[
				['Health', `${Math.round(battery.health_pct)}% of original capacity`],
				[
					'Full charge',
					`${(battery.batteries.reduce((s: number, x: any) => s + x.full_mwh, 0) / 1000).toFixed(1)} Wh, was ${(battery.batteries.reduce((s: number, x: any) => s + x.design_mwh, 0) / 1000).toFixed(1)} Wh new`
				],
				[
					'Lasts about',
					`${minutes(battery.runtime_now_min)}, was ${minutes(battery.runtime_new_min)} new`
				],
				['Charge cycles', b.cycles >= 0 ? String(b.cycles) : 'Not reported by the battery'],
				[
					'Wear',
					battery.loss_per_year > 0
						? `about ${battery.loss_per_year.toFixed(1)}% a year, recorded since ${battery.tracked_since}`
						: `recorded since ${battery.tracked_since || 'today'}`
				]
			]
				.map(([l, v]) => `<tr><th>${esc(l)}</th><td>${esc(v)}</td></tr>`)
				.join('')}</table></section>`
		);
	}

	if (network?.steps?.length) {
		out.push(
			`<section><h2>Network doctor</h2><p class="lead">${esc(network.summary)}</p>${statusTable(
				esc,
				network.steps.map((s: any) => [
					s.name,
					statusCell(esc, LEVEL_STATUS[s.status] ?? 'unknown', s.detail || 'Not checked', s.fix)
				])
			)}</section>`
		);
	}

	const installed = (devTools ?? []).filter((t: any) => t.status !== 'missing');
	if (installed.length) {
		out.push(
			`<section><h2>Developer tools</h2>${statusTable(
				esc,
				installed.map((t: any) => [
					t.name,
					statusCell(
						esc,
						LEVEL_STATUS[t.status] ?? 'unknown',
						`${t.version || 'no version'} · ${t.copies[0]?.path ?? ''}${t.copies.length > 1 ? ` (+${t.copies.length - 1} more)` : ''}`,
						t.issues.map((i: any) => i.text).join(' ')
					)
				])
			)}</section>`
		);
	}

	if (slow) {
		const rows: [string, string][] = [
			['Processor', `${Math.round(slow.cpu.percent)}% busy`],
			['Memory', `${Math.round(slow.memory.used_pct)}% used`],
			['Drive', slow.disk.busy_pct >= 0 ? `${Math.round(slow.disk.busy_pct)}% busy` : 'n/a']
		];
		out.push(
			`<section class="wide"><h2>Slowness check</h2><p class="lead">${esc(slow.calm ? 'Nothing was slowing the PC down when this report was made.' : slow.summary)}</p>${statusTable(
				esc,
				[
					...rows.map(([l, v]): [string, string] => [l, esc(v)]),
					...slow.findings.map((f: any): [string, string] => [
						f.title,
						statusCell(esc, LEVEL_STATUS[f.level] ?? 'unknown', f.detail, f.fix)
					])
				]
			)}</section>`
		);
	}
	return out.join('');
}

function writeReport(input: ReportInput, esc: Esc): string {
	const { systemInfo, specs, security, drives } = input;
	const sections = buildSpecSections(specs).map((s) => ({
		...s,
		rows: input.hidePersonal ? s.rows.filter(([label]) => label !== 'Serial number') : s.rows
	}));

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
			.map(([label, value]) => `<tr><th>${esc(label)}</th><td>${esc(value)}</td></tr>`)
			.join('')}</table>`;

	const specHtml = sections
		.map((s) => `<section><h2>${esc(s.title)}</h2>${table(s.rows)}</section>`)
		.join('');

	const driveHtml =
		drives && drives.length > 0
			? `<section><h2>Drives</h2>${table(
					drives.map(
						(d): Row => [
							d.path,
							`${formatBytes(d.free_bytes)} free of ${formatBytes(d.total_bytes)} (${d.used_pct.toFixed(0)}% used) · ${d.fs_type}`
						]
					)
				)}</section>`
			: '';

	const securityHtml = security
		? `<section class="wide"><h2>Security check: ${security.passed} of ${security.checks.length} passed</h2><table>${security.checks
				.map(
					(c: any) =>
						`<tr><th>${esc(c.title)}</th><td><span class="status ${esc(c.status)}">${STATUS_LABEL[c.status] ?? esc(c.status)}</span> ${esc(c.detail)}${c.advice ? `<div class="advice">${esc(c.advice)}</div>` : ''}</td></tr>`
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
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>System report: ${esc(title)}</title>
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
.lead { margin: 0 0 8px; font-weight: 600; }
footer { margin-top: 20px; color: var(--muted); font-size: 12px; }
@media print { body { background: #fff; padding: 0; } section { border: 1px solid var(--line); break-inside: avoid; } }
</style>
</head>
<body>
<main>
<header><h1>${esc(title)}</h1><p>System report · ${esc(generated)}</p></header>
<div class="grid">${specHtml}${driveHtml}${extraSections(input, esc)}${securityHtml}</div>
<footer>Generated by SysInfo Pro. ${input.hidePersonal ? "Serial numbers, the PC's name, the user name, network addresses and emails are left out." : 'This report contains your device serial number and PC name; remove them before sharing publicly.'}</footer>
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

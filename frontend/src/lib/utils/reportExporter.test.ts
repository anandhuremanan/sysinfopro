import { describe, expect, it } from 'vitest';
import { escapeHtml, generateReportHTML } from './reportExporter';

const systemInfo = {
	host_platform: 'Microsoft Windows 11 Pro',
	host_platform_version: '10.0.22631',
	cpu_model: 'Intel Core i5',
	cpu_cores: 10,
	total_ram_mb: 16384
};

describe('escapeHtml', () => {
	it('escapes markup characters', () => {
		expect(escapeHtml('<b>"a" & b</b>')).toBe('&lt;b&gt;&quot;a&quot; &amp; b&lt;/b&gt;');
		expect(escapeHtml(undefined)).toBe('');
	});
});

describe('generateReportHTML', () => {
	const network = {
		summary: 'Router at 192.168.1.1 owned by dev <script>alert(1)</script>',
		steps: [{ name: 'Router', status: 'ok', detail: 'Router 192.168.1.1 answered', fix: '' }]
	};

	it('escapes machine data, so nothing in it can run as markup', () => {
		const html = generateReportHTML({ systemInfo, network });
		expect(html).not.toContain('<script>alert(1)</script>');
		expect(html).toContain('&lt;script&gt;');
	});

	it('forbids scripts and outside files with a content policy', () => {
		const html = generateReportHTML({ systemInfo });
		expect(html).toContain('Content-Security-Policy');
		expect(html).toContain("default-src 'none'");
	});

	it('leaves out personal details when asked', () => {
		const html = generateReportHTML({
			systemInfo,
			network,
			hidePersonal: true,
			privateTerms: ['dev']
		});
		expect(html).not.toContain('192.168.1.1');
		expect(html).not.toMatch(/\bdev\b/);
		// the redactor is per report: a later report without hiding is untouched
		const plain = generateReportHTML({ systemInfo, network });
		expect(plain).toContain('192.168.1.1');
	});
});

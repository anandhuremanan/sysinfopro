import { describe, expect, it } from 'vitest';
import { AUDIENCES, audienceOf, getTool, matchTool, tools } from './tools';

describe('matchTool', () => {
	const slow = getTool('slow');

	it('matches everything when nothing is typed', () => {
		expect(matchTool(slow, '')).toBe(1);
		expect(matchTool(slow, '   ')).toBe(1);
	});

	it('scores a name prefix above a keyword hit', () => {
		const byName = matchTool(slow, 'why');
		const byKeyword = matchTool(slow, 'lag');
		expect(byName).toBeGreaterThan(byKeyword);
		expect(byKeyword).toBeGreaterThan(0);
	});

	it('needs every typed word to match', () => {
		expect(matchTool(slow, 'why zebra')).toBe(0);
	});

	it('ignores case', () => {
		expect(matchTool(slow, 'WHY SLOW')).toBeGreaterThan(0);
	});
});

describe('tools', () => {
	it('have unique ids', () => {
		expect(new Set(tools.map((t) => t.id)).size).toBe(tools.length);
	});

	it('each name at least one audience, and only known ones', () => {
		const known = new Set(AUDIENCES.map((a) => a.id));
		for (const tool of tools) {
			const list = audienceOf[tool.id];
			expect(list, tool.id).toBeDefined();
			expect(list.length, tool.id).toBeGreaterThan(0);
			for (const a of list) expect(known.has(a)).toBe(true);
		}
	});

	it('give every everyday user something to use', () => {
		expect(tools.filter((t) => audienceOf[t.id].includes('everyday')).length).toBeGreaterThan(8);
	});
});

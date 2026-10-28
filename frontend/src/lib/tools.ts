export type ToolId =
	| 'specs'
	| 'security'
	| 'devcache'
	| 'changes'
	| 'locks'
	| 'growth'
	| 'health'
	| 'space'
	| 'ports'
	| 'stress'
	| 'startup'
	| 'env'
	| 'software'
	| 'speed'
	| 'wsl'
	| 'netdoctor'
	| 'devdoctor'
	| 'dupes'
	| 'slow'
	| 'battery'
	| 'handover'
	| 'gitsweep'
	| 'move'
	| 'shred';

export type Tool = {
	id: ToolId;
	name: string;
	hint: string;
	group: string;
	// 24x24 stroke icon path
	icon: string;
	// other words people might search for
	keywords?: string;
};

// Live status shown on a tool tile; count renders as an attention badge
export type ToolStatus = { hint?: string; count?: number };

export const tools: Tool[] = [
	{
		id: 'specs',
		name: 'Specs',
		hint: 'Full inventory',
		group: 'System',
		keywords: 'hardware cpu gpu ram memory serial bios report',
		icon: 'M9 3v2M15 3v2M9 19v2M15 19v2M3 9h2M3 15h2M19 9h2M19 15h2M7 5h10a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2zM10 10h4v4h-4z'
	},
	{
		id: 'security',
		name: 'Security check',
		hint: 'Protection status',
		group: 'System',
		keywords: 'antivirus firewall bitlocker updates defender secure boot uac',
		icon: 'M12 3l7 3v5c0 4.5-3 8.5-7 10-4-1.5-7-5.5-7-10V6l7-3zM9 12l2 2 4-4'
	},
	{
		id: 'slow',
		name: 'Why so slow',
		hint: 'Right now',
		group: 'System',
		keywords: 'slow lag sluggish performance cpu memory ram disk busy hang freeze high usage',
		icon: 'M6 3h12M6 21h12M7 3c0 5 5 6 5 9s-5 4-5 9M17 3c0 5-5 6-5 9s5 4 5 9'
	},
	{
		id: 'battery',
		name: 'Battery',
		hint: 'Health & history',
		group: 'System',
		keywords: 'battery health wear capacity cycles charge laptop runtime',
		icon: 'M3 8h15a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V9a1 1 0 0 1 1-1zM22 11v2M6 11v2M10 11v2'
	},
	{
		id: 'handover',
		name: 'Hand over',
		hint: 'Before you sell',
		group: 'System',
		keywords: 'sell give away hand over reset wipe new pc accounts sign out privacy',
		icon: 'M7 11l3-3 3 2 4-4M3 14l4 1 5 4 9-6M3 14V9M21 13V8'
	},
	{
		id: 'changes',
		name: 'What changed',
		hint: 'Since last review',
		group: 'System',
		keywords: 'diff snapshot new installed startup ports',
		icon: 'M12 7v5l3 2M4.5 8.5A8.5 8.5 0 1 1 3.5 12M4 4v4.5h4.5'
	},
	{
		id: 'health',
		name: 'Disk & health',
		hint: 'Clean up space',
		group: 'Clean up',
		keywords: 'cleaner temp temporary cache browser drive health ssd nvme wear temperature',
		icon: 'M3 13h18M5 13l2-7h10l2 7v5a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1v-5zM16 16h.01M12 16h.01'
	},
	{
		id: 'space',
		name: 'Disk usage',
		hint: 'Find big files',
		group: 'Clean up',
		keywords: 'disk usage big files folders storage tree size',
		icon: 'M12 3v9h9a9 9 0 1 1-9-9zM15 3.5A9 9 0 0 1 20.5 9H15V3.5z'
	},
	{
		id: 'growth',
		name: 'Disk growth',
		hint: 'What is growing',
		group: 'Clean up',
		keywords: 'growing filling watch folder',
		icon: 'M3 17l6-6 4 4 8-8M15 7h6v6'
	},
	{
		id: 'startup',
		name: 'Startup',
		hint: 'Startup apps',
		group: 'Clean up',
		keywords: 'boot autostart login',
		icon: 'M12 3v9M6.3 6.3a8 8 0 1 0 11.4 0'
	},
	{
		id: 'software',
		name: 'Software',
		hint: 'Check versions',
		group: 'Clean up',
		keywords: 'versions update upgrade winget outdated',
		icon: 'M21 8l-9-5-9 5v8l9 5 9-5V8zM3 8l9 5 9-5M12 13v8'
	},
	{
		id: 'speed',
		name: 'Speed test',
		hint: 'Internet speed',
		group: 'Network',
		keywords: 'internet download upload ping bandwidth',
		icon: 'M12 14l4-4M4 18a9 9 0 1 1 16 0'
	},
	{
		id: 'netdoctor',
		name: 'Network doctor',
		hint: 'Why is it offline',
		group: 'Network',
		keywords: 'internet not working offline wifi dns router proxy hosts no connection troubleshoot',
		icon: 'M12 20h.01M8.5 16.5a5 5 0 0 1 7 0M5 13a10 10 0 0 1 14 0M2 9.5a15 15 0 0 1 20 0'
	},
	{
		id: 'ports',
		name: 'Ports',
		hint: 'Listening ports',
		group: 'Network',
		keywords: 'listening localhost kill process port in use',
		icon: 'M9 2v6M15 2v6M6 8h12v4a6 6 0 0 1-12 0V8zM12 18v4'
	},
	{
		id: 'devcache',
		name: 'Dev caches',
		hint: 'Reclaim space',
		group: 'Developer',
		keywords: 'npm pnpm yarn pip nuget gradle maven cargo go node_modules venv target build clean',
		icon: 'M12 3l9 5-9 5-9-5 9-5zM3 13l9 5 9-5M3 17l9 5 9-5'
	},
	{
		id: 'wsl',
		name: 'WSL & Docker',
		hint: 'Virtual disks',
		group: 'Developer',
		keywords: 'linux ubuntu docker vhdx virtual disk compact',
		icon: 'M4 4h16v6H4zM4 14h16v6H4zM8 7h.01M8 17h.01'
	},
	{
		id: 'locks',
		name: 'File locks',
		hint: 'Who has it open',
		group: 'System',
		keywords: 'in use open handle cannot delete locked',
		icon: 'M7 11V8a5 5 0 0 1 10 0v3M6 11h12a1 1 0 0 1 1 1v8a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1v-8a1 1 0 0 1 1-1zM12 15v2.5'
	},
	{
		id: 'gitsweep',
		name: 'Git projects',
		hint: 'Unpushed work',
		group: 'Developer',
		keywords: 'git repo repository unpushed uncommitted commit push stash backup branch lost work',
		icon: 'M6 3v12M18 9a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM6 21a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM18 9a9 9 0 0 1-9 9'
	},
	{
		id: 'devdoctor',
		name: 'Dev doctor',
		hint: 'Which version runs',
		group: 'Developer',
		keywords:
			'python node java git go dotnet version wrong shadowed java_home nvm pyenv toolchain environment',
		icon: 'M9 3h6M10 3v6l-5 9a2 2 0 0 0 1.8 3h10.4a2 2 0 0 0 1.8-3l-5-9V3M7.5 15h9'
	},
	{
		id: 'dupes',
		name: 'Duplicates',
		hint: 'Same file twice',
		group: 'Clean up',
		keywords: 'duplicate files copies same identical photos downloads free space',
		icon: 'M8 8h11v13H8zM5 16H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h11a1 1 0 0 1 1 1v1'
	},
	{
		id: 'move',
		name: 'Move off C:',
		hint: 'Make room',
		group: 'Clean up',
		keywords:
			'move relocate migrate drive space c: cache npm pip go gradle nuget wsl chrome edge firefox browser profile junction',
		icon: 'M3 5h8v14H3zM14 12h7M18 8l4 4-4 4'
	},
	{
		id: 'shred',
		name: 'File shredder',
		hint: 'Delete for good',
		group: 'Clean up',
		keywords: 'shred wipe secure delete erase overwrite privacy recover forensic free space cipher',
		icon: 'M6 10V4a1 1 0 0 1 1-1h8l3 3v4M3 10h18v4H3zM6 14v7M10 14v5M14 14v7M18 14v4'
	},
	{
		id: 'env',
		name: 'PATH',
		hint: 'PATH entries',
		group: 'Developer',
		keywords: 'environment variables path broken duplicate',
		icon: 'M4 6l6 6-6 6M12 19h8'
	},
	{
		id: 'stress',
		name: 'Stress test',
		hint: '10 s benchmark',
		group: 'System',
		keywords: 'benchmark performance cpu load',
		icon: 'M13 2L4 14h7l-1 8 9-12h-7l1-8z'
	}
];

// Tools shown on the home row; the sixth slot is "All tools"
export const pinnedTools: ToolId[] = ['specs', 'space', 'devcache', 'ports', 'env'];

export const ALL_TOOLS_ICON = 'M4 4h6v6H4zM14 4h6v6h-6zM4 14h6v6H4zM14 14h6v6h-6z';

export function getTool(id: ToolId): Tool {
	return tools.find((t) => t.id === id) as Tool;
}

// How well a tool matches what was typed: 0 for no match, higher is better.
// Every word typed has to appear in the name, hint, group or keywords.
export function matchTool(tool: Tool, query: string): number {
	const words = query.trim().toLowerCase().split(/\s+/).filter(Boolean);
	if (words.length === 0) return 1;
	const name = tool.name.toLowerCase();
	const rest = `${tool.hint} ${tool.group} ${tool.keywords ?? ''}`.toLowerCase();
	let score = 0;
	for (const word of words) {
		if (name.startsWith(word)) score += 3;
		else if (name.includes(word)) score += 2;
		else if (rest.includes(word)) score += 1;
		else return 0;
	}
	return score;
}

// Who each tool helps. A tool can help several kinds of people, and people are
// usually more than one kind (a developer who games is also an everyday user),
// so the All tools filter shows tools matching any kind that is turned on.
export type Audience = 'everyday' | 'developer' | 'gaming';
export const AUDIENCES: { id: Audience; label: string }[] = [
	{ id: 'everyday', label: 'Everyday' },
	{ id: 'developer', label: 'Developer' },
	{ id: 'gaming', label: 'Gaming' }
];

// which kinds of tools to show; remembered on this PC. All on = every tool.
const AUDIENCE_KEY = 'sysinfopro-tool-audience';

export function loadAudience(): Audience[] {
	try {
		const saved = JSON.parse(localStorage.getItem(AUDIENCE_KEY) ?? 'null');
		if (Array.isArray(saved) && saved.length) return saved;
	} catch {
		// private window or blocked storage: fall back to everything
	}
	return AUDIENCES.map((a) => a.id);
}

export function saveAudience(audience: Audience[]) {
	try {
		localStorage.setItem(AUDIENCE_KEY, JSON.stringify(audience));
	} catch {
		// not remembered; still applies now
	}
}

// turns one kind on or off; at least one stays on, so the list is never empty
export function toggleAudience(audience: Audience[], id: Audience): Audience[] {
	if (audience.includes(id) && audience.length === 1) return audience;
	return audience.includes(id) ? audience.filter((a) => a !== id) : [...audience, id];
}
const E: Audience = 'everyday';
const D: Audience = 'developer';
const G: Audience = 'gaming';
export const audienceOf: Record<ToolId, Audience[]> = {
	specs: [E, D, G],
	security: [E, D],
	slow: [E, D, G],
	battery: [E, D],
	handover: [E, D],
	changes: [E, D],
	stress: [G],
	health: [E, D, G],
	space: [E, D, G],
	growth: [E, D],
	startup: [E, G],
	software: [E, D],
	dupes: [E],
	devcache: [D],
	wsl: [D],
	locks: [E, D],
	gitsweep: [D],
	devdoctor: [D],
	env: [D],
	speed: [E, G],
	netdoctor: [E, D, G],
	ports: [D],
	move: [E, D],
	shred: [E, D]
};

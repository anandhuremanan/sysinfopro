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
	| 'wsl';

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
		icon: 'M4 7h16M10 11v6M14 11v6M6 7l1 12a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-12M9 7V4h6v3'
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
		group: 'Developer',
		keywords: 'in use open handle cannot delete locked',
		icon: 'M7 11V8a5 5 0 0 1 10 0v3M6 11h12a1 1 0 0 1 1 1v8a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1v-8a1 1 0 0 1 1-1zM12 15v2.5'
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

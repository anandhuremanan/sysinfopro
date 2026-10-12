export type ToolId =
	| 'specs'
	| 'security'
	| 'devcache'
	| 'health'
	| 'space'
	| 'ports'
	| 'stress'
	| 'startup'
	| 'env'
	| 'software'
	| 'speed';

export type Tool = {
	id: ToolId;
	name: string;
	hint: string;
	group: string;
	// 24x24 stroke icon path
	icon: string;
};

// Live status shown on a tool tile; count renders as an attention badge
export type ToolStatus = { hint?: string; count?: number };

export const tools: Tool[] = [
	{
		id: 'specs',
		name: 'Specs',
		hint: 'Full inventory',
		group: 'System',
		icon: 'M9 3v2M15 3v2M9 19v2M15 19v2M3 9h2M3 15h2M19 9h2M19 15h2M7 5h10a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2zM10 10h4v4h-4z'
	},
	{
		id: 'security',
		name: 'Security check',
		hint: 'Protection status',
		group: 'System',
		icon: 'M12 3l7 3v5c0 4.5-3 8.5-7 10-4-1.5-7-5.5-7-10V6l7-3zM9 12l2 2 4-4'
	},
	{
		id: 'health',
		name: 'Disk & health',
		hint: 'Clean up space',
		group: 'Clean up',
		icon: 'M4 7h16M10 11v6M14 11v6M6 7l1 12a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-12M9 7V4h6v3'
	},
	{
		id: 'space',
		name: 'Disk usage',
		hint: 'Find big files',
		group: 'Clean up',
		icon: 'M12 3v9h9a9 9 0 1 1-9-9zM15 3.5A9 9 0 0 1 20.5 9H15V3.5z'
	},
	{
		id: 'startup',
		name: 'Startup',
		hint: 'Startup apps',
		group: 'Clean up',
		icon: 'M12 3v9M6.3 6.3a8 8 0 1 0 11.4 0'
	},
	{
		id: 'software',
		name: 'Software',
		hint: 'Check versions',
		group: 'Clean up',
		icon: 'M21 8l-9-5-9 5v8l9 5 9-5V8zM3 8l9 5 9-5M12 13v8'
	},
	{
		id: 'speed',
		name: 'Speed test',
		hint: 'Internet speed',
		group: 'Network',
		icon: 'M12 14l4-4M4 18a9 9 0 1 1 16 0'
	},
	{
		id: 'ports',
		name: 'Ports',
		hint: 'Listening ports',
		group: 'Network',
		icon: 'M9 2v6M15 2v6M6 8h12v4a6 6 0 0 1-12 0V8zM12 18v4'
	},
	{
		id: 'devcache',
		name: 'Dev caches',
		hint: 'Reclaim space',
		group: 'Developer',
		icon: 'M12 3l9 5-9 5-9-5 9-5zM3 13l9 5 9-5M3 17l9 5 9-5'
	},
	{
		id: 'env',
		name: 'PATH',
		hint: 'PATH entries',
		group: 'Developer',
		icon: 'M4 6l6 6-6 6M12 19h8'
	},
	{
		id: 'stress',
		name: 'Stress test',
		hint: '10 s benchmark',
		group: 'System',
		icon: 'M13 2L4 14h7l-1 8 9-12h-7l1-8z'
	}
];

// Tools shown on the home row; the sixth slot is "All tools"
export const pinnedTools: ToolId[] = ['specs', 'space', 'devcache', 'ports', 'env'];

export const ALL_TOOLS_ICON = 'M4 4h6v6H4zM14 4h6v6h-6zM4 14h6v6H4zM14 14h6v6h-6z';

export function getTool(id: ToolId): Tool {
	return tools.find((t) => t.id === id) as Tool;
}

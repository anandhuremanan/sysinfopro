// Small helpers for Windows paths shown on screen.

// "D:\work\app\" -> "app"; a drive root stays as it is
export function folderName(path: string): string {
	return (
		path
			.replace(/[\\/]$/, '')
			.split(/[\\/]/)
			.pop() || path
	);
}

// the folder a file or folder sits in: "D:\a\b.txt" -> "D:\a"
export function parentOf(path: string): string {
	const i = Math.max(path.lastIndexOf('\\'), path.lastIndexOf('/'));
	return i > 0 ? path.slice(0, i) : path;
}

// "C:" means "the current folder on C", so a drive root always ends in a separator
export function driveRoot(path: string): string {
	return /^[A-Za-z]:$/.test(path) ? path + '\\' : path;
}

// Where something is, inside the folder that was searched: the part that tells
// results apart. "D:\work\a\b" under "D:\work" -> "a"; directly in the searched
// folder -> topLabel(root)
export function placeOf(
	path: string,
	root: string,
	topLabel: (root: string) => string = (r) => folderName(r) || r
): string {
	const base = root.replace(/[\\/]+$/, '');
	const parent = parentOf(path);
	if (base && parent.toLowerCase().startsWith(base.toLowerCase())) {
		return parent.slice(base.length).replace(/^[\\/]+/, '') || topLabel(base);
	}
	return parent;
}

// the icon for "Show in File Explorer", used by every screen that lists files
export const REVEAL_ICON =
	'M14 4h6v6M20 4l-9 9M18 14v4a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4';

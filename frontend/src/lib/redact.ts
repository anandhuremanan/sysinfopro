// Hides personal details in text that leaves the PC: shared reports and copied stats.

export type Redactor = (text: string) => string;
const escapeRe = (t: string) => t.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

// IPv4 octets must each be 0-255; a longer dotted number is a version, not an address
function hideIPv4(text: string): string {
	return text.replace(
		/(?<![\w.])(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})(?!\.?\d)(?!\w)/g,
		(all, ...parts: string[]) =>
			parts.slice(0, 4).every((n) => Number(n) <= 255) ? '(hidden)' : all
	);
}

// Hides what identifies the PC or its user: the user folder in paths, emails,
// network addresses, and the given names (whole words only, so a name like "dev"
// does not touch "Developer")
export function makeRedactor(terms: string[]): Redactor {
	const words = [...new Set(terms.map((t) => (t ?? '').trim()).filter(Boolean))]
		.sort((a, b) => b.length - a.length)
		.map(escapeRe);
	const named = words.length
		? new RegExp(`(?<![\\p{L}\\p{N}_])(?:${words.join('|')})(?![\\p{L}\\p{N}_])`, 'giu')
		: null;
	return (text) => {
		let out = text
			// C:\Users\<name>\..., whatever the name (also 8.3 names like ANAND~1)
			// (one or more slashes: JSON doubles each backslash)
			.replace(/([A-Za-z]:[\\/]+Users[\\/]+)[^\\/\s"'<>]+/gi, '$1(hidden)')
			.replace(/[\w.+-]+@[\w-]+(\.[\w-]+)+/g, '(email hidden)')
			// MAC addresses, with or without separators
			.replace(/\b(?:[0-9a-f]{2}[:-]){5}[0-9a-f]{2}\b/gi, '(hidden)')
			.replace(/\b(?=[0-9a-f]*[a-f])(?=[0-9a-f]*\d)[0-9a-f]{12}\b/gi, '(hidden)')
			// IPv6: eight groups, or a shortened form with "::"; times like 12:30:45 are left alone
			.replace(
				/(?<![\w:.])(?:(?:[0-9a-f]{1,4}:){7}[0-9a-f]{1,4}|(?:[0-9a-f]{1,4}:){1,7}:(?:[0-9a-f]{1,4}(?::[0-9a-f]{1,4}){0,6})?|::[0-9a-f]{1,4}(?::[0-9a-f]{1,4}){0,6})(?:%\w+)?(?![\w:])/gi,
				'(hidden)'
			);
		out = hideIPv4(out);
		if (named) out = out.replace(named, '(hidden)');
		return out;
	};
}

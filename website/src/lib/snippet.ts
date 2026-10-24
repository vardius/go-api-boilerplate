/**
 * Cuts top-level declarations out of a Go file imported with `?raw`, so a page shows real code
 * without pasting it.
 *
 * Each start is the beginning of a top-level line, such as `type counter struct` or
 * `func (c *counter) Apply`. A declaration runs from its doc comment to the line that closes it,
 * and the declarations are joined with a blank line, in the order given.
 *
 * A start that matches no line, or more than one, fails the build: a renamed or duplicated
 * declaration can't leave a stale sample on the site.
 */
export function goDecls(source: string, file: string, ...starts: string[]): string {
	const lines = source.split('\n');
	return starts.map((start) => decl(lines, file, start)).join('\n\n');
}

function decl(lines: string[], file: string, start: string): string {
	const matches = lines.flatMap((line, i) => (line.startsWith(start) ? [i] : []));
	if (matches.length !== 1) {
		throw new Error(`${file}: ${matches.length} top-level lines start with "${start}"; want exactly one`);
	}
	const first = matches[0];

	let from = first;
	while (from > 0 && lines[from - 1].startsWith('//')) from--;

	let to = first;
	const opener = lines[first].trimEnd();
	if (opener.endsWith('{') || opener.endsWith('(')) {
		const closer = opener.endsWith('{') ? '}' : ')';
		to = lines.findIndex((line, i) => i > first && line.trimEnd() === closer);
		if (to === -1) throw new Error(`${file}: "${start}" never closes`);
	}
	return lines.slice(from, to + 1).join('\n');
}

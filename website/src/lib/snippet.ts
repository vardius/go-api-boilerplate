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

/**
 * Cuts one block out of a YAML file imported with `?raw`, such as a schema of an OpenAPI document,
 * so a page shows the contract itself.
 *
 * The path names a key at each level, `'components', 'schemas', 'Problem'`, and the block runs from
 * that key's line to the line before the next key at its level or above, dedented to the key.
 *
 * A key that matches no line at its level, or more than one, fails the build.
 */
export function yamlBlock(source: string, file: string, ...path: string[]): string {
	const lines = source.split('\n');
	const indent = (line: string) => line.length - line.trimStart().length;
	const content = (line: string) => line.trim() !== '' && !line.trimStart().startsWith('#');

	let from = -1;
	let to = lines.length;
	let level = 0;
	for (const [depth, key] of path.entries()) {
		const name = path.slice(0, depth + 1).join('.');
		const first = lines.findIndex((line, i) => i > from && i < to && content(line));
		if (first === -1) throw new Error(`${file}: nothing under ${name}`);
		level = indent(lines[first]);
		const matches = lines.flatMap((line, i) =>
			i >= first && i < to && indent(line) === level && line.trimStart().startsWith(`${key}:`) ? [i] : [],
		);
		if (matches.length !== 1) throw new Error(`${file}: ${matches.length} keys ${name}; want exactly one`);
		from = matches[0];
		const end = lines.findIndex((line, i) => i > from && i < to && content(line) && indent(line) <= level);
		if (end !== -1) to = end;
	}
	while (to > from && lines[to - 1].trim() === '') to--;
	return lines
		.slice(from, to)
		.map((line) => line.slice(level))
		.join('\n');
}

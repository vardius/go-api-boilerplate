// A Sätteri hast plugin that copies each table's column headings onto its body cells as data-label,
// so that on a narrow screen custom.css can stack every row into a labelled card.

/** @param {readonly any[]} nodes */
const elements = (nodes) => nodes.filter((node) => node.type === 'element');

/** @returns {import('satteri').HastPluginDefinition} */
function tableLabels() {
	/** @type {WeakMap<object, string[]>} the headings of each table seen in this document */
	const headings = new WeakMap();
	return {
		name: 'table-labels',
		element: {
			filter: ['td'],
			visit(td, ctx) {
				const row = ctx.parent(td);
				const body = row && ctx.parent(row);
				const table = body && ctx.parent(body);
				if (body?.type !== 'element' || body.tagName !== 'tbody' || table?.type !== 'element') return;

				if (!headings.has(table)) {
					const head = elements(table.children).find((node) => node.tagName === 'thead');
					const headRow = head && elements(head.children).find((node) => node.tagName === 'tr');
					const cells = headRow ? elements(headRow.children).filter((node) => node.tagName === 'th') : [];
					headings.set(table, cells.map((th) => ctx.textContent(th).trim()));
				}

				const column = elements(row.children.slice(0, ctx.indexOf(td))).length;
				const label = headings.get(table)?.[column];
				if (label) ctx.setProperty(td, 'data-label', label);
			},
		},
	};
}

// A factory, so each document starts with its own cache.
export default () => tableLabels;

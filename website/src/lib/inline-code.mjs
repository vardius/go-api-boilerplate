// A Sätteri hast plugin that gives inline code sensible line breaks, so routes, paths and commands
// wrap cleanly in narrow columns instead of splitting mid-word:
//   - a <wbr> after each slash, so `/api/user/v1/me` may break between its segments;
//   - each word that starts with a dash in <span class="nowrap">, so `up --wait` never becomes "up -" and "-wait".
// Copying the code still gives the original text.

const flag = /^-{1,2}[A-Za-z]/;

/** @param {string} text */
function pieces(text) {
	/** @type {any[]} */
	const out = [];
	for (const word of text.split(/(\s+)/)) {
		if (word === '') continue;
		if (flag.test(word)) {
			out.push({ type: 'element', tagName: 'span', properties: { class: 'nowrap' }, children: [{ type: 'text', value: word }] });
			continue;
		}
		const parts = word.split(/(?<=\/)(?=.)/);
		parts.forEach((part, i) => {
			if (i > 0) out.push({ type: 'element', tagName: 'wbr', properties: {}, children: [] });
			out.push({ type: 'text', value: part });
		});
	}
	return out;
}

/** @returns {import('satteri').HastPluginDefinition} */
function inlineCode() {
	return {
		name: 'inline-code',
		element: {
			filter: ['code'],
			visit(code, ctx) {
				const parent = ctx.parent(code);
				if (parent?.type === 'element' && parent.tagName === 'pre') return;
				const [only, ...rest] = code.children;
				if (!only || rest.length > 0 || only.type !== 'text') return;

				const children = pieces(only.value);
				if (children.length < 2) return;
				return { type: 'element', tagName: 'code', properties: { ...code.properties }, children };
			},
		},
	};
}

export default () => inlineCode;

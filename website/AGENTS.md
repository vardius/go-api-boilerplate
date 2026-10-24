# website

The documentation site: Astro Starlight, published to https://vardius.github.io/go-api-boilerplate/ by
`.github/workflows/website.yml` on every push to master that touches it. Pages live in `src/content/docs/`.

## Rules

- **Code samples are imported, never pasted.** Add an export to `src/lib/samples.ts` (`go()` cuts
  declarations from a Go file by their first line; other files are imported whole with `?raw`) and
  spread it into a page: `<Code {...sample} />`. A renamed declaration fails the build: fix the sample.
- **Every page has a one-sentence `description`** in its frontmatter.
- **Never invent** an env var, endpoint, flag or config key. Take it from the code.
- Text that depends on code not written yet carries a verify note: `{/* verify: … */}` in `.mdx`,
  `<!-- verify: … -->` in `.md` and `.astro`. Check it against the code, then delete the note.
- Everything after `<!-- generated -->` in `reference/environment-variables.md` belongs to the generator.
- **Flawless on a 320 px phone with 1.3× text**, in light and dark: nothing leaves its box. Code wraps
  inside its frame; tables stack into labelled rows in a narrow column (`src/lib/table-labels.mjs`);
  inline code breaks after slashes, never inside a flag (`src/lib/inline-code.mjs`).
- Colours are tokens in `src/styles/custom.css`; contrast is AA in both themes.
- Plain, short sentences; why before how; no filler.

## Done means

```sh
npm ci && npm run build
mkdir -p ../_site && rm -rf ../_site/go-api-boilerplate && cp -R dist ../_site/go-api-boilerplate
lychee --offline --include-fragments --root-dir "$PWD/../_site" --index-files index.html '../_site/**/*.html'
```

Both pass, and every changed page was checked at 320 px and 1280 px in light and dark.

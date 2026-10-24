// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import { satteri } from '@astrojs/markdown-satteri';
import inlineCode from './src/lib/inline-code.mjs';
import tableLabels from './src/lib/table-labels.mjs';

export default defineConfig({
	site: 'https://vardius.github.io',
	base: '/go-api-boilerplate',
	trailingSlash: 'always',
	// Pages import their code samples from the Go and proto files in the repository, so the dev server may read outside this folder.
	vite: { server: { fs: { allow: ['..'] } } },
	// For narrow columns: tables stack into labelled rows, so each column heading is copied onto its
	// cells, and inline code breaks after slashes but never inside a flag.
	markdown: { processor: satteri({ hastPlugins: [tableLabels(), inlineCode()] }) },
	integrations: [
		starlight({
			title: 'go-api-boilerplate',
			description:
				'A Go API template with passwordless sign-in, an OAuth2 token service, an event-sourced user service and a transactional outbox.',
			logo: { src: './src/assets/mark.svg', alt: '' },
			favicon: '/favicon.svg',
			components: { Hero: './src/components/Hero.astro' },
			// Long lines wrap instead of scrolling under the copy button on phones.
			expressiveCode: { defaultProps: { wrap: true } },
			social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/vardius/go-api-boilerplate' }],
			editLink: { baseUrl: 'https://github.com/vardius/go-api-boilerplate/edit/master/website/' },
			customCss: ['./src/styles/custom.css'],
			sidebar: [
				{
					label: 'Start here',
					items: ['introduction', 'quick-start', 'architecture', 'project-layout'],
				},
				{
					label: 'Concepts',
					items: [
						'concepts/aggregates-and-events',
						'concepts/event-store-and-concurrency',
						'concepts/cqrs-and-projections',
						'concepts/outbox-and-side-effects',
						'concepts/authentication',
						'concepts/errors',
						'concepts/observability',
					],
				},
				{
					label: 'Guides',
					items: [
						'guides/add-a-command',
						'guides/add-a-projection',
						'guides/add-an-endpoint',
						'guides/add-an-rpc',
						'guides/add-a-service',
						'guides/testing',
						'guides/configuration',
						'guides/deploy-to-kubernetes',
						'guides/working-with-ai-agents',
					],
				},
				{
					label: 'Reference',
					items: [
						'reference/rest-api',
						'reference/environment-variables',
						'reference/proto',
						'reference/migrating-from-2022',
						'reference/changelog',
					],
				},
			],
		}),
	],
});

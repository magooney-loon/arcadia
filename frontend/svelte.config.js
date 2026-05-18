import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	compilerOptions: {
		// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
		runes: ({ filename }) => (filename.split(/[/\\]/).includes('node_modules') ? undefined : true)
	},
	kit: {
		// SPA fallback lets dynamic routes (e.g. /wallet/[address], /tx/[hash])
		// be served client-side from a single index.html.
		adapter: adapter({ fallback: '200.html' })
	}
};

export default config;

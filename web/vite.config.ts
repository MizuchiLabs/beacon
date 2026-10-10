import adapter from '@sveltejs/adapter-static';
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig, type Plugin } from 'vite';

function goServerUrl(): Plugin {
	const port = process.env.BEACON_PORT || '3000';
	const url = `http://localhost:${port}`;
	return {
		name: 'go-server-url',
		configureServer(server) {
			server.printUrls = () => server.config.logger.info(`  ➜  Dashboard: ${url}`);
		}
	};
}

export default defineConfig({
	server: { port: 5173, strictPort: true },
	plugins: [
		goServerUrl(),
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			adapter: adapter({ fallback: 'index.html', precompress: true })
		})
	]
});

import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import { compression } from 'vite-plugin-compression2';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit(), compression()],
	server: {
		proxy: {
			'^/(api|openapi|docs|schemas)': {
				target: 'http://localhost:3000',
				changeOrigin: true
			}
		}
	}
});

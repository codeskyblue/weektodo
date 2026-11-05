import adapter from '@sveltejs/adapter-static';
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

const backend = process.env.WEEKTODO_BACKEND_URL || 'http://127.0.0.1:8787';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			adapter: adapter({
				pages: 'build',
				assets: 'build',
				precompress: false,
				strict: true
			}),
			paths: {
				relative: false
			}
		})
	],
	server: {
		proxy: {
			'/api': backend
		}
	}
});

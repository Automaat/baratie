import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	resolve: {
		alias: {
			cookie: new URL('./src/lib/server/cookie-v2-compat.js', import.meta.url).pathname
		},
		conditions: ['browser']
	},
	test: {
		globals: true,
		environment: 'jsdom',
		include: ['src/**/*.{test,spec}.{js,ts}'],
		coverage: {
			provider: 'v8',
			reporter: ['text', 'json', 'html', 'lcov'],
			// Coverage is gated on the pure-logic modules (utils + stores). UI
			// components and routes are exercised by the Playwright e2e suite.
			include: ['src/lib/utils/**', 'src/lib/stores/**'],
			// Thresholds are a ratchet against the current baseline, not a target.
			// Raise them as coverage improves so regressions can't slip past CI.
			thresholds: {
				statements: 60,
				branches: 60,
				functions: 60,
				lines: 60
			}
		}
	}
});

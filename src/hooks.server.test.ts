import { describe, it, expect, vi } from 'vitest';
import type { RequestEvent } from '@sveltejs/kit';

vi.mock('$env/dynamic/public', () => ({ env: {} }));

const { handle } = await import('./hooks.server');

const expiredToken = `x.${btoa(JSON.stringify({ username: 'u', is_admin: false, exp: 1 }))}.y`;

function run(pathname: string, token?: string) {
	const del = vi.fn();
	const event = {
		url: new URL(`https://baratie.test${pathname}`),
		cookies: { get: () => token, delete: del },
		locals: {}
	} as unknown as RequestEvent;
	const resolve = vi.fn(async () => new Response('resolved'));
	return { promise: handle({ event, resolve }), resolve, del };
}

describe('handle', () => {
	it.each(['/api/auth/oidc/status', '/api/auth/oidc/login', '/api/auth/oidc/callback'])(
		'lets unauthenticated %s through',
		async (path) => {
			const { promise, resolve } = run(path);
			const res = await promise;
			expect(resolve).toHaveBeenCalledOnce();
			expect(await res.text()).toBe('resolved');
		}
	);

	it('keeps a stale token on the OIDC callback so the new session cookie wins', async () => {
		const { promise, del } = run('/api/auth/oidc/callback', expiredToken);
		await promise;
		expect(del).not.toHaveBeenCalled();
	});

	it('requires a session for paths that only start with the OIDC prefix', async () => {
		const { promise, resolve } = run('/api/auth/oidc/..%2F..%2Fmetrics');
		const res = await promise;
		expect(resolve).not.toHaveBeenCalled();
		expect(res.status).toBe(401);
	});

	it('still clears a stale token and rejects other API calls', async () => {
		const { promise, del, resolve } = run('/api/recipes', expiredToken);
		const res = await promise;
		expect(del).toHaveBeenCalledWith('brt_token', { path: '/' });
		expect(resolve).not.toHaveBeenCalled();
		expect(res.status).toBe(401);
	});
});

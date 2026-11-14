import { describe, it, expect, vi } from 'vitest';
import type { RequestEvent } from './$types';

vi.mock('$env/dynamic/private', () => ({ env: { API_PROXY_TARGET: 'http://backend.test' } }));

const { GET } = await import('./+server');

interface Call {
	url: string;
	init: RequestInit;
}

function proxyEvent(path: string, cookieJar: Record<string, string>, upstream: Response) {
	const calls: Call[] = [];
	const fetchFn = vi.fn(async (url: string, init: RequestInit) => {
		calls.push({ url, init });
		return upstream;
	});
	const event = {
		request: new Request(`https://baratie.test/api/${path}`),
		params: { path: path.split('?')[0] },
		url: new URL(`https://baratie.test/api/${path}`),
		cookies: { get: (name: string) => cookieJar[name] },
		fetch: fetchFn
	} as unknown as RequestEvent;
	return { event, calls };
}

function redirectWithCookies(location: string, cookies: string[]): Response {
	const headers = new Headers({ location });
	for (const c of cookies) {
		headers.append('set-cookie', c);
	}
	return new Response(null, { status: 302, headers });
}

describe('/api proxy', () => {
	it('passes OIDC callback 302, Location and every Set-Cookie through unchanged', async () => {
		const setCookies = [
			'brt_oidc_state=; Path=/api/auth/oidc; Max-Age=0; HttpOnly; SameSite=Lax',
			'brt_token=jwt.value.sig; Path=/; HttpOnly; SameSite=Lax'
		];
		const { event, calls } = proxyEvent(
			'auth/oidc/callback?code=abc&state=xyz',
			{ brt_oidc_state: 'payload.sig', other: 'ignored' },
			redirectWithCookies('/', setCookies)
		);

		const res = await GET(event);

		expect(res.status).toBe(302);
		expect(res.headers.get('location')).toBe('/');
		expect(res.headers.getSetCookie()).toEqual(setCookies);
		expect(calls).toHaveLength(1);
		expect(calls[0].url).toBe('http://backend.test/api/auth/oidc/callback?code=abc&state=xyz');
		expect(calls[0].init.redirect).toBe('manual');
		expect(new Headers(calls[0].init.headers).get('cookie')).toBe('brt_oidc_state=payload.sig');
	});

	it('passes the OIDC login redirect to the provider through', async () => {
		const target = 'https://auth.test/api/oidc/authorization?client_id=baratie';
		const stateCookie = 'brt_oidc_state=p.s; Path=/api/auth/oidc; Max-Age=600; HttpOnly';
		const { event } = proxyEvent('auth/oidc/login', {}, redirectWithCookies(target, [stateCookie]));

		const res = await GET(event);

		expect(res.status).toBe(302);
		expect(res.headers.get('location')).toBe(target);
		expect(res.headers.getSetCookie()).toEqual([stateCookie]);
	});

	it('drops backend Set-Cookie and the state cookie outside the OIDC flow', async () => {
		const upstream = new Response('{"ok":true}', {
			status: 200,
			headers: { 'content-type': 'application/json', 'set-cookie': 'brt_token=x; Path=/' }
		});
		const { event, calls } = proxyEvent(
			'recipes',
			{ brt_token: 'session-jwt', brt_oidc_state: 'p.s' },
			upstream
		);

		const res = await GET(event);

		expect(res.status).toBe(200);
		expect(res.headers.getSetCookie()).toEqual([]);
		const sent = new Headers(calls[0].init.headers);
		expect(sent.get('authorization')).toBe('Bearer session-jwt');
		expect(sent.get('cookie')).toBeNull();
	});

	it.each(['auth/oidc/../../metrics', 'auth/oidc/./status', '..'])(
		'rejects dot segments in %s without calling the backend',
		async (path) => {
			const { event, calls } = proxyEvent(path, {}, new Response(null, { status: 200 }));

			const res = await GET(event);

			expect(res.status).toBe(404);
			expect(calls).toHaveLength(0);
		}
	);
});

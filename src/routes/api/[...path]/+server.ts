// Server-side proxy for the Go backend.
//
// The browser only ever talks to this SvelteKit origin; this route forwards
// /api/* to the backend and attaches the session token from the brt_token
// cookie as a Bearer header. Keeps the JWT out of client-side JavaScript.
import { env } from '$env/dynamic/private';
import { OIDC_API_PREFIX, OIDC_STATE_COOKIE } from '$lib/utils/oidc';
import type { RequestHandler } from './$types';

function backendURL(path: string, search: string): string {
	const base = env.API_PROXY_TARGET ?? 'http://localhost:8000';
	return `${base}/api/${path}${search}`;
}

function isOidcPath(path: string): boolean {
	return `/api/${path}`.startsWith(OIDC_API_PREFIX);
}

/**
 * Copies content-type and location, so redirects pass through untouched.
 * Set-Cookie is forwarded only for the OIDC flow, which sets its state cookie
 * and the session through the backend. Elsewhere it is dropped on purpose: the
 * /login action calls this proxy through SvelteKit's internal fetch, which
 * would apply the backend's session cookie on top of the one the action sets.
 */
function responseHeadersFrom(upstream: Response, oidc: boolean): Headers {
	const headers = new Headers();
	const contentType = upstream.headers.get('content-type');
	if (contentType) {
		headers.set('content-type', contentType);
	}
	const location = upstream.headers.get('location');
	if (location) {
		headers.set('location', location);
	}
	if (oidc) {
		for (const cookie of upstream.headers.getSetCookie()) {
			headers.append('set-cookie', cookie);
		}
	}
	return headers;
}

const proxy: RequestHandler = async ({ request, params, cookies, url, fetch }) => {
	const oidc = isOidcPath(params.path);
	const headers = new Headers();
	const contentType = request.headers.get('content-type');
	if (contentType) {
		headers.set('content-type', contentType);
	}
	const token = cookies.get('brt_token');
	if (token) {
		headers.set('authorization', `Bearer ${token}`);
	}
	const oidcState = oidc ? cookies.get(OIDC_STATE_COOKIE) : undefined;
	if (oidcState) {
		headers.set('cookie', `${OIDC_STATE_COOKIE}=${oidcState}`);
	}

	const init: RequestInit = { method: request.method, headers, redirect: 'manual' };
	if (request.method !== 'GET' && request.method !== 'HEAD') {
		init.body = await request.text();
	}

	const upstream = await fetch(backendURL(params.path, url.search), init);
	const body = await upstream.arrayBuffer();
	// 204/304 are null-body statuses — Response throws if handed a body, even
	// an empty one (e.g. the backend's 204 on DELETE).
	const responseBody = body.byteLength > 0 ? body : null;
	return new Response(responseBody, {
		status: upstream.status,
		headers: responseHeadersFrom(upstream, oidc)
	});
};

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const PATCH = proxy;
export const DELETE = proxy;

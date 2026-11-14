import { describe, it, expect, vi } from 'vitest';
import { fetchOidcEnabled, oidcErrorMessage } from './oidc';

describe('oidcErrorMessage', () => {
	it.each([
		['oidc_no_user', /Brak konta Baratie/],
		['oidc_failed', /nie powiodło się/],
		['oidc_unavailable', /niedostępna/]
	])('maps %s to a readable message', (code, message) => {
		expect(oidcErrorMessage(code)).toMatch(message);
	});

	it.each([null, '', 'bogus', 'toString'])('returns null for %j', (code) => {
		expect(oidcErrorMessage(code)).toBeNull();
	});
});

describe('fetchOidcEnabled', () => {
	const respond = (body: unknown, status = 200) =>
		vi.fn<typeof fetch>().mockResolvedValue(
			new Response(JSON.stringify(body), {
				status,
				headers: { 'content-type': 'application/json' }
			})
		);

	it('is true when the backend reports enabled', async () => {
		const fetchFn = respond({ enabled: true });
		await expect(fetchOidcEnabled(fetchFn)).resolves.toBe(true);
		expect(fetchFn).toHaveBeenCalledWith('/api/auth/oidc/status');
	});

	it.each([
		['disabled', respond({ enabled: false })],
		['non-boolean', respond({ enabled: 'yes' })],
		['error status', respond({ detail: 'Not found' }, 404)],
		['network failure', vi.fn<typeof fetch>().mockRejectedValue(new TypeError('fetch failed'))]
	])('is false when %s', async (_name, fetchFn) => {
		await expect(fetchOidcEnabled(fetchFn)).resolves.toBe(false);
	});
});

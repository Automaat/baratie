import { test, expect } from '@playwright/test';

// Where the Go backend listens (same value the webServer config uses). The
// token is exercised straight against the backend, bypassing the SvelteKit
// proxy, to prove pure `Authorization: Bearer` auth.
const API_URL = process.env.E2E_API_URL ?? 'http://127.0.0.1:8000';

// Full personal-access-token lifecycle through the real stack: mint a token in
// the settings UI, call the backend with it as a bearer, revoke it, and confirm
// the same call is then rejected.
test('create, use and revoke an API token', async ({ page, playwright }) => {
	const name = `e2e-token-${Date.now()}`;

	await page.goto('/settings/tokens');
	await expect(page.getByRole('heading', { name: 'Tokeny API' })).toBeVisible();

	await page.getByLabel('Nazwa', { exact: true }).fill(name);
	await page.getByRole('button', { name: 'Utwórz token' }).click();

	// The secret is shown once in a modal; capture it before closing.
	const dialog = page.getByRole('dialog');
	await expect(dialog).toBeVisible();
	const token = await dialog.locator('input[readonly]').inputValue();
	expect(token).toMatch(/^brt_pat_/);
	await dialog.getByRole('button', { name: 'Gotowe' }).click();
	await expect(page.getByRole('cell', { name, exact: true })).toBeVisible();

	// A fresh context carries no session cookie (cookies ignore port, so the
	// logged-in brt_token on 127.0.0.1 would otherwise reach the backend and
	// mask the bearer token we want to test).
	const apiCtx = await playwright.request.newContext();
	try {
		const authed = await apiCtx.get(`${API_URL}/api/auth/me`, {
			headers: { Authorization: `Bearer ${token}` }
		});
		expect(authed.status()).toBe(200);

		// Revoke through the UI (row action + confirm dialog).
		const row = page.getByRole('row', { name: new RegExp(name) });
		await row.getByRole('button', { name: 'Unieważnij' }).click();
		await page.getByRole('dialog').getByRole('button', { name: 'Unieważnij' }).click();
		await expect(page.getByRole('cell', { name, exact: true })).toBeHidden();

		const revoked = await apiCtx.get(`${API_URL}/api/auth/me`, {
			headers: { Authorization: `Bearer ${token}` }
		});
		expect(revoked.status()).toBe(401);
	} finally {
		await apiCtx.dispose();
	}
});

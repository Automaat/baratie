import { test as setup, expect } from '@playwright/test';
import { ADMIN_USERNAME, ADMIN_PASSWORD, STORAGE_STATE } from './credentials';

// Logs in once and saves the session cookie; the test projects reuse this
// storage state so they start authenticated.
setup('authenticate', async ({ page }) => {
	await page.goto('/login');
	await page.fill('input[name="username"]', ADMIN_USERNAME);
	await page.fill('input[name="password"]', ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Zaloguj' }).click();

	await page.waitForURL('/');
	await expect(page.getByRole('heading', { name: 'Pulpit' })).toBeVisible();

	await page.context().storageState({ path: STORAGE_STATE });
});

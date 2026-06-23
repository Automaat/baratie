import { test, expect } from '@playwright/test';

// End-to-end create flow through the real backend: open the foods modal, fill
// the name + macros, save, and confirm the new food shows up in the table.
test('creates a food', async ({ page }) => {
	const name = `Test Food ${Date.now()}`;

	await page.goto('/foods');
	await expect(page.getByRole('heading', { name: 'Baza produktów' })).toBeVisible();

	const dialog = page.getByRole('dialog');
	await expect(async () => {
		await page.getByRole('button', { name: 'Nowy produkt' }).click();
		await expect(dialog).toBeVisible({ timeout: 1000 });
	}).toPass();

	await dialog.locator('input[type="text"]').first().fill(name);
	// First macro input is kcal per 100 g.
	await dialog.locator('input[type="number"]').first().fill('165');
	await dialog.getByRole('button', { name: 'Zapisz' }).click();

	await expect(dialog).toBeHidden();
	await expect(page.getByRole('cell', { name, exact: true })).toBeVisible();
});

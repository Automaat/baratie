import { test, expect } from '@playwright/test';

// End-to-end create flow through the real backend: open the modal, fill the
// form, save, and confirm the new recipe shows up in the list.
test('creates a recipe', async ({ page }) => {
	const name = `Test Recipe ${Date.now()}`;

	await page.goto('/recipes');
	await expect(page.getByRole('heading', { name: 'Przepisy' })).toBeVisible();

	const dialog = page.getByRole('dialog');
	// Retry the open click until the modal appears — guards against clicking
	// before SvelteKit has hydrated the button's handler.
	await expect(async () => {
		await page.getByRole('button', { name: 'Nowy przepis' }).click();
		await expect(dialog).toBeVisible({ timeout: 1000 });
	}).toPass();

	// The name is the first text input in the modal form.
	await dialog.locator('input[type="text"]').first().fill(name);
	await dialog.getByRole('button', { name: 'Zapisz' }).click();

	await expect(dialog).toBeHidden();
	await expect(page.getByRole('heading', { name, exact: true })).toBeVisible();
});

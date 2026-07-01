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
	// First number input under the nutrition fieldset is calories per serving.
	await dialog.locator('fieldset input[type="number"]').first().fill('500');
	await dialog.getByRole('button', { name: 'Zapisz' }).click();

	await expect(dialog).toBeHidden();
	await expect(page.getByRole('heading', { name, exact: true })).toBeVisible();

	// The macros round-trip and render on the new recipe card.
	const card = page.locator('article', {
		has: page.getByRole('heading', { name, exact: true })
	});
	await expect(card.getByText('500 kcal')).toBeVisible();
});

test('opens a recipe detail page from the list', async ({ page }) => {
	const name = `Detail Recipe ${Date.now()}`;

	await page.goto('/recipes');

	const dialog = page.getByRole('dialog');
	await expect(async () => {
		await page.getByRole('button', { name: 'Nowy przepis' }).click();
		await expect(dialog).toBeVisible({ timeout: 1000 });
	}).toPass();

	await dialog.getByLabel('Nazwa').fill(name);
	await dialog.getByLabel('Składniki (jeden na linię)').fill('2 jajka\n100 g mąki');
	await dialog.getByLabel('Instrukcje').fill('Wymieszaj składniki.\nSmaż na patelni.');
	await dialog.locator('fieldset input[type="number"]').first().fill('500');
	await dialog.getByRole('button', { name: 'Zapisz' }).click();

	await expect(dialog).toBeHidden();

	const card = page.locator('article', {
		has: page.getByRole('heading', { name, exact: true })
	});
	await card.getByRole('link', { name: 'Zobacz' }).click();

	await expect(page.getByRole('heading', { name, exact: true })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Składniki' })).toBeVisible();
	await expect(page.getByText('2 jajka')).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Instrukcje' })).toBeVisible();
	await expect(page.getByText('Wymieszaj składniki.')).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Wartości odżywcze' })).toBeVisible();
	await expect(page.getByText('500 kcal')).toBeVisible();
});

test('shows not found for a missing recipe detail page', async ({ page }) => {
	await page.goto('/recipes/999999999');

	await expect(page.getByRole('heading', { name: '404' })).toBeVisible();
	await expect(page.getByText('Nie znaleziono przepisu')).toBeVisible();
});

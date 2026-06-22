import { test, expect } from '@playwright/test';

// Runs on both desktop and mobile projects: each main section renders its
// heading once authenticated. Direct navigation keeps the check viewport-
// agnostic (desktop sidebar vs mobile bottom bar).
const sections = [
	{ path: '/', heading: 'Pulpit' },
	{ path: '/recipes', heading: 'Przepisy' },
	{ path: '/pantry', heading: 'Spiżarnia' },
	{ path: '/meal-plan', heading: 'Plan posiłków' },
	{ path: '/settings', heading: 'Ustawienia' }
];

for (const section of sections) {
	test(`renders ${section.heading}`, async ({ page }) => {
		await page.goto(section.path);
		await expect(page.getByRole('heading', { name: section.heading, exact: true })).toBeVisible();
	});
}

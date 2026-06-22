import { describe, it, expect, afterEach } from 'vitest';
import { navPrefs, DEFAULT_PINNED, MAX_PINNED } from './navPrefs.svelte';

describe('navPrefs', () => {
	afterEach(() => navPrefs.reset());

	it('starts from the defaults', () => {
		expect([...navPrefs.pinned]).toEqual([...DEFAULT_PINNED]);
	});

	it('drops unknown hrefs and duplicates when setting', () => {
		navPrefs.set(['/recipes', '/recipes', '/nope', '/pantry']);
		expect([...navPrefs.pinned]).toEqual(['/recipes', '/pantry']);
	});

	it('caps the pinned list at MAX_PINNED', () => {
		navPrefs.set(['/', '/recipes', '/pantry', '/meal-plan', '/settings']);
		expect(navPrefs.pinned.length).toBe(MAX_PINNED);
	});

	it('falls back to defaults when given no valid entries', () => {
		navPrefs.set(['/nonexistent']);
		expect([...navPrefs.pinned]).toEqual([...DEFAULT_PINNED]);
	});
});

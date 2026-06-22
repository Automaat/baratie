import { describe, it, expect, beforeEach } from 'vitest';
import { toast } from './toast.svelte';

describe('toast store', () => {
	beforeEach(() => {
		// Clear any leftover toasts between tests.
		for (const item of [...toast.items]) {
			toast.dismiss(item.id);
		}
	});

	it('pushes and dismisses toasts', () => {
		const id = toast.success('saved', 0);
		expect(toast.items).toHaveLength(1);
		expect(toast.items[0]).toMatchObject({ kind: 'success', message: 'saved' });
		toast.dismiss(id);
		expect(toast.items).toHaveLength(0);
	});

	it('supports error and info kinds', () => {
		toast.error('boom', 0);
		toast.info('fyi', 0);
		const kinds = toast.items.map((t) => t.kind);
		expect(kinds).toContain('error');
		expect(kinds).toContain('info');
	});

	it('auto-dismisses after the given duration', async () => {
		toast.info('temp', 10);
		expect(toast.items).toHaveLength(1);
		await new Promise((resolve) => setTimeout(resolve, 30));
		expect(toast.items).toHaveLength(0);
	});
});

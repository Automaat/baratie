import { describe, it, expect } from 'vitest';
import { confirm, confirmDialog } from './confirm.svelte';

describe('confirm dialog', () => {
	it('resolves true when confirmed', async () => {
		const promise = confirm({ title: 'T', message: 'M' });
		expect(confirmDialog.current).not.toBeNull();
		confirmDialog.confirm();
		await expect(promise).resolves.toBe(true);
		expect(confirmDialog.current).toBeNull();
	});

	it('resolves false when cancelled', async () => {
		const promise = confirm({ title: 'T', message: 'M' });
		confirmDialog.cancel();
		await expect(promise).resolves.toBe(false);
	});

	it('runs an async onConfirm handler and resolves true', async () => {
		let ran = false;
		const promise = confirm({
			title: 'T',
			message: 'M',
			onConfirm: async () => {
				ran = true;
			}
		});
		confirmDialog.confirm();
		await expect(promise).resolves.toBe(true);
		expect(ran).toBe(true);
	});

	it('resolves false when the handler throws', async () => {
		const promise = confirm({
			title: 'T',
			message: 'M',
			onConfirm: async () => {
				throw new Error('boom');
			}
		});
		confirmDialog.confirm();
		await expect(promise).resolves.toBe(false);
	});
});

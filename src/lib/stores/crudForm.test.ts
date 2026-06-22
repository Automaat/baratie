import { describe, it, expect } from 'vitest';
import { CrudForm } from './crudForm.svelte';

interface Row {
	id: number;
	name: string;
}

describe('CrudForm', () => {
	it('tracks create vs edit mode', () => {
		const form = new CrudForm<Row>();
		expect(form.open).toBe(false);
		expect(form.isEditing).toBe(false);

		form.openCreate();
		expect(form.open).toBe(true);
		expect(form.isEditing).toBe(false);

		form.openEdit({ id: 1, name: 'a' });
		expect(form.isEditing).toBe(true);
		expect(form.editing).toMatchObject({ id: 1 });

		form.close();
		expect(form.open).toBe(false);
		expect(form.editing).toBeNull();
	});

	it('submit returns true and closes on success', async () => {
		const form = new CrudForm<Row>();
		form.openCreate();
		const ok = await form.submit(async () => {});
		expect(ok).toBe(true);
		expect(form.open).toBe(false);
		expect(form.saving).toBe(false);
	});

	it('submit returns false and keeps the error on failure', async () => {
		const form = new CrudForm<Row>();
		form.openCreate();
		const ok = await form.submit(async () => {
			throw new Error('nope');
		});
		expect(ok).toBe(false);
		expect(form.open).toBe(true);
		expect(form.error).toBe('nope');
		expect(form.saving).toBe(false);
	});
});

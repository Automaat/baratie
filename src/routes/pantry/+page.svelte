<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import { api } from '$lib/apiClient';
	import { invalidateAll } from '$app/navigation';
	import { CrudForm } from '$lib/stores/crudForm.svelte';
	import { confirm } from '$lib/stores/confirm.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { formatDate, formatQuantity } from '$lib/utils/format';
	import { Refrigerator, Plus, Pencil, Trash2 } from 'lucide-svelte';
	import type { PantryItem } from './+page';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
	const items = $derived(data.items);

	// Common categories offered in the picker; the column accepts any string.
	const CATEGORIES = [
		'warzywa',
		'owoce',
		'nabiał',
		'mięso',
		'pieczywo',
		'przyprawy',
		'napoje',
		'mrożonki',
		'other'
	];

	const form = new CrudForm<PantryItem>();

	const emptyForm = () => ({
		name: '',
		quantity: 1,
		unit: 'szt',
		category: 'other',
		expires_on: ''
	});

	let formData = $state(emptyForm());

	$effect(() => {
		const editing = form.editing;
		if (editing) {
			formData = {
				name: editing.name,
				quantity: editing.quantity,
				unit: editing.unit,
				category: editing.category,
				expires_on: editing.expires_on ?? ''
			};
		} else if (form.open) {
			formData = emptyForm();
		}
	});

	function payload() {
		return {
			name: formData.name,
			quantity: Number(formData.quantity),
			unit: formData.unit,
			category: formData.category,
			expires_on: formData.expires_on || null
		};
	}

	async function handleSubmit() {
		const editing = form.editing;
		await form.submit(async () => {
			if (editing) {
				await api.put(`/api/pantry/${editing.id}`, payload());
			} else {
				await api.post('/api/pantry', payload());
			}
			await invalidateAll();
		});
	}

	async function handleDelete(item: PantryItem) {
		const ok = await confirm({
			title: 'Usunąć produkt?',
			message: `Czy na pewno usunąć „${item.name}” ze spiżarni?`,
			danger: true,
			confirmText: 'Usuń'
		});
		if (!ok) return;
		try {
			await api.del(`/api/pantry/${item.id}`);
			await invalidateAll();
			toast.success('Produkt usunięty');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Nie udało się usunąć');
		}
	}

	function isExpiringSoon(expires: string | null): boolean {
		if (!expires) return false;
		return (new Date(expires).getTime() - Date.now()) / 86_400_000 <= 7;
	}
</script>

<div class="space-y-6">
	<header class="flex items-center justify-between gap-4 flex-wrap">
		<div class="flex items-center gap-3">
			<Refrigerator class="text-primary-500" size={28} />
			<div>
				<h1 class="h2 font-bold">Spiżarnia</h1>
				<p class="text-sm text-surface-700-300">{items.length} produktów</p>
			</div>
		</div>
		<button type="button" class="btn preset-filled-primary-500" onclick={() => form.openCreate()}>
			<Plus size={18} />
			<span>Dodaj produkt</span>
		</button>
	</header>

	{#if items.length === 0}
		<div class="card preset-tonal-surface p-8 text-center">
			<Refrigerator class="mx-auto mb-3 text-surface-500" size={40} />
			<p class="text-surface-700-300">Spiżarnia jest pusta. Dodaj pierwszy produkt.</p>
		</div>
	{:else}
		<div class="card preset-filled-surface-50-950 p-4 table-cards">
			<table class="table">
				<thead>
					<tr>
						<th>Produkt</th>
						<th>Ilość</th>
						<th>Kategoria</th>
						<th>Termin</th>
						<th></th>
					</tr>
				</thead>
				<tbody>
					{#each items as item (item.id)}
						<tr>
							<td data-label="Produkt" class="font-semibold">{item.name}</td>
							<td data-label="Ilość">{formatQuantity(item.quantity, item.unit)}</td>
							<td data-label="Kategoria">
								<span class="badge preset-tonal-surface">{item.category}</span>
							</td>
							<td data-label="Termin">
								{#if item.expires_on}
									<span
										class={isExpiringSoon(item.expires_on)
											? 'text-warning-600-400 font-semibold'
											: ''}
									>
										{formatDate(item.expires_on)}
									</span>
								{:else}
									—
								{/if}
							</td>
							<td>
								<div class="flex gap-1 justify-end">
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Edytuj"
										onclick={() => form.openEdit(item)}
									>
										<Pencil size={16} />
									</button>
									<button
										type="button"
										class="btn-icon btn-icon-sm preset-tonal-error"
										aria-label="Usuń"
										onclick={() => handleDelete(item)}
									>
										<Trash2 size={16} />
									</button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<Modal
	open={form.open}
	title={form.isEditing ? 'Edytuj produkt' : 'Dodaj produkt'}
	confirmText={form.saving ? 'Zapisywanie...' : 'Zapisz'}
	confirmDisabled={form.saving || !formData.name}
	onConfirm={handleSubmit}
	onCancel={() => form.close()}
>
	<form class="space-y-3" onsubmit={(e) => e.preventDefault()}>
		<label class="label">
			<span>Nazwa</span>
			<input class="input" type="text" bind:value={formData.name} required />
		</label>
		<div class="grid grid-cols-2 gap-3">
			<label class="label">
				<span>Ilość</span>
				<input class="input" type="number" min="0" step="0.01" bind:value={formData.quantity} />
			</label>
			<label class="label">
				<span>Jednostka</span>
				<input class="input" type="text" bind:value={formData.unit} placeholder="szt, kg, l" />
			</label>
		</div>
		<label class="label">
			<span>Kategoria</span>
			<select class="select" bind:value={formData.category}>
				{#each CATEGORIES as category (category)}
					<option value={category}>{category}</option>
				{/each}
			</select>
		</label>
		<label class="label">
			<span>Termin przydatności (opcjonalnie)</span>
			<input class="input" type="date" bind:value={formData.expires_on} />
		</label>
		{#if form.error}
			<div class="card preset-tonal-error-500 p-2 text-sm" role="alert">{form.error}</div>
		{/if}
	</form>
</Modal>

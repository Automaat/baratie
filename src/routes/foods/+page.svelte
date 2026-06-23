<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import { api } from '$lib/apiClient';
	import { invalidateAll } from '$app/navigation';
	import { CrudForm } from '$lib/stores/crudForm.svelte';
	import { confirm } from '$lib/stores/confirm.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { formatQuantity } from '$lib/utils/format';
	import { Apple, Plus, Pencil, Trash2 } from 'lucide-svelte';
	import type { Food } from './+page';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
	const foods = $derived(data.foods);

	const form = new CrudForm<Food>();

	const emptyForm = () => ({
		name: '',
		kcal_per_100g: 0,
		protein_per_100g: 0,
		carbs_per_100g: 0,
		fat_per_100g: 0
	});

	let formData = $state(emptyForm());

	$effect(() => {
		const editing = form.editing;
		if (editing) {
			formData = {
				name: editing.name,
				kcal_per_100g: editing.kcal_per_100g,
				protein_per_100g: editing.protein_per_100g,
				carbs_per_100g: editing.carbs_per_100g,
				fat_per_100g: editing.fat_per_100g
			};
		} else if (form.open) {
			formData = emptyForm();
		}
	});

	function payload() {
		return {
			name: formData.name,
			kcal_per_100g: Number(formData.kcal_per_100g),
			protein_per_100g: Number(formData.protein_per_100g),
			carbs_per_100g: Number(formData.carbs_per_100g),
			fat_per_100g: Number(formData.fat_per_100g)
		};
	}

	async function handleSubmit() {
		const editing = form.editing;
		await form.submit(async () => {
			if (editing) {
				await api.put(`/api/foods/${editing.id}`, payload());
			} else {
				await api.post('/api/foods', payload());
			}
			await invalidateAll();
		});
	}

	async function handleDelete(food: Food) {
		const ok = await confirm({
			title: 'Usunąć produkt?',
			message: `Czy na pewno usunąć „${food.name}” z bazy produktów?`,
			danger: true,
			confirmText: 'Usuń'
		});
		if (!ok) return;
		try {
			await api.del(`/api/foods/${food.id}`);
			await invalidateAll();
			toast.success('Produkt usunięty');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Nie udało się usunąć');
		}
	}
</script>

<div class="space-y-6">
	<header class="flex items-center justify-between gap-4 flex-wrap">
		<div class="flex items-center gap-3">
			<Apple class="text-primary-500" size={28} />
			<div>
				<h1 class="h2 font-bold">Baza produktów</h1>
				<p class="text-sm text-surface-700-300">{foods.length} produktów · wartości na 100 g</p>
			</div>
		</div>
		<button type="button" class="btn preset-filled-primary-500" onclick={() => form.openCreate()}>
			<Plus size={18} />
			<span>Nowy produkt</span>
		</button>
	</header>

	{#if foods.length === 0}
		<div class="card preset-tonal-surface p-8 text-center">
			<Apple class="mx-auto mb-3 text-surface-500" size={40} />
			<p class="text-surface-700-300">Brak produktów. Dodaj pierwszy produkt do bazy.</p>
		</div>
	{:else}
		<div class="card preset-filled-surface-50-950 p-4 table-cards">
			<table class="table">
				<thead>
					<tr>
						<th>Produkt</th>
						<th>kcal</th>
						<th>Białko</th>
						<th>Węgl.</th>
						<th>Tłuszcz</th>
						<th></th>
					</tr>
				</thead>
				<tbody>
					{#each foods as food (food.id)}
						<tr>
							<td data-label="Produkt" class="font-semibold">{food.name}</td>
							<td data-label="kcal">{formatQuantity(food.kcal_per_100g)}</td>
							<td data-label="Białko">{formatQuantity(food.protein_per_100g)} g</td>
							<td data-label="Węgl.">{formatQuantity(food.carbs_per_100g)} g</td>
							<td data-label="Tłuszcz">{formatQuantity(food.fat_per_100g)} g</td>
							<td>
								<div class="flex gap-1 justify-end">
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Edytuj"
										onclick={() => form.openEdit(food)}
									>
										<Pencil size={16} />
									</button>
									<button
										type="button"
										class="btn-icon btn-icon-sm preset-tonal-error"
										aria-label="Usuń"
										onclick={() => handleDelete(food)}
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
	title={form.isEditing ? 'Edytuj produkt' : 'Nowy produkt'}
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
		<p class="text-sm text-surface-700-300">Wartości odżywcze na 100 g</p>
		<div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
			<label class="label">
				<span>Kalorie</span>
				<input class="input" type="number" min="0" step="1" bind:value={formData.kcal_per_100g} />
			</label>
			<label class="label">
				<span>Białko (g)</span>
				<input
					class="input"
					type="number"
					min="0"
					step="0.1"
					bind:value={formData.protein_per_100g}
				/>
			</label>
			<label class="label">
				<span>Węgl. (g)</span>
				<input
					class="input"
					type="number"
					min="0"
					step="0.1"
					bind:value={formData.carbs_per_100g}
				/>
			</label>
			<label class="label">
				<span>Tłuszcz (g)</span>
				<input class="input" type="number" min="0" step="0.1" bind:value={formData.fat_per_100g} />
			</label>
		</div>
		{#if form.error}
			<div class="card preset-tonal-error-500 p-2 text-sm" role="alert">{form.error}</div>
		{/if}
	</form>
</Modal>

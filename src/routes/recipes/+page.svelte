<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import { api } from '$lib/apiClient';
	import { invalidateAll } from '$app/navigation';
	import { CrudForm } from '$lib/stores/crudForm.svelte';
	import { confirm } from '$lib/stores/confirm.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { formatMinutes, formatQuantity } from '$lib/utils/format';
	import { BookOpen, Plus, Pencil, Trash2, Clock, Users, Flame, X } from 'lucide-svelte';
	import type { Recipe } from './+page';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
	const recipes = $derived(data.recipes);
	const foods = $derived(data.foods);

	// Units the backend understands for structured ingredients.
	const UNITS = ['g', 'kg', 'mg', 'ml', 'l', 'szt'];

	type IngredientRow = { key: number; food_id: number; amount: number; unit: string };
	let structuredRows = $state<IngredientRow[]>([]);
	let rowKey = 0;

	// Whether the structured ingredients would yield computed macros — mirrors
	// the backend rule (mass unit + amount + a food carrying macro data). When
	// true the manual macro fields are overwritten on save, so disable them.
	const computedUsable = $derived(
		structuredRows.some((row) => {
			if (Number(row.food_id) <= 0 || Number(row.amount) <= 0 || row.unit === 'szt') return false;
			const food = foods.find((f) => f.id === Number(row.food_id));
			return (
				!!food &&
				(food.kcal_per_100g > 0 ||
					food.protein_per_100g > 0 ||
					food.carbs_per_100g > 0 ||
					food.fat_per_100g > 0)
			);
		})
	);

	const form = new CrudForm<Recipe>();

	const emptyForm = () => ({
		name: '',
		description: '',
		instructions: '',
		ingredients: '',
		tags: '',
		servings: 2,
		prep_minutes: 0,
		cook_minutes: 0,
		calories_kcal: 0,
		protein_g: 0,
		carbs_g: 0,
		fat_g: 0
	});

	let formData = $state(emptyForm());

	$effect(() => {
		const editing = form.editing;
		if (editing) {
			formData = {
				name: editing.name,
				description: editing.description,
				instructions: editing.instructions,
				ingredients: editing.ingredients.join('\n'),
				tags: editing.tags.join(', '),
				servings: editing.servings,
				prep_minutes: editing.prep_minutes,
				cook_minutes: editing.cook_minutes,
				calories_kcal: editing.calories_kcal,
				protein_g: editing.protein_g,
				carbs_g: editing.carbs_g,
				fat_g: editing.fat_g
			};
			structuredRows = editing.ingredients_structured.map((i) => ({
				key: rowKey++,
				food_id: i.food_id,
				amount: i.amount,
				unit: i.unit
			}));
		} else if (form.open) {
			formData = emptyForm();
			structuredRows = [];
		}
	});

	function addIngredient() {
		structuredRows = [...structuredRows, { key: rowKey++, food_id: 0, amount: 0, unit: 'g' }];
	}

	function removeIngredient(index: number) {
		structuredRows = structuredRows.filter((_, i) => i !== index);
	}

	// Structured rows with a chosen food, mapped to the API shape.
	function ingredientPayload() {
		return structuredRows
			.filter((r) => Number(r.food_id) > 0)
			.map((r) => ({ food_id: Number(r.food_id), amount: Number(r.amount), unit: r.unit }));
	}

	function payload() {
		return {
			name: formData.name,
			description: formData.description,
			instructions: formData.instructions,
			ingredients: formData.ingredients
				.split('\n')
				.map((s) => s.trim())
				.filter(Boolean),
			tags: formData.tags
				.split(',')
				.map((s) => s.trim())
				.filter(Boolean),
			servings: Number(formData.servings),
			prep_minutes: Number(formData.prep_minutes),
			cook_minutes: Number(formData.cook_minutes),
			calories_kcal: Number(formData.calories_kcal),
			protein_g: Number(formData.protein_g),
			carbs_g: Number(formData.carbs_g),
			fat_g: Number(formData.fat_g)
		};
	}

	async function handleSubmit() {
		const editing = form.editing;
		await form.submit(async () => {
			const ingredients = ingredientPayload();
			if (editing) {
				await api.put(`/api/recipes/${editing.id}`, payload());
				// Full-replace structured ingredients (applies adds, edits, removals).
				await api.put(`/api/recipes/${editing.id}/ingredients`, { ingredients });
			} else {
				const created = await api.post<Recipe>('/api/recipes', payload());
				if (ingredients.length > 0) {
					await api.put(`/api/recipes/${created.id}/ingredients`, { ingredients });
				}
			}
			await invalidateAll();
		});
	}

	async function handleDelete(recipe: Recipe) {
		const ok = await confirm({
			title: 'Usunąć przepis?',
			message: `Czy na pewno usunąć „${recipe.name}”? Tej operacji nie da się cofnąć.`,
			danger: true,
			confirmText: 'Usuń'
		});
		if (!ok) return;
		try {
			await api.del(`/api/recipes/${recipe.id}`);
			await invalidateAll();
			toast.success('Przepis usunięty');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Nie udało się usunąć');
		}
	}
</script>

<div class="space-y-6">
	<header class="flex items-center justify-between gap-4 flex-wrap">
		<div class="flex items-center gap-3">
			<BookOpen class="text-primary-500" size={28} />
			<div>
				<h1 class="h2 font-bold">Przepisy</h1>
				<p class="text-sm text-surface-700-300">{recipes.length} w kolekcji</p>
			</div>
		</div>
		<button type="button" class="btn preset-filled-primary-500" onclick={() => form.openCreate()}>
			<Plus size={18} />
			<span>Nowy przepis</span>
		</button>
	</header>

	{#if recipes.length === 0}
		<div class="card preset-tonal-surface p-8 text-center">
			<BookOpen class="mx-auto mb-3 text-surface-500" size={40} />
			<p class="text-surface-700-300">Brak przepisów. Dodaj swój pierwszy przepis.</p>
		</div>
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
			{#each recipes as recipe (recipe.id)}
				<article class="card preset-tonal-surface p-4 space-y-3 flex flex-col">
					<div class="flex items-start justify-between gap-2">
						<h2 class="h5 font-bold min-w-0">{recipe.name}</h2>
						<div class="flex gap-1 shrink-0">
							<button
								type="button"
								class="btn-icon btn-icon-sm"
								aria-label="Edytuj"
								onclick={() => form.openEdit(recipe)}
							>
								<Pencil size={16} />
							</button>
							<button
								type="button"
								class="btn-icon btn-icon-sm preset-tonal-error"
								aria-label="Usuń"
								onclick={() => handleDelete(recipe)}
							>
								<Trash2 size={16} />
							</button>
						</div>
					</div>

					{#if recipe.description}
						<p class="text-sm text-surface-700-300 line-clamp-2">{recipe.description}</p>
					{/if}

					<div class="flex flex-wrap gap-3 text-xs text-surface-700-300">
						<span class="flex items-center gap-1"><Users size={14} /> {recipe.servings} porcji</span
						>
						<span class="flex items-center gap-1"
							><Clock size={14} /> {formatMinutes(recipe.total_minutes)}</span
						>
						<span>{recipe.ingredients.length} składników</span>
					</div>

					{#if recipe.calories_kcal > 0 || recipe.protein_g > 0 || recipe.carbs_g > 0 || recipe.fat_g > 0}
						<div class="flex flex-wrap gap-3 text-xs text-surface-700-300">
							<span class="flex items-center gap-1">
								<Flame size={14} />
								{formatQuantity(recipe.calories_kcal)} kcal
							</span>
							<span>B {formatQuantity(recipe.protein_g)} g</span>
							<span>W {formatQuantity(recipe.carbs_g)} g</span>
							<span>T {formatQuantity(recipe.fat_g)} g</span>
							<span class="text-surface-500">na porcję</span>
						</div>
					{/if}

					{#if recipe.tags.length > 0}
						<div class="flex flex-wrap gap-1 mt-auto">
							{#each recipe.tags as tag (tag)}
								<span class="badge preset-tonal-primary text-xs">{tag}</span>
							{/each}
						</div>
					{/if}
				</article>
			{/each}
		</div>
	{/if}
</div>

<Modal
	open={form.open}
	title={form.isEditing ? 'Edytuj przepis' : 'Nowy przepis'}
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
		<label class="label">
			<span>Opis</span>
			<textarea class="textarea" rows="2" bind:value={formData.description}></textarea>
		</label>
		<div class="grid grid-cols-3 gap-3">
			<label class="label">
				<span>Porcje</span>
				<input class="input" type="number" min="1" bind:value={formData.servings} />
			</label>
			<label class="label">
				<span>Przygot. (min)</span>
				<input class="input" type="number" min="0" bind:value={formData.prep_minutes} />
			</label>
			<label class="label">
				<span>Gotowanie (min)</span>
				<input class="input" type="number" min="0" bind:value={formData.cook_minutes} />
			</label>
		</div>
		<label class="label">
			<span>Składniki (jeden na linię)</span>
			<textarea class="textarea" rows="4" bind:value={formData.ingredients}></textarea>
		</label>
		<fieldset class="space-y-2">
			<legend class="text-sm font-semibold">Składniki strukturalne (z bazy produktów)</legend>
			{#if foods.length === 0}
				<p class="text-sm text-surface-700-300">
					Brak produktów w bazie. Dodaj je w
					<a href="/foods" class="underline">Bazie produktów</a>, aby liczyć makro automatycznie.
				</p>
			{:else}
				{#each structuredRows as row, i (row.key)}
					<div class="flex gap-2 items-end">
						<label class="label flex-1">
							<span class="sr-only">Produkt</span>
							<select class="select" bind:value={row.food_id}>
								<option value={0} disabled>Wybierz produkt…</option>
								{#each foods as food (food.id)}
									<option value={food.id}>{food.name}</option>
								{/each}
							</select>
						</label>
						<label class="label w-24">
							<span class="sr-only">Ilość</span>
							<input class="input" type="number" min="0" step="0.1" bind:value={row.amount} />
						</label>
						<label class="label w-20">
							<span class="sr-only">Jednostka</span>
							<select class="select" bind:value={row.unit}>
								{#each UNITS as u (u)}
									<option value={u}>{u}</option>
								{/each}
							</select>
						</label>
						<button
							type="button"
							class="btn-icon btn-icon-sm preset-tonal-error"
							aria-label="Usuń składnik"
							onclick={() => removeIngredient(i)}
						>
							<X size={16} />
						</button>
					</div>
				{/each}
				<button type="button" class="btn btn-sm preset-tonal-surface" onclick={addIngredient}>
					<Plus size={16} />
					<span>Dodaj składnik</span>
				</button>
			{/if}
		</fieldset>
		<label class="label">
			<span>Instrukcje</span>
			<textarea class="textarea" rows="4" bind:value={formData.instructions}></textarea>
		</label>
		<label class="label">
			<span>Tagi (po przecinku)</span>
			<input
				class="input"
				type="text"
				bind:value={formData.tags}
				placeholder="szybkie, wegańskie"
			/>
		</label>
		<fieldset class="space-y-2">
			<legend class="text-sm font-semibold">Wartości odżywcze (na porcję)</legend>
			{#if computedUsable}
				<p class="text-xs text-surface-700-300">
					Przeliczane automatycznie ze składników strukturalnych.
				</p>
			{/if}
			<div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
				<label class="label">
					<span>Kalorie (kcal)</span>
					<input
						class="input"
						type="number"
						min="0"
						step="1"
						disabled={computedUsable}
						bind:value={formData.calories_kcal}
					/>
				</label>
				<label class="label">
					<span>Białko (g)</span>
					<input
						class="input"
						type="number"
						min="0"
						step="0.1"
						disabled={computedUsable}
						bind:value={formData.protein_g}
					/>
				</label>
				<label class="label">
					<span>Węgl. (g)</span>
					<input
						class="input"
						type="number"
						min="0"
						step="0.1"
						disabled={computedUsable}
						bind:value={formData.carbs_g}
					/>
				</label>
				<label class="label">
					<span>Tłuszcz (g)</span>
					<input
						class="input"
						type="number"
						min="0"
						step="0.1"
						disabled={computedUsable}
						bind:value={formData.fat_g}
					/>
				</label>
			</div>
		</fieldset>
		{#if form.error}
			<div class="card preset-tonal-error-500 p-2 text-sm" role="alert">{form.error}</div>
		{/if}
	</form>
</Modal>

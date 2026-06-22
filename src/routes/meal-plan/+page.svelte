<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import { api } from '$lib/apiClient';
	import { invalidateAll } from '$app/navigation';
	import { CrudForm } from '$lib/stores/crudForm.svelte';
	import { confirm } from '$lib/stores/confirm.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { formatDate, mealTypeLabel, MEAL_TYPE_LABELS } from '$lib/utils/format';
	import { CalendarDays, Plus, Pencil, Trash2 } from 'lucide-svelte';
	import type { MealEntry } from './+page';
	import type { Recipe } from '../recipes/+page';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
	const entries = $derived(data.entries);
	const recipes = $derived(data.recipes as Recipe[]);

	const grouped = $derived.by(() => {
		const map = new Map<string, MealEntry[]>();
		for (const entry of entries) {
			const list = map.get(entry.plan_date) ?? [];
			list.push(entry);
			map.set(entry.plan_date, list);
		}
		return [...map.entries()].sort((a, b) => a[0].localeCompare(b[0]));
	});

	const form = new CrudForm<MealEntry>();

	const emptyForm = () => ({
		plan_date: data.today,
		meal_type: 'dinner',
		recipe_id: null as number | null,
		note: ''
	});

	let formData = $state(emptyForm());

	$effect(() => {
		const editing = form.editing;
		if (editing) {
			formData = {
				plan_date: editing.plan_date,
				meal_type: editing.meal_type,
				recipe_id: editing.recipe_id,
				note: editing.note
			};
		} else if (form.open) {
			formData = emptyForm();
		}
	});

	function payload() {
		return {
			plan_date: formData.plan_date,
			meal_type: formData.meal_type,
			recipe_id: formData.recipe_id ? Number(formData.recipe_id) : null,
			note: formData.note
		};
	}

	async function handleSubmit() {
		const editing = form.editing;
		await form.submit(async () => {
			if (editing) {
				await api.put(`/api/meal-plan/${editing.id}`, payload());
			} else {
				await api.post('/api/meal-plan', payload());
			}
			await invalidateAll();
		});
	}

	async function handleDelete(entry: MealEntry) {
		const ok = await confirm({
			title: 'Usunąć posiłek?',
			message: 'Czy na pewno usunąć ten zaplanowany posiłek?',
			danger: true,
			confirmText: 'Usuń'
		});
		if (!ok) return;
		try {
			await api.del(`/api/meal-plan/${entry.id}`);
			await invalidateAll();
			toast.success('Posiłek usunięty');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Nie udało się usunąć');
		}
	}

	function label(entry: MealEntry): string {
		return entry.recipe_name ?? (entry.note || '—');
	}
</script>

<div class="space-y-6">
	<header class="flex items-center justify-between gap-4 flex-wrap">
		<div class="flex items-center gap-3">
			<CalendarDays class="text-primary-500" size={28} />
			<div>
				<h1 class="h2 font-bold">Plan posiłków</h1>
				<p class="text-sm text-surface-700-300">Najbliższe 2 tygodnie</p>
			</div>
		</div>
		<button type="button" class="btn preset-filled-primary-500" onclick={() => form.openCreate()}>
			<Plus size={18} />
			<span>Zaplanuj posiłek</span>
		</button>
	</header>

	{#if grouped.length === 0}
		<div class="card preset-tonal-surface p-8 text-center">
			<CalendarDays class="mx-auto mb-3 text-surface-500" size={40} />
			<p class="text-surface-700-300">Brak zaplanowanych posiłków.</p>
		</div>
	{:else}
		<div class="space-y-4">
			{#each grouped as [date, dayEntries] (date)}
				<section class="card preset-tonal-surface p-4 space-y-2">
					<h2 class="font-bold text-sm text-surface-700-300">{formatDate(date)}</h2>
					<ul class="space-y-1">
						{#each dayEntries as entry (entry.id)}
							<li class="flex items-center justify-between gap-3 py-1">
								<div class="min-w-0 flex items-center gap-2">
									<span class="badge preset-tonal-primary text-xs shrink-0"
										>{mealTypeLabel(entry.meal_type)}</span
									>
									<span class="truncate">{label(entry)}</span>
								</div>
								<div class="flex gap-1 shrink-0">
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Edytuj"
										onclick={() => form.openEdit(entry)}
									>
										<Pencil size={16} />
									</button>
									<button
										type="button"
										class="btn-icon btn-icon-sm preset-tonal-error"
										aria-label="Usuń"
										onclick={() => handleDelete(entry)}
									>
										<Trash2 size={16} />
									</button>
								</div>
							</li>
						{/each}
					</ul>
				</section>
			{/each}
		</div>
	{/if}
</div>

<Modal
	open={form.open}
	title={form.isEditing ? 'Edytuj posiłek' : 'Zaplanuj posiłek'}
	confirmText={form.saving ? 'Zapisywanie...' : 'Zapisz'}
	confirmDisabled={form.saving || !formData.plan_date}
	onConfirm={handleSubmit}
	onCancel={() => form.close()}
>
	<form class="space-y-3" onsubmit={(e) => e.preventDefault()}>
		<div class="grid grid-cols-2 gap-3">
			<label class="label">
				<span>Data</span>
				<input class="input" type="date" bind:value={formData.plan_date} required />
			</label>
			<label class="label">
				<span>Posiłek</span>
				<select class="select" bind:value={formData.meal_type}>
					{#each Object.entries(MEAL_TYPE_LABELS) as [value, text] (value)}
						<option {value}>{text}</option>
					{/each}
				</select>
			</label>
		</div>
		<label class="label">
			<span>Przepis (opcjonalnie)</span>
			<select class="select" bind:value={formData.recipe_id}>
				<option value={null}>—</option>
				{#each recipes as recipe (recipe.id)}
					<option value={recipe.id}>{recipe.name}</option>
				{/each}
			</select>
		</label>
		<label class="label">
			<span>Notatka</span>
			<input
				class="input"
				type="text"
				bind:value={formData.note}
				placeholder="np. resztki z obiadu"
			/>
		</label>
		{#if form.error}
			<div class="card preset-tonal-error-500 p-2 text-sm" role="alert">{form.error}</div>
		{/if}
	</form>
</Modal>

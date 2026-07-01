<script lang="ts">
	import { formatMinutes, formatQuantity } from '$lib/utils/format';
	import { ArrowLeft, BookOpen, Clock, Flame, ListChecks, Scale, Tags, Users } from 'lucide-svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const recipe = $derived(data.recipe);
	const instructionSteps = $derived(
		recipe.instructions
			.split('\n')
			.map((line) => line.trim())
			.filter(Boolean)
	);
	const hasNutrition = $derived(
		recipe.calories_kcal > 0 || recipe.protein_g > 0 || recipe.carbs_g > 0 || recipe.fat_g > 0
	);
</script>

<div class="space-y-6">
	<header class="space-y-4">
		<a href="/recipes" class="btn btn-sm preset-tonal-surface w-fit">
			<ArrowLeft size={16} />
			<span>Przepisy</span>
		</a>

		<div class="flex flex-col gap-3">
			<div class="flex items-center gap-3 text-primary-500">
				<BookOpen size={28} />
				<h1 class="h2 font-bold text-surface-950-50">{recipe.name}</h1>
			</div>
			{#if recipe.description}
				<p class="max-w-3xl text-surface-700-300">{recipe.description}</p>
			{/if}
		</div>
	</header>

	<section class="grid grid-cols-2 lg:grid-cols-4 gap-3" aria-label="Szczegóły przepisu">
		<div class="card preset-tonal-surface p-4 space-y-1">
			<div class="flex items-center gap-2 text-sm text-surface-700-300">
				<Users size={16} />
				<span>Porcje</span>
			</div>
			<p class="h4 font-bold">{recipe.servings}</p>
		</div>
		<div class="card preset-tonal-surface p-4 space-y-1">
			<div class="flex items-center gap-2 text-sm text-surface-700-300">
				<Clock size={16} />
				<span>Czas</span>
			</div>
			<p class="h4 font-bold">{formatMinutes(recipe.total_minutes)}</p>
		</div>
		<div class="card preset-tonal-surface p-4 space-y-1">
			<div class="flex items-center gap-2 text-sm text-surface-700-300">
				<Scale size={16} />
				<span>Przygot.</span>
			</div>
			<p class="h4 font-bold">{formatMinutes(recipe.prep_minutes)}</p>
		</div>
		<div class="card preset-tonal-surface p-4 space-y-1">
			<div class="flex items-center gap-2 text-sm text-surface-700-300">
				<Flame size={16} />
				<span>Gotowanie</span>
			</div>
			<p class="h4 font-bold">{formatMinutes(recipe.cook_minutes)}</p>
		</div>
	</section>

	<div class="grid grid-cols-1 lg:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)] gap-6">
		<div class="space-y-6">
			<section class="card preset-tonal-surface p-5 space-y-4">
				<div class="flex items-center gap-2">
					<ListChecks class="text-primary-500" size={20} />
					<h2 class="h4 font-bold">Składniki</h2>
				</div>

				{#if recipe.ingredients_structured.length > 0}
					<ul class="space-y-2">
						{#each recipe.ingredients_structured as ingredient (ingredient.id)}
							<li
								class="flex items-baseline justify-between gap-4 border-b border-surface-200-800 pb-2 last:border-b-0 last:pb-0"
							>
								<span class="font-medium">{ingredient.food_name}</span>
								<span class="text-sm text-surface-700-300 whitespace-nowrap">
									{formatQuantity(ingredient.amount, ingredient.unit)}
								</span>
							</li>
						{/each}
					</ul>
				{:else if recipe.ingredients.length > 0}
					<ul class="list-disc space-y-2 pl-5">
						{#each recipe.ingredients as ingredient, index (index)}
							<li>{ingredient}</li>
						{/each}
					</ul>
				{:else}
					<p class="text-sm text-surface-700-300">Brak zapisanych składników.</p>
				{/if}
			</section>

			{#if hasNutrition}
				<section class="card preset-tonal-surface p-5 space-y-4">
					<div class="flex items-center gap-2">
						<Flame class="text-primary-500" size={20} />
						<h2 class="h4 font-bold">Wartości odżywcze</h2>
					</div>
					<div class="grid grid-cols-2 gap-3 text-sm">
						<div>
							<p class="text-surface-700-300">Kalorie</p>
							<p class="font-bold">{formatQuantity(recipe.calories_kcal)} kcal</p>
						</div>
						<div>
							<p class="text-surface-700-300">Białko</p>
							<p class="font-bold">{formatQuantity(recipe.protein_g)} g</p>
						</div>
						<div>
							<p class="text-surface-700-300">Węglowodany</p>
							<p class="font-bold">{formatQuantity(recipe.carbs_g)} g</p>
						</div>
						<div>
							<p class="text-surface-700-300">Tłuszcz</p>
							<p class="font-bold">{formatQuantity(recipe.fat_g)} g</p>
						</div>
					</div>
					<p class="text-xs text-surface-700-300">Na porcję</p>
				</section>
			{/if}

			{#if recipe.tags.length > 0}
				<section class="card preset-tonal-surface p-5 space-y-3">
					<div class="flex items-center gap-2">
						<Tags class="text-primary-500" size={20} />
						<h2 class="h4 font-bold">Tagi</h2>
					</div>
					<div class="flex flex-wrap gap-2">
						{#each recipe.tags as tag (tag)}
							<span class="badge preset-tonal-primary">{tag}</span>
						{/each}
					</div>
				</section>
			{/if}
		</div>

		<section class="card preset-tonal-surface p-5 space-y-4">
			<h2 class="h4 font-bold">Instrukcje</h2>
			{#if instructionSteps.length > 0}
				<ol class="space-y-4">
					{#each instructionSteps as step, index (index)}
						<li class="grid grid-cols-[2rem_minmax(0,1fr)] gap-3">
							<span
								class="flex h-8 w-8 items-center justify-center rounded-container preset-filled-primary-500 text-sm font-bold"
							>
								{index + 1}
							</span>
							<p class="pt-1 leading-relaxed">{step}</p>
						</li>
					{/each}
				</ol>
			{:else}
				<p class="text-sm text-surface-700-300">Brak zapisanych instrukcji.</p>
			{/if}
		</section>
	</div>
</div>

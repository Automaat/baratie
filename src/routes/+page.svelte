<script lang="ts">
	import MetricCard from '$lib/components/MetricCard.svelte';
	import CategoryChart from '$lib/components/CategoryChart.svelte';
	import { formatDate, mealTypeLabel } from '$lib/utils/format';
	import { LayoutDashboard, CalendarDays, BookOpen } from 'lucide-svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const recipes = $derived(data.recipes);
	const pantry = $derived(data.pantry);
	const meals = $derived(data.meals);

	// Pantry items expiring within 7 days (inclusive of today).
	const expiringSoon = $derived(
		pantry.filter((item) => {
			if (!item.expires_on) return false;
			const days = (new Date(item.expires_on).getTime() - Date.now()) / 86_400_000;
			return days <= 7;
		}).length
	);

	const pantryByCategory = $derived.by(() => {
		const counts = new Map<string, number>();
		for (const item of pantry) {
			counts.set(item.category, (counts.get(item.category) ?? 0) + 1);
		}
		return [...counts.entries()].map(([name, value]) => ({ name, value }));
	});

	const upcoming = $derived(
		[...meals].sort((a, b) => a.plan_date.localeCompare(b.plan_date)).slice(0, 8)
	);
</script>

<div class="space-y-6">
	<header class="flex items-center gap-3">
		<LayoutDashboard class="text-primary-500" size={28} />
		<h1 class="h2 font-bold">Pulpit</h1>
	</header>

	<div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
		<MetricCard label="Przepisy" value={recipes.length} color="blue" />
		<MetricCard label="W spiżarni" value={pantry.length} />
		<MetricCard
			label="Kończy się"
			value={expiringSoon}
			color={expiringSoon > 0 ? 'yellow' : 'neutral'}
			secondary="w ciągu 7 dni"
		/>
		<MetricCard label="Posiłki w tym tygodniu" value={meals.length} color="green" />
	</div>

	<div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
		<section class="card preset-tonal-surface p-4 space-y-3">
			<h2 class="h5 font-bold flex items-center gap-2">
				<CalendarDays size={18} />
				Najbliższe posiłki
			</h2>
			{#if upcoming.length === 0}
				<p class="text-sm text-surface-700-300">
					Brak zaplanowanych posiłków. Zajrzyj do <a href="/meal-plan" class="underline"
						>planu posiłków</a
					>.
				</p>
			{:else}
				<ul class="space-y-2">
					{#each upcoming as meal (meal.id)}
						<li
							class="flex items-center justify-between gap-3 text-sm border-b border-surface-200-800 pb-2 last:border-0"
						>
							<div class="min-w-0">
								<span class="font-semibold">{meal.recipe_name ?? (meal.note || '—')}</span>
								<span class="text-surface-700-300"> · {mealTypeLabel(meal.meal_type)}</span>
							</div>
							<span class="text-surface-700-300 shrink-0">{formatDate(meal.plan_date)}</span>
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<section class="card preset-tonal-surface p-4 space-y-3">
			<h2 class="h5 font-bold flex items-center gap-2">
				<BookOpen size={18} />
				Spiżarnia wg kategorii
			</h2>
			{#if pantryByCategory.length === 0}
				<p class="text-sm text-surface-700-300">Spiżarnia jest pusta.</p>
			{:else}
				<CategoryChart data={pantryByCategory} />
			{/if}
		</section>
	</div>
</div>

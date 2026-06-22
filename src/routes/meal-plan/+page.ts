import { error } from '@sveltejs/kit';
import { api, ApiError } from '$lib/apiClient';
import type { Recipe } from '../recipes/+page';
import type { PageLoad } from './$types';

export interface MealEntry {
	id: number;
	plan_date: string;
	meal_type: string;
	recipe_id: number | null;
	recipe_name: string | null;
	note: string;
	created_at: string;
}

function isoDate(d: Date): string {
	return d.toISOString().slice(0, 10);
}

export const load: PageLoad = async ({ fetch }) => {
	const today = new Date();
	const start = new Date(today);
	const end = new Date(today);
	end.setDate(today.getDate() + 13);

	try {
		const [entries, recipes] = await Promise.all([
			api.get<MealEntry[]>('/api/meal-plan', {
				fetch,
				query: { date_from: isoDate(start), date_to: isoDate(end) }
			}),
			// Recipes power the picker; degrade to empty rather than failing the page.
			api.get<Recipe[]>('/api/recipes', { fetch }).catch(() => [] as Recipe[])
		]);
		return { entries, recipes, today: isoDate(today) };
	} catch (err) {
		if (err instanceof ApiError) {
			throw error(err.status, 'Nie udało się załadować planu posiłków');
		}
		throw err;
	}
};

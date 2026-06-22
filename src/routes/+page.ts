import { error } from '@sveltejs/kit';
import { api, ApiError } from '$lib/apiClient';
import type { Recipe } from './recipes/+page';
import type { PantryItem } from './pantry/+page';
import type { MealEntry } from './meal-plan/+page';
import type { PageLoad } from './$types';

function isoDate(d: Date): string {
	return d.toISOString().slice(0, 10);
}

export const load: PageLoad = async ({ fetch }) => {
	const today = new Date();
	const weekEnd = new Date(today);
	weekEnd.setDate(today.getDate() + 6);

	try {
		const [recipes, pantry, meals] = await Promise.all([
			api.get<Recipe[]>('/api/recipes', { fetch }),
			api.get<PantryItem[]>('/api/pantry', { fetch }),
			api.get<MealEntry[]>('/api/meal-plan', {
				fetch,
				query: { date_from: isoDate(today), date_to: isoDate(weekEnd) }
			})
		]);
		return { recipes, pantry, meals, today: isoDate(today) };
	} catch (err) {
		if (err instanceof ApiError) {
			throw error(err.status, 'Nie udało się załadować pulpitu');
		}
		throw err;
	}
};

import { error } from '@sveltejs/kit';
import { api, ApiError } from '$lib/apiClient';
import type { Food } from '../foods/+page';
import type { PageLoad } from './$types';

export interface RecipeIngredient {
	id: number;
	food_id: number;
	food_name: string;
	amount: number;
	unit: string;
	kcal_per_100g: number;
	protein_per_100g: number;
	carbs_per_100g: number;
	fat_per_100g: number;
}

export interface Recipe {
	id: number;
	name: string;
	description: string;
	instructions: string;
	ingredients: string[];
	tags: string[];
	servings: number;
	prep_minutes: number;
	cook_minutes: number;
	total_minutes: number;
	calories_kcal: number;
	protein_g: number;
	carbs_g: number;
	fat_g: number;
	ingredients_structured: RecipeIngredient[];
	created_at: string;
}

export const load: PageLoad = async ({ fetch }) => {
	try {
		const [recipes, foods] = await Promise.all([
			api.get<Recipe[]>('/api/recipes', { fetch }),
			// Foods power the structured-ingredient picker; degrade to empty.
			api.get<Food[]>('/api/foods', { fetch }).catch(() => [] as Food[])
		]);
		return { recipes, foods };
	} catch (err) {
		if (err instanceof ApiError) {
			throw error(err.status, 'Nie udało się załadować przepisów');
		}
		throw err;
	}
};

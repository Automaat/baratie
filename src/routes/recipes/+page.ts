import { error } from '@sveltejs/kit';
import { api, ApiError } from '$lib/apiClient';
import type { PageLoad } from './$types';

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
	created_at: string;
}

export const load: PageLoad = async ({ fetch }) => {
	try {
		const recipes = await api.get<Recipe[]>('/api/recipes', { fetch });
		return { recipes };
	} catch (err) {
		if (err instanceof ApiError) {
			throw error(err.status, 'Nie udało się załadować przepisów');
		}
		throw err;
	}
};

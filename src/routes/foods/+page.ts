import { error } from '@sveltejs/kit';
import { api, ApiError } from '$lib/apiClient';
import type { PageLoad } from './$types';

export interface Food {
	id: number;
	name: string;
	kcal_per_100g: number;
	protein_per_100g: number;
	carbs_per_100g: number;
	fat_per_100g: number;
	created_at: string;
}

export const load: PageLoad = async ({ fetch }) => {
	try {
		const foods = await api.get<Food[]>('/api/foods', { fetch });
		return { foods };
	} catch (err) {
		if (err instanceof ApiError) {
			throw error(err.status, 'Nie udało się załadować bazy produktów');
		}
		throw err;
	}
};

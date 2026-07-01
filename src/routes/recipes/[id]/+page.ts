import { error } from '@sveltejs/kit';
import { api, ApiError } from '$lib/apiClient';
import type { Recipe } from '../+page';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, params }) => {
	const id = Number(params.id);
	if (!Number.isInteger(id) || id <= 0) {
		throw error(404, 'Nie znaleziono przepisu');
	}

	try {
		const recipe = await api.get<Recipe>(`/api/recipes/${id}`, { fetch });
		return { recipe };
	} catch (err) {
		if (err instanceof ApiError) {
			throw error(
				err.status,
				err.status === 404 ? 'Nie znaleziono przepisu' : 'Nie udało się załadować przepisu'
			);
		}
		throw err;
	}
};

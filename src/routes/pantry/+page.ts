import { error } from '@sveltejs/kit';
import { api, ApiError } from '$lib/apiClient';
import type { PageLoad } from './$types';

export interface PantryItem {
	id: number;
	name: string;
	quantity: number;
	unit: string;
	category: string;
	expires_on: string | null;
	created_at: string;
}

export const load: PageLoad = async ({ fetch }) => {
	try {
		const items = await api.get<PantryItem[]>('/api/pantry', { fetch });
		return { items };
	} catch (err) {
		if (err instanceof ApiError) {
			throw error(err.status, 'Nie udało się załadować spiżarni');
		}
		throw err;
	}
};

import { error } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import type { PageLoad } from './$types';

export interface ApiToken {
	id: number;
	name: string;
	scope: string;
	created_at: string;
	expires_at: string | null;
	last_used_at: string | null;
}

export const load: PageLoad = async ({ fetch }) => {
	const apiUrl = resolveApiUrl();
	const response = await fetch(`${apiUrl}/api/auth/tokens`);
	if (!response.ok) {
		throw error(response.status, 'Nie udało się pobrać tokenów');
	}
	return { tokens: (await response.json()) as ApiToken[] };
};

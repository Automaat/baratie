import { error, redirect } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import type { PageLoad } from './$types';

export interface AppUser {
	id: number;
	username: string;
	is_admin: boolean;
	name: string | null;
	surname: string | null;
	created_at: string;
}

export const load: PageLoad = async ({ fetch, parent }) => {
	const { user } = await parent();
	if (!user?.isAdmin) {
		redirect(303, '/');
	}

	const apiUrl = resolveApiUrl();
	const response = await fetch(`${apiUrl}/api/auth/users`);
	if (!response.ok) {
		throw error(response.status, 'Nie udało się pobrać użytkowników');
	}
	return { users: (await response.json()) as AppUser[] };
};

import { describe, it, expect, vi, afterEach } from 'vitest';
import { cleanup, render, screen } from '@testing-library/svelte';
import LoginPage from './+page.svelte';

vi.mock('$app/forms', () => ({ enhance: () => ({ destroy: () => {} }) }));

afterEach(cleanup);

function renderLogin(
	data: { oidcEnabled: boolean; oidcError: string | null },
	form: { error?: string } | null = null
) {
	return render(LoginPage, { props: { data, form } as never });
}

describe('login page', () => {
	it('shows the Authelia button as a full-page link when OIDC is enabled', () => {
		renderLogin({ oidcEnabled: true, oidcError: null });

		const link = screen.getByRole('link', { name: /Zaloguj przez Authelię/ });
		expect(link.getAttribute('href')).toBe('/api/auth/oidc/login');
		expect(link.hasAttribute('data-sveltekit-reload')).toBe(true);
		expect(screen.getByRole('button', { name: 'Zaloguj' })).toBeTruthy();
	});

	it('hides the Authelia button when OIDC is disabled', () => {
		renderLogin({ oidcEnabled: false, oidcError: null });

		expect(screen.queryByRole('link', { name: /Authelię/ })).toBeNull();
		expect(screen.queryByRole('alert')).toBeNull();
	});

	it('shows the OIDC error message from the load data', () => {
		renderLogin({ oidcEnabled: true, oidcError: 'Authelia jest chwilowo niedostępna' });

		expect(screen.getByRole('alert').textContent).toBe('Authelia jest chwilowo niedostępna');
	});

	it('prefers the password form error over a stale OIDC error', () => {
		renderLogin(
			{ oidcEnabled: true, oidcError: 'Authelia jest chwilowo niedostępna' },
			{ error: 'Nieprawidłowa nazwa użytkownika lub hasło' }
		);

		expect(screen.getByRole('alert').textContent).toBe('Nieprawidłowa nazwa użytkownika lub hasło');
	});
});

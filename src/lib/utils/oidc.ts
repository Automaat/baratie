export const OIDC_STATE_COOKIE = 'brt_oidc_state';
export const OIDC_LOGIN_PATH = '/api/auth/oidc/login';

const OIDC_API_PATHS = new Set([
	'/api/auth/oidc/status',
	OIDC_LOGIN_PATH,
	'/api/auth/oidc/callback'
]);

// isOidcApiPath matches the OIDC endpoints exactly; a prefix match would let
// a decoded `..` segment reach other backend routes without a session.
export function isOidcApiPath(pathname: string): boolean {
	return OIDC_API_PATHS.has(pathname);
}

const OIDC_ERROR_MESSAGES: Record<string, string> = {
	oidc_no_user:
		'Brak konta Baratie dla użytkownika z Authelii — poproś administratora o jego założenie',
	oidc_failed: 'Logowanie przez Authelię nie powiodło się — spróbuj ponownie',
	oidc_unavailable: 'Authelia jest chwilowo niedostępna — zaloguj się hasłem lub spróbuj później'
};

// oidcErrorMessage maps the backend's /login?error=<code> to a Polish message;
// unknown or missing codes yield null so nothing is shown.
export function oidcErrorMessage(code: string | null): string | null {
	if (!code || !Object.hasOwn(OIDC_ERROR_MESSAGES, code)) {
		return null;
	}
	return OIDC_ERROR_MESSAGES[code];
}

// fetchOidcEnabled asks the backend whether OIDC login is configured. Any
// failure counts as disabled so the password form always works.
export async function fetchOidcEnabled(fetchFn: typeof fetch): Promise<boolean> {
	try {
		const response = await fetchFn('/api/auth/oidc/status');
		if (!response.ok) {
			return false;
		}
		const body = (await response.json()) as { enabled?: unknown };
		return body.enabled === true;
	} catch {
		return false;
	}
}

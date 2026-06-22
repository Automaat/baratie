// Shared e2e credentials. The backend seeds this admin account on startup from
// the BRT_ADMIN_* env vars (see playwright.config.ts / the E2E workflow).
export const ADMIN_USERNAME = process.env.BRT_ADMIN_USERNAME ?? 'e2e-admin';
export const ADMIN_PASSWORD = process.env.BRT_ADMIN_PASSWORD ?? 'e2e-admin-password';
export const JWT_SECRET = process.env.BRT_JWT_SECRET ?? 'e2e-test-jwt-secret';

// Where auth.setup.ts persists the logged-in storage state for reuse.
export const STORAGE_STATE = 'e2e/.auth/user.json';

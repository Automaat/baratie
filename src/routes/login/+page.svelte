<script lang="ts">
	import { enhance } from '$app/forms';
	import { ChefHat, KeyRound } from 'lucide-svelte';
	import { OIDC_LOGIN_PATH } from '$lib/utils/oidc';
	import type { ActionData, PageData } from './$types';

	let { data, form }: { data: PageData; form: ActionData } = $props();
	let submitting = $state(false);
	const errorMessage = $derived(form?.error ?? data.oidcError);
</script>

<div
	class="flex min-h-screen items-center justify-center bg-surface-50-950 text-surface-950-50 p-4"
>
	<div class="card preset-filled-surface-50-950 w-full max-w-sm shadow-xl p-6 space-y-5">
		<div class="flex items-center gap-2 text-lg font-bold">
			<ChefHat class="text-primary-500" size={24} />
			<span>Baratie</span>
		</div>

		<form
			method="POST"
			class="space-y-4"
			use:enhance={() => {
				submitting = true;
				return async ({ update }) => {
					await update();
					submitting = false;
				};
			}}
		>
			<label class="label">
				<span class="font-semibold text-sm">Nazwa użytkownika</span>
				<input
					name="username"
					type="text"
					class="input"
					autocomplete="username"
					value={form?.username ?? ''}
					required
				/>
			</label>

			<label class="label">
				<span class="font-semibold text-sm">Hasło</span>
				<input
					name="password"
					type="password"
					class="input"
					autocomplete="current-password"
					required
				/>
			</label>

			<label class="flex items-center gap-2 text-sm">
				<input name="remember_me" type="checkbox" class="checkbox" />
				<span>Zapamiętaj mnie</span>
			</label>

			{#if errorMessage}
				<div class="card preset-filled-error-500 p-3 text-sm" role="alert">{errorMessage}</div>
			{/if}

			<button type="submit" class="btn preset-filled-primary-500 w-full" disabled={submitting}>
				{submitting ? 'Logowanie...' : 'Zaloguj'}
			</button>
		</form>

		{#if data.oidcEnabled}
			<div class="flex items-center gap-3 text-xs text-surface-600-400">
				<hr class="hr flex-1" />
				<span>lub</span>
				<hr class="hr flex-1" />
			</div>
			<a href={OIDC_LOGIN_PATH} class="btn preset-tonal-primary w-full" data-sveltekit-reload>
				<KeyRound size={18} />
				<span>Zaloguj przez Authelię</span>
			</a>
		{/if}
	</div>
</div>

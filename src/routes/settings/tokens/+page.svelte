<script lang="ts">
	import { api } from '$lib/apiClient';
	import { invalidateAll } from '$app/navigation';
	import { toast } from '$lib/stores/toast.svelte';
	import { confirm } from '$lib/stores/confirm.svelte';
	import { formatDate } from '$lib/utils/format';
	import Modal from '$lib/components/Modal.svelte';
	import type { ApiToken } from './+page';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
	const tokens = $derived(data.tokens);

	let name = $state('');
	let expiresAt = $state('');
	let creating = $state(false);

	// The plaintext secret is shown exactly once, right after creation.
	let newToken = $state<string | null>(null);
	let copied = $state(false);

	async function createToken(event: Event): Promise<void> {
		event.preventDefault();
		creating = true;
		try {
			const created = await api.post<{ token: string }>('/api/auth/tokens', {
				name,
				expires_at: expiresAt || null
			});
			newToken = created.token;
			copied = false;
			name = '';
			expiresAt = '';
			await invalidateAll();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Wystąpił błąd');
		} finally {
			creating = false;
		}
	}

	async function copyToken(): Promise<void> {
		if (!newToken) return;
		try {
			await navigator.clipboard.writeText(newToken);
			copied = true;
		} catch {
			toast.error('Nie udało się skopiować — skopiuj ręcznie');
		}
	}

	async function revoke(token: ApiToken): Promise<void> {
		const ok = await confirm({
			title: 'Unieważnić token?',
			message: `Token „${token.name}” przestanie działać natychmiast. Tej operacji nie można cofnąć.`,
			danger: true,
			confirmText: 'Unieważnij'
		});
		if (!ok) return;
		try {
			await api.del(`/api/auth/tokens/${token.id}`);
			await invalidateAll();
			toast.success('Token unieważniony');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Nie udało się unieważnić');
		}
	}
</script>

<div class="space-y-6 max-w-3xl">
	<div>
		<h2 class="h4 font-semibold">Tokeny API</h2>
		<p class="text-sm text-surface-700-300">
			Długoterminowe tokeny dla klientów programistycznych (np. asystent AI). Używaj nagłówka
			<code class="code">Authorization: Bearer &lt;token&gt;</code>. Token nie wygasa po 24 h.
		</p>
	</div>

	<div class="card preset-filled-surface-50-950 p-5 space-y-4">
		<h3 class="h5 font-semibold">Utwórz token</h3>
		<form class="grid gap-3 sm:grid-cols-2" onsubmit={createToken}>
			<label class="label">
				<span class="font-semibold text-sm">Nazwa</span>
				<input bind:value={name} type="text" class="input" maxlength="100" required />
			</label>
			<label class="label">
				<span class="font-semibold text-sm">Wygasa (opcjonalnie)</span>
				<input bind:value={expiresAt} type="date" class="input" />
			</label>
			<div class="sm:col-span-2">
				<button type="submit" class="btn preset-filled-primary-500" disabled={creating}>
					{creating ? 'Tworzenie...' : 'Utwórz token'}
				</button>
			</div>
		</form>
	</div>

	<div class="card preset-filled-surface-50-950 p-5 table-cards">
		{#if tokens.length === 0}
			<p class="text-sm text-surface-700-300">Brak tokenów.</p>
		{:else}
			<table class="table">
				<thead>
					<tr>
						<th>Nazwa</th>
						<th>Utworzono</th>
						<th>Wygasa</th>
						<th>Ostatnio użyty</th>
						<th></th>
					</tr>
				</thead>
				<tbody>
					{#each tokens as token (token.id)}
						<tr>
							<td data-label="Nazwa">{token.name}</td>
							<td data-label="Utworzono">{formatDate(token.created_at)}</td>
							<td data-label="Wygasa"
								>{token.expires_at ? formatDate(token.expires_at) : 'Nigdy'}</td
							>
							<td data-label="Ostatnio użyty">{formatDate(token.last_used_at)}</td>
							<td>
								<button
									type="button"
									class="btn btn-sm preset-tonal-error"
									onclick={() => revoke(token)}
								>
									Unieważnij
								</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>
</div>

<Modal
	open={newToken !== null}
	title="Nowy token utworzony"
	confirmText="Gotowe"
	hideFooter={false}
	onConfirm={() => (newToken = null)}
	onCancel={() => (newToken = null)}
>
	<div class="space-y-3">
		<p class="text-sm text-surface-700-300">
			Skopiuj token teraz — nie zobaczysz go ponownie po zamknięciu tego okna.
		</p>
		<div class="flex items-center gap-2">
			<input
				type="text"
				class="input font-mono text-sm"
				aria-label="Nowy token API"
				value={newToken}
				readonly
			/>
			<button type="button" class="btn preset-tonal-surface shrink-0" onclick={copyToken}>
				{copied ? 'Skopiowano' : 'Kopiuj'}
			</button>
		</div>
	</div>
</Modal>

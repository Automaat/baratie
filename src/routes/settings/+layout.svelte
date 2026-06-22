<script lang="ts">
	import { page } from '$app/stores';
	import type { LayoutData } from './$types';

	let { children, data }: { children: import('svelte').Snippet; data: LayoutData } = $props();

	const tabs = $derived([
		{ href: '/settings/navigation', label: 'Nawigacja' },
		...(data.user?.isAdmin ? [{ href: '/settings/users', label: 'Użytkownicy' }] : [])
	]);
</script>

<div class="space-y-6">
	<h1 class="h2 font-bold">Ustawienia</h1>

	<div class="tabs">
		{#each tabs as tab (tab.href)}
			<a href={tab.href} class="tab" class:active={$page.url.pathname === tab.href}>
				{tab.label}
			</a>
		{/each}
	</div>

	{@render children?.()}
</div>

<style>
	.tabs {
		display: flex;
		gap: var(--size-1);
		border-bottom: 2px solid var(--surface-3);
		margin-bottom: var(--size-6);
		overflow-x: auto;
	}

	.tab {
		padding: var(--size-2) var(--size-4);
		font-size: var(--font-size-1);
		font-weight: 500;
		color: var(--color-text-3);
		text-decoration: none;
		border-bottom: 2px solid transparent;
		margin-bottom: -2px;
		white-space: nowrap;
		transition:
			color 0.15s,
			border-color 0.15s;
	}

	.tab:hover {
		color: var(--color-text-1);
	}

	.tab.active {
		color: var(--color-primary);
		border-bottom-color: var(--color-primary);
		font-weight: 600;
	}
</style>

<script lang="ts">
	import { NAV_ROUTES } from '$lib/nav/routes';
	import { navPrefs, MAX_PINNED } from '$lib/stores/navPrefs.svelte';
	import { toast } from '$lib/stores/toast.svelte';

	const pinned = $derived(navPrefs.pinned);

	function toggle(href: string) {
		const current = [...navPrefs.pinned];
		if (current.includes(href)) {
			navPrefs.set(current.filter((h) => h !== href));
			return;
		}
		if (current.length >= MAX_PINNED) {
			toast.info(`Możesz przypiąć maksymalnie ${MAX_PINNED} pozycji`);
			return;
		}
		navPrefs.set([...current, href]);
	}
</script>

<div class="space-y-4 max-w-xl">
	<div>
		<h2 class="h4 font-semibold">Pasek nawigacji (mobilny)</h2>
		<p class="text-sm text-surface-700-300">
			Wybierz do {MAX_PINNED} pozycji widocznych na dolnym pasku na telefonie. Pozostałe trafią do menu
			„Więcej”.
		</p>
	</div>

	<ul class="space-y-2">
		{#each NAV_ROUTES as item (item.href)}
			<li>
				<label class="flex items-center gap-3 card preset-tonal-surface p-3 cursor-pointer">
					<input
						type="checkbox"
						class="checkbox"
						checked={pinned.includes(item.href)}
						onchange={() => toggle(item.href)}
					/>
					<item.icon size={18} />
					<span>{item.label}</span>
				</label>
			</li>
		{/each}
	</ul>

	<button type="button" class="btn preset-tonal-surface" onclick={() => navPrefs.reset()}>
		Przywróć domyślne
	</button>
</div>

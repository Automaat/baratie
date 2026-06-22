<script lang="ts">
	import { page } from '$app/stores';
	import { AlertTriangle } from 'lucide-svelte';
</script>

<div class="flex flex-col items-center justify-center gap-4 py-16 text-center">
	<AlertTriangle class="text-error-500" size={48} />
	<h1 class="h3 font-bold">{$page.status}</h1>
	<p class="text-surface-700-300">{$page.error?.message ?? 'Coś poszło nie tak'}</p>
	<a href="/" class="btn preset-filled-primary-500">Wróć do pulpitu</a>
</div>

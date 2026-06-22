<script lang="ts">
	interface Props {
		label: string;
		value: number | string | null | undefined;
		suffix?: string;
		color?: 'green' | 'blue' | 'red' | 'yellow' | 'neutral';
		// Optional muted line beneath the value.
		secondary?: string;
	}

	let { label, value, suffix = '', color = 'neutral', secondary }: Props = $props();

	const isEmpty = $derived(value == null || (typeof value === 'number' && Number.isNaN(value)));
	const displayValue = $derived(isEmpty ? '—' : `${value}${suffix}`);

	const valueClass = $derived(
		{
			green: 'text-success-600-400',
			red: 'text-error-600-400',
			blue: 'text-primary-600-400',
			yellow: 'text-warning-600-400',
			neutral: 'text-surface-950-50'
		}[color]
	);
</script>

<div class="card preset-filled-surface-100-900 p-4 space-y-1">
	<div class="text-sm opacity-75">{label}</div>
	<div class="text-2xl font-bold {valueClass}">{displayValue}</div>
	{#if secondary}
		<div class="text-xs text-surface-600-400">{secondary}</div>
	{/if}
</div>

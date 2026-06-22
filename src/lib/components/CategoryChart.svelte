<script lang="ts">
	import { onMount } from 'svelte';
	import type { ECharts } from 'echarts';

	interface Props {
		data: { name: string; value: number }[];
	}

	let { data }: Props = $props();

	let el = $state<HTMLDivElement | null>(null);
	let chart: ECharts | null = null;

	function render() {
		if (!chart) return;
		chart.setOption({
			grid: { left: 8, right: 8, top: 16, bottom: 24, containLabel: true },
			tooltip: { trigger: 'axis' },
			xAxis: { type: 'category', data: data.map((d) => d.name) },
			yAxis: { type: 'value', minInterval: 1 },
			series: [
				{
					type: 'bar',
					data: data.map((d) => d.value),
					itemStyle: { borderRadius: [4, 4, 0, 0] }
				}
			]
		});
	}

	onMount(() => {
		let disposed = false;
		const resize = () => chart?.resize();
		void import('echarts').then((echarts) => {
			if (disposed || !el) return;
			chart = echarts.init(el);
			render();
			window.addEventListener('resize', resize);
		});
		return () => {
			disposed = true;
			window.removeEventListener('resize', resize);
			chart?.dispose();
			chart = null;
		};
	});

	$effect(() => {
		// Re-render whenever the data changes and the chart is ready.
		void data;
		render();
	});
</script>

<div bind:this={el} class="w-full h-64"></div>

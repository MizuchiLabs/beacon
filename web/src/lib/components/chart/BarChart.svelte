<script lang="ts">
	import type { MonitorStats } from '$lib/api/generated/types.gen';
	import { scaleBand } from 'd3-scale';
	import { BarChart } from 'layerchart';
	import { cubicInOut } from 'svelte/easing';
	import * as Chart from '$lib/components/ui/chart';

	interface Props {
		monitor: MonitorStats;
	}
	let { monitor }: Props = $props();

	const chartData = $derived(
		(monitor.data_points ?? []).map((dp, idx) => ({
			id: idx,
			timestamp: new Date(dp.timestamp),
			up: dp.up_ratio,
			degraded: dp.degraded_ratio,
			down: dp.down_ratio
		}))
	);

	const chartConfig = {
		up: { label: 'Operational', color: 'var(--chart-3)' },
		degraded: { label: 'Degraded', color: 'var(--chart-4)' },
		down: { label: 'Down', color: 'var(--destructive)' }
	} satisfies Chart.ChartConfig;
</script>

<Chart.Container config={chartConfig} class="h-14 w-full">
	<BarChart
		data={chartData}
		xScale={scaleBand().padding(0.2)}
		x="id"
		axis={false}
		rule={false}
		grid={false}
		series={[
			{
				key: 'down',
				label: 'Down',
				color: chartConfig.down.color,
				props: { rounded: 'bottom' }
			},
			{
				key: 'degraded',
				label: 'Degraded',
				color: chartConfig.degraded.color
			},
			{
				key: 'up',
				label: 'Operational',
				color: chartConfig.up.color
			}
		]}
		seriesLayout="stack"
		props={{
			bars: {
				stroke: 'none',
				radius: 3,
				motion: {
					x: { type: 'tween', duration: 500, easing: cubicInOut },
					y: { type: 'tween', duration: 500, easing: cubicInOut },
					width: { type: 'tween', duration: 500, easing: cubicInOut },
					height: { type: 'tween', duration: 500, easing: cubicInOut }
				}
			}
		}}
		padding={{ top: 0, bottom: 0, left: 0, right: 0 }}
	>
		{#snippet tooltip()}
			<Chart.Tooltip
				labelFormatter={(idx: number) => {
					const point = chartData[idx];
					return point?.timestamp.toLocaleString(undefined, {
						month: 'short',
						day: 'numeric',
						hour: 'numeric',
						minute: '2-digit'
					});
				}}
				class="items-start border bg-background/95 text-xs shadow-xl backdrop-blur"
			/>
		{/snippet}
	</BarChart>
</Chart.Container>

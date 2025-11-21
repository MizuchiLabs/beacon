<script lang="ts">
	import type { MonitorStats } from '$lib/api/queries';
	import { scaleBand, scaleLinear } from 'd3-scale';
	import { BarChart as LayerBarChart, type ChartContextValue } from 'layerchart';
	import { cubicInOut } from 'svelte/easing';
	import * as Chart from '$lib/components/ui/chart';

	interface Props {
		monitor: MonitorStats;
	}
	let { monitor }: Props = $props();

	const chartData = $derived(
		monitor.data_points.map((dp, idx) => {
			// Check if we have pre-calculated ratios from backend
			if (dp.up_ratio !== undefined && dp.up_ratio !== null) {
				return {
					id: idx,
					timestamp: new Date(dp.timestamp),
					up: dp.up_ratio,
					degraded: dp.degraded_ratio ?? 0,
					down: dp.down_ratio ?? 0
				};
			}

			// Fallback: calculate from individual point (for timeseries data)
			const isDown = !dp.is_up;
			const isDegraded = dp.is_up && dp.response_time && dp.response_time > 500;
			const isUp = dp.is_up && (!dp.response_time || dp.response_time <= 500);

			return {
				id: idx,
				timestamp: new Date(dp.timestamp),
				up: isUp ? 1 : 0,
				degraded: isDegraded ? 1 : 0,
				down: isDown ? 1 : 0
			};
		})
	);

	const chartConfig = {
		up: { label: 'Operational', color: 'var(--chart-3)' },
		degraded: { label: 'Degraded', color: 'var(--chart-4)' },
		down: { label: 'Down', color: 'var(--destructive)' }
	} satisfies Chart.ChartConfig;

	let context = $state<ChartContextValue>();
</script>

<Chart.Container config={chartConfig} class="h-14 w-full">
	<LayerBarChart
		bind:context
		data={chartData}
		xScale={scaleBand().padding(0.2)}
		yScale={scaleLinear().domain([0, 1]).nice()}
		x="id"
		y={(d) => d.up + d.degraded + d.down}
		axis={false}
		rule={false}
		grid={false}
		series={[
			{
				key: 'down',
				label: 'Down',
				color: chartConfig.down.color
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
				rx: 3,
				ry: 3,
				stroke: 'none',
				initialY: 100,
				initialHeight: 0,
				motion: {
					x: { type: 'tween', duration: 500, easing: cubicInOut },
					y: { type: 'tween', duration: 500, easing: cubicInOut },
					width: { type: 'tween', duration: 500, easing: cubicInOut },
					height: { type: 'tween', duration: 500, easing: cubicInOut }
				}
			}
		}}
		padding={{ top: 0, bottom: 0, left: 8, right: 8 }}
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
	</LayerBarChart>
</Chart.Container>

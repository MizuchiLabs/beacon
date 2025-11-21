<script lang="ts">
	import type { MonitorStats } from '$lib/api/queries';
	import * as Chart from '$lib/components/ui/chart';
	import { AreaChart, Area, ChartClipPath } from 'layerchart';
	import { scaleUtc } from 'd3-scale';
	import { curveCatmullRom } from 'd3-shape';
	import { cubicInOut } from 'svelte/easing';
	import { ChartContainer } from '$lib/components/ui/chart';

	interface Props {
		monitor: MonitorStats;
	}
	let { monitor }: Props = $props();

	// Transform data for chart
	const chartData = $derived(
		monitor.data_points.map((dp) => ({
			date: new Date(dp.timestamp),
			response_time: dp.response_time || 0,
			is_up: dp.is_up
		}))
	);

	const chartConfig = {
		response_time: {
			label: 'Response Time',
			color: 'hsl(var(--primary))'
		}
	} satisfies Chart.ChartConfig;
</script>

<ChartContainer config={chartConfig} class="h-full w-full">
	<AreaChart
		data={chartData}
		x="date"
		xScale={scaleUtc()}
		series={[
			{
				key: 'response_time',
				label: 'Response Time',
				color: 'var(--primary)'
			}
		]}
		padding={{ top: 0, bottom: 0, left: 0, right: 0 }}
		props={{
			area: { curve: curveCatmullRom.alpha(0.5) },
			grid: { x: false, y: false },
			yAxis: { format: () => '' }
		}}
	>
		{#snippet marks({ series, getAreaProps })}
			<defs>
				<linearGradient id="fillGradient-{monitor.id}" x1="0" y1="0" x2="0" y2="1">
					<stop offset="0%" stop-color="var(--primary)" stop-opacity="0.25" />
					<stop offset="100%" stop-color="var(--primary)" stop-opacity="0" />
				</linearGradient>
			</defs>

			<ChartClipPath
				initialWidth={0}
				motion={{ width: { type: 'tween', duration: 800, easing: cubicInOut } }}
			>
				{#each series as s, i (s.key)}
					<Area {...getAreaProps(s, i)} fill="url(#fillGradient-{monitor.id})" stroke="none" />
				{/each}
			</ChartClipPath>
		{/snippet}

		{#snippet tooltip()}
			<Chart.Tooltip
				labelFormatter={(v: Date) => {
					return v.toLocaleString(undefined, {
						month: 'short',
						day: 'numeric',
						hour: 'numeric',
						minute: '2-digit'
					});
				}}
				class="border bg-background/95 text-xs shadow-xl backdrop-blur"
				indicator="dot"
			/>
		{/snippet}
	</AreaChart>
</ChartContainer>

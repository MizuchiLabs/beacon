<script lang="ts">
	import type { MonitorStats } from '$lib/api/generated/types.gen';
	import * as Chart from '$lib/components/ui/chart';
	import { Highlight, Spline } from 'layerchart';
	import { BarChart as BarChartCanvas } from 'layerchart/canvas';
	import { scaleBand } from 'd3-scale';
	import { cn } from '$lib/utils.js';
	import { cubicInOut } from 'svelte/easing';

	interface Props {
		monitor: MonitorStats;
		showTicks?: boolean;
		class?: string;
	}
	let { monitor, showTicks = false, class: className }: Props = $props();

	const chartConfig = {
		missing: { label: 'No data', color: 'var(--muted-foreground)' },
		down: { label: 'Down', color: 'var(--color-chart-5)' },
		degraded: { label: 'Degraded', color: 'var(--color-chart-4)' },
		up: { label: 'Operational', color: 'var(--color-chart-3)' }
	} satisfies Chart.ChartConfig;

	const points = $derived(monitor.data_points ?? []);

	const maxAvg = $derived(Math.max(0, ...points.map((p) => p.avg_ms ?? 0)));

	const chartData = $derived(
		points.map((p) => {
			const total = p.total || 0;
			return {
				date: new Date(p.timestamp),
				missing: !p.has_data && p.expected > 0 ? 1 : 0,
				down: total ? p.down / total : 0,
				degraded: total ? p.degraded / total : 0,
				up: total ? p.up / total : 0,
				avg_norm: p.avg_ms == null ? null : maxAvg > 0 ? p.avg_ms / maxAvg : 0,
				avg_ms: p.avg_ms,
				upCount: p.up,
				degradedCount: p.degraded,
				downCount: p.down,
				total: p.total,
				has_data: p.has_data
			};
		})
	);

	const series = [
		{
			key: 'missing',
			label: 'No data',
			color: 'var(--muted-foreground)',
			props: { fillOpacity: 0.3 }
		},
		{ key: 'down', label: 'Down', color: 'var(--color-chart-5)' },
		{ key: 'degraded', label: 'Degraded', color: 'var(--color-chart-4)' },
		{
			key: 'up',
			label: 'Operational',
			color: 'var(--color-chart-3)',
			props: { fillOpacity: 0.55 }
		}
	];

	const hasLine = $derived(chartData.some((d) => d.avg_norm != null));

	const ticks = $derived.by(() => {
		if (!showTicks || points.length < 3) return [];
		const first = new Date(points[0].timestamp);
		const last = new Date(points[points.length - 1].timestamp);
		const format = new Intl.DateTimeFormat(
			undefined,
			last.getTime() - first.getTime() > 3 * 86_400_000
				? { month: 'short', day: 'numeric' }
				: { month: 'short', day: 'numeric', hour: 'numeric' }
		);
		return [points[0], points[Math.floor(points.length / 2)], points[points.length - 1]].map((p) =>
			format.format(new Date(p.timestamp))
		);
	});
</script>

<div class={cn('flex w-full flex-col', className)}>
	<div class="relative min-h-0 w-full flex-1">
		<Chart.Container config={chartConfig} class="absolute inset-0 h-full w-full">
			<BarChartCanvas
				data={chartData}
				x="date"
				xScale={scaleBand().padding(0.15)}
				yDomain={[0, 1]}
				axis={false}
				rule={false}
				grid={false}
				{series}
				seriesLayout="stack"
				props={{
					bars: {
						stroke: 'none',
						motion: { type: 'tween', duration: 500, easing: cubicInOut }
					},
					highlight: { area: false },
					xAxis: { format: (d) => d.slice(0, 3) }
				}}
				padding={{ top: 0, bottom: 0, left: 0, right: 0 }}
			>
				{#snippet belowMarks()}
					<Highlight area={{ class: 'fill-muted/30' }} />
				{/snippet}
				{#snippet aboveMarks()}
					{#if hasLine}
						<Spline
							x="date"
							y="avg_norm"
							stroke="var(--primary)"
							strokeWidth={1.5}
							fillOpacity={0}
							motion={{ type: 'none' }}
						/>
					{/if}
				{/snippet}
				{#snippet tooltip()}
					<Chart.Tooltip
						labelFormatter={(v: Date) => {
							return v.toLocaleDateString(undefined, {
								month: 'short',
								day: 'numeric',
								hour: 'numeric',
								minute: '2-digit'
							});
						}}
					/>
				{/snippet}
			</BarChartCanvas>
		</Chart.Container>

		{#if chartData.length === 0}
			<div class="grid h-full place-items-center text-xs text-muted-foreground">
				No checks in this window
			</div>
		{/if}
	</div>

	{#if ticks.length === 3}
		<div class="grid grid-cols-3 pt-1.5 text-[11px] text-muted-foreground/70">
			<span>{ticks[0]}</span>
			<span class="text-center">{ticks[1]}</span>
			<span class="text-right">{ticks[2]}</span>
		</div>
	{/if}
</div>

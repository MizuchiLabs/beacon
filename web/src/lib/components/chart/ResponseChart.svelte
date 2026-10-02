<script lang="ts">
	import type { DataPoint } from '#lib/api/generated/types.gen.js';
	import * as Chart from '#lib/components/ui/chart/index.js';
	import { formatMs } from '#lib/status.js';
	import { cn } from '#lib/utils.js';
	import { scaleTime } from 'd3-scale';
	import { AreaChart } from 'layerchart';

	interface Props {
		points: DataPoint[];
		threshold: number;
		ranges?: { start: number; end: number; class: string }[];
		class?: string;
	}
	let { points, threshold, ranges = [], class: className }: Props = $props();

	const chartConfig = {
		avg_ms: { label: 'Avg response (ms)', color: 'var(--primary)' }
	} satisfies Chart.ChartConfig;

	const chartData = $derived(
		points.map((p) => ({ date: new Date(p.timestamp), avg_ms: p.avg_ms ?? null }))
	);
	const maxMs = $derived(Math.max(0, ...chartData.map((d) => d.avg_ms ?? 0)));
	const hasData = $derived(maxMs > 0);
	// The threshold line only helps when it sits near the data, a far away one
	// would squash the curve into the floor.
	const showThreshold = $derived(threshold <= maxMs * 2);
	const yMax = $derived(Math.max(maxMs, showThreshold ? threshold : 0) * 1.15);

	const shaded = $derived.by(() => {
		if (chartData.length < 2) return [];
		const first = chartData[0].date.getTime();
		const last = chartData[chartData.length - 1].date.getTime();
		return ranges
			.filter((r) => r.end > first && r.start < last)
			.map((r) => ({
				type: 'range' as const,
				layer: 'below' as const,
				x: [new Date(Math.max(r.start, first)), new Date(Math.min(r.end, last))],
				class: r.class
			}));
	});

	const tickFormat = $derived.by(() => {
		const span =
			points.length > 1
				? Date.parse(points.at(-1)!.timestamp) - Date.parse(points[0].timestamp)
				: 0;
		return new Intl.DateTimeFormat(
			undefined,
			span > 3 * 86_400_000
				? { month: 'short', day: 'numeric' }
				: { hour: 'numeric', minute: '2-digit' }
		);
	});
</script>

<div class={cn('relative w-full', className)}>
	{#if hasData}
		<Chart.Container config={chartConfig} class="h-full w-full">
			<AreaChart
				data={chartData}
				x="date"
				xScale={scaleTime()}
				yDomain={[0, yMax]}
				series={[{ key: 'avg_ms', label: 'Avg response (ms)', color: 'var(--primary)' }]}
				grid={{ x: false, y: true }}
				rule={false}
				padding={{ left: 44, bottom: 20, top: 4, right: 4 }}
				annotations={[
					...shaded,
					...(showThreshold
						? [
								{
									type: 'line' as const,
									y: threshold,
									props: { line: { class: 'stroke-chart-4 [stroke-dasharray:4_4]' } }
								}
							]
						: [])
				]}
				props={{
					area: {
						fillOpacity: 0.15,
						line: { class: 'stroke-2' },
						defined: (d: { avg_ms: number | null }) => d.avg_ms != null,
						motion: 'tween'
					},
					grid: { class: 'stroke-border/60' },
					xAxis: { ticks: 4, format: (d: Date) => tickFormat.format(d) },
					yAxis: { ticks: 3, format: (v: number) => formatMs(Math.round(v)) }
				}}
			>
				{#snippet tooltip()}
					<Chart.Tooltip
						labelFormatter={(v: Date) =>
							v.toLocaleString(undefined, {
								month: 'short',
								day: 'numeric',
								hour: 'numeric',
								minute: '2-digit'
							})}
					/>
				{/snippet}
			</AreaChart>
		</Chart.Container>
	{:else}
		<div class="grid h-full place-items-center text-xs text-muted-foreground">
			No response times in this window
		</div>
	{/if}
</div>

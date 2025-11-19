<script lang="ts">
	import type { MonitorStats } from '$lib/api/queries';
	import * as Card from '$lib/components/ui/card';
	import * as Chart from '$lib/components/ui/chart';
	import { AreaChart, Area, ChartClipPath } from 'layerchart';
	import { scaleUtc } from 'd3-scale';
	import { curveCatmullRom } from 'd3-shape';
	import { cubicInOut } from 'svelte/easing';
	import { Badge } from '$lib/components/ui/badge';
	import { ChartContainer } from '$lib/components/ui/chart';
	import CircleCheckIcon from '@lucide/svelte/icons/circle-check';
	import CircleXIcon from '@lucide/svelte/icons/circle-x';

	interface Props {
		monitor: MonitorStats;
		timeRange: string;
	}

	let { monitor, timeRange }: Props = $props();

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
			color: 'var(--chart-1)'
		}
	} satisfies Chart.ChartConfig;

	// Get tick count based on time range
	const tickCount = $derived.by(() => {
		const seconds = parseInt(timeRange);
		if (seconds <= 86400) return 6; // 24h
		if (seconds <= 604800) return 7; // 7d
		if (seconds <= 1209600) return 7; // 14d
		return 10; // 30d
	});
</script>

<Card.Root class="overflow-hidden bg-card">
	<Card.Header class="pb-3">
		<div class="flex items-start justify-between gap-2">
			<div class="min-w-0 flex-1 space-y-1">
				<Card.Title class="truncate text-base">{monitor.name}</Card.Title>
				<Card.Description class="truncate text-xs">{monitor.url}</Card.Description>
			</div>
			<Badge variant={monitor.uptime_pct >= 95 ? 'default' : 'destructive'} class="shrink-0">
				{monitor.uptime_pct.toFixed(2)}%
			</Badge>
			<Badge variant="outline" class="shrink-0">
				Every {monitor.check_interval}s
			</Badge>
		</div>
	</Card.Header>

	<Card.Content class="bg-card">
		<ChartContainer config={chartConfig} class="aspect-auto h-[150px] w-full px-4">
			<AreaChart
				data={chartData}
				x="date"
				xScale={scaleUtc()}
				series={[
					{
						key: 'response_time',
						label: 'Response Time',
						color: chartConfig.response_time.color
					}
				]}
				props={{
					area: {
						curve: curveCatmullRom.alpha(0.5),
						'fill-opacity': 0.4,
						line: { class: 'stroke-1' }
					},
					xAxis: {
						ticks: tickCount,
						format: (v) => {
							const seconds = parseInt(timeRange);
							if (seconds <= 86400) {
								return v.toLocaleTimeString('en-US', {
									hour: 'numeric',
									hour12: false
								});
							}
							return v.toLocaleDateString('en-US', {
								month: 'short',
								day: 'numeric'
							});
						}
					},
					yAxis: {
						format: (v) => `${v}ms`
					}
				}}
			>
				{#snippet marks({ series, getAreaProps })}
					<defs>
						<!-- Response time gradient: fast (low opacity) to slow (high opacity) -->
						<linearGradient id="strokeGradient-{monitor.id}" x1="0" y1="1" x2="0" y2="0">
							<stop offset="0%" stop-color="var(--primary)" stop-opacity="0.4" />
							<stop offset="30%" stop-color="var(--primary)" stop-opacity="0.7" />
							<stop offset="70%" stop-color="var(--primary)" stop-opacity="0.9" />
							<stop offset="100%" stop-color="var(--primary)" stop-opacity="1" />
						</linearGradient>

						<!-- Fill gradient: subtle at bottom, more visible at top -->
						<linearGradient id="fillGradient-{monitor.id}" x1="0" y1="0" x2="0" y2="1">
							<stop offset="0%" stop-color="var(--primary)" stop-opacity="0.35" />
							<stop offset="50%" stop-color="var(--primary)" stop-opacity="0.15" />
							<stop offset="100%" stop-color="var(--primary)" stop-opacity="0.05" />
						</linearGradient>
					</defs>
					<ChartClipPath
						initialWidth={0}
						motion={{
							width: { type: 'tween', duration: 800, easing: cubicInOut }
						}}
					>
						{#each series as s, i (s.key)}
							<Area
								{...getAreaProps(s, i)}
								fill="url(#fillGradient-{monitor.id})"
								stroke="url(#strokeGradient-{monitor.id})"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
							/>
						{/each}
					</ChartClipPath>
				{/snippet}

				{#snippet tooltip()}
					<Chart.Tooltip
						labelFormatter={(v: Date) => {
							return v.toLocaleString('en-US', {
								month: 'short',
								day: 'numeric',
								hour: 'numeric',
								minute: '2-digit'
							});
						}}
						indicator="dot"
					/>
				{/snippet}
			</AreaChart>
		</ChartContainer>
	</Card.Content>

	<Card.Footer class="border-t">
		<div class="flex w-full flex-col items-center justify-between gap-4 text-xs sm:flex-row">
			<!-- Percentiles -->
			<div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
				{#if monitor.percentiles.p50}
					<Badge variant={monitor.percentiles.p50 >= 500 ? 'destructive' : 'outline'}>
						P50: {monitor.percentiles.p50}ms
					</Badge>
				{/if}
				{#if monitor.percentiles.p90}
					<Badge variant={monitor.percentiles.p90 >= 500 ? 'destructive' : 'outline'}>
						P90: {monitor.percentiles.p90}ms
					</Badge>
				{/if}
				{#if monitor.percentiles.p95}
					<Badge variant={monitor.percentiles.p95 >= 500 ? 'destructive' : 'outline'}>
						P95: {monitor.percentiles.p95}ms
					</Badge>
				{/if}
				{#if monitor.percentiles.p99}
					<Badge variant={monitor.percentiles.p99 >= 500 ? 'destructive' : 'outline'}>
						P99: {monitor.percentiles.p99}ms
					</Badge>
				{/if}
			</div>

			<!-- Avg response time -->
			<div class="flex items-center gap-1.5 text-muted-foreground">
				<Badge variant="outline">
					{#if monitor.uptime_pct >= 95}
						<CircleCheckIcon class="size-3.5 text-green-500" />
					{:else}
						<CircleXIcon class="size-3.5 text-destructive" />
					{/if}
					<span class="font-medium">
						Avg: {monitor.avg_response_time ?? 'N/A'}{#if monitor.avg_response_time}ms{/if}
					</span>
				</Badge>
			</div>
		</div>
	</Card.Footer>
</Card.Root>

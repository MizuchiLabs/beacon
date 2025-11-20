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
	import ActivityIcon from '@lucide/svelte/icons/activity';

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

	// Helper for stat color
	const getStatColor = (val: number | undefined | null) => {
		if (!val) return 'text-muted-foreground';
		if (val >= 500) return 'text-destructive';
		if (val >= 200) return 'text-amber-500';
		return 'text-foreground';
	};
</script>

<Card.Root
	class="group overflow-hidden border bg-card transition-all hover:bg-card hover:shadow-md"
>
	<Card.Header class="pb-2">
		<div class="flex items-start justify-between gap-4">
			<div class="flex min-w-0 flex-1 flex-col gap-1">
				<div class="flex items-center gap-2">
					{#if monitor.uptime_pct >= 95}
						<div class="relative flex h-2.5 w-2.5">
							<span
								class="absolute inline-flex h-full w-full animate-ping rounded-full bg-chart-4 opacity-75"
							></span>
							<span class="relative inline-flex h-2.5 w-2.5 rounded-full bg-chart-4"></span>
						</div>
					{:else}
						<div class="h-2.5 w-2.5 rounded-full bg-destructive shadow-sm"></div>
					{/if}
					<h3 class="truncate text-base leading-none font-semibold tracking-tight">
						{monitor.name}
					</h3>
				</div>
				<div class="flex gap-2 text-xs text-muted-foreground">
					<a
						href={monitor.url}
						target="_blank"
						rel="noreferrer"
						class="truncate transition-colors hover:text-foreground hover:underline"
					>
						{monitor.url}
					</a>
					<span>•</span>
					<span>{monitor.check_interval}s interval</span>
				</div>
			</div>

			<div class="flex flex-col items-end">
				<span
					class="text-xl font-bold tracking-tight {monitor.uptime_pct >= 95
						? 'text-chart-4'
						: 'text-destructive'}"
				>
					{monitor.uptime_pct.toFixed(2)}%
				</span>
				<span class="text-[10px] font-medium text-muted-foreground/70 uppercase">Uptime</span>
			</div>
		</div>
	</Card.Header>

	<Card.Content class="p-0">
		<!-- Minimal Chart area with refined gradient and glow -->
		<div class="h-[100px] w-full">
			<ChartContainer config={chartConfig} class="h-full w-full">
				<AreaChart
					data={chartData}
					x="date"
					xScale={scaleUtc()}
					series={[
						{
							key: 'response_time',
							label: 'Response Time',
							color: 'var(--primary)' // used for tooltip
						}
					]}
					padding={{ top: 10, bottom: 0, left: 0, right: 0 }}
					props={{
						area: { curve: curveCatmullRom.alpha(0.5) },
						grid: { x: false, y: false }
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
								<Area
									{...getAreaProps(s, i)}
									fill="url(#fillGradient-{monitor.id})"
									stroke="none"
								/>
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
		</div>
	</Card.Content>

	<Card.Footer class="grid grid-cols-3 gap-2 border-t px-4 text-xs sm:grid-cols-5">
		<div class="col-span-1 flex flex-col gap-0.5">
			<span class="text-muted-foreground">Avg</span>
			<span class="font-medium {getStatColor(monitor.avg_response_time)}">
				{monitor.avg_response_time ?? '-'}ms
			</span>
		</div>

		<div class="col-span-1 flex flex-col gap-0.5 border-l pl-3">
			<span class="text-muted-foreground">P50</span>
			<span class="font-medium {getStatColor(monitor.percentiles.p50)}">
				{monitor.percentiles.p50 ?? '-'}ms
			</span>
		</div>

		<div class="col-span-1 flex flex-col gap-0.5 border-l pl-3">
			<span class="text-muted-foreground">P95</span>
			<span class="font-medium {getStatColor(monitor.percentiles.p95)}">
				{monitor.percentiles.p95 ?? '-'}ms
			</span>
		</div>

		<div class="col-span-1 hidden flex-col gap-0.5 border-l pl-3 sm:flex">
			<span class="text-muted-foreground">P99</span>
			<span class="font-medium {getStatColor(monitor.percentiles.p99)}">
				{monitor.percentiles.p99 ?? '-'}ms
			</span>
		</div>

		<div class="col-span-1 ml-auto hidden items-end justify-end sm:flex">
			<Badge variant="outline" class="bg-background/50 font-normal text-muted-foreground">
				<ActivityIcon class="mr-1 size-3 opacity-70" />
				<span>Live</span>
			</Badge>
		</div>
	</Card.Footer>
</Card.Root>

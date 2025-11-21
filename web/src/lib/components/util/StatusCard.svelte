<script lang="ts">
	import type { MonitorStats } from '$lib/api/queries';
	import * as Card from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import ActivityIcon from '@lucide/svelte/icons/activity';
	import AreaChart from '$lib/components/chart/AreaChart.svelte';
	import BarChart from '$lib/components/chart/BarChart.svelte';

	interface Props {
		monitor: MonitorStats;
		chartType: string;
	}
	let { monitor, chartType }: Props = $props();

	// Helper for stat color
	const getStatColor = (val: number | undefined | null) => {
		if (!val) return 'text-muted-foreground';
		if (val >= 500) return 'text-chart-4';
		if (val >= 200) return 'text-chart-2';
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
								class="absolute inline-flex h-full w-full animate-ping rounded-full bg-chart-3 opacity-75"
							></span>
							<span class="relative inline-flex h-2.5 w-2.5 rounded-full bg-chart-3"></span>
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
						? 'text-chart-3'
						: 'text-destructive'}"
				>
					{monitor.uptime_pct.toFixed(2)}%
				</span>
				<span class="text-[10px] font-medium text-muted-foreground/70 uppercase">Uptime</span>
			</div>
		</div>
	</Card.Header>

	<Card.Content class="h-12 w-full p-0">
		{#if chartType === 'bars'}
			<BarChart {monitor} />
		{:else}
			<AreaChart {monitor} />
		{/if}
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

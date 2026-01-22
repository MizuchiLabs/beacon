<script lang="ts">
	import type { MonitorStats } from '$lib/api/queries';
	import * as Card from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import AreaChart from '$lib/components/chart/AreaChart.svelte';
	import BarChart from '$lib/components/chart/BarChart.svelte';
	import { cn } from 'tailwind-variants';
	import { ClockIcon, DotIcon, ExternalLinkIcon } from '@lucide/svelte';

	interface Props {
		monitor: MonitorStats;
		chartType: string;
	}
	let { monitor, chartType }: Props = $props();

	const status = $derived(
		monitor.uptime_pct >= 99
			? { label: 'Operational', class: 'bg-emerald-500/15 text-emerald-600 border-emerald-500/20' }
			: monitor.uptime_pct >= 95
				? { label: 'Degraded', class: 'bg-amber-500/15 text-amber-600 border-amber-500/20' }
				: { label: 'Down', class: 'bg-red-500/15 text-red-600 border-red-500/20' }
	);

	const uptimeColor = $derived(
		monitor.uptime_pct >= 99
			? 'text-emerald-600'
			: monitor.uptime_pct >= 95
				? 'text-amber-600'
				: 'text-red-600'
	);

	const getLatencyClass = (ms: number | undefined | null) => {
		if (!ms) return 'text-muted-foreground';
		if (ms < 200) return 'text-emerald-600';
		if (ms < 500) return 'text-amber-600';
		return 'text-red-600';
	};
</script>

<Card.Root
	class="group relative overflow-hidden border-border/50 bg-linear-to-b from-card to-card/80 pb-0 transition-all duration-300 hover:border-border hover:shadow-lg"
>
	<Card.Header class="flex items-start justify-between">
		<div class="space-y-0.5">
			<h3 class="truncate font-semibold tracking-tight">
				{monitor.name}
			</h3>

			<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
				<a
					href={monitor.url}
					target="_blank"
					rel="noreferrer"
					class="group/link flex items-center gap-1.5 truncate transition-colors hover:text-foreground"
				>
					<span class="truncate">{monitor.url}</span>
					<ExternalLinkIcon
						class="hidden size-3 shrink-0 transition-opacity group-hover/link:block"
					/>
					<DotIcon class="block size-3 shrink-0 transition-opacity group-hover/link:hidden" />
				</a>
				<span class="flex items-center gap-1 text-muted-foreground/70">
					<ClockIcon class="size-3" />
					{monitor.check_interval}s
				</span>
			</div>
		</div>

		<span class={cn('text-xl font-bold tracking-tight tabular-nums', uptimeColor)}>
			{monitor.uptime_pct.toFixed(2)}%
		</span>
	</Card.Header>

	<Card.Content class="p-0">
		<div class="relative">
			<div
				class="pointer-events-none absolute inset-y-0 left-0 z-10 w-4 bg-linear-to-r from-card to-transparent"
			></div>
			<div
				class="pointer-events-none absolute inset-y-0 right-0 z-10 w-4 bg-linear-to-l from-card to-transparent"
			></div>
			{#if chartType === 'bars'}
				<BarChart {monitor} />
			{:else}
				<AreaChart {monitor} />
			{/if}
		</div>

		<div class="grid grid-cols-3 gap-2 border-t bg-muted px-4 py-4 text-xs sm:grid-cols-5">
			<div class="col-span-1 flex flex-col gap-0.5">
				<span class="text-muted-foreground">Avg</span>
				<span class="font-medium {getLatencyClass(monitor.avg_response_time)}">
					{monitor.avg_response_time ?? '-'}ms
				</span>
			</div>

			<div class="col-span-1 flex flex-col gap-0.5 border-l pl-3">
				<span class="text-muted-foreground">P50</span>
				<span class="font-medium {getLatencyClass(monitor.percentiles.p50)}">
					{monitor.percentiles.p50 ?? '-'}ms
				</span>
			</div>

			<div class="col-span-1 flex flex-col gap-0.5 border-l pl-3">
				<span class="text-muted-foreground">P95</span>
				<span class="font-medium {getLatencyClass(monitor.percentiles.p95)}">
					{monitor.percentiles.p95 ?? '-'}ms
				</span>
			</div>

			<div class="col-span-1 hidden flex-col gap-0.5 border-l pl-3 sm:flex">
				<span class="text-muted-foreground">P99</span>
				<span class="font-medium {getLatencyClass(monitor.percentiles.p99)}">
					{monitor.percentiles.p99 ?? '-'}ms
				</span>
			</div>

			<div class="col-span-1 ml-auto hidden items-center justify-end sm:flex">
				<Badge variant="outline" class={cn('text-xs font-medium', status.class)}>
					{#if monitor.uptime_pct >= 95}
						<span class="relative mr-1.5 flex h-1.5 w-1.5">
							<span
								class="absolute inline-flex h-full w-full animate-ping rounded-full bg-current opacity-75"
							></span>
							<span class="relative inline-flex h-1.5 w-1.5 rounded-full bg-current"></span>
						</span>
					{/if}
					{status.label}
				</Badge>
			</div>
		</div>
	</Card.Content>
</Card.Root>

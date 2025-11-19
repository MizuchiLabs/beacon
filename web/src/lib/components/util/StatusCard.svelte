<script lang="ts">
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import { Area, AreaChart } from 'layerchart';
	import { scaleUtc } from 'd3-scale';
	import { curveMonotoneX } from 'd3-shape';
	import { useMonitorStatus, useMonitorStats, type ChartDataPoint } from '$lib/api/queries';
	import { Badge } from '$lib/components/ui/badge';
	import { Globe, Clock, TrendingUp, AlertCircle } from '@lucide/svelte';
	import ChartContainer from '../ui/chart/chart-container.svelte';

	interface Props {
		monitorId: number;
	}

	let { monitorId }: Props = $props();
	let timeRange = $state<'7' | '14' | '30'>('7');

	const statusQuery = useMonitorStatus(monitorId);
	const statsQuery = useMonitorStats(monitorId, timeRange);

	const status = $derived(statusQuery.data);
	const chartData = $derived(statsQuery.data || []);

	const isOnline = $derived(status?.last_status === 'up');
	const uptimePercent = $derived(() => {
		if (chartData.length === 0) return 0;
		const total = chartData.reduce((sum, d) => sum + d.total_checks, 0);
		const successful = chartData.reduce((sum, d) => sum + d.successful_checks, 0);
		return total > 0 ? (successful / total) * 100 : 0;
	});

	const avgResponseTime = $derived(() => {
		const validData = chartData.filter((d) => d.response_time !== null);
		if (validData.length === 0) return 0;
		const sum = validData.reduce((acc, d) => acc + (d.response_time || 0), 0);
		return Math.round(sum / validData.length);
	});

	const chartConfig = {
		uptime: {
			label: 'Uptime',
			color: isOnline ? 'hsl(var(--success))' : 'hsl(var(--destructive))'
		}
	};

	const formattedData = $derived(
		chartData.map((d) => ({
			date: new Date(d.timestamp),
			uptime: d.uptime_percent
		}))
	);
</script>

<Card.Root class="w-full max-w-2xl">
	<Card.Header class="pb-4">
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-3">
				<div class="flex size-10 items-center justify-center rounded-lg bg-muted">
					<Globe class="size-5 text-muted-foreground" />
				</div>
				<div>
					<Card.Title class="text-lg">{status?.name || 'Loading...'}</Card.Title>
					<Card.Description class="text-sm">{status?.url || ''}</Card.Description>
				</div>
			</div>
			<Badge variant={isOnline ? 'default' : 'destructive'} class="h-6">
				{isOnline ? 'Online' : 'Offline'}
			</Badge>
		</div>
	</Card.Header>

	<Card.Content class="space-y-4">
		<!-- Stats Grid -->
		<div class="grid grid-cols-3 gap-4">
			<div class="flex flex-col gap-1">
				<span class="text-xs text-muted-foreground">Uptime</span>
				<div class="flex items-baseline gap-1">
					<span class="text-2xl font-bold">{uptimePercent().toFixed(1)}</span>
					<span class="text-sm text-muted-foreground">%</span>
				</div>
			</div>
			<div class="flex flex-col gap-1">
				<span class="text-xs text-muted-foreground">Response Time</span>
				<div class="flex items-baseline gap-1">
					<span class="text-2xl font-bold">{avgResponseTime()}</span>
					<span class="text-sm text-muted-foreground">ms</span>
				</div>
			</div>
			<div class="flex flex-col gap-1">
				<span class="text-xs text-muted-foreground">Last Check</span>
				<div class="text-sm font-medium">
					{status?.last_check_at ? new Date(status.last_check_at).toLocaleTimeString() : 'Never'}
				</div>
			</div>
		</div>

		<!-- Chart -->
		<div class="space-y-2">
			<div class="flex items-center justify-between">
				<h4 class="text-sm font-medium">Uptime History</h4>
				<Select.Root type="single" bind:value={timeRange}>
					<Select.Trigger class="h-8 w-32 text-xs" aria-label="Select time range">
						Last {timeRange} days
					</Select.Trigger>
					<Select.Content class="rounded-lg">
						<Select.Item value="7" class="text-xs">Last 7 days</Select.Item>
						<Select.Item value="14" class="text-xs">Last 14 days</Select.Item>
						<Select.Item value="30" class="text-xs">Last 30 days</Select.Item>
					</Select.Content>
				</Select.Root>
			</div>

			<ChartContainer config={chartConfig} class="h-[120px] w-full">
				<AreaChart
					data={formattedData}
					x="date"
					xScale={scaleUtc()}
					series={[{ key: 'uptime', label: 'Uptime %', color: chartConfig.uptime.color }]}
					props={{
						area: {
							curve: curveMonotoneX,
							'fill-opacity': 0.2,
							line: { class: 'stroke-2' }
						},
						xAxis: {
							ticks: 5,
							format: (v) => v.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
						},
						yAxis: {
							format: (v) => `${v}%`,
							domain: [0, 100]
						}
					}}
				/>
			</ChartContainer>
		</div>
	</Card.Content>
</Card.Root>

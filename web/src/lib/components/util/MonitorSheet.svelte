<script lang="ts">
	import { resolve } from '$app/paths';
	import { getIncidents, useMonitorPercentiles, type MonitorStats } from '$lib/api/queries';
	import StatusChart from '$lib/components/chart/StatusChart.svelte';
	import * as Alert from '$lib/components/ui/alert';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Empty from '$lib/components/ui/empty';
	import * as Item from '$lib/components/ui/item';
	import { Separator } from '$lib/components/ui/separator';
	import * as Sheet from '$lib/components/ui/sheet';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import SubscribeBell from '$lib/components/util/SubscribeBell.svelte';
	import { pushNotifications } from '$lib/stores/push.svelte';
	import {
		affectsMonitor,
		ago,
		certTextClass,
		durationText,
		formatMs,
		incidentSeverity,
		incidentStatus,
		intervalText,
		latencyTextClass,
		statusMeta,
		targetOf,
		typeLabel,
		uptimeTextClass
	} from '$lib/status.js';
	import { cn } from '$lib/utils.js';
	import { CircleAlertIcon, ExternalLinkIcon } from '@lucide/svelte';

	interface Props {
		monitor: MonitorStats | null;
		open: boolean;
		onOpenChange: (open: boolean) => void;
	}
	let { monitor, open, onOpenChange }: Props = $props();

	const incidentsQuery = getIncidents();
	const percentilesQuery = useMonitorPercentiles(() => (open ? monitor?.id : undefined));

	// Layerchart walks every point on each render, plain objects keep that
	// cheap compared to the query's reactive proxies.
	const chartMonitor = $derived(monitor ? $state.snapshot(monitor) : null);
	const meta = $derived(monitor ? statusMeta[monitor.status] : null);
	const checks = $derived((monitor?.data_points ?? []).reduce((sum, p) => sum + p.total, 0));
	const incidents = $derived(
		monitor ? (incidentsQuery.data ?? []).filter((i) => affectsMonitor(i, monitor.name)) : []
	);
	const percentiles = $derived.by(() => {
		const p = percentilesQuery.data?.percentiles;
		if (!p) return [];
		return [
			{ label: 'P50', ms: p.p50 },
			{ label: 'P75', ms: p.p75 },
			{ label: 'P90', ms: p.p90 },
			{ label: 'P95', ms: p.p95 },
			{ label: 'P99', ms: p.p99 }
		];
	});
	const percentileMax = $derived(percentiles.at(-1)?.ms ?? 0);
	const monthDay = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric' });
</script>

<Sheet.Root {open} {onOpenChange}>
	<Sheet.Content class="sm:max-w-xl">
		{#if monitor && chartMonitor && meta}
			<Sheet.Header>
				<div class="flex items-start justify-between gap-3">
					<div class="min-w-0">
						<div class="flex items-center gap-2">
							<Sheet.Title>{monitor.name}</Sheet.Title>
							{#if typeLabel(monitor.type)}
								<Badge variant="secondary">{typeLabel(monitor.type)}</Badge>
							{/if}
						</div>
						<Sheet.Description class="mt-1 flex items-center">
							{#if monitor.url.startsWith('http')}
								<a
									href={monitor.url}
									target="_blank"
									rel="noreferrer"
									class="flex min-w-0 items-center gap-1 transition-colors hover:text-foreground"
								>
									<span class="truncate">{monitor.url}</span>
									<ExternalLinkIcon class="size-3 shrink-0" />
								</a>
							{:else}
								<span class="truncate">{monitor.url || targetOf(monitor)}</span>
							{/if}
						</Sheet.Description>
					</div>
					<div class="flex shrink-0 items-center gap-1.5 pr-6">
						<SubscribeBell monitorId={monitor.id} />
						<Badge variant="outline">
							<span class={cn('size-1.5 rounded-full', meta.dot)}></span>
							{meta.label}
						</Badge>
					</div>
				</div>
				<div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
					<span>
						Checked {monitor.last_checked_at ? ago(new Date(monitor.last_checked_at)) : 'never'}
					</span>
					<span>Every {intervalText(monitor.check_interval)}</span>
					{#if monitor.group}
						<span>{monitor.group}</span>
					{/if}
					{#if monitor.days_remaining != null}
						<span
							class={cn(
								'tabular-nums',
								monitor.ignore_cert_expiry
									? 'text-muted-foreground'
									: certTextClass(monitor.days_remaining)
							)}
						>
							Cert expires in {monitor.days_remaining}d
						</span>
					{/if}
				</div>
			</Sheet.Header>

			<Separator />

			<div class="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto p-5">
				<Card.Root size="sm">
					<Card.Content>
						<div class="grid grid-cols-3 gap-3 text-center">
							<div class="flex flex-col gap-0.5">
								<span class="text-[11px] text-muted-foreground">Uptime</span>
								<span
									class={cn(
										'text-base font-semibold tabular-nums',
										uptimeTextClass(monitor.uptime_pct)
									)}
								>
									{monitor.uptime_pct == null ? '-' : `${monitor.uptime_pct.toFixed(2)}%`}
								</span>
							</div>
							<div class="flex flex-col gap-0.5">
								<span class="text-[11px] text-muted-foreground">Avg response</span>
								<span
									class={cn(
										'text-base font-semibold tabular-nums',
										latencyTextClass(monitor.avg_response_time, monitor.degraded_threshold)
									)}
								>
									{formatMs(monitor.avg_response_time)}
								</span>
							</div>
							<div class="flex flex-col gap-0.5">
								<span class="text-[11px] text-muted-foreground">Checks</span>
								<span class="text-base font-semibold tabular-nums">{checks.toLocaleString()}</span>
							</div>
						</div>
					</Card.Content>
				</Card.Root>

				<section class="flex flex-col gap-2">
					<h3 class="text-xs font-medium text-muted-foreground">Uptime and response time</h3>
					<StatusChart monitor={chartMonitor} showTicks class="h-44" />
				</section>

				{#if percentiles.length > 0}
					<section class="flex flex-col gap-3">
						<div class="flex items-baseline justify-between">
							<h3 class="text-xs font-medium text-muted-foreground">Response time percentiles</h3>
							<span class="text-[11px] text-muted-foreground">
								degraded above {formatMs(monitor.degraded_threshold)}
							</span>
						</div>
						<div class="relative h-2 rounded-full bg-muted">
							{#each percentiles as row (row.label)}
								<Tooltip.Root>
									<Tooltip.Trigger
										class="absolute top-1/2 -translate-x-1/2 -translate-y-1/2 cursor-default"
										style="left: {Math.min(100, (row.ms / percentileMax) * 100)}%"
										aria-label="{row.label} response time {formatMs(row.ms)}"
									>
										<div
											class={cn(
												'size-3 rounded-full border-2 border-background transition-transform hover:scale-125',
												row.ms > monitor.degraded_threshold ? 'bg-chart-4' : 'bg-chart-3'
											)}
										></div>
									</Tooltip.Trigger>
									<Tooltip.Content>{row.label} · {formatMs(row.ms)}</Tooltip.Content>
								</Tooltip.Root>
							{/each}
						</div>
					</section>
				{/if}

				<section class="flex flex-col gap-2">
					<div class="flex items-center justify-between">
						<h3 class="text-xs font-medium text-muted-foreground">Incidents</h3>
						<Button variant="ghost" size="xs" href={resolve('/events')}>View all</Button>
					</div>
					{#if incidents.length === 0}
						<Empty.Root>
							<Empty.Header>
								<Empty.Description>No incidents recorded for this monitor</Empty.Description>
							</Empty.Header>
						</Empty.Root>
					{:else}
						<Item.Group>
							{#each incidents.slice(0, 5) as incident (incident.id)}
								{@const severity = incidentSeverity(incident.severity)}
								{@const status = incidentStatus(incident.status)}
								<Item.Root variant="outline" size="xs" role="listitem">
									<Item.Content class="min-w-0">
										<Item.Title>{incident.title}</Item.Title>
										<Item.Description>
											{monthDay.format(new Date(incident.started_at))}
											· {durationText(incident.started_at, incident.resolved_at)}
										</Item.Description>
									</Item.Content>
									<Item.Actions>
										<Badge variant={status.variant}>{status.label}</Badge>
										<Badge variant={severity.variant}>{severity.label}</Badge>
									</Item.Actions>
								</Item.Root>
							{/each}
						</Item.Group>
						{#if incidents.length > 5}
							<p class="text-center text-xs text-muted-foreground">
								and {incidents.length - 5} more
							</p>
						{/if}
					{/if}
				</section>

				{#if pushNotifications.error}
					<Alert.Root variant="destructive">
						<CircleAlertIcon />
						<Alert.Description>{pushNotifications.error}</Alert.Description>
					</Alert.Root>
				{/if}
			</div>
		{/if}
	</Sheet.Content>
</Sheet.Root>

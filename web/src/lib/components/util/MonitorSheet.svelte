<script lang="ts">
	import { resolve } from '$app/paths';
	import { getIncidents, useMonitorPercentiles, type MonitorStats } from '#lib/api/queries.js';
	import ResponseChart from '#lib/components/chart/ResponseChart.svelte';
	import UptimeBar from '#lib/components/chart/UptimeBar.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import * as Item from '#lib/components/ui/item/index.js';
	import { Separator } from '#lib/components/ui/separator/index.js';
	import * as Sheet from '#lib/components/ui/sheet/index.js';
	import SubscribeBell from '#lib/components/util/SubscribeBell.svelte';
	import { timeRange } from '#lib/range.svelte.js';
	import { pushNotifications } from '#lib/stores/push.svelte.js';
	import {
		affectsMonitor,
		ago,
		certTextClass,
		durationText,
		formatMs,
		incidentLevel,
		incidentSeverity,
		currentStatus,
		incidentWindow,
		intervalText,
		isUpcoming,
		latencyTextClass,
		statusMeta,
		targetOf,
		typeLabel,
		uptimeTextClass
	} from '#lib/status.js';
	import { cn } from '#lib/utils.js';
	import {
		CircleAlertIcon,
		ClockIcon,
		ExternalLinkIcon,
		FolderIcon,
		RefreshCwIcon,
		ShieldCheckIcon
	} from '@lucide/svelte';

	interface Props {
		monitor: MonitorStats | null;
		open: boolean;
		onOpenChange: (open: boolean) => void;
	}
	let { monitor, open, onOpenChange }: Props = $props();

	let panel = $state<HTMLElement | null>(null);

	const incidentsQuery = getIncidents();
	const percentilesQuery = useMonitorPercentiles(() => (open ? monitor?.id : undefined));

	const meta = $derived(monitor ? statusMeta[monitor.status] : null);
	// Layerchart walks every point on each render, plain objects keep that
	// cheap compared to the query's reactive proxies.
	const points = $derived(monitor?.data_points ? $state.snapshot(monitor.data_points) : []);
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
			{ label: 'P99', ms: p.p99 }
		];
	});
	const fills: Record<string, string> = {
		down: 'fill-chart-5/10',
		degraded: 'fill-chart-4/10',
		maintenance: 'fill-chart-2/10'
	};
	const incidentMarks = $derived(
		incidents.map((i) => {
			const [start, end] = incidentWindow(i);
			const level = incidentLevel(i);
			return { start, end, title: i.title, class: statusMeta[level].dot, fill: fills[level] };
		})
	);
	const p95 = $derived(percentilesQuery.data?.percentiles?.p95 ?? null);
	const monthDay = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric' });
</script>

<Sheet.Root {open} {onOpenChange}>
	<!-- Focus goes to the panel. The first control can be the bell, and its tooltip would open over the close button. -->
	<Sheet.Content
		bind:ref={panel}
		class="sm:max-w-xl"
		onOpenAutoFocus={(event) => {
			event.preventDefault();
			panel?.focus();
		}}
	>
		{#if monitor && meta}
			<Sheet.Header>
				<div class="flex min-w-0 flex-wrap items-center gap-2 pr-8">
					<Sheet.Title>{monitor.name}</Sheet.Title>
					{#if typeLabel(monitor.type)}
						<Badge variant="secondary">{typeLabel(monitor.type)}</Badge>
					{/if}
					<Badge variant="outline" class={meta.badge}>{meta.label}</Badge>
				</div>
				<div class="flex items-center justify-between gap-3">
					<Sheet.Description class="flex min-w-0 items-center">
						{#if monitor.url.startsWith('http')}
							<a
								href={monitor.url}
								target="_blank"
								rel="noreferrer"
								class="flex min-w-0 items-center gap-1 transition-colors hover:text-foreground"
							>
								<span class="truncate font-mono">{monitor.url}</span>
								<ExternalLinkIcon class="size-3 shrink-0" />
							</a>
						{:else}
							<span class={cn('truncate', monitor.type !== 'push' && 'font-mono')}>
								{monitor.url || targetOf(monitor)}
							</span>
						{/if}
					</Sheet.Description>
					<SubscribeBell monitorId={monitor.id} />
				</div>
				<div
					class="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground [&_svg]:size-3.5 [&_svg]:shrink-0"
				>
					<span class="flex items-center gap-1.5">
						<ClockIcon />
						Checked {monitor.last_checked_at ? ago(new Date(monitor.last_checked_at)) : 'never'}
					</span>
					<span class="flex items-center gap-1.5">
						<RefreshCwIcon />
						Every {intervalText(monitor.check_interval)}
					</span>
					{#if monitor.group}
						<span class="flex items-center gap-1.5">
							<FolderIcon />
							{monitor.group}
						</span>
					{/if}
					{#if monitor.days_remaining != null}
						<span
							class={cn(
								'flex items-center gap-1.5 tabular-nums',
								monitor.ignore_cert_expiry
									? 'text-muted-foreground'
									: certTextClass(monitor.days_remaining)
							)}
						>
							<ShieldCheckIcon />
							Cert expires in {monitor.days_remaining}d
						</span>
					{/if}
				</div>
			</Sheet.Header>

			<Separator />

			<div class="flex min-h-0 flex-1 flex-col gap-8 overflow-y-auto p-5">
				<div class="grid grid-cols-3 gap-4">
					<div class="flex flex-col gap-1">
						<span class="text-xs text-muted-foreground">Uptime · {timeRange.entry.label}</span>
						<span
							class={cn(
								'text-2xl font-semibold tracking-tight tabular-nums',
								uptimeTextClass(monitor.uptime_pct)
							)}
						>
							{monitor.uptime_pct == null ? '-' : `${monitor.uptime_pct.toFixed(2)}%`}
						</span>
					</div>
					<div class="flex flex-col gap-1">
						<span class="text-xs text-muted-foreground">Avg response</span>
						<span
							class={cn(
								'text-2xl font-semibold tracking-tight tabular-nums',
								latencyTextClass(monitor.avg_response_time, monitor.degraded_threshold)
							)}
						>
							{formatMs(monitor.avg_response_time)}
						</span>
					</div>
					<div class="flex flex-col gap-1">
						<span class="text-xs text-muted-foreground">P95</span>
						<span
							class={cn(
								'text-2xl font-semibold tracking-tight tabular-nums',
								latencyTextClass(p95, monitor.degraded_threshold)
							)}
						>
							{formatMs(p95)}
						</span>
					</div>
				</div>

				<section class="flex flex-col gap-2">
					<h3 class="text-xs font-medium text-muted-foreground">Uptime</h3>

					<UptimeBar
						{points}
						interval={monitor.check_interval}
						alert={monitor.status === 'down'}
						marks={incidentMarks}
					/>
				</section>

				<section class="flex flex-col gap-2">
					<div class="flex items-baseline justify-between">
						<h3 class="text-xs font-medium text-muted-foreground">Response time</h3>
						<span class="flex items-center gap-1.5 text-xs text-muted-foreground">
							<span class="w-3 border-t-2 border-dashed border-chart-4"></span>
							degraded above {formatMs(monitor.degraded_threshold)}
						</span>
					</div>
					<ResponseChart
						{points}
						threshold={monitor.degraded_threshold}
						ranges={incidentMarks.map((m) => ({ start: m.start, end: m.end, class: m.fill }))}
						class="h-44"
					/>
					{#if percentiles.length > 0}
						<dl class="grid grid-cols-4 gap-2 pt-2">
							{#each percentiles as row (row.label)}
								<div class="flex flex-col gap-0.5">
									<dt class="text-xs text-muted-foreground">{row.label}</dt>
									<dd
										class={cn(
											'text-sm font-medium tabular-nums',
											row.ms > monitor.degraded_threshold && 'text-chart-4'
										)}
									>
										{formatMs(row.ms)}
									</dd>
								</div>
							{/each}
						</dl>
					{/if}
				</section>

				<section class="flex flex-col gap-2">
					<div class="flex items-center justify-between">
						<h3 class="text-xs font-medium text-muted-foreground">Incidents</h3>
						<Button variant="ghost" size="xs" href={resolve('events')}>View all</Button>
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
								<Item.Root variant="outline" size="xs" role="listitem">
									{#snippet child({ props })}
										<a href={resolve('/events/[id]', { id: incident.id })} {...props}>
											<Item.Content class="min-w-0">
												<Item.Title>{incident.title}</Item.Title>
												<Item.Description>
													{monthDay.format(new Date(incident.started_at))}
													{#if !isUpcoming(incident)}
														· {durationText(incident.started_at, incident.resolved_at)}
													{/if}
													· {currentStatus(incident).label}
												</Item.Description>
											</Item.Content>
											<Item.Actions>
												<Badge variant={severity.variant} class={severity.class}>
													<severity.icon data-icon="inline-start" />
													{severity.label}
												</Badge>
											</Item.Actions>
										</a>
									{/snippet}
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

<script lang="ts">
	import type { MonitorStats } from '$lib/api/generated/types.gen';
	import * as Sheet from '$lib/components/ui/sheet';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import * as Separator from '$lib/components/ui/separator';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import StatusChart from '$lib/components/chart/StatusChart.svelte';
	import SubscribeBell from '$lib/components/util/SubscribeBell.svelte';
	import { pushNotifications } from '$lib/stores/push.svelte';
	import { getIncidents } from '$lib/api/queries';
	import {
		affectsMonitor,
		ago,
		formatMs,
		incidentSeverity,
		incidentStatus,
		latencyTextClass,
		statusMeta,
		uptimeTextClass
	} from '$lib/status.js';
	import { resolve } from '$app/paths';
	import { ExternalLinkIcon } from '@lucide/svelte';
	import { cn } from '$lib/utils.js';

	interface Props {
		monitor: MonitorStats | null;
		open: boolean;
		onOpenChange: (open: boolean) => void;
	}
	let { monitor, open, onOpenChange }: Props = $props();

	const incidentsQuery = $derived(getIncidents());
	const incidents = $derived.by(() => {
		if (!monitor) return [];
		return (incidentsQuery.data ?? [])
			.filter((i) => affectsMonitor(i, monitor.name))
			.sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime());
	});

	const meta = $derived(monitor ? statusMeta[monitor.status] : null);
	const checks = $derived((monitor?.data_points ?? []).reduce((sum, p) => sum + p.total, 0));
	const percentileRows = $derived.by(() => {
		const p = monitor?.percentiles;
		if (!p) return [];
		return [
			{ label: 'P50', ms: p.p50 },
			{ label: 'P75', ms: p.p75 },
			{ label: 'P90', ms: p.p90 },
			{ label: 'P95', ms: p.p95 },
			{ label: 'P99', ms: p.p99 }
		];
	});
	const percentileMax = $derived(percentileRows[percentileRows.length - 1]?.ms ?? 0);

	function duration(startedAt: string, resolvedAt: string | null | undefined) {
		const end = resolvedAt ? new Date(resolvedAt) : new Date();
		const minutes = Math.floor((end.getTime() - new Date(startedAt).getTime()) / 60000);
		if (minutes < 60) return `${minutes}m`;
		return `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
	}
</script>

<Sheet.Root {open} {onOpenChange}>
	<Sheet.Content class="sm:max-w-xl">
		{#if monitor && meta}
			<Sheet.Header>
				<div class="flex items-start justify-between gap-3">
					<div class="min-w-0">
						<Sheet.Title class="truncate">{monitor.name}</Sheet.Title>
						<Sheet.Description class="mt-1 flex items-center gap-2 text-xs">
							<a
								href={monitor.url}
								target="_blank"
								rel="noreferrer"
								class="flex min-w-0 items-center gap-1 transition-colors hover:text-foreground"
							>
								<span class="truncate">{monitor.url}</span>
								<ExternalLinkIcon class="size-3 shrink-0" />
							</a>
						</Sheet.Description>
					</div>
					<div class="flex shrink-0 items-center gap-1.5 pr-6">
						<SubscribeBell monitorId={monitor.id} />
						<Badge variant="outline" class={cn('gap-1.5 text-xs', meta.badge)}>
							{meta.label}
						</Badge>
					</div>
				</div>
				<div class="flex items-center gap-4 text-xs text-muted-foreground">
					<span>
						Checked
						{monitor.last_checked_at ? ago(new Date(monitor.last_checked_at)) : 'never'}
					</span>
					<span>Every {monitor.check_interval}s</span>
				</div>
			</Sheet.Header>

			<Separator.Root />

			<div class="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto p-5">
				<section class="flex flex-col gap-2">
					<div class="flex items-baseline justify-between">
						<h3 class="text-xs font-medium text-muted-foreground">Uptime and response time</h3>
						<span class="text-sm font-semibold tabular-nums {uptimeTextClass(monitor.uptime_pct)}">
							{monitor.uptime_pct === null ? '-' : `${monitor.uptime_pct.toFixed(2)}%`}
						</span>
					</div>
					<StatusChart {monitor} showTicks class="h-44" />
				</section>

				{#if percentileRows.length > 0}
					<section class="flex flex-col gap-3">
						<h3 class="text-xs font-medium text-muted-foreground">Response time percentiles</h3>
						<div class="relative h-2 rounded-full bg-muted">
							{#each percentileRows as row (row.label)}
								<Tooltip.Root>
									<Tooltip.Trigger
										class="absolute top-1/2 size-3 -translate-x-1/2 -translate-y-1/2 cursor-default rounded-full border-2 border-background bg-muted-foreground/50 p-0 transition-colors hover:bg-muted-foreground/80"
										style="left: {Math.min(100, (row.ms / percentileMax) * 100)}%"
										aria-label="{row.label} response time {formatMs(row.ms)}"
									></Tooltip.Trigger>
									<Tooltip.Content class="px-2 py-1">
										{row.label} · {formatMs(row.ms)}
									</Tooltip.Content>
								</Tooltip.Root>
							{/each}
						</div>
					</section>
				{/if}

				<section class="flex flex-col gap-2">
					<div class="flex items-center justify-between">
						<h3 class="text-xs font-medium text-muted-foreground">Incidents</h3>
						<Button
							variant="ghost"
							size="sm"
							class="h-7 rounded-md px-2 text-xs"
							href={resolve('/events')}
						>
							View all
						</Button>
					</div>
					{#if incidents.length === 0}
						<p
							class="rounded-lg border border-dashed px-3 py-4 text-center text-xs text-muted-foreground"
						>
							No incidents recorded for this monitor
						</p>
					{:else}
						<ul class="flex flex-col gap-2">
							{#each incidents.slice(0, 5) as incident (incident.id)}
								<li class="flex flex-col gap-1 rounded-lg border px-3 py-2">
									<div class="flex items-center justify-between gap-2">
										<span class="truncate text-xs font-medium">{incident.title}</span>
										<Badge
											variant={incidentSeverity(incident.severity).variant}
											class="text-[10px]"
										>
											{incidentSeverity(incident.severity).label}
										</Badge>
									</div>
									<div class="flex items-center gap-2 text-[11px] text-muted-foreground">
										<Badge variant={incidentStatus(incident.status).variant} class="text-[10px]">
											{incidentStatus(incident.status).label}
										</Badge>
										<span>
											{new Date(incident.started_at).toLocaleDateString(undefined, {
												month: 'short',
												day: 'numeric'
											})}
											· {duration(incident.started_at, incident.resolved_at)}
										</span>
									</div>
								</li>
							{/each}
							{#if incidents.length > 5}
								<li class="text-center text-[11px] text-muted-foreground">
									and {incidents.length - 5} more
								</li>
							{/if}
						</ul>
					{/if}
				</section>

				<section class="grid grid-cols-3 gap-3 rounded-lg border px-3 py-3 text-center">
					<div class="flex flex-col gap-0.5">
						<span class="text-[11px] text-muted-foreground">Checks</span>
						<span class="text-xs font-medium tabular-nums">{checks.toLocaleString()}</span>
					</div>
					<div class="flex flex-col gap-0.5">
						<span class="text-[11px] text-muted-foreground">Avg response</span>
						<span
							class={cn(
								'text-xs font-medium tabular-nums',
								latencyTextClass(monitor.avg_response_time)
							)}
						>
							{formatMs(monitor.avg_response_time)}
						</span>
					</div>
					<div class="flex flex-col gap-0.5">
						<span class="text-[11px] text-muted-foreground">Interval</span>
						<span class="text-xs font-medium tabular-nums">{monitor.check_interval}s</span>
					</div>
				</section>

				{#if pushNotifications.error}
					<p class="text-chart-error text-xs">{pushNotifications.error}</p>
				{/if}
			</div>
		{/if}
	</Sheet.Content>
</Sheet.Root>

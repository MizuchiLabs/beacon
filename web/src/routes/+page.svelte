<script lang="ts">
	import { resolve } from '$app/paths';
	import type { MonitorStats } from '$lib/api/generated/types.gen';
	import { getIncidents, useMonitorStats } from '$lib/api/queries';
	import * as Alert from '$lib/components/ui/alert';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import * as Empty from '$lib/components/ui/empty';
	import * as Item from '$lib/components/ui/item';
	import { Separator } from '$lib/components/ui/separator';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import MonitorRow from '$lib/components/util/MonitorRow.svelte';
	import MonitorSheet from '$lib/components/util/MonitorSheet.svelte';
	import OverallBanner from '$lib/components/util/OverallBanner.svelte';
	import TimeRange from '$lib/components/util/TimeRange.svelte';
	import {
		aggregateStatus,
		ago,
		durationText,
		incidentSeverity,
		isActiveIncident,
		statusMeta
	} from '$lib/status.js';
	import { ArrowRightIcon, CircleCheckIcon } from '@lucide/svelte';

	const statsQuery = $derived(useMonitorStats());
	const incidentsQuery = $derived(getIncidents());

	let sheetOpen = $state(false);
	let selected = $state<MonitorStats | null>(null);

	function openMonitor(monitor: MonitorStats) {
		selected = monitor;
		sheetOpen = true;
	}

	const activeIncidents = $derived((incidentsQuery.data ?? []).filter(isActiveIncident));
	const pastIncidents = $derived.by(() => {
		const cutoff = Date.now() - 30 * 24 * 60 * 60 * 1000;
		return (incidentsQuery.data ?? [])
			.filter(
				(incident) =>
					!isActiveIncident(incident) && new Date(incident.started_at).getTime() >= cutoff
			)
			.slice(0, 5);
	});

	let now = $state(Date.now());
	$effect(() => {
		const timer = setInterval(() => (now = Date.now()), 10_000);
		return () => clearInterval(timer);
	});

	const updatedAgo = $derived.by(() => {
		void now;
		return statsQuery.dataUpdatedAt ? ago(new Date(statsQuery.dataUpdatedAt)) : null;
	});

	// The favicon doubles as a passive status light for pinned tabs.
	let faviconHref = $state('');
	$effect(() => {
		const monitors = statsQuery.data ?? [];
		if (monitors.length === 0) return;
		const style = getComputedStyle(document.documentElement);
		const token = statusMeta[aggregateStatus(monitors)].token;
		const color = style.getPropertyValue(token).trim();
		const canvas = document.createElement('canvas');
		canvas.width = 64;
		canvas.height = 64;
		const ctx = canvas.getContext('2d');
		if (!ctx) return;
		ctx.fillStyle = color;
		ctx.beginPath();
		ctx.arc(32, 32, 26, 0, Math.PI * 2);
		ctx.fill();
		faviconHref = canvas.toDataURL('image/png');
	});

	const monthDay = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric' });
</script>

<svelte:head>
	{#if faviconHref}
		<link rel="icon" type="image/png" href={faviconHref} />
	{/if}
</svelte:head>

<div class="mx-auto w-full space-y-4 p-6 sm:max-w-4xl">
	{#if statsQuery.isPending}
		<div class="flex flex-col gap-2">
			<Skeleton class="h-12 w-full rounded-xl" />
			<Skeleton class="h-24 w-full rounded-xl" />
			<Skeleton class="h-24 w-full rounded-xl" />
			<Skeleton class="h-24 w-full rounded-xl" />
		</div>
	{:else if statsQuery.isError}
		<Empty.Root class="border border-dashed">
			<Empty.Header>
				<Empty.Title>Could not load monitors</Empty.Title>
				<Empty.Description>Try refreshing the page.</Empty.Description>
			</Empty.Header>
		</Empty.Root>
	{:else if statsQuery.data?.length === 0}
		<Empty.Root class="border border-dashed">
			<Empty.Header>
				<Empty.Title>No monitors configured</Empty.Title>
				<Empty.Description>Add a monitor to your config file to start tracking.</Empty.Description>
			</Empty.Header>
		</Empty.Root>
	{:else}
		<div class="flex flex-col gap-4 sm:flex-row sm:items-center">
			<OverallBanner monitors={statsQuery.data ?? []} {updatedAgo} />
			<TimeRange />
		</div>

		{#if incidentsQuery.isSuccess && activeIncidents.length > 0}
			<div class="flex flex-col gap-2">
				{#each activeIncidents as incident (incident.id)}
					{@const severity = incidentSeverity(incident.severity)}
					{@const latest = incident.updates?.at(-1)}
					<Alert.Root variant={severity.variant === 'destructive' ? 'destructive' : 'default'}>
						<severity.icon />
						<Alert.Title class="flex flex-wrap items-center gap-2">
							{incident.title}
							<Badge variant={severity.variant} class="gap-1">
								<severity.icon class="size-3" />
								{severity.label}
							</Badge>
						</Alert.Title>
						<Alert.Description>
							{latest?.message ?? incident.description}
							{#if latest}
								· {ago(new Date(latest.created_at))}
							{/if}
							{#if incident.affected_monitors?.length}
								· affects {incident.affected_monitors.join(', ')}
							{/if}
						</Alert.Description>
					</Alert.Root>
				{/each}
			</div>
		{/if}

		<div class="flex flex-col divide-y rounded-xl border bg-card">
			{#each statsQuery.data as monitor (monitor.id)}
				<MonitorRow {monitor} onOpen={openMonitor} />
			{/each}
		</div>

		{#if incidentsQuery.isSuccess}
			<section class="space-y-2 pt-2">
				<div class="flex items-center justify-between">
					<h2 class="text-sm font-medium text-muted-foreground">Past incidents</h2>
					<Button
						variant="ghost"
						size="sm"
						class="h-7 gap-1 rounded-md px-2 text-xs text-muted-foreground"
						href={resolve('/events')}
					>
						View all
						<ArrowRightIcon />
					</Button>
				</div>

				<div class="overflow-hidden rounded-xl border bg-card">
					{#if pastIncidents.length === 0}
						<div
							class="flex items-center justify-center gap-2 px-4 py-6 text-sm text-muted-foreground"
						>
							<CircleCheckIcon class="size-4 text-chart-3" />
							No incidents in the last 30 days
						</div>
					{:else}
						{#each pastIncidents as incident, i (incident.id)}
							{#if i > 0}
								<Separator />
							{/if}
							{@const severity = incidentSeverity(incident.severity)}
							<Item.Root size="sm">
								<Item.Media class="shrink-0">
									<CircleCheckIcon class="size-4 text-chart-3" />
								</Item.Media>
								<Item.Content class="min-w-0">
									<Item.Title class="truncate text-sm">{incident.title}</Item.Title>
									<Item.Description class="text-xs">
										{monthDay.format(new Date(incident.started_at))}
										· {durationText(incident.started_at, incident.resolved_at)}
										{#if incident.affected_monitors?.length}
											· affects {incident.affected_monitors.join(', ')}
										{/if}
									</Item.Description>
								</Item.Content>
								<Item.Actions class="shrink-0">
									<Badge variant={severity.variant} class="gap-1 text-xs">
										<severity.icon class="size-3" />
										{severity.label}
									</Badge>
								</Item.Actions>
							</Item.Root>
						{/each}
					{/if}
				</div>
			</section>
		{/if}
	{/if}
</div>

<MonitorSheet monitor={selected} open={sheetOpen} onOpenChange={(v) => (sheetOpen = v)} />

<script lang="ts">
	import { resolve } from '$app/paths';
	import { getIncidents, useConfig, useMonitorStats } from '$lib/api/queries';
	import * as Alert from '$lib/components/ui/alert';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import * as Empty from '$lib/components/ui/empty';
	import * as InputGroup from '$lib/components/ui/input-group';
	import * as Item from '$lib/components/ui/item';
	import { Kbd } from '$lib/components/ui/kbd';
	import { Separator } from '$lib/components/ui/separator';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import MonitorRow from '$lib/components/util/MonitorRow.svelte';
	import MonitorSheet from '$lib/components/util/MonitorSheet.svelte';
	import OverallBanner from '$lib/components/util/OverallBanner.svelte';
	import TimeRange from '$lib/components/util/TimeRange.svelte';
	import { timeRange, timeRanges } from '$lib/range.svelte';
	import {
		aggregateStatus,
		ago,
		durationText,
		groupMonitors,
		incidentSeverity,
		isActiveIncident,
		statusMeta,
		statusTitle,
		targetOf
	} from '$lib/status.js';
	import { ArrowRightIcon, CheckIcon, CircleCheckIcon, SearchIcon } from '@lucide/svelte';
	import { useInterval, watch } from 'runed';
	import { toast } from 'svelte-sonner';
	import { SvelteSet } from 'svelte/reactivity';

	// A filter only earns its space once the list gets long.
	const filterThreshold = 7;

	const statsQuery = useMonitorStats();
	const incidentsQuery = getIncidents();
	const configQuery = useConfig();

	const monitors = $derived(statsQuery.data ?? []);
	const brand = $derived(configQuery.data?.title ?? 'Beacon');

	let sheetOpen = $state(false);
	let selectedId = $state<number | null>(null);
	// Derived from the live query, so the open sheet updates on every refetch.
	const selected = $derived(monitors.find((m) => m.id === selectedId) ?? null);

	function openMonitor(id: number) {
		selectedId = id;
		sheetOpen = true;
	}

	let filter = $state('');
	let filterInput = $state<HTMLInputElement | null>(null);
	const showFilter = $derived(monitors.length >= filterThreshold || filter !== '');
	const groups = $derived.by(() => {
		const needle = filter.trim().toLowerCase();
		const matches = needle
			? monitors.filter((m) =>
					[m.name, m.group, targetOf(m)].some((field) => field.toLowerCase().includes(needle))
				)
			: monitors;
		return groupMonitors(matches);
	});

	// Status changes between two polls get a toast and a short row flash.
	const flashing = new SvelteSet<number>();
	watch(
		() => statsQuery.data,
		(current, previous) => {
			if (!current || !previous) return;
			const before = new Map(previous.map((m) => [m.id, m.status]));
			for (const m of current) {
				const was = before.get(m.id);
				if (!was || was === m.status || was === 'unknown' || m.status === 'unknown') continue;

				flashing.add(m.id);
				setTimeout(() => flashing.delete(m.id), 1800);
				if (m.status === 'down') toast.error(`${m.name} is down`);
				else if (was === 'down') toast.success(`${m.name} is back up`);
				else if (m.status === 'degraded') toast.warning(`${m.name} is degraded`);
				else toast.success(`${m.name} is operational again`);
			}
		}
	);

	function onkeydown(event: KeyboardEvent) {
		if (event.metaKey || event.ctrlKey || event.altKey || event.defaultPrevented) return;
		const target = event.target as HTMLElement;
		if (target.closest('input, textarea, select, [contenteditable="true"], [role="dialog"]'))
			return;

		if (event.key === '/' && filterInput) {
			event.preventDefault();
			filterInput.focus();
			return;
		}
		const range = timeRanges[Number(event.key) - 1];
		if (range) timeRange.current = range.value;
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

	const clock = useInterval(10_000);
	const updatedAgo = $derived.by(() => {
		void clock.counter;
		return statsQuery.dataUpdatedAt ? ago(new Date(statsQuery.dataUpdatedAt)) : null;
	});

	// The favicon doubles as a passive status light for pinned tabs.
	let faviconHref = $state('');
	$effect(() => {
		if (monitors.length === 0) return;
		const style = getComputedStyle(document.documentElement);
		const token = statusMeta[aggregateStatus(monitors)].token;
		const canvas = document.createElement('canvas');
		canvas.width = 64;
		canvas.height = 64;
		const ctx = canvas.getContext('2d');
		if (!ctx) return;
		ctx.fillStyle = style.getPropertyValue(token).trim();
		ctx.beginPath();
		ctx.arc(32, 32, 26, 0, Math.PI * 2);
		ctx.fill();
		faviconHref = canvas.toDataURL('image/png');
	});

	const monthDay = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric' });
</script>

<svelte:window {onkeydown} />

<svelte:head>
	<title>{statusTitle(monitors, brand)}</title>
	{#if faviconHref}
		<link rel="icon" type="image/png" href={faviconHref} />
	{/if}
</svelte:head>

<div class="mx-auto w-full space-y-4 p-6 sm:max-w-4xl">
	{#if statsQuery.isError && !statsQuery.data}
		<div class="rounded-3xl border border-dashed">
			<Empty.Root>
				<Empty.Header>
					<Empty.Title>Could not load monitors</Empty.Title>
					<Empty.Description>Try refreshing the page.</Empty.Description>
				</Empty.Header>
			</Empty.Root>
		</div>
	{:else if !statsQuery.data}
		<div class="flex flex-col gap-2">
			<Skeleton class="h-12 w-full" />
			<Skeleton class="h-24 w-full" />
			<Skeleton class="h-24 w-full" />
			<Skeleton class="h-24 w-full" />
		</div>
	{:else if statsQuery.data?.length === 0}
		<div class="rounded-3xl border border-dashed">
			<Empty.Root>
				<Empty.Header>
					<Empty.Title>No monitors configured</Empty.Title>
					<Empty.Description>Add a monitor to your config file to start tracking.</Empty.Description
					>
				</Empty.Header>
			</Empty.Root>
		</div>
	{:else}
		<div class="flex flex-col gap-4 sm:flex-row sm:items-center">
			<OverallBanner {monitors} {updatedAgo} />
			<TimeRange />
		</div>

		{#if incidentsQuery.isSuccess && activeIncidents.length > 0}
			<div class="flex flex-col gap-2">
				{#each activeIncidents as incident (incident.id)}
					{@const severity = incidentSeverity(incident.severity)}
					{@const latest = incident.updates?.at(-1)}
					<Alert.Root variant={severity.variant === 'destructive' ? 'destructive' : 'default'}>
						<severity.icon />
						<Alert.Title class="flex flex-wrap items-center">
							{incident.title}
							<Badge variant={severity.variant} class="ml-2">
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

		{#if showFilter}
			<InputGroup.Root>
				<InputGroup.Input
					bind:ref={filterInput}
					bind:value={filter}
					type="search"
					placeholder="Filter monitors"
					aria-label="Filter monitors"
					onkeydown={(event) => {
						if (event.key === 'Escape') {
							filter = '';
							filterInput?.blur();
						}
					}}
				/>
				<InputGroup.Addon>
					<SearchIcon />
				</InputGroup.Addon>
				<InputGroup.Addon align="inline-end">
					<Kbd>/</Kbd>
				</InputGroup.Addon>
			</InputGroup.Root>
		{/if}

		{#each groups as group (group.name)}
			<section class="space-y-2">
				{#if group.name}
					<h2 class="flex items-center gap-2 px-1 text-sm font-medium text-muted-foreground">
						{group.name}
						<span class="text-xs tabular-nums opacity-60">{group.monitors.length}</span>
					</h2>
				{/if}
				<div class="flex flex-col divide-y overflow-hidden rounded-2xl border bg-card">
					{#each group.monitors as monitor (monitor.id)}
						<MonitorRow {monitor} flash={flashing.has(monitor.id)} onOpen={openMonitor} />
					{/each}
				</div>
			</section>
		{:else}
			<p class="py-8 text-center text-sm text-muted-foreground">No monitor matches "{filter}"</p>
		{/each}

		{#if incidentsQuery.isSuccess}
			<section class="space-y-2 pt-2">
				<div class="flex items-center justify-between">
					<h2 class="text-sm font-medium text-muted-foreground">Past incidents</h2>
					<Button variant="ghost" size="xs" href={resolve('/events')}>
						View all
						<ArrowRightIcon />
					</Button>
				</div>

				<div class="overflow-hidden rounded-xl border bg-card">
					{#if pastIncidents.length === 0}
						<Empty.Root>
							<Empty.Header>
								<Empty.Media variant="icon">
									<CheckIcon />
								</Empty.Media>
								<Empty.Title>Smooth Sailing</Empty.Title>
								<Empty.Description>No incidents in the last 30 days.</Empty.Description>
							</Empty.Header>
						</Empty.Root>
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
									<Item.Title>{incident.title}</Item.Title>
									<Item.Description>
										{monthDay.format(new Date(incident.started_at))}
										· {durationText(incident.started_at, incident.resolved_at)}
										{#if incident.affected_monitors?.length}
											· affects {incident.affected_monitors.join(', ')}
										{/if}
									</Item.Description>
								</Item.Content>
								<Item.Actions class="shrink-0">
									<Badge variant={severity.variant}>
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

<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { getIncidents, useConfig, useMonitorStats } from '#lib/api/queries.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import * as InputGroup from '#lib/components/ui/input-group/index.js';
	import { Kbd } from '#lib/components/ui/kbd/index.js';
	import { Separator } from '#lib/components/ui/separator/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import IncidentRow from '#lib/components/util/IncidentRow.svelte';
	import MonitorRow from '#lib/components/util/MonitorRow.svelte';
	import MonitorSheet from '#lib/components/util/MonitorSheet.svelte';
	import OverallBanner from '#lib/components/util/OverallBanner.svelte';
	import TimeRange from '#lib/components/util/TimeRange.svelte';
	import { usePageStatus } from '#lib/hooks/page-status.svelte.js';
	import { timeRange, timeRanges } from '#lib/range.svelte.js';
	import {
		ago,
		currentStatus,
		groupMonitors,
		incidentLevel,
		incidentSeverity,
		isActiveIncident,
		isUpcoming,
		statusTitle,
		targetOf
	} from '#lib/status.js';
	import { cn } from '#lib/utils.js';
	import {
		ArrowRightIcon,
		CalendarClockIcon,
		CheckIcon,
		SearchIcon,
		SearchXIcon
	} from '@lucide/svelte';
	import { useInterval, watch } from 'runed';
	import { toast } from 'svelte-sonner';
	import { SvelteSet } from 'svelte/reactivity';

	// A filter only earns its space once the list gets long.
	const filterThreshold = 7;

	const statsQuery = useMonitorStats();
	const incidentsQuery = getIncidents();
	const configQuery = useConfig();
	const status = usePageStatus();

	const monitors = $derived(statsQuery.data ?? []);
	const brand = $derived(configQuery.data?.title ?? 'Beacon');

	// The open monitor is kept in the url, so a link to it can be shared.
	const linkedId = Number(page.url.searchParams.get('m')) || null;
	let selectedId = $state(linkedId);
	let sheetOpen = $state(linkedId !== null);
	// Derived from the live query, so the open sheet updates on every refetch.
	const selected = $derived(monitors.find((m) => m.id === selectedId) ?? null);

	function setSheet(open: boolean, id = selectedId) {
		selectedId = id;
		sheetOpen = open;
		const url = new URL(page.url.href);
		if (open && id !== null) url.searchParams.set('m', String(id));
		else url.searchParams.delete('m');
		goto(url, { state: page.state, shallow: true, replace: true });
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

	const incidents = $derived(incidentsQuery.data ?? []);
	const activeIncidents = $derived(incidents.filter(isActiveIncident));
	// Soonest first, the API sorts newest first.
	const upcoming = $derived(incidents.filter(isUpcoming).toReversed());
	const pastIncidents = $derived.by(() => {
		const cutoff = Date.now() - 30 * 24 * 60 * 60 * 1000;
		return incidents
			.filter((i) => i.status === 'resolved' && Date.parse(i.started_at) >= cutoff)
			.slice(0, 5);
	});

	const clock = useInterval(10_000);
	const updatedAgo = $derived.by(() => {
		void clock.counter;
		return statsQuery.dataUpdatedAt ? ago(new Date(statsQuery.dataUpdatedAt)) : null;
	});

	const whenFormat = new Intl.DateTimeFormat(undefined, {
		weekday: 'short',
		month: 'short',
		day: 'numeric',
		hour: 'numeric',
		minute: '2-digit'
	});
	const timeFormat = new Intl.DateTimeFormat(undefined, { hour: 'numeric', minute: '2-digit' });
</script>

<svelte:window {onkeydown} />
<svelte:head>
	<title>{status.stale ? `offline · ${brand}` : statusTitle(monitors, brand)}</title>
</svelte:head>

<div class="mx-auto flex w-full flex-col gap-4 p-6 sm:max-w-4xl">
	{#if statsQuery.isError && !statsQuery.data}
		<Empty.Root>
			<Empty.Header>
				<Empty.Title>Could not load monitors</Empty.Title>
				<Empty.Description>Try refreshing the page.</Empty.Description>
			</Empty.Header>
		</Empty.Root>
	{:else if !statsQuery.data}
		<div class="flex flex-col gap-2">
			<Skeleton class="h-12 w-full" />
			<Skeleton class="h-24 w-full" />
			<Skeleton class="h-24 w-full" />
			<Skeleton class="h-24 w-full" />
		</div>
	{:else if statsQuery.data?.length === 0}
		<Empty.Root>
			<Empty.Header>
				<Empty.Title>No monitors configured</Empty.Title>
				<Empty.Description>Add a monitor to your config file to start tracking.</Empty.Description>
			</Empty.Header>
		</Empty.Root>
	{:else}
		<OverallBanner {updatedAgo} />

		{#if activeIncidents.length > 0 || upcoming.length > 0}
			<div class="flex flex-col gap-2">
				{#each activeIncidents as incident (incident.id)}
					{@const severity = incidentSeverity(incident.severity)}
					{@const stage = currentStatus(incident)}
					{@const latest = incident.updates?.at(-1)}
					<a
						href={resolve('/events/[id]', { id: incident.id })}
						class="group rounded-2xl outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50"
					>
						<Alert.Root variant={incidentLevel(incident) === 'down' ? 'destructive' : 'default'}>
							<severity.icon />
							<Alert.Title class="flex flex-wrap items-center">
								<span class="group-hover:underline">{incident.title}</span>
								<Badge variant={stage.variant} class="ml-2">
									<stage.icon data-icon="inline-start" />
									{stage.label}
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
					</a>
				{/each}
				{#each upcoming as incident (incident.id)}
					<a
						href={resolve('/events/[id]', { id: incident.id })}
						class="group rounded-2xl outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50"
					>
						<Alert.Root>
							<CalendarClockIcon />
							<Alert.Title>
								<span class="group-hover:underline">{incident.title}</span>
							</Alert.Title>
							<Alert.Description>
								Scheduled for {whenFormat.format(new Date(incident.started_at))}
								{#if incident.ends_at}
									to {timeFormat.format(new Date(incident.ends_at))}
								{/if}
								{#if incident.affected_monitors?.length}
									· affects {incident.affected_monitors.join(', ')}
								{/if}
							</Alert.Description>
						</Alert.Root>
					</a>
				{/each}
			</div>
		{/if}

		<div class="flex flex-wrap items-center justify-between gap-2 pt-2">
			<h2 class="px-1 text-sm font-medium text-muted-foreground">
				Uptime · {timeRange.entry.title.toLowerCase()}
			</h2>
			<TimeRange />
		</div>

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
			<section
				class={cn(
					'flex flex-col gap-2 transition-[opacity,filter] duration-500',
					status.stale && 'opacity-50 grayscale'
				)}
			>
				{#if group.name}
					<h2 class="flex items-center gap-2 px-1 text-sm font-medium text-muted-foreground">
						{group.name}
						<Badge variant="secondary">{group.monitors.length}</Badge>
					</h2>
				{/if}
				<div role="list" class="flex flex-col overflow-hidden rounded-2xl border bg-card">
					{#each group.monitors as monitor, i (monitor.id)}
						{#if i > 0}
							<Separator />
						{/if}

						<MonitorRow
							{monitor}
							flash={flashing.has(monitor.id)}
							onOpen={(id) => setSheet(true, id)}
						/>
					{/each}
				</div>
			</section>
		{:else}
			<Empty.Root>
				<Empty.Header>
					<Empty.Media variant="icon">
						<SearchXIcon />
					</Empty.Media>
					<Empty.Title>No matches</Empty.Title>
					<Empty.Description>No monitor matches "{filter}".</Empty.Description>
				</Empty.Header>
			</Empty.Root>
		{/each}

		{#if incidentsQuery.isSuccess}
			<section class="flex flex-col gap-2 pt-2">
				<div class="flex items-center justify-between">
					<h2 class="px-1 text-sm font-medium text-muted-foreground">Past incidents</h2>
					<Button variant="ghost" size="xs" href={resolve('events')}
						>View all <ArrowRightIcon data-icon="inline-end" /></Button
					>
				</div>

				<div class="flex flex-col overflow-hidden rounded-2xl border bg-card">
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
							<IncidentRow {incident} />
						{/each}
					{/if}
				</div>
			</section>
		{/if}
	{/if}
</div>

<MonitorSheet monitor={selected} open={sheetOpen && selected !== null} onOpenChange={setSheet} />

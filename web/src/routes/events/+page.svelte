<script lang="ts">
	import { getIncidents, useConfig } from '#lib/api/queries.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import { Separator } from '#lib/components/ui/separator/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import IncidentCard from '#lib/components/util/IncidentCard.svelte';
	import IncidentRow from '#lib/components/util/IncidentRow.svelte';
	import { isActiveIncident, isUpcoming } from '#lib/status.js';
	import { CheckIcon, RssIcon } from '@lucide/svelte';

	const monthFormat = new Intl.DateTimeFormat(undefined, { month: 'long', year: 'numeric' });

	const incidentsQuery = getIncidents();
	const configQuery = useConfig();
	const incidents = $derived(incidentsQuery.data ?? []);

	const open = $derived(incidents.filter((i) => isActiveIncident(i) || isUpcoming(i)));
	const months = $derived.by(() => {
		const groups = new Map<string, typeof incidents>();
		for (const incident of incidents) {
			if (incident.status !== 'resolved') continue;
			const month = monthFormat.format(new Date(incident.started_at));
			groups.set(month, [...(groups.get(month) ?? []), incident]);
		}
		return [...groups];
	});
</script>

<svelte:head>
	<title>Incidents · {configQuery.data?.title ?? 'Beacon'}</title>
	<link rel="alternate" type="application/atom+xml" title="Incidents" href="/incidents.atom" />
</svelte:head>

<div class="mx-auto flex w-full flex-col gap-6 p-6 sm:max-w-4xl">
	<div class="flex items-center justify-between gap-2">
		<h1 class="text-xl font-semibold tracking-tight">Incidents</h1>
		<Button variant="ghost" size="sm" href="/incidents.atom" data-sveltekit-reload>
			<RssIcon data-icon="inline-start" />
			Feed
		</Button>
	</div>

	{#if incidentsQuery.isPending}
		<Skeleton class="h-40 w-full" />
		<Skeleton class="h-24 w-full" />
	{:else if incidentsQuery.isError}
		<Empty.Root>
			<Empty.Header>
				<Empty.Title>Could not load incidents</Empty.Title>
				<Empty.Description>Try refreshing the page.</Empty.Description>
			</Empty.Header>
		</Empty.Root>
	{:else if incidents.length === 0}
		<Empty.Root>
			<Empty.Header>
				<Empty.Media variant="icon">
					<CheckIcon />
				</Empty.Media>
				<Empty.Title>No incidents found</Empty.Title>
				<Empty.Description>Everything is working as expected.</Empty.Description>
			</Empty.Header>
		</Empty.Root>
	{:else}
		{#each open as incident (incident.id)}
			<IncidentCard {incident} link />
		{/each}

		{#each months as [month, list] (month)}
			<section class="flex flex-col gap-2">
				<h2 class="flex items-center gap-2 px-1 text-sm font-medium text-muted-foreground">
					{month}
					<Badge variant="secondary">{list.length}</Badge>
				</h2>
				<div class="flex flex-col overflow-hidden rounded-2xl border bg-card">
					{#each list as incident, i (incident.id)}
						{#if i > 0}
							<Separator />
						{/if}
						<IncidentRow {incident} />
					{/each}
				</div>
			</section>
		{/each}
	{/if}
</div>

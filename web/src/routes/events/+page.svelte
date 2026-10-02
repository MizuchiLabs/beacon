<script lang="ts">
	import { resolve } from '$app/paths';
	import { getIncidents } from '#lib/api/queries.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import * as Item from '#lib/components/ui/item/index.js';
	import { Separator } from '#lib/components/ui/separator/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import IncidentCard from '#lib/components/util/IncidentCard.svelte';
	import { durationText, incidentSeverity, isActiveIncident, isUpcoming } from '#lib/status.js';
	import { CheckIcon, ChevronRightIcon, RssIcon } from '@lucide/svelte';

	const monthFormat = new Intl.DateTimeFormat(undefined, { month: 'long', year: 'numeric' });
	const dayFormat = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric' });

	const incidentsQuery = getIncidents();
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
	<link rel="alternate" type="application/atom+xml" title="Incidents" href="/incidents.atom" />
</svelte:head>

<div class="mx-auto flex w-full flex-col gap-6 p-6 sm:max-w-3xl">
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
						{@const severity = incidentSeverity(incident.severity)}
						<Item.Root size="sm">
							{#snippet child({ props })}
								<a href={resolve('/events/[id]', { id: incident.id })} {...props}>
									<Item.Content class="min-w-0">
										<Item.Title>{incident.title}</Item.Title>
										<Item.Description>
											{dayFormat.format(new Date(incident.started_at))}
											· {durationText(incident.started_at, incident.resolved_at)}
											{#if incident.affected_monitors?.length}
												· {incident.affected_monitors.join(', ')}
											{/if}
										</Item.Description>
									</Item.Content>
									<Item.Actions class="shrink-0">
										<Badge variant={severity.variant}>
											<severity.icon data-icon="inline-start" />
											{severity.label}
										</Badge>
										<ChevronRightIcon class="size-4 text-muted-foreground/50" />
									</Item.Actions>
								</a>
							{/snippet}
						</Item.Root>
					{/each}
				</div>
			</section>
		{/each}
	{/if}
</div>

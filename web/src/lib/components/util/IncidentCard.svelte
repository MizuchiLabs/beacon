<script lang="ts">
	import { resolve } from '$app/paths';
	import type { Incident } from '$lib/api/queries';
	import { Badge } from '$lib/components/ui/badge';
	import * as Card from '$lib/components/ui/card';
	import IncidentTimeline from '$lib/components/util/IncidentTimeline.svelte';
	import { currentStatus, durationText, incidentSeverity, isUpcoming } from '$lib/status.js';

	interface Props {
		incident: Incident;
		link?: boolean;
	}
	let { incident, link = false }: Props = $props();

	const severity = $derived(incidentSeverity(incident.severity));
	const status = $derived(currentStatus(incident));
	const format = new Intl.DateTimeFormat(undefined, {
		month: 'short',
		day: 'numeric',
		year: 'numeric',
		hour: 'numeric',
		minute: '2-digit',
		timeZoneName: 'short'
	});
</script>

<Card.Root>
	<Card.Header>
		<div class="flex flex-wrap items-center gap-2">
			<Badge variant={severity.variant}>
				<severity.icon data-icon="inline-start" />
				{severity.label}
			</Badge>
			<Badge variant={status.variant}>
				<status.icon data-icon="inline-start" />
				{status.label}
			</Badge>
		</div>
		<Card.Title>
			{#if link}
				<a href={resolve('/events/[id]', { id: incident.id })} class="hover:underline">
					{incident.title}
				</a>
			{:else}
				{incident.title}
			{/if}
		</Card.Title>
		{#if incident.description}
			<Card.Description>{incident.description}</Card.Description>
		{/if}
		<p class="text-xs text-muted-foreground">
			{#if isUpcoming(incident)}
				Starts {format.format(new Date(incident.started_at))}
			{:else}
				Started {format.format(new Date(incident.started_at))}
				· {incident.resolved_at ? 'lasted' : 'ongoing for'}
				{durationText(incident.started_at, incident.resolved_at)}
			{/if}
			{#if incident.ends_at && !incident.resolved_at}
				· planned until {format.format(new Date(incident.ends_at))}
			{/if}
			· {incident.affected_monitors?.length
				? `affects ${incident.affected_monitors.join(', ')}`
				: 'affects all monitors'}
		</p>
	</Card.Header>
	{#if incident.updates?.length}
		<Card.Content>
			<IncidentTimeline updates={incident.updates} />
		</Card.Content>
	{/if}
</Card.Root>

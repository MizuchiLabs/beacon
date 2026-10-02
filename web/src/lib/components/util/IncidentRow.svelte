<script lang="ts">
	import { resolve } from '$app/paths';
	import type { Incident } from '#lib/api/queries.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as Item from '#lib/components/ui/item/index.js';
	import { durationText, incidentSeverity } from '#lib/status.js';
	import { ChevronRightIcon } from '@lucide/svelte';

	interface Props {
		incident: Incident;
	}
	let { incident }: Props = $props();

	const severity = $derived(incidentSeverity(incident.severity));
	const dayFormat = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric' });
</script>

<Item.Root>
	{#snippet child({ props })}
		<a href={resolve('/events/[id]', { id: incident.id })} {...props}>
			<Item.Content class="min-w-0">
				<Item.Title>{incident.title}</Item.Title>
				<Item.Description>
					{dayFormat.format(new Date(incident.started_at))}
					· {durationText(incident.started_at, incident.resolved_at)}
					{#if incident.affected_monitors?.length}
						· affects {incident.affected_monitors.join(', ')}
					{/if}
				</Item.Description>
			</Item.Content>
			<Item.Actions class="shrink-0">
				<Badge variant={severity.variant} class={severity.class}>
					<severity.icon data-icon="inline-start" />
					{severity.label}
				</Badge>
				<ChevronRightIcon class="size-4 text-muted-foreground/50" />
			</Item.Actions>
		</a>
	{/snippet}
</Item.Root>

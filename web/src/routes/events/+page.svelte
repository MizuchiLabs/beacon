<script lang="ts">
	import { getIncidents } from '$lib/api/queries';
	import * as Card from '$lib/components/ui/card';
	import * as Empty from '$lib/components/ui/empty';
	import { Badge } from '$lib/components/ui/badge';
	import { Separator } from '$lib/components/ui/separator';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { durationText, incidentSeverity, incidentStatus } from '$lib/status.js';
	import { CheckIcon, ClockIcon } from '@lucide/svelte';

	const incidents = getIncidents();
	const dateTimeFormat = new Intl.DateTimeFormat(undefined, {
		month: 'short',
		day: 'numeric',
		year: 'numeric',
		hour: '2-digit',
		minute: '2-digit',
		timeZoneName: 'short'
	});
</script>

<div class="mx-auto flex w-full flex-col gap-4 p-6 sm:max-w-4xl">
	{#if incidents.isSuccess && (incidents.data?.length ?? 0) > 0}
		{#each incidents.data ?? [] as incident (incident.id ?? incident.started_at)}
			{@const severity = incidentSeverity(incident.severity)}
			{@const status = incidentStatus(incident.status)}

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
					<Card.Title>{incident.title}</Card.Title>
					<Card.Description>{incident.description}</Card.Description>
					<Card.Action>
						<Badge variant="outline">
							<ClockIcon data-icon="inline-start" />
							{durationText(incident.started_at, incident.resolved_at)}
						</Badge>
					</Card.Action>
					{#if (incident.affected_monitors?.length ?? 0) > 0}
						<div class="flex flex-wrap gap-2 pt-2">
							{#each incident.affected_monitors! as monitor (monitor)}
								<Badge variant="secondary">{monitor}</Badge>
							{/each}
						</div>
					{/if}
				</Card.Header>

				{#if (incident.updates?.length ?? 0) > 0}
					<Card.Content>
						<div class="flex flex-col gap-4">
							<Separator />
							<h3 class="text-sm font-semibold">Timeline</h3>
							<ol
								class="relative flex flex-col gap-4 pl-6 before:absolute before:top-2 before:left-2 before:h-[calc(100%-1rem)] before:w-px before:bg-border"
							>
								{#each incident.updates! as update (update.created_at)}
									{@const updateStatus = incidentStatus(update.status)}
									<li class="relative flex flex-col gap-1">
										<span
											class="absolute top-1 -left-6 flex size-4 items-center justify-center rounded-full bg-background"
										>
											<span class="size-2 rounded-full bg-primary"></span>
										</span>
										<div class="flex items-center gap-2">
											<Badge variant={updateStatus.variant}>
												<updateStatus.icon data-icon="inline-start" />
												{updateStatus.label}
											</Badge>
											<span class="text-xs text-muted-foreground">
												{dateTimeFormat.format(new Date(update.created_at))}
											</span>
										</div>
										<p class="text-sm">{update.message}</p>
									</li>
								{/each}
							</ol>
						</div>
					</Card.Content>
				{/if}

				<Card.Footer>
					<div class="flex w-full flex-wrap items-center justify-between gap-2">
						<span>Started: {dateTimeFormat.format(new Date(incident.started_at))}</span>
						{#if incident.resolved_at}
							<span>Resolved: {dateTimeFormat.format(new Date(incident.resolved_at))}</span>
						{/if}
					</div>
				</Card.Footer>
			</Card.Root>
		{/each}
	{:else if incidents.isPending}
		<Skeleton class="h-40 w-full" />
		<Skeleton class="h-40 w-full" />
	{:else if incidents.isError}
		<Empty.Root>
			<Empty.Header>
				<Empty.Title>Could not load incidents</Empty.Title>
				<Empty.Description>Try refreshing the page.</Empty.Description>
			</Empty.Header>
		</Empty.Root>
	{:else}
		<Empty.Root>
			<Empty.Header>
				<Empty.Media variant="icon">
					<CheckIcon />
				</Empty.Media>
				<Empty.Title>No incidents found</Empty.Title>
				<Empty.Description>Everything is working as expected.</Empty.Description>
			</Empty.Header>
		</Empty.Root>
	{/if}
</div>

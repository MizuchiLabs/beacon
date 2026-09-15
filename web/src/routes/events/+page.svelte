<script lang="ts">
	import { getIncidents, useConfig } from '$lib/api/queries';
	import * as Card from '$lib/components/ui/card';
	import * as Empty from '$lib/components/ui/empty';
	import { Badge } from '$lib/components/ui/badge';
	import { Separator } from '$lib/components/ui/separator';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { durationText, incidentSeverity, incidentStatus } from '$lib/status.js';
	import { ActivityIcon, CheckIcon, ClockIcon } from '@lucide/svelte';

	const configQuery = $derived(useConfig());
	let incidents = $derived(getIncidents());
	const dateTimeFormat = new Intl.DateTimeFormat(undefined, {
		month: 'short',
		day: 'numeric',
		year: 'numeric',
		hour: '2-digit',
		minute: '2-digit',
		timeZoneName: 'short'
	});
</script>

<div class="mx-auto w-full space-y-6 p-6 sm:max-w-4xl">
	{#if incidents.isSuccess && (incidents.data?.length ?? 0) > 0}
		<div class="flex flex-col gap-4">
			{#each incidents.data ?? [] as incident (incident.id ?? incident.started_at)}
				{@const severity = incidentSeverity(incident.severity)}
				{@const status = incidentStatus(incident.status)}

				<Card.Root class="overflow-hidden">
					<Card.Header>
						<div class="flex items-start justify-between gap-4">
							<div class="flex-1 space-y-2">
								<div class="flex flex-wrap items-center gap-2">
									<Badge variant={severity.variant} class="gap-1">
										<severity.icon class="h-3 w-3" />
										{severity.label}
									</Badge>
									<Badge variant={status.variant} class="gap-1">
										<status.icon class="h-3 w-3" />
										{status.label}
									</Badge>
									{#if (incident.affected_monitors?.length ?? 0) > 0}
										<Badge variant="outline" class="gap-1">
											<ActivityIcon class="h-3 w-3" />
											{incident.affected_monitors!.length} services
										</Badge>
									{/if}
								</div>

								<Card.Title class="text-xl">{incident.title}</Card.Title>
								<Card.Description class="text-base">
									{incident.description}
								</Card.Description>
							</div>

							<div class="flex items-center gap-1 text-sm whitespace-nowrap text-muted-foreground">
								<ClockIcon class="h-4 w-4" />
								{durationText(incident.started_at, incident.resolved_at)}
							</div>
						</div>

						{#if (incident.affected_monitors?.length ?? 0) > 0}
							<div class="flex flex-wrap gap-2 pt-2">
								{#each incident.affected_monitors! as monitor (monitor)}
									<Badge variant="secondary" class="font-mono text-xs">
										{monitor}
									</Badge>
								{/each}
							</div>
						{/if}
					</Card.Header>

					{#if (incident.updates?.length ?? 0) > 0}
						<Card.Content class="pt-0">
							<Separator class="mb-4" />

							<div class="space-y-4">
								<h3 class="text-sm font-semibold">Timeline</h3>
								<div
									class="relative space-y-4 pl-6 before:absolute before:top-2 before:left-2 before:h-[calc(100%-1rem)] before:w-px before:bg-border"
								>
									{#each incident.updates! as update (update.created_at)}
										{@const updateStatus = incidentStatus(update.status)}

										<div class="relative">
											<div
												class="absolute top-1 -left-6 flex h-4 w-4 items-center justify-center rounded-full border-2 border-background bg-background"
											>
												<div class="h-2 w-2 rounded-full bg-primary"></div>
											</div>

											<div class="space-y-1">
												<div class="flex items-center gap-2 text-sm">
													<Badge variant={updateStatus.variant} class="h-5 gap-1 text-xs">
														<updateStatus.icon class="h-2.5 w-2.5" />
														{updateStatus.label}
													</Badge>
													<span class="text-xs text-muted-foreground">
														{dateTimeFormat.format(new Date(update.created_at))}
													</span>
												</div>
												<p class="text-sm">{update.message}</p>
											</div>
										</div>
									{/each}
								</div>
							</div>
						</Card.Content>
					{/if}

					<Card.Footer class="border-t text-xs text-muted-foreground">
						<div class="flex w-full items-center justify-between">
							<span>Started: {dateTimeFormat.format(new Date(incident.started_at))}</span>
							{#if incident.resolved_at}
								<span>Resolved: {dateTimeFormat.format(new Date(incident.resolved_at))}</span>
							{/if}
						</div>
					</Card.Footer>
				</Card.Root>
			{/each}
		</div>
	{:else if configQuery.isPending || incidents.isPending}
		<div class="flex flex-col gap-4">
			<Skeleton class="h-40 w-full rounded-xl" />
			<Skeleton class="h-40 w-full rounded-xl" />
		</div>
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

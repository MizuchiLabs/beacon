<script lang="ts">
	import { useIncidents } from '$lib/api/queries';
	import * as Card from '$lib/components/ui/card';
	import * as Empty from '$lib/components/ui/empty';
	import { Badge } from '$lib/components/ui/badge';
	import { Separator } from '$lib/components/ui/separator';
	import {
		ActivityIcon,
		CheckIcon,
		CircleAlertIcon,
		ClockIcon,
		InfoIcon,
		SearchIcon,
		TriangleAlertIcon,
		WrenchIcon
	} from '@lucide/svelte';

	let incidents = $derived(useIncidents());

	function getSeverityConfig(severity: string) {
		switch (severity) {
			case 'critical':
				return { variant: 'destructive' as const, icon: CircleAlertIcon, label: 'Critical' };
			case 'major':
				return { variant: 'default' as const, icon: TriangleAlertIcon, label: 'Major' };
			case 'minor':
				return { variant: 'secondary' as const, icon: InfoIcon, label: 'Minor' };
			case 'maintenance':
				return { variant: 'outline' as const, icon: WrenchIcon, label: 'Maintenance' };
			default:
				return { variant: 'secondary' as const, icon: InfoIcon, label: severity };
		}
	}

	function getStatusConfig(status: string) {
		switch (status) {
			case 'investigating':
				return { variant: 'default' as const, icon: SearchIcon, label: 'Investigating' };
			case 'identified':
				return { variant: 'default' as const, icon: TriangleAlertIcon, label: 'Identified' };
			case 'monitoring':
				return { variant: 'secondary' as const, icon: ActivityIcon, label: 'Monitoring' };
			case 'resolved':
				return { variant: 'outline' as const, icon: CheckIcon, label: 'Resolved' };
			default:
				return { variant: 'secondary' as const, icon: InfoIcon, label: status };
		}
	}

	function formatDate(dateString: string | null | undefined) {
		if (!dateString) return 'N/A';

		try {
			const date = new Date(dateString);
			if (isNaN(date.getTime())) return 'N/A';

			return new Intl.DateTimeFormat(undefined, {
				month: 'short',
				day: 'numeric',
				year: 'numeric',
				hour: '2-digit',
				minute: '2-digit',
				timeZoneName: 'short'
			}).format(date);
		} catch (e) {
			console.error('Error formatting date:', dateString, e);
			return 'N/A';
		}
	}

	function getDuration(startedAt: string, resolvedAt: string | null | undefined) {
		try {
			const start = new Date(startedAt);
			if (isNaN(start.getTime())) return 'N/A';

			const end = resolvedAt ? new Date(resolvedAt) : new Date();
			if (isNaN(end.getTime())) return 'N/A';

			const diffMs = end.getTime() - start.getTime();

			const hours = Math.floor(diffMs / (1000 * 60 * 60));
			const minutes = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60));

			if (hours > 0) {
				return `${hours}h ${minutes}m`;
			}
			return `${minutes}m`;
		} catch (e) {
			console.error('Error calculating duration:', e);
			return 'N/A';
		}
	}
</script>

<div class="mx-auto w-full space-y-6 p-6 sm:max-w-4xl">
	<div class="space-y-2">
		<h1 class="text-3xl font-bold tracking-tight">Incident History</h1>
		<p class="text-muted-foreground">
			Past incidents and maintenance events affecting our services
		</p>
	</div>

	{#if incidents.isSuccess && incidents.data.length > 0}
		<div class="flex flex-col gap-4">
			{#each incidents.data || [] as incident (incident.id)}
				{@const severity = getSeverityConfig(incident.severity)}
				{@const status = getStatusConfig(incident.status)}

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
									{#if incident.affected_monitors?.length > 0}
										<Badge variant="outline" class="gap-1">
											<ActivityIcon class="h-3 w-3" />
											{incident.affected_monitors.length} services
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
								{getDuration(incident.started_at, incident.resolved_at)}
							</div>
						</div>

						{#if incident.affected_monitors?.length > 0}
							<div class="flex flex-wrap gap-2 pt-2">
								{#each incident.affected_monitors as monitor}
									<Badge variant="secondary" class="font-mono text-xs">
										{monitor}
									</Badge>
								{/each}
							</div>
						{/if}
					</Card.Header>

					{#if incident.updates?.length > 0}
						<Card.Content class="pt-0">
							<Separator class="mb-4" />

							<div class="space-y-4">
								<h3 class="text-sm font-semibold">Timeline</h3>
								<div
									class="relative space-y-4 pl-6 before:absolute before:top-2 before:left-[7px] before:h-[calc(100%-1rem)] before:w-px before:bg-border"
								>
									{#each incident.updates as update}
										{@const updateStatus = getStatusConfig(update.status)}

										<div class="relative">
											<div
												class="absolute top-1 -left-[25px] flex h-4 w-4 items-center justify-center rounded-full border-2 border-background bg-background"
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
														{formatDate(update.created_at)}
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
							<span>Started: {formatDate(incident.started_at)}</span>
							{#if incident.resolved_at}
								<span>Resolved: {formatDate(incident.resolved_at)}</span>
							{/if}
						</div>
					</Card.Footer>
				</Card.Root>
			{/each}
		</div>
	{:else}
		<Empty.Root class="border border-dashed py-12">
			<Empty.Header>
				<Empty.Media variant="icon">
					<CheckIcon class="h-12 w-12 text-primary" />
				</Empty.Media>
				<Empty.Title class="text-2xl">No incidents found</Empty.Title>
				<Empty.Description>
					Everything is working as expected. All systems operational.
				</Empty.Description>
			</Empty.Header>
		</Empty.Root>
	{/if}
</div>

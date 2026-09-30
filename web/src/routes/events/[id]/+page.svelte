<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { getIncident, useConfig } from '$lib/api/queries';
	import { Button } from '$lib/components/ui/button';
	import * as Empty from '$lib/components/ui/empty';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import IncidentCard from '$lib/components/util/IncidentCard.svelte';
	import { ArrowLeftIcon, SearchXIcon } from '@lucide/svelte';

	const incidentQuery = getIncident(() => page.params.id ?? '');
	const configQuery = useConfig();
	const brand = $derived(configQuery.data?.title ?? 'Beacon');
</script>

<svelte:head>
	<title>{incidentQuery.data ? `${incidentQuery.data.title} · ${brand}` : brand}</title>
</svelte:head>

<div class="mx-auto flex w-full flex-col gap-4 p-6 sm:max-w-3xl">
	<div>
		<Button variant="ghost" size="sm" href={resolve('/events')}>
			<ArrowLeftIcon data-icon="inline-start" />
			All incidents
		</Button>
	</div>

	{#if incidentQuery.data}
		<IncidentCard incident={incidentQuery.data} />
	{:else if incidentQuery.isPending}
		<Skeleton class="h-64 w-full" />
	{:else}
		<Empty.Root>
			<Empty.Header>
				<Empty.Media variant="icon">
					<SearchXIcon />
				</Empty.Media>
				<Empty.Title>Incident not found</Empty.Title>
				<Empty.Description>It may have been renamed or removed.</Empty.Description>
			</Empty.Header>
		</Empty.Root>
	{/if}
</div>

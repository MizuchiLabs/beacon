<script lang="ts">
	import { useMonitors } from '$lib/api/queries';
	import { useMonitorStatus } from '$lib/api/queries';
	// import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';
	import StatusCard from '$lib/components/util/StatusCard.svelte';
	import { AlertCircle, CheckCircle2 } from '@lucide/svelte';

	const monitorsQuery = useMonitors();

	// Calculate overall system status
	const allStatuses = $derived(monitorsQuery.data?.map((m) => useMonitorStatus(m.id)) || []);

	const allOperational = $derived(allStatuses.every((status) => status.data?.last_status === 'up'));

	const someDown = $derived(allStatuses.some((status) => status.data?.last_status !== 'up'));
</script>

<div class="container py-8">
	<!-- Overall Status Banner -->
	<div class="mb-8">
		{#if monitorsQuery.isLoading}
			<!-- <Skeleton class="h-24 w-full" /> -->
		{:else if allOperational && monitorsQuery.data && monitorsQuery.data.length > 0}
			<Alert class="border-green-200 bg-green-50 dark:border-green-900 dark:bg-green-950">
				<CheckCircle2 class="h-5 w-5 text-green-600 dark:text-green-400" />
				<AlertDescription class="text-lg font-semibold text-green-900 dark:text-green-100">
					All Systems Operational
				</AlertDescription>
			</Alert>
		{:else if someDown}
			<Alert variant="destructive">
				<AlertCircle class="h-5 w-5" />
				<AlertDescription class="text-lg font-semibold">
					Some Services Are Experiencing Issues
				</AlertDescription>
			</Alert>
		{/if}
	</div>

	<!-- Page Header -->
	<div class="mb-8">
		<h1 class="text-3xl font-bold tracking-tight">Service Status</h1>
		<p class="mt-2 text-muted-foreground">
			Current status and performance of all monitored services
		</p>
	</div>

	<!-- Status Cards -->
	<div class="space-y-4">
		{#if monitorsQuery.isLoading}
			{#each Array(3) as _}
				<!-- <Skeleton class="h-40 w-full" /> -->
			{/each}
		{:else if monitorsQuery.isError}
			<Alert variant="destructive">
				<AlertCircle class="h-4 w-4" />
				<AlertDescription>
					Failed to load monitors: {monitorsQuery.error?.message}
				</AlertDescription>
			</Alert>
		{:else if monitorsQuery.data && monitorsQuery.data.length > 0}
			{#each monitorsQuery.data as monitor (monitor.id)}
				{@const statusQuery = useMonitorStatus(monitor.id)}
				{#if statusQuery.data}
					<StatusCard monitor={statusQuery.data} />
				{:else if statusQuery.isLoading}
					<!-- <Skeleton class="h-40 w-full" /> -->
				{/if}
			{/each}
		{:else}
			<Alert>
				<AlertCircle class="h-4 w-4" />
				<AlertDescription>
					No monitors configured yet. Add your first monitor to get started.
				</AlertDescription>
			</Alert>
		{/if}
	</div>
</div>

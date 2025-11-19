<script lang="ts">
	import { useMonitors } from '$lib/api/queries';
	import StatusCard from '$lib/components/util/StatusCard.svelte';
	import { LoaderCircle } from '@lucide/svelte';

	const monitorsQuery = useMonitors();
	const monitors = $derived(monitorsQuery.data || []);
</script>

<div class="container mx-auto py-8">
	<div class="mb-8 text-center">
		<h1 class="text-3xl font-bold">System Status</h1>
		<p class="mt-2 text-muted-foreground">
			Monitoring {monitors.length}
			{monitors.length === 1 ? 'service' : 'services'}
		</p>
	</div>

	<div class="flex flex-col items-center gap-6">
		{#if monitorsQuery.isLoading}
			<div class="flex items-center gap-2 text-muted-foreground">
				<LoaderCircle class="size-4 animate-spin" />
				Loading monitors...
			</div>
		{:else if monitors.length === 0}
			<p class="text-muted-foreground">No monitors configured yet.</p>
		{:else}
			{#each monitors as monitor (monitor.id)}
				<StatusCard monitorId={monitor.id} />
			{/each}
		{/if}
	</div>
</div>

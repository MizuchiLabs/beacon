<script lang="ts">
	import { useConfig, useMonitorStats } from '$lib/api/queries';
	import * as Select from '$lib/components/ui/select';
	import * as Empty from '$lib/components/ui/empty/index.js';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import StatusCard from '$lib/components/util/StatusCard.svelte';
	import { GlobeIcon } from '@lucide/svelte';

	let configQuery = $derived(useConfig());
	let timeRange = $state('86400'); // 24 hours in seconds

	const timeRanges = [
		{ label: 'Last 24 hours', value: '86400' },
		{ label: 'Last 7 days', value: '604800' },
		{ label: 'Last 14 days', value: '1209600' },
		{ label: 'Last 30 days', value: '2592000' }
	];

	let selectedRange = $derived(timeRanges.find((t) => t.value === timeRange));
	const statsQuery = $derived(useMonitorStats(timeRange));
</script>

<div class="mx-auto w-full space-y-6 p-6 sm:max-w-3xl">
	<!-- Header with Global Time Selector -->
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-3xl font-bold tracking-tight">
				{configQuery.data?.title}
			</h1>
			<p class="text-muted-foreground">
				{configQuery.data?.description}
			</p>
		</div>

		<Select.Root type="single" bind:value={timeRange}>
			<Select.Trigger
				class="w-[150px] rounded-lg bg-card dark:bg-card"
				aria-label="Select time range"
			>
				{selectedRange ? selectedRange.label : 'Last 24 hours'}
			</Select.Trigger>
			<Select.Content class="rounded-xl">
				{#each timeRanges as { label, value }}
					<Select.Item {value} class="rounded-lg">{label}</Select.Item>
				{/each}
			</Select.Content>
		</Select.Root>
	</div>

	<!-- Monitor Grid -->
	{#if statsQuery.isPending}
		<div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
			{#each Array(6) as _}
				<Skeleton class="h-[300px] rounded-lg" />
			{/each}
		</div>
	{:else if statsQuery.isError}
		<Empty.Root class="border border-dashed">
			<Empty.Header>
				<Empty.Media variant="icon">
					<GlobeIcon />
				</Empty.Media>
				<Empty.Title>Failed to load monitors</Empty.Title>
				<Empty.Description>
					{statsQuery.error.message}
				</Empty.Description>
			</Empty.Header>
		</Empty.Root>
	{:else if statsQuery.data?.length === 0}
		<Empty.Root class="border border-dashed">
			<Empty.Header>
				<Empty.Media variant="icon">
					<GlobeIcon />
				</Empty.Media>
				<Empty.Title>No monitors configured</Empty.Title>
				<Empty.Description>Add your first monitor to get started</Empty.Description>
			</Empty.Header>
		</Empty.Root>
	{:else}
		<div class="flex flex-col gap-4">
			{#each statsQuery.data || [] as monitor (monitor.id)}
				<StatusCard {monitor} />
			{/each}
		</div>
	{/if}
</div>

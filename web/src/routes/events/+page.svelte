<script lang="ts">
	import * as Empty from '$lib/components/ui/empty/index.js';
	import { CheckIcon } from '@lucide/svelte';

	let incidents = [];
</script>

<div class="mx-auto w-full space-y-6 p-6 sm:max-w-3xl">
	{#if incidents.length === 0}
		<Empty.Root class="border border-dashed">
			<Empty.Header>
				<Empty.Media variant="icon">
					<CheckIcon class="text-primary" />
				</Empty.Media>
				<Empty.Title>No incidents found</Empty.Title>
				<Empty.Description>Everything is working as expected</Empty.Description>
			</Empty.Header>
		</Empty.Root>
	{:else}
		<div class="flex flex-col gap-4">
			<!-- {#each incidents as incident (incident.id)} -->
			<!-- 	<StatusCard {incident} /> -->
			<!-- {/each} -->
		</div>
	{/if}
</div>

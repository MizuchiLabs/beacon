<script lang="ts">
	import { Bell, Orbit } from '@lucide/svelte';
	import Button from '../ui/button/button.svelte';
	import { pushNotifications } from '$lib/stores/push.svelte';
	import { onMount } from 'svelte';
	import SubscribeModal from './SubscribeModal.svelte';

	let showSubscriptionDialog = $state(false);

	onMount(() => {
		pushNotifications.checkSupport();
	});

	let hasSubscriptions = $derived(pushNotifications.subscribedMonitorIds.length > 0);
	let subscribedCount = $derived(pushNotifications.subscribedMonitorIds.length);
</script>

<SubscribeModal bind:open={showSubscriptionDialog} />

<header class="fixed top-4 right-0 left-0 z-50 flex justify-center px-4">
	<div
		class="flex min-h-12 items-center justify-between rounded-full border-x border-b px-4 py-2 shadow-lg backdrop-blur-md sm:min-w-xl"
	>
		<a href="/" class="flex items-center gap-4">
			<Orbit class="size-6 text-primary" />
			<!-- <Logo class="size-6" /> -->
		</a>

		<nav class="ml-8 flex items-center font-mono">
			<Button variant="ghost" href="/" class="rounded-full" size="sm">Status</Button>
			<Button variant="ghost" href="/events" class="rounded-full" size="sm">Events</Button>
		</nav>

		<div class="flex items-center gap-2">
			<Button
				variant="outline"
				class="rounded-full hover:text-primary"
				size="sm"
				onclick={() => (showSubscriptionDialog = true)}
			>
				<Bell class={hasSubscriptions ? 'fill-current' : ''} />
				Subscribe
				{#if hasSubscriptions}
					<span class="ml-1 text-xs">({subscribedCount})</span>
				{/if}
			</Button>
		</div>
	</div>
</header>

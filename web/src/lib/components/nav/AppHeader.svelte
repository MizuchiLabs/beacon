<script lang="ts">
	import { resolve } from '$app/paths';
	import { useConfig } from '$lib/api/queries';
	import Beacon from '$lib/assets/beacon.svelte';
	import { Button } from '$lib/components/ui/button';
	import { pushNotifications } from '$lib/stores/push.svelte';
	import { Bell, Moon, Sun } from '@lucide/svelte';
	import { mode, toggleMode } from 'mode-watcher';
	import { onMount } from 'svelte';
	import SubscribeModal from './SubscribeModal.svelte';

	let showSubscriptionDialog = $state(false);

	onMount(() => {
		pushNotifications.checkSupport();
	});

	const configQuery = $derived(useConfig());
	const brand = $derived(configQuery.data?.title ?? 'Beacon');

	let hasSubscriptions = $derived(pushNotifications.subscribedMonitorIds.length > 0);
	let subscribedCount = $derived(pushNotifications.subscribedMonitorIds.length);
</script>

<SubscribeModal bind:open={showSubscriptionDialog} />

<header class="pointer-events-none sticky z-50 mx-auto mt-4 mb-6 w-full max-w-4xl px-4 sm:px-6">
	<div class="flex items-center justify-between gap-3">
		<a
			href={resolve('/')}
			class="pointer-events-auto flex h-10 min-w-0 items-center gap-2 rounded-full border bg-background/80 px-3.5 shadow-sm backdrop-blur-md outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50"
		>
			<Beacon class="size-5 shrink-0" />
			<span class="truncate text-sm font-semibold tracking-tight">{brand}</span>
		</a>

		<div
			class="pointer-events-auto flex h-10 shrink-0 items-center gap-1 rounded-full border bg-background/80 px-2 shadow-sm backdrop-blur-md"
		>
			<Button
				variant="ghost"
				size="icon-sm"
				class="relative rounded-full"
				aria-label="Subscribe to notifications"
				onclick={() => (showSubscriptionDialog = true)}
			>
				<Bell class={hasSubscriptions ? 'fill-current' : ''} />
				{#if hasSubscriptions}
					<span
						class="absolute -top-0.5 -right-0.5 flex size-3.5 items-center justify-center rounded-full bg-primary text-[9px] leading-none font-semibold text-primary-foreground"
					>
						{subscribedCount}
					</span>
				{/if}
			</Button>
			<Button
				variant="ghost"
				size="icon-sm"
				class="rounded-full"
				onclick={toggleMode}
				aria-label="Toggle theme"
			>
				{#if mode.current === 'light'}
					<Moon />
				{:else}
					<Sun />
				{/if}
			</Button>
		</div>
	</div>
</header>

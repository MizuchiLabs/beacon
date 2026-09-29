<script lang="ts">
	import { resolve } from '$app/paths';
	import { useConfig } from '$lib/api/queries';
	import Beacon from '$lib/assets/beacon.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { pushNotifications } from '$lib/stores/push.svelte';
	import { cn } from '$lib/utils.js';
	import { Bell, Moon, Sun } from '@lucide/svelte';
	import { mode, toggleMode } from 'mode-watcher';
	import { onMount } from 'svelte';
	import SubscribeModal from './SubscribeModal.svelte';

	let showSubscriptionDialog = $state(false);

	onMount(() => {
		pushNotifications.init();
	});

	const configQuery = useConfig();
	const brand = $derived(configQuery.data?.title ?? 'Beacon');
	const logoURL = $derived(configQuery.data?.logo_url ?? null);

	const subscribedCount = $derived(pushNotifications.subscribed.size);
</script>

<SubscribeModal bind:open={showSubscriptionDialog} />

<header class="pointer-events-none sticky z-50 mx-auto mt-4 mb-6 w-full max-w-4xl px-4 sm:px-6">
	<div class="flex items-center justify-between gap-3">
		<div
			class="pointer-events-auto flex h-10 min-w-0 items-center rounded-full border bg-background/80 px-1.5 shadow-sm backdrop-blur-md"
		>
			<Button href={resolve('/')} variant="ghost" class="min-w-0">
				{#if logoURL}
					<img src={logoURL} alt="" class="size-5 shrink-0 rounded-sm object-contain" />
				{:else}
					<Beacon class="size-5" />
				{/if}
				<span class="truncate font-semibold tracking-tight">{brand}</span>
			</Button>
		</div>

		<div
			class="pointer-events-auto flex h-10 shrink-0 items-center gap-1 rounded-full border bg-background/80 px-1.5 shadow-sm backdrop-blur-md"
		>
			<Button
				variant="ghost"
				size="icon-sm"
				class="relative"
				aria-label="Downtime alerts"
				onclick={() => (showSubscriptionDialog = true)}
			>
				<Bell class={cn(subscribedCount > 0 && 'fill-current')} />
				{#if subscribedCount > 0}
					<Badge class="absolute -top-1.5 -right-1.5">
						{subscribedCount}
					</Badge>
				{/if}
			</Button>
			<Button variant="ghost" size="icon-sm" onclick={toggleMode} aria-label="Toggle theme">
				{#if mode.current === 'light'}
					<Moon />
				{:else}
					<Sun />
				{/if}
			</Button>
		</div>
	</div>
</header>

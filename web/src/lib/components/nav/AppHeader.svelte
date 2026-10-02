<script lang="ts">
	import { resolve } from '$app/paths';
	import { getIncidents, useConfig, useMonitorStats } from '#lib/api/queries.js';
	import Beacon from '#lib/assets/beacon.svelte';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { pageStatus, statusMeta } from '#lib/status.js';
	import { pushNotifications } from '#lib/stores/push.svelte.js';
	import { cn } from '#lib/utils.js';
	import { Bell, Moon, Sun } from '@lucide/svelte';
	import { mode, toggleMode } from 'mode-watcher';
	import { onMount } from 'svelte';
	import SubscribeModal from './SubscribeModal.svelte';

	let showSubscriptionDialog = $state(false);

	onMount(() => {
		pushNotifications.init();
	});

	const configQuery = useConfig();
	const logoURL = $derived(configQuery.data?.logo_url ?? null);

	const statsQuery = useMonitorStats();
	const incidentsQuery = getIncidents();
	const monitors = $derived(statsQuery.data ?? []);
	const meta = $derived(statusMeta[pageStatus(monitors, incidentsQuery.data ?? []).status]);

	// The favicon doubles as a passive status light for pinned tabs.
	const faviconHref = $derived.by(() => {
		if (monitors.length === 0) return '';
		const canvas = document.createElement('canvas');
		canvas.width = 64;
		canvas.height = 64;
		const ctx = canvas.getContext('2d');
		if (!ctx) return '';
		// Same shapes as favicon.svg, only the dot color changes.
		ctx.strokeStyle = '#8a5cf0';
		ctx.lineWidth = 12;
		ctx.beginPath();
		ctx.arc(32, 32, 26, 0, Math.PI * 2);
		ctx.stroke();
		ctx.fillStyle = getComputedStyle(document.documentElement).getPropertyValue(meta.token).trim();
		ctx.beginPath();
		ctx.arc(32, 32, 12, 0, Math.PI * 2);
		ctx.fill();
		return canvas.toDataURL('image/png');
	});

	const subscribedCount = $derived(pushNotifications.subscribed.size);
</script>

<!-- The only icon link on the page. With several, browsers pick by type and the live one loses. -->
<svelte:head>
	<link rel="icon" href={faviconHref || '/favicon.svg'} />
</svelte:head>

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
					<Beacon class={cn('size-5 transition-colors duration-500', meta.text)} />
				{/if}
				{#if configQuery.data?.title}
					<span class="truncate font-semibold tracking-tight">{configQuery.data.title}</span>
				{/if}
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

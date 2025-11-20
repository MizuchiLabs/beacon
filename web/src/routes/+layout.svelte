<script lang="ts">
	import './layout.css';
	import { Toaster } from '$lib/components/ui/sonner/index.js';
	import { QueryClientProvider } from '@tanstack/svelte-query';
	import { queryClient } from '$lib/api/client';
	import AppHeader from '$lib/components/nav/AppHeader.svelte';
	import AppFooter from '$lib/components/nav/AppFooter.svelte';
	import GridPattern from '$lib/components/util/GridPattern.svelte';
	import { cn } from '$lib/utils';
	import { ModeWatcher } from 'mode-watcher';
	import { onMount } from 'svelte';
	import { dev } from '$app/environment';

	let { children } = $props();

	onMount(() => {
		if ('serviceWorker' in navigator) {
			navigator.serviceWorker
				.register('/service-worker.js', {
					type: dev ? 'module' : 'classic'
				})
				.then(
					(registration) => {
						console.log('Service Worker registered:', registration);
					},
					(error) => {
						console.error('Service Worker registration failed:', error);
					}
				);
		}
	});
</script>

<ModeWatcher />
<Toaster />

<QueryClientProvider client={queryClient}>
	<div class="flex min-h-screen flex-col">
		<GridPattern
			width={40}
			height={40}
			x={-1}
			y={-1}
			class={cn('mask-[linear-gradient(to_bottom_left,white,transparent,transparent)]')}
		/>

		<AppHeader />
		<main class="mt-20 mb-12 flex-1">
			{@render children?.()}
		</main>
		<AppFooter />
	</div>
</QueryClientProvider>

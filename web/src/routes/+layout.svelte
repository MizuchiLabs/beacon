<script lang="ts">
	import { dev } from '$app/environment';
	import { queryClient } from '$lib/api/client';
	import AppFooter from '$lib/components/nav/AppFooter.svelte';
	import { Toaster } from '$lib/components/ui/sonner/index.js';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import { ModeWatcher } from 'mode-watcher';
	import { onMount } from 'svelte';
	import './layout.css';
	import AppHeader from '$lib/components/nav/AppHeader.svelte';
	import { QueryClientProvider } from '@tanstack/svelte-query';
	import NoiseTexture from '$lib/components/magic/noise-texture/noise-texture.svelte';

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
<NoiseTexture noiseOpacity={0.1} />

<QueryClientProvider client={queryClient}>
	<Tooltip.Provider>
		<div class="flex min-h-screen flex-col">
			<AppHeader />
			<main class="z-10 mb-12 flex-1">
				{@render children?.()}
			</main>
			<AppFooter />
		</div>
	</Tooltip.Provider>
</QueryClientProvider>

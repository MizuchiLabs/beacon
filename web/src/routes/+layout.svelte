<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { Toaster } from '$lib/components/ui/sonner/index.js';
	import { QueryClientProvider } from '@tanstack/svelte-query';
	import { queryClient } from '$lib/api/client';
	import AppHeader from '$lib/components/nav/AppHeader.svelte';
	import AppFooter from '$lib/components/nav/AppFooter.svelte';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="flex min-h-screen flex-col">
	<AppHeader />

	<main class="flex-1">
		<Toaster />

		<QueryClientProvider client={queryClient}>
			{@render children?.()}
		</QueryClientProvider>
	</main>

	<AppFooter />
</div>

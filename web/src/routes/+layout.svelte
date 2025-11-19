<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { Toaster } from '$lib/components/ui/sonner/index.js';
	import { QueryClientProvider } from '@tanstack/svelte-query';
	import { queryClient } from '$lib/api/client';
	import AppHeader from '$lib/components/nav/AppHeader.svelte';
	import AppFooter from '$lib/components/nav/AppFooter.svelte';
	import { theme } from '$lib/stores/theme';
	import { onMount } from 'svelte';

	let { children } = $props();

	onMount(() => {
		// Apply initial theme
		const unsubscribe = theme.subscribe((currentTheme) => {
			document.documentElement.classList.toggle('dark', currentTheme === 'dark');
		});
		return unsubscribe;
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="flex min-h-screen flex-col">
	<AppHeader />

	<main class="my-12 flex-1">
		<Toaster />

		<QueryClientProvider client={queryClient}>
			{@render children?.()}
		</QueryClientProvider>
	</main>

	<AppFooter />
</div>

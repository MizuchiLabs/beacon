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

	let { children } = $props();
</script>

<ModeWatcher />
<Toaster />

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
		<QueryClientProvider client={queryClient}>
			{@render children?.()}
		</QueryClientProvider>
	</main>

	<AppFooter />
</div>

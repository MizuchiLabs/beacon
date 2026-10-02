<script lang="ts">
	import { queryClient } from '#lib/api/client.js';
	import AppFooter from '#lib/components/nav/AppFooter.svelte';
	import { Toaster } from '#lib/components/ui/sonner/index.js';
	import * as Tooltip from '#lib/components/ui/tooltip/index.js';
	import { ModeWatcher } from 'mode-watcher';
	import './layout.css';
	import AppHeader from '#lib/components/nav/AppHeader.svelte';
	import { QueryClientProvider } from '@tanstack/svelte-query';
	import NoiseTexture from '#lib/components/magic/noise-texture/noise-texture.svelte';
	import StatusGlow from '#lib/components/util/StatusGlow.svelte';

	let { children } = $props();
</script>

<!-- Same values as --background in layout.css. -->
<ModeWatcher themeColors={{ dark: '#0f0f1a', light: '#f5f5ff' }} />
<Toaster />

<QueryClientProvider client={queryClient}>
	<StatusGlow />
	<NoiseTexture noiseOpacity={0.1} class="fixed" />
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

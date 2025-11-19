<script lang="ts">
	import { Telescope } from '@lucide/svelte';
</script>

<header class="fixed top-0 right-0 left-0 z-50 flex justify-center px-4">
	<div
		class="flex min-h-12 items-center gap-1 rounded-b-xl border border-border/40 bg-background/80 px-5 py-2 shadow-lg backdrop-blur-md"
	>
		<a href="/" class="flex items-center gap-4">
			<Telescope class="size-6 text-primary" />
			<!-- <Logo class="size-6" /> -->
			<span class="font-mono text-sm font-semibold">Beacon</span>
		</a>

		<nav class="ml-4 flex items-center">
			<a
				href="/"
				class="relative rounded-full px-3 py-1.5 text-sm font-medium transition-all duration-200 hover:bg-card/80"
			>
				Status
			</a>
			<a
				href="/events"
				class="relative rounded-full px-3 py-1.5 text-sm font-medium transition-all duration-200 hover:bg-card/80"
			>
				Events
			</a>
		</nav>
	</div>
</header>

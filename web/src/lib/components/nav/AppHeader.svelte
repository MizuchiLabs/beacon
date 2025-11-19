<script lang="ts">
	import { Telescope } from '@lucide/svelte';
</script>

<header
	class="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-backdrop-filter:bg-background/60"
>
	<div class="container flex h-16 items-center justify-between">
		<div class="flex items-center gap-2">
			<Telescope class="h-6 w-6 text-primary" />
			<h1 class="text-xl font-bold">Beacon</h1>
		</div>
		<nav class="flex items-center gap-6">
			<a href="/" class="text-sm font-medium transition-colors hover:text-primary">Status</a>
			<a href="/events" class="text-sm font-medium transition-colors hover:text-primary">
				Events
			</a>
		</nav>
	</div>
</header>

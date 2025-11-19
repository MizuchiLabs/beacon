<script lang="ts">
	import { Bell, Orbit } from '@lucide/svelte';
	import Button from '../ui/button/button.svelte';
</script>

<header class="fixed top-4 right-0 left-0 z-50 flex justify-center px-4">
	<div
		class="flex min-h-12 items-center justify-between rounded-full border-x border-b px-4 py-2 shadow-lg backdrop-blur-md sm:min-w-xl"
	>
		<a href="/" class="flex items-center gap-4">
			<Orbit class="size-6 text-primary" />
			<!-- <Logo class="size-6" /> -->
		</a>

		<nav class="ml-8 flex items-center font-mono">
			<Button variant="ghost" href="/" class="rounded-full" size="sm">Status</Button>
			<Button variant="ghost" href="/events" class="rounded-full" size="sm">Events</Button>
		</nav>

		<div class="flex items-center gap-2">
			<Button variant="outline" href="/" class="rounded-full hover:text-primary" size="sm">
				<Bell />
				Subscribe
			</Button>
		</div>
	</div>
</header>

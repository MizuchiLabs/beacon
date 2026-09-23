<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { Button } from '$lib/components/ui/button';
	import { Radio } from '@lucide/svelte';

	const isNotFound = $derived(page.status === 404);
	const title = $derived(isNotFound ? 'No signal from this page' : `Error ${page.status}`);
	const description = $derived(
		isNotFound
			? "It doesn't exist, it moved, or it was never monitored in the first place."
			: (page.error?.message ?? 'Something went wrong.')
	);
</script>

<svelte:head>
	<title>{isNotFound ? '404' : `Error ${page.status}`}</title>
</svelte:head>

<div
	class="flex min-h-[calc(100dvh-8rem)] flex-col items-center justify-center gap-6 p-6 text-center"
>
	<div class="relative flex size-16 items-center justify-center rounded-full border bg-card">
		<span
			class="absolute inset-0 animate-ping rounded-full bg-muted-foreground/15 animation-duration-[3s] motion-reduce:hidden"
		>
		</span>
		<Radio class="size-7 opacity-60" />
	</div>

	<div class="space-y-1.5">
		<h1 class="text-2xl font-semibold tracking-tight">{title}</h1>
		<p class="text-sm text-muted-foreground">{description}</p>
	</div>

	<Button href={resolve('/')} size="sm">Back to status</Button>
</div>

<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
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

<Empty.Root class="min-h-[calc(100dvh-8rem)]">
	<Empty.Header>
		<Empty.Media variant="icon">
			<Radio />
		</Empty.Media>
		<Empty.Title>{title}</Empty.Title>
		<Empty.Description>{description}</Empty.Description>
	</Empty.Header>
	<Empty.Content>
		<Button href={resolve('/')} size="sm">Back to status</Button>
	</Empty.Content>
</Empty.Root>

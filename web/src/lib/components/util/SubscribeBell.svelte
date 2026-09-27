<script lang="ts">
	import * as Tooltip from '$lib/components/ui/tooltip';
	import { pushNotifications } from '$lib/stores/push.svelte';
	import { BellIcon, BellRingIcon, LoaderCircleIcon } from '@lucide/svelte';
	import { cn } from '$lib/utils.js';

	interface Props {
		monitorId: number;
		class?: string;
	}
	let { monitorId, class: className }: Props = $props();

	const subscribed = $derived(pushNotifications.subscribed.has(monitorId));
</script>

{#if pushNotifications.supported}
	<Tooltip.Root>
		<Tooltip.Trigger>
			{#snippet child({ props })}
				<button
					type="button"
					{...props}
					class={cn(
						'rounded-md p-1 transition-colors hover:bg-muted focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-none',
						subscribed ? 'text-primary' : 'text-muted-foreground',
						className
					)}
					onclick={() => pushNotifications.toggle(monitorId)}
					disabled={pushNotifications.loading}
					aria-label={subscribed ? 'Unsubscribe from alerts' : 'Subscribe to alerts'}
					aria-pressed={subscribed}
				>
					{#if pushNotifications.loading}
						<LoaderCircleIcon class="size-4 animate-spin" />
					{:else if subscribed}
						<BellRingIcon class="size-4" />
					{:else}
						<BellIcon class="size-4" />
					{/if}
				</button>
			{/snippet}
		</Tooltip.Trigger>
		<Tooltip.Content>
			{subscribed ? 'You get alerts for this monitor' : 'Notify me about downtime'}
		</Tooltip.Content>
	</Tooltip.Root>
{/if}

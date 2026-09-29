<script lang="ts">
	import { Spinner } from '$lib/components/ui/spinner';
	import { Toggle } from '$lib/components/ui/toggle';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import { pushNotifications } from '$lib/stores/push.svelte';
	import { BellIcon, BellRingIcon } from '@lucide/svelte';

	interface Props {
		monitorId: number;
	}
	let { monitorId }: Props = $props();

	const subscribed = $derived(pushNotifications.subscribed.has(monitorId));
</script>

{#if pushNotifications.supported}
	<Tooltip.Root>
		<Tooltip.Trigger>
			{#snippet child({ props })}
				<Toggle
					{...props}
					size="sm"
					bind:pressed={() => subscribed, () => pushNotifications.toggle(monitorId)}
					disabled={pushNotifications.loading}
					aria-label={subscribed ? 'Unsubscribe from alerts' : 'Subscribe to alerts'}
				>
					{#if pushNotifications.loading}
						<Spinner />
					{:else if subscribed}
						<BellRingIcon />
					{:else}
						<BellIcon />
					{/if}
				</Toggle>
			{/snippet}
		</Tooltip.Trigger>
		<Tooltip.Content>
			{subscribed ? 'You get alerts for this monitor' : 'Notify me about downtime'}
		</Tooltip.Content>
	</Tooltip.Root>
{/if}

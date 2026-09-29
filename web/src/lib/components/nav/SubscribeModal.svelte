<script lang="ts">
	import { useMonitorStats } from '$lib/api/queries';
	import * as Alert from '$lib/components/ui/alert';
	import { Button } from '$lib/components/ui/button';
	import { Checkbox } from '$lib/components/ui/checkbox';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Field from '$lib/components/ui/field';
	import { ScrollArea } from '$lib/components/ui/scroll-area';
	import { Spinner } from '$lib/components/ui/spinner';
	import { pushNotifications } from '$lib/stores/push.svelte';
	import { targetOf } from '$lib/status.js';
	import { CircleAlertIcon } from '@lucide/svelte';

	let { open = $bindable(false) } = $props();

	const monitorsQuery = useMonitorStats();
	const monitors = $derived(monitorsQuery.data ?? []);
	const unsubscribed = $derived(
		monitors.filter((m) => !pushNotifications.subscribed.has(m.id)).map((m) => m.id)
	);
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="sm:max-w-lg">
		<Dialog.Header>
			<Dialog.Title>Downtime alerts</Dialog.Title>
			<Dialog.Description>
				Pick the monitors this browser should notify you about.
			</Dialog.Description>
		</Dialog.Header>

		{#if !pushNotifications.supported}
			<Alert.Root>
				<CircleAlertIcon />
				<Alert.Title>Not supported here</Alert.Title>
				<Alert.Description>
					This browser can't receive push notifications. On iOS, add the page to your home screen
					first.
				</Alert.Description>
			</Alert.Root>
		{:else if pushNotifications.permission === 'denied'}
			<Alert.Root variant="destructive">
				<CircleAlertIcon />
				<Alert.Title>Notifications are blocked</Alert.Title>
				<Alert.Description>
					Allow notifications for this site in your browser settings.
				</Alert.Description>
			</Alert.Root>
		{:else}
			{#if pushNotifications.error}
				<Alert.Root variant="destructive">
					<CircleAlertIcon />
					<Alert.Description>{pushNotifications.error}</Alert.Description>
				</Alert.Root>
			{/if}

			<ScrollArea class="-mx-1 *:data-[slot=scroll-area-viewport]:max-h-96">
				<div class="px-1">
					<Field.Set>
						<Field.Legend class="sr-only">Monitors</Field.Legend>
						<Field.Group data-slot="checkbox-group">
							{#each monitors as monitor (monitor.id)}
								<Field.Label for="subscribe-{monitor.id}">
									<Field.Field orientation="horizontal">
										<Checkbox
											id="subscribe-{monitor.id}"
											checked={pushNotifications.subscribed.has(monitor.id)}
											disabled={pushNotifications.loading}
											onCheckedChange={() => pushNotifications.toggle(monitor.id)}
										/>
										<Field.Content class="min-w-0">
											<Field.Title class="w-full min-w-0"
												><span class="truncate">{monitor.name}</span></Field.Title
											>
											<Field.Description
												><span class="block truncate">{targetOf(monitor)}</span></Field.Description
											>
										</Field.Content>
									</Field.Field>
								</Field.Label>
							{/each}
						</Field.Group>
					</Field.Set>
				</div>
			</ScrollArea>

			<Dialog.Footer class="sm:justify-between">
				<Button
					variant="secondary"
					size="sm"
					onclick={() => pushNotifications.unsubscribe(...pushNotifications.subscribed)}
					disabled={pushNotifications.loading || pushNotifications.subscribed.size === 0}
				>
					Unsubscribe all
				</Button>
				<Button
					size="sm"
					onclick={() => pushNotifications.subscribe(...unsubscribed)}
					disabled={pushNotifications.loading || unsubscribed.length === 0}
				>
					{#if pushNotifications.loading}
						<Spinner data-icon="inline-start" />
					{/if}
					Subscribe to all
				</Button>
			</Dialog.Footer>
		{/if}
	</Dialog.Content>
</Dialog.Root>

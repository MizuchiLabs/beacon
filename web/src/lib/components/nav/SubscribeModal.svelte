<script lang="ts">
	import { useMonitorStats } from '$lib/api/queries';
	import * as Alert from '$lib/components/ui/alert';
	import { Button } from '$lib/components/ui/button';
	import { Checkbox } from '$lib/components/ui/checkbox';
	import * as Dialog from '$lib/components/ui/dialog';
	import { pushNotifications } from '$lib/stores/push.svelte';
	import { targetOf } from '$lib/status.js';
	import { BellIcon, BellOffIcon, CircleAlertIcon, LoaderCircleIcon } from '@lucide/svelte';

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
				<Alert.Description
					>Allow notifications for this site in your browser settings.</Alert.Description
				>
			</Alert.Root>
		{:else}
			{#if pushNotifications.error}
				<Alert.Root variant="destructive">
					<CircleAlertIcon />
					<Alert.Description>{pushNotifications.error}</Alert.Description>
				</Alert.Root>
			{/if}

			<div class="-mx-1 flex max-h-96 flex-col gap-1.5 overflow-y-auto px-1">
				{#each monitors as monitor (monitor.id)}
					{@const isSubscribed = pushNotifications.subscribed.has(monitor.id)}
					<label
						class="flex cursor-pointer items-center gap-3 rounded-xl border p-3 transition-colors hover:bg-accent has-data-[state=checked]:border-primary/40 has-data-[state=checked]:bg-accent/60"
					>
						<Checkbox
							checked={isSubscribed}
							disabled={pushNotifications.loading}
							onCheckedChange={() => pushNotifications.toggle(monitor.id)}
						/>
						<div class="min-w-0 flex-1">
							<div class="truncate text-sm font-medium">{monitor.name}</div>
							<div class="truncate text-xs text-muted-foreground">{targetOf(monitor)}</div>
						</div>
						{#if isSubscribed}
							<BellIcon class="size-4 text-primary" />
						{:else}
							<BellOffIcon class="size-4 text-muted-foreground" />
						{/if}
					</label>
				{/each}
			</div>

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
						<LoaderCircleIcon class="animate-spin" />
					{/if}
					Subscribe to all
				</Button>
			</Dialog.Footer>
		{/if}
	</Dialog.Content>
</Dialog.Root>

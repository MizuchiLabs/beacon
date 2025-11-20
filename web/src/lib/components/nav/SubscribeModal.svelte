<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import Button from '$lib/components/ui/button/button.svelte';
	import { Checkbox } from '$lib/components/ui/checkbox';
	import { Bell, BellOff, LoaderCircle } from '@lucide/svelte';
	import { pushNotifications } from '$lib/stores/push.svelte';
	import { useMonitorStats } from '$lib/api/queries';

	let { open = $bindable(false) } = $props();

	const monitorsQuery = useMonitorStats();

	let monitors = $derived(monitorsQuery.data || []);
	let loading = $derived(pushNotifications.loading);
	let error = $derived(pushNotifications.error);
	let subscribedMonitorIDs = $derived(pushNotifications.subscribedMonitorIds);
	let hasPermission = $derived(pushNotifications.hasPermission);

	async function handleToggleSubscription(monitorId: number, subscribe: boolean) {
		if (subscribe) {
			await pushNotifications.subscribeToMonitor(monitorId);
		} else {
			await pushNotifications.unsubscribeFromMonitor(monitorId);
		}
	}

	async function handleSubscribeAll() {
		for (const monitor of monitors) {
			if (!subscribedMonitorIDs.includes(monitor.id)) {
				await pushNotifications.subscribeToMonitor(monitor.id);
			}
		}
	}

	async function handleUnsubscribeAll() {
		for (const monitorId of subscribedMonitorIDs) {
			await pushNotifications.unsubscribeFromMonitor(monitorId);
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="sm:max-w-[500px]">
		<Dialog.Header>
			<Dialog.Title>Subscribe to Notifications</Dialog.Title>
			<Dialog.Description>Choose which monitors to receive notifications for</Dialog.Description>
		</Dialog.Header>

		{#if !hasPermission}
			<div
				class="rounded-lg border border-yellow-200 bg-yellow-50 p-4 text-sm text-yellow-800 dark:border-yellow-800 dark:bg-yellow-950 dark:text-yellow-200"
			>
				<p class="mb-2 font-medium">Notification Permission Required</p>
				<p>You need to grant notification permission to receive alerts when monitors go down.</p>
				<Button
					variant="outline"
					size="sm"
					class="mt-3"
					onclick={() => pushNotifications.requestPermission()}
				>
					Grant Permission
				</Button>
			</div>
		{:else}
			<div class="space-y-4">
				{#if error}
					<div
						class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800 dark:border-red-800 dark:bg-red-950 dark:text-red-200"
					>
						{error}
					</div>
				{/if}

				<div class="max-h-[400px] space-y-2 overflow-y-auto">
					{#each monitors as monitor (monitor.id)}
						{@const isSubscribed = subscribedMonitorIDs.includes(monitor.id)}
						<label
							class="flex cursor-pointer items-center gap-3 rounded-lg border p-3 transition-colors hover:bg-accent"
							class:bg-accent={isSubscribed}
						>
							<Checkbox
								checked={isSubscribed}
								disabled={loading}
								onCheckedChange={(checked) =>
									handleToggleSubscription(monitor.id, checked === true)}
							/>
							<div class="flex-1">
								<div class="font-medium">{monitor.name}</div>
								<div class="text-sm text-muted-foreground">{monitor.url}</div>
							</div>
							{#if isSubscribed}
								<Bell class="size-4 text-primary" />
							{:else}
								<BellOff class="size-4 text-muted-foreground" />
							{/if}
						</label>
					{/each}
				</div>

				{#if loading}
					<div class="flex items-center justify-center gap-2 py-2 text-sm text-muted-foreground">
						<LoaderCircle class="size-4 animate-spin" />
						Processing...
					</div>
				{/if}
			</div>

			<div class="grid grid-cols-2 gap-2 text-sm text-muted-foreground">
				<Button variant="outline" size="sm" onclick={handleSubscribeAll} disabled={loading}>
					Subscribe All
				</Button>
				<Button
					variant="secondary"
					size="sm"
					onclick={handleUnsubscribeAll}
					disabled={loading || subscribedMonitorIDs.length === 0}
				>
					Unsubscribe All
				</Button>
			</div>
		{/if}
	</Dialog.Content>
</Dialog.Root>

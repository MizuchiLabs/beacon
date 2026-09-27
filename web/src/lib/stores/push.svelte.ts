import {
	getVapidPublicKey,
	listSubscriptions,
	subscribeToMonitor,
	unsubscribeFromMonitor
} from '$lib/api/generated/sdk.gen';
import { SvelteSet } from 'svelte/reactivity';

class PushNotifications {
	supported = $state(false);
	permission = $state<NotificationPermission>('default');
	loading = $state(false);
	error = $state<string | null>(null);
	subscribed = new SvelteSet<number>();

	get hasPermission() {
		return this.permission === 'granted';
	}

	// The server knows which monitors this browser gets alerts for, so a
	// subscription it dropped does not linger in the UI.
	async init() {
		this.supported = 'serviceWorker' in navigator && 'PushManager' in window;
		if (!this.supported) return;
		this.permission = Notification.permission;

		const subscription = await this.existing();
		if (!subscription) return;
		const { data } = await listSubscriptions({ body: { endpoint: subscription.endpoint } });
		this.subscribed.clear();
		for (const id of data?.monitor_ids ?? []) this.subscribed.add(id);
	}

	async requestPermission() {
		this.permission = await Notification.requestPermission();
		return this.permission;
	}

	toggle(monitorId: number) {
		return this.subscribed.has(monitorId) ? this.unsubscribe(monitorId) : this.subscribe(monitorId);
	}

	subscribe(...monitorIds: number[]) {
		return this.run('Could not subscribe', async () => {
			const subscription = await this.ensure();
			const { p256dh = '', auth = '' } = subscription.toJSON().keys ?? {};
			for (const id of monitorIds) {
				const { error } = await subscribeToMonitor({
					path: { id },
					body: { endpoint: subscription.endpoint, keys: { p256dh, auth } }
				});
				if (error) throw new Error('the server rejected the subscription');
				this.subscribed.add(id);
			}
		});
	}

	unsubscribe(...monitorIds: number[]) {
		return this.run('Could not unsubscribe', async () => {
			const subscription = await this.existing();
			for (const id of monitorIds) {
				if (subscription) {
					await unsubscribeFromMonitor({
						path: { id },
						body: { endpoint: subscription.endpoint }
					});
				}
				this.subscribed.delete(id);
			}
			// One browser subscription serves every monitor, drop it once unused.
			if (this.subscribed.size === 0) await subscription?.unsubscribe();
		});
	}

	private async run(failure: string, work: () => Promise<void>) {
		this.loading = true;
		this.error = null;
		try {
			await work();
		} catch (err) {
			this.error = `${failure}: ${err instanceof Error ? err.message : 'push service error'}`;
		} finally {
			this.loading = false;
		}
	}

	private async existing() {
		const registration = await navigator.serviceWorker.ready;
		return registration.pushManager.getSubscription();
	}

	private async ensure() {
		if ((await this.requestPermission()) !== 'granted') {
			throw new Error('notification permission denied');
		}
		const current = await this.existing();
		if (current) return current;

		const { data, error } = await getVapidPublicKey();
		if (error || !data) throw new Error('could not load the server key');
		const registration = await navigator.serviceWorker.ready;
		return registration.pushManager.subscribe({
			userVisibleOnly: true,
			applicationServerKey: decodeBase64Url(data.publicKey)
		});
	}
}

function decodeBase64Url(value: string): Uint8Array<ArrayBuffer> {
	const base64 = value.trim().replace(/-/g, '+').replace(/_/g, '/');
	const padded = base64 + '='.repeat((4 - (base64.length % 4)) % 4);
	return Uint8Array.from(atob(padded), (c) => c.charCodeAt(0));
}

export const pushNotifications = new PushNotifications();

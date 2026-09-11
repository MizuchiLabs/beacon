import {
	getVapidPublicKey,
	subscribeToMonitor,
	unsubscribeFromMonitor
} from '$lib/api/generated/sdk.gen';
import { SvelteSet } from 'svelte/reactivity';

interface PushSubscriptionState {
	supported: boolean;
	permission: NotificationPermission;
	subscriptions: SvelteSet<number>;
	loading: boolean;
	error: string | null;
}

class PushNotificationStore {
	private state = $state<PushSubscriptionState>({
		supported: false,
		permission: 'default',
		subscriptions: new SvelteSet(),
		loading: false,
		error: null
	});

	get supported() {
		return this.state.supported;
	}

	get permission() {
		return this.state.permission;
	}

	get loading() {
		return this.state.loading;
	}

	get error() {
		return this.state.error;
	}

	get hasPermission() {
		return this.state.permission === 'granted';
	}

	get subscribedMonitorIds() {
		return [...this.state.subscriptions];
	}

	checkSupport() {
		const supported = 'serviceWorker' in navigator && 'PushManager' in window;
		this.state.supported = supported;
		if (supported) {
			this.state.permission = Notification.permission;
			this.loadSubscriptions();
		}
		return supported;
	}

	async requestPermission(): Promise<NotificationPermission> {
		if (!('Notification' in window)) {
			return 'denied';
		}

		const permission = await Notification.requestPermission();
		this.state.permission = permission;
		return permission;
	}

	private async getVAPIDPublicKey(): Promise<string> {
		const { data, error } = await getVapidPublicKey();
		if (error || !data) {
			throw new Error('Failed to fetch VAPID public key');
		}
		return data.publicKey;
	}

	async subscribeToMonitor(monitorID: number): Promise<boolean> {
		this.state.loading = true;
		this.state.error = null;

		try {
			const registration = await navigator.serviceWorker.ready;
			if (!registration) {
				throw new Error('Service worker not ready');
			}

			// Request permission if not granted
			const permission = await this.requestPermission();
			if (permission !== 'granted') {
				throw new Error('Notification permission denied');
			}

			// One browser subscription serves every monitor
			let subscription = await registration.pushManager.getSubscription();

			if (!subscription) {
				const vapidPublicKey = await this.getVAPIDPublicKey();

				subscription = await registration.pushManager.subscribe({
					userVisibleOnly: true,
					applicationServerKey: this.urlBase64ToUint8Array(vapidPublicKey)
				});
			}

			const { error } = await subscribeToMonitor({
				path: { id: monitorID },
				body: {
					endpoint: subscription.endpoint,
					keys: {
						p256dh: this.arrayBufferToBase64(subscription.getKey('p256dh')),
						auth: this.arrayBufferToBase64(subscription.getKey('auth'))
					}
				}
			});

			if (error) {
				throw new Error('Failed to save subscription on server');
			}

			this.state.subscriptions.add(monitorID);
			this.state.loading = false;
			this.saveSubscriptions();
			return true;
		} catch (error) {
			this.state.loading = false;
			this.state.error =
				error instanceof Error
					? `Registration failed: ${error.message}`
					: 'Registration failed - push service error';
			return false;
		}
	}

	async unsubscribeFromMonitor(monitorID: number): Promise<boolean> {
		this.state.loading = true;
		this.state.error = null;

		try {
			const registration = await navigator.serviceWorker.ready;
			const subscription = await registration.pushManager.getSubscription();

			if (subscription) {
				await unsubscribeFromMonitor({
					path: { id: monitorID },
					body: { endpoint: subscription.endpoint }
				});
			}

			this.state.subscriptions.delete(monitorID);
			this.state.loading = false;
			this.saveSubscriptions();

			// Drop the browser subscription once no monitor uses it
			if (this.state.subscriptions.size === 0 && subscription) {
				await subscription.unsubscribe();
			}

			return true;
		} catch (error) {
			console.error('Failed to unsubscribe:', error);
			this.state.loading = false;
			this.state.error = 'Failed to unsubscribe';
			return false;
		}
	}

	private saveSubscriptions() {
		localStorage.setItem('monitor-subscriptions', JSON.stringify([...this.state.subscriptions]));
	}

	private loadSubscriptions() {
		try {
			const stored = localStorage.getItem('monitor-subscriptions');
			if (stored) {
				const monitorIds: number[] = JSON.parse(stored);
				monitorIds.forEach((id) => this.state.subscriptions.add(id));
			}
		} catch (error) {
			console.error('Failed to load subscriptions:', error);
		}
	}

	private urlBase64ToUint8Array(base64String: string): BufferSource {
		// Remove any whitespace
		base64String = base64String.trim();

		// Add padding if needed
		const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
		const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');

		try {
			const rawData = window.atob(base64);
			const outputArray = new Uint8Array(rawData.length);
			for (let i = 0; i < rawData.length; i++) {
				outputArray[i] = rawData.charCodeAt(i);
			}
			return outputArray;
		} catch (error) {
			console.error('Failed to decode VAPID key:', error, 'Key:', base64String);
			throw new Error('Invalid VAPID public key format');
		}
	}

	private arrayBufferToBase64(buffer: ArrayBuffer | null): string {
		if (!buffer) return '';
		const bytes = new Uint8Array(buffer);
		let binary = '';
		for (let i = 0; i < bytes.byteLength; i++) {
			binary += String.fromCharCode(bytes[i]);
		}
		return window.btoa(binary);
	}
}

export const pushNotifications = new PushNotificationStore();

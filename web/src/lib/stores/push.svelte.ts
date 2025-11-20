import { BackendURL } from '$lib/api/queries';
import { SvelteMap } from 'svelte/reactivity';

interface PushSubscriptionState {
	supported: boolean;
	permission: NotificationPermission;
	subscriptions: SvelteMap<number, PushSubscription>;
	loading: boolean;
	error: string | null;
}

class PushNotificationStore {
	private state = $state<PushSubscriptionState>({
		supported: false,
		permission: 'default',
		subscriptions: new SvelteMap(),
		loading: false,
		error: null
	});

	get supported() {
		return this.state.supported;
	}

	get permission() {
		return this.state.permission;
	}

	get subscriptions() {
		return this.state.subscriptions;
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
		return Array.from(this.state.subscriptions.keys());
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
		const response = await fetch(`${BackendURL}/vapid-public-key`);
		const data = await response.json();
		return data.publicKey;
	}

	async subscribeToMonitor(monitorID: number): Promise<boolean> {
		this.state.loading = true;
		this.state.error = null;

		try {
			// Get service worker registration
			const registration = await navigator.serviceWorker.ready;
			if (!registration) {
				throw new Error('Service worker not ready');
			}

			// Request permission if not granted
			const permission = await this.requestPermission();
			if (permission !== 'granted') {
				throw new Error('Notification permission denied');
			}

			// Check for existing subscription
			let subscription = await registration.pushManager.getSubscription();

			if (!subscription) {
				// Get VAPID public key
				const vapidPublicKey = await this.getVAPIDPublicKey();

				// Subscribe to push notifications
				subscription = await registration.pushManager.subscribe({
					userVisibleOnly: true,
					applicationServerKey: this.urlBase64ToUint8Array(vapidPublicKey)
				});
			}

			// Send subscription to backend
			const response = await fetch(`${BackendURL}/monitor/${monitorID}/subscribe`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					endpoint: subscription.endpoint,
					keys: {
						p256dh: this.arrayBufferToBase64(subscription.getKey('p256dh')),
						auth: this.arrayBufferToBase64(subscription.getKey('auth'))
					}
				})
			});

			if (!response.ok) {
				throw new Error('Failed to save subscription on server');
			}

			// Save subscription locally
			this.state.subscriptions.set(monitorID, subscription);
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
			const subscription = this.state.subscriptions.get(monitorID);

			if (subscription) {
				// Notify backend to remove association
				await fetch(`${BackendURL}/monitor/${monitorID}/unsubscribe`, {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json'
					},
					body: JSON.stringify({
						endpoint: subscription.endpoint
					})
				});
			}

			// Remove from local state
			this.state.subscriptions.delete(monitorID);
			this.state.loading = false;
			this.saveSubscriptions();

			// If no more subscriptions, unsubscribe from push manager
			if (this.state.subscriptions.size === 0) {
				const registration = await navigator.serviceWorker.ready;
				const pushSubscription = await registration.pushManager.getSubscription();
				if (pushSubscription) {
					await pushSubscription.unsubscribe();
				}
			}

			return true;
		} catch (error) {
			console.error('Failed to unsubscribe:', error);
			this.state.loading = false;
			this.state.error = 'Failed to unsubscribe';
			return false;
		}
	}

	isSubscribed(monitorID: number): boolean {
		return this.state.subscriptions.has(monitorID);
	}

	private saveSubscriptions() {
		const subscriptionData = Array.from(this.state.subscriptions.keys());
		localStorage.setItem('monitor-subscriptions', JSON.stringify(subscriptionData));
	}

	private loadSubscriptions() {
		try {
			const stored = localStorage.getItem('monitor-subscriptions');
			if (stored) {
				const monitorIds: number[] = JSON.parse(stored);
				// Store just the IDs - actual subscriptions are managed by the browser
				monitorIds.forEach((id) => {
					this.state.subscriptions.set(id, {} as PushSubscription);
				});
			}
		} catch (error) {
			console.error('Failed to load subscriptions:', error);
		}
	}

	private urlBase64ToUint8Array(base64String: string): Uint8Array {
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

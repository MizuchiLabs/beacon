/// <reference no-default-lib="true"/>
/// <reference lib="esnext" />
/// <reference lib="webworker" />
/// <reference lib="dom" />
/// <reference types="@sveltejs/kit" />

import { build, files, version } from '$service-worker';

const self = globalThis.self as unknown as ServiceWorkerGlobalScope;

const CACHE = `cache-${version}`;

const ASSETS = [...build, ...files];

self.addEventListener('install', (event) => {
	async function addFilesToCache() {
		const cache = await caches.open(CACHE);
		await cache.addAll(ASSETS);
	}

	// Force the waiting service worker to become the active service worker
	self.skipWaiting();
	event.waitUntil(addFilesToCache());
});

self.addEventListener('activate', (event) => {
	async function deleteOldCaches() {
		for (const key of await caches.keys()) {
			if (key !== CACHE) await caches.delete(key);
		}
		// Take control of all pages immediately
		await self.clients.claim();
	}

	event.waitUntil(deleteOldCaches());
});

self.addEventListener('fetch', (event) => {
	if (event.request.method !== 'GET') return;

	const url = new URL(event.request.url);

	// Ignore extension URLs and other weird schemes
	if (url.protocol === 'chrome-extension:' || url.protocol === 'moz-extension:') {
		return;
	}

	// API responses are never cached, a stale status page is worse than none
	if (url.pathname.startsWith('/api/')) {
		return;
	}

	async function respond() {
		const cache = await caches.open(CACHE);

		if (ASSETS.includes(url.pathname)) {
			const response = await cache.match(url.pathname);
			if (response) {
				return response;
			}
		}

		try {
			const response = await fetch(event.request);

			if (!(response instanceof Response)) {
				throw new Error('invalid response from fetch');
			}

			if (response.status === 200) {
				cache.put(event.request, response.clone());
			}

			return response;
		} catch (err) {
			const response = await cache.match(event.request);
			if (response) {
				return response;
			}
			throw err;
		}
	}

	event.respondWith(respond());
});

self.addEventListener('push', (event) => {
	if (!event.data) return;

	try {
		const data = event.data?.json() || {};
		const title = data.title || 'Monitor Alert';
		const options: NotificationOptions = {
			body: data.body || 'A monitored service is down',
			icon: '/web-app-manifest-192x192.png',
			badge: '/favicon-96x96.png',
			data: {
				url: data.url || '/',
				monitorId: data.monitorId
			},
			tag: data.tag || `monitor-${data.monitorId}`,
			requireInteraction: true,
			actions: [
				{
					action: 'view',
					title: 'View Status'
				},
				{
					action: 'close',
					title: 'Dismiss'
				}
			]
		};

		event.waitUntil(self.registration.showNotification(title, options));
	} catch (error) {
		console.error('Error handling push event:', error);
	}
});

// Focus an open dashboard tab instead of stacking up new ones.
self.addEventListener('notificationclick', (event) => {
	event.notification.close();
	if (event.action === 'close') return;

	async function focusOrOpen() {
		const tabs = await self.clients.matchAll({ type: 'window', includeUncontrolled: true });
		const tab = tabs.find((client) => new URL(client.url).origin === self.location.origin);
		const url = event.notification.data?.url || '/';
		if (!tab) return self.clients.openWindow(url);
		if (url !== '/') await tab.navigate(url).catch(() => {});
		return tab.focus();
	}
	event.waitUntil(focusOrOpen());
});

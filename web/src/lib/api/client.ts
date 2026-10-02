import { browser } from '$app/env';
import { QueryClient } from '@tanstack/svelte-query';
import { client } from './generated/client.gen';
import { toast } from 'svelte-sonner';

// Relative base URL: same-origin in production, routed to the Go backend by
// the Vite dev proxy in development.
client.setConfig({ baseUrl: '/' });

// Refetch intervals live next to each query in queries.ts.
export const queryClient = new QueryClient({
	defaultOptions: {
		queries: {
			enabled: browser,
			retry: false,
			staleTime: 15_000,
			refetchOnWindowFocus: true,
			// Keeps the favicon and tab title status light fresh in pinned tabs.
			refetchIntervalInBackground: true
		},
		mutations: {
			retry: false,
			onError: (err) => {
				if (err instanceof Error) toast.error(err.message);
			}
		}
	}
});

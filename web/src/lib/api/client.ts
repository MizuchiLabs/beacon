import { browser } from '$app/environment';
import { QueryClient } from '@tanstack/svelte-query';
import { client } from './generated/client.gen';
import { getMonitorsQueryKey } from './generated/@tanstack/svelte-query.gen';
import { toast } from 'svelte-sonner';

// Relative base URL: same-origin in production, routed to the Go backend by
// the Vite dev proxy in development.
client.setConfig({ baseUrl: '/' });

export const queryClient = new QueryClient({
	defaultOptions: {
		queries: {
			enabled: browser,
			retry: false,
			refetchOnMount: true,
			refetchOnReconnect: true,
			refetchOnWindowFocus: true,
			refetchIntervalInBackground: true,
			refetchInterval: 300000 // 5min
		},
		mutations: {
			retry: false,
			onError: (err) => {
				if (err instanceof Error) toast.error(err.message);
			}
		}
	}
});

queryClient.setQueryDefaults(getMonitorsQueryKey(), { refetchInterval: 60000 });

import { browser } from '$app/environment';
import { QueryClient } from '@tanstack/svelte-query';
import { toast } from 'svelte-sonner';

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

import { createQuery, keepPreviousData } from '@tanstack/svelte-query';
import {
	getConfigOptions,
	getIncidentsOptions,
	getMonitorsOptions
} from './generated/@tanstack/svelte-query.gen';
import { timeRange } from '$lib/range.svelte';

export type { ConfigBody, Incident, IncidentUpdate, MonitorStats } from './generated/types.gen';

export function useConfig() {
	return createQuery(() => getConfigOptions());
}

export function useMonitorStats() {
	return createQuery(() => ({
		...getMonitorsOptions({ query: { seconds: Number(timeRange.current) } }),
		staleTime: 30_000,
		refetchInterval: 30_000,
		placeholderData: keepPreviousData
	}));
}

export function getIncidents() {
	return createQuery(() => ({
		...getIncidentsOptions(),
		refetchInterval: 60_000
	}));
}

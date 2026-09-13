import { createQuery } from '@tanstack/svelte-query';
import {
	getConfigOptions,
	getIncidentsOptions,
	getMonitorsOptions
} from './generated/@tanstack/svelte-query.gen';
import { timeRange } from '$lib/range.svelte';

export type { ConfigBody, Incident, IncidentUpdate, MonitorStats } from './generated/types.gen';

const DEFAULT_WINDOW = 86400;

export function useConfig() {
	return createQuery(() => getConfigOptions());
}

export function useMonitorStats() {
	let seconds = Number(timeRange.current ?? DEFAULT_WINDOW);
	return createQuery(() => ({
		...getMonitorsOptions({ query: { seconds } }),
		refetchInterval: 30_000
	}));
}

export function getIncidents() {
	return createQuery(() => ({
		...getIncidentsOptions(),
		refetchInterval: 60_000
	}));
}

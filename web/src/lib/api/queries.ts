import { createQuery, keepPreviousData } from '@tanstack/svelte-query';
import {
	getConfigOptions,
	getIncidentOptions,
	getIncidentsOptions,
	getMonitorPercentilesOptions,
	getMonitorsOptions
} from './generated/@tanstack/svelte-query.gen';
import { timeRange } from '$lib/range.svelte';

export type {
	ConfigBody,
	Incident,
	IncidentUpdate,
	MonitorStats,
	Percentiles
} from './generated/types.gen';

export function useConfig() {
	return createQuery(() => getConfigOptions());
}

export function useMonitorStats() {
	return createQuery(() => ({
		...getMonitorsOptions({ query: { seconds: Number(timeRange.current) } }),
		refetchInterval: 30_000,
		placeholderData: keepPreviousData
	}));
}

export function useMonitorPercentiles(id: () => number | undefined) {
	return createQuery(() => ({
		...getMonitorPercentilesOptions({
			path: { id: id() ?? 0 },
			query: { seconds: Number(timeRange.current) }
		}),
		enabled: id() !== undefined,
		refetchInterval: 60_000,
		placeholderData: keepPreviousData
	}));
}

export function getIncidents() {
	return createQuery(() => ({
		...getIncidentsOptions(),
		refetchInterval: 60_000
	}));
}

export function getIncident(id: () => string) {
	return createQuery(() => ({
		...getIncidentOptions({ path: { id: id() } }),
		refetchInterval: 60_000,
		retry: false
	}));
}

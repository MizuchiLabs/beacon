import { createQuery, keepPreviousData } from '@tanstack/svelte-query';
import {
	getConfigOptions,
	getIncidentOptions,
	getIncidentsOptions,
	getMonitorChecksOptions,
	getMonitorPercentilesOptions,
	getMonitorsOptions
} from './generated/@tanstack/svelte-query.gen';
import { timeRange } from '#lib/range.svelte.js';

export type {
	CheckResult,
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

// The sheet's long view, independent of the page's range. The query key
// matches the page's when the range is 90d, so both share one cache entry.
export function useMonitorHistory(enabled: () => boolean) {
	return createQuery(() => ({
		...getMonitorsOptions({ query: { seconds: 90 * 86_400 } }),
		enabled: enabled(),
		refetchInterval: 300_000
	}));
}

export function useMonitorChecks(
	id: () => number | undefined,
	options: () => { failed: boolean; limit: number }
) {
	return createQuery(() => ({
		...getMonitorChecksOptions({ path: { id: id() ?? 0 }, query: options() }),
		enabled: id() !== undefined,
		refetchInterval: 60_000,
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

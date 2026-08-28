import { createQuery } from '@tanstack/svelte-query';
import {
	getConfigOptions,
	getIncidentsOptions,
	getMonitorsOptions
} from './generated/@tanstack/svelte-query.gen';

export type { ConfigBody, Incident, IncidentUpdate, MonitorStats } from './generated/types.gen';

const DEFAULT_WINDOW = 86400;

export function useConfig() {
	return createQuery(() => getConfigOptions());
}

export function useMonitorStats(seconds: number = DEFAULT_WINDOW) {
	return createQuery(() => getMonitorsOptions({ query: { seconds } }));
}

export function useIncidents() {
	return createQuery(() => getIncidentsOptions());
}

import { env } from '$env/dynamic/public';
import { createQuery } from '@tanstack/svelte-query';

// Types
export interface Monitor {
	id: number;
	name: string;
	url: string;
	check_interval: number;
	created_at: string;
	updated_at: string;
}

export interface MonitorStats {
	id: number;
	name: string;
	url: string;
	check_interval: number;
	uptime_pct: number;
	avg_response_time: number | null;
	percentiles: {
		p50: number | null;
		p75: number | null;
		p90: number | null;
		p95: number | null;
		p99: number | null;
	};
	data_points: ChartDataPoint[];
}

export interface ChartDataPoint {
	timestamp: string;
	response_time: number | null;
	is_up: boolean;
}

export const BackendURL = env.PUBLIC_BACKEND_URL || 'http://localhost:3000' + '/api';

async function fetchAPI<T>(endpoint: string, options?: RequestInit): Promise<T> {
	const response = await fetch(`${BackendURL}${endpoint}`, {
		headers: {
			'Content-Type': 'application/json',
			...options?.headers
		},
		...options
	});

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: 'Unknown error' }));
		throw new Error(error.error || `HTTP ${response.status}`);
	}

	if (response.status === 204) {
		return {} as T;
	}

	return response.json();
}

// API functions
export const api = {
	monitors: {
		list: () => fetchAPI<Monitor[]>('/monitors'),
		get: (id: number) => fetchAPI<Monitor>(`/monitors/${id}`),
		getStats: (seconds = '86400') => fetchAPI<MonitorStats[]>(`/monitors/stats?seconds=${seconds}`)
	}
};

// Query Hooks
export function useMonitors() {
	return createQuery(() => ({
		queryKey: ['monitors'],
		queryFn: api.monitors.list
	}));
}

export function useMonitor(id: number) {
	return createQuery(() => ({
		queryKey: ['monitors', id],
		queryFn: () => api.monitors.get(id),
		enabled: id > 0
	}));
}

export function useMonitorStats(seconds = '86400') {
	return createQuery(() => ({
		queryKey: ['monitors', 'stats', seconds],
		queryFn: () => api.monitors.getStats(seconds),
		enabled: seconds !== '',
		refetchInterval: 60000 // Refresh every minute
	}));
}

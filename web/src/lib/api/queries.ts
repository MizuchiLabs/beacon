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

export interface MonitorStatus {
	id: number;
	name: string;
	url: string;
	check_interval: number;
	created_at: string;
	updated_at: string;
	last_check_at: string | null;
	last_status: string | null;
	last_response_time: number | null;
}

export interface UptimeStats {
	total_checks: number;
	successful_checks: number;
	avg_response_time: number;
	uptime_percentage: number;
}

export interface Check {
	id: number;
	monitor_id: number;
	status: string;
	response_time: number;
	error_message: string | null;
	checked_at: string;
}

export interface CreateMonitorRequest {
	name: string;
	url: string;
	check_interval?: number;
}

export interface UpdateMonitorRequest {
	name?: string;
	url?: string;
	check_interval?: number;
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
		getStatus: (id: number) => fetchAPI<MonitorStatus>(`/monitors/${id}/status`),
		getUptimeStats: (id: number) => fetchAPI<UptimeStats>(`/monitors/${id}/uptime`),
		getCheckHistory: (id: number, limit?: number) =>
			fetchAPI<Check[]>(`/monitors/${id}/checks${limit ? `?limit=${limit}` : ''}`)
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

export function useMonitorStatus(id: number) {
	return createQuery(() => ({
		queryKey: ['monitors', id, 'status'],
		queryFn: () => api.monitors.getStatus(id),
		enabled: id > 0
	}));
}

export function useUptimeStats(id: number) {
	return createQuery(() => ({
		queryKey: ['monitors', id, 'uptime'],
		queryFn: () => api.monitors.getUptimeStats(id),
		enabled: id > 0
	}));
}

export function useCheckHistory(id: number, limit?: number) {
	return createQuery(() => ({
		queryKey: ['monitors', id, 'checks', limit],
		queryFn: () => api.monitors.getCheckHistory(id, limit),
		enabled: id > 0
	}));
}

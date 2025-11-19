import { createQuery, createMutation } from '@tanstack/svelte-query';
import { queryClient } from './client';

// Types
export interface Monitor {
	id: number;
	name: string;
	url: string;
	check_interval: number;
	is_active: boolean;
	created_at: string;
	updated_at: string;
}

export interface MonitorStatus {
	id: number;
	name: string;
	url: string;
	check_interval: number;
	is_active: boolean;
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
	is_active?: boolean;
}

// API Client Functions
const API_BASE = '/api';

async function fetchAPI<T>(endpoint: string, options?: RequestInit): Promise<T> {
	const response = await fetch(`${API_BASE}${endpoint}`, {
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
		create: (data: CreateMonitorRequest) =>
			fetchAPI<Monitor>('/monitors', {
				method: 'POST',
				body: JSON.stringify(data)
			}),
		update: (id: number, data: UpdateMonitorRequest) =>
			fetchAPI<Monitor>(`/monitors/${id}`, {
				method: 'PUT',
				body: JSON.stringify(data)
			}),
		delete: (id: number) =>
			fetchAPI<void>(`/monitors/${id}`, {
				method: 'DELETE'
			}),
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

// Mutation Hooks
export function useCreateMonitor() {
	return createMutation(() => ({
		mutationFn: api.monitors.create,
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['monitors'] });
		}
	}));
}

export function useUpdateMonitor() {
	return createMutation(() => ({
		mutationFn: ({ id, data }: { id: number; data: UpdateMonitorRequest }) =>
			api.monitors.update(id, data),
		onSuccess: (_, variables) => {
			queryClient.invalidateQueries({ queryKey: ['monitors'] });
			queryClient.invalidateQueries({ queryKey: ['monitors', variables.id] });
			queryClient.invalidateQueries({ queryKey: ['monitors', variables.id, 'status'] });
		}
	}));
}

export function useDeleteMonitor() {
	return createMutation(() => ({
		mutationFn: api.monitors.delete,
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['monitors'] });
		}
	}));
}

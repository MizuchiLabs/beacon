import type { Incident, MonitorStats } from '$lib/api/generated/types.gen';

import type { Component } from 'svelte';
import {
	ActivityIcon,
	CheckIcon,
	CircleAlertIcon,
	InfoIcon,
	SearchIcon,
	TriangleAlertIcon,
	WrenchIcon
} from '@lucide/svelte';
export type MonitorStatus = MonitorStats['status'];

export interface StatusMeta {
	label: string;
	badge: string;
	dot: string;
	stripe: string;
	text: string;
	token: string;
}

export const statusMeta: Record<MonitorStatus, StatusMeta> = {
	operational: {
		label: 'Operational',
		badge: 'bg-chart-3/15 text-chart-3 border-chart-3/20',
		dot: 'bg-chart-3',
		stripe: 'border-l-chart-3/70',
		text: 'text-chart-3',
		token: '--chart-3'
	},
	degraded: {
		label: 'Degraded',
		badge: 'bg-chart-4/15 text-chart-4 border-chart-4/20',
		dot: 'bg-chart-4',
		stripe: 'border-l-chart-4/70',
		text: 'text-chart-4',
		token: '--chart-4'
	},
	down: {
		label: 'Down',
		badge: 'bg-chart-5/15 text-chart-5 border-chart-5/20',
		dot: 'bg-chart-5',
		stripe: 'border-l-chart-5/70',
		text: 'text-chart-5',
		token: '--chart-5'
	},
	unknown: {
		label: 'Unknown',
		badge: 'bg-muted text-muted-foreground border-border',
		dot: 'bg-muted-foreground',
		stripe: 'border-l-border',
		text: 'text-muted-foreground',
		token: '--muted-foreground'
	}
};

export function aggregateStatus(monitors: MonitorStats[]): MonitorStatus {
	if (monitors.length === 0) return 'unknown';
	if (monitors.some((m) => m.status === 'down')) return 'down';
	if (monitors.some((m) => m.status === 'degraded')) return 'degraded';
	if (monitors.every((m) => m.status === 'operational')) return 'operational';
	return 'unknown';
}

export function aggregatePhrase(monitors: MonitorStats[]): string {
	const down = monitors.filter((m) => m.status === 'down').length;
	if (down === monitors.length && monitors.length > 0) return 'Major Outage';
	switch (aggregateStatus(monitors)) {
		case 'operational':
			return 'All Systems Operational';
		case 'degraded':
			return 'Degraded Performance';
		case 'down':
			return 'Partial Outage';
		default:
			return 'Status Unavailable';
	}
}

export function latencyTextClass(ms: number | null | undefined): string {
	if (ms == null) return 'text-muted-foreground';
	if (ms < 200) return 'text-chart-3';
	if (ms < 500) return 'text-chart-4';
	return 'text-chart-5';
}

export function uptimeTextClass(pct: number | null): string {
	if (pct === null) return 'text-muted-foreground';
	if (pct >= 99) return 'text-chart-3';
	if (pct >= 95) return 'text-chart-4';
	return 'text-chart-5';
}

const relative = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' });

export function ago(date: Date): string {
	const seconds = Math.round((Date.now() - date.getTime()) / 1000);
	if (seconds < 120) return relative.format(-Math.max(seconds, 1), 'second');
	if (seconds < 7200) return relative.format(-Math.round(seconds / 60), 'minute');
	return relative.format(-Math.round(seconds / 3600), 'hour');
}

export function formatMs(ms: number | null | undefined): string {
	return ms == null ? '-' : `${ms}ms`;
}

type IncidentBadge = {
	variant: 'destructive' | 'default' | 'secondary' | 'outline';
	label: string;
	icon: Component;
};

export function incidentSeverity(severity: string | undefined): IncidentBadge {
	switch (severity) {
		case 'critical':
			return { variant: 'destructive', label: 'Critical', icon: CircleAlertIcon };
		case 'major':
			return { variant: 'default', label: 'Major', icon: TriangleAlertIcon };
		case 'minor':
			return { variant: 'secondary', label: 'Minor', icon: InfoIcon };
		case 'maintenance':
			return { variant: 'outline', label: 'Maintenance', icon: WrenchIcon };
		default:
			return { variant: 'secondary', label: severity ?? 'Unknown', icon: InfoIcon };
	}
}

export function incidentStatus(status: string | undefined): IncidentBadge {
	switch (status) {
		case 'investigating':
			return { variant: 'default', label: 'Investigating', icon: SearchIcon };
		case 'identified':
			return { variant: 'default', label: 'Identified', icon: TriangleAlertIcon };
		case 'monitoring':
			return { variant: 'secondary', label: 'Monitoring', icon: ActivityIcon };
		case 'resolved':
			return { variant: 'outline', label: 'Resolved', icon: CheckIcon };
		default:
			return { variant: 'secondary', label: status ?? 'Unknown', icon: InfoIcon };
	}
}

export function affectsMonitor(incident: Incident, monitorName: string): boolean {
	const affected = incident.affected_monitors;
	return !affected || affected.length === 0 || affected.includes(monitorName);
}

export function isActiveIncident(incident: Incident): boolean {
	return incident.status !== 'resolved' && !incident.resolved_at;
}

export function durationText(startedAt: string, resolvedAt?: string | null): string {
	const end = resolvedAt ? new Date(resolvedAt) : new Date();
	const minutes = Math.floor((end.getTime() - new Date(startedAt).getTime()) / 60000);
	if (minutes < 60) return `${minutes}m`;
	return `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
}

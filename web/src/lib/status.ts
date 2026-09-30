import type { Incident, MonitorStats } from '$lib/api/generated/types.gen';

import type { Component } from 'svelte';
import {
	ActivityIcon,
	CalendarClockIcon,
	CheckIcon,
	CircleAlertIcon,
	InfoIcon,
	SearchIcon,
	TriangleAlertIcon,
	WrenchIcon
} from '@lucide/svelte';
export type MonitorStatus = MonitorStats['status'];
// What the page as a whole shows, monitors plus posted incidents.
export type PageStatus = MonitorStatus | 'maintenance';

export interface StatusMeta {
	label: string;
	dot: string;
	text: string;
	badge: string;
	token: string;
}

export const statusMeta: Record<PageStatus, StatusMeta> = {
	operational: {
		label: 'Operational',
		dot: 'bg-chart-3',
		text: 'text-chart-3',
		badge: 'border-chart-3/40 bg-chart-3/10 text-chart-3',
		token: '--chart-3'
	},
	degraded: {
		label: 'Degraded',
		dot: 'bg-chart-4',
		text: 'text-chart-4',
		badge: 'border-chart-4/40 bg-chart-4/10 text-chart-4',
		token: '--chart-4'
	},
	down: {
		label: 'Down',
		dot: 'bg-chart-5',
		text: 'text-chart-5',
		badge: 'border-chart-5/40 bg-chart-5/10 text-chart-5',
		token: '--chart-5'
	},
	maintenance: {
		label: 'Maintenance',
		dot: 'bg-chart-2',
		text: 'text-chart-2',
		badge: 'border-chart-2/40 bg-chart-2/10 text-chart-2',
		token: '--chart-2'
	},
	unknown: {
		label: 'Unknown',
		dot: 'bg-muted-foreground',
		text: 'text-muted-foreground',
		badge: 'text-muted-foreground',
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

// Names the broken monitor when there is only one, counts them otherwise.
export function aggregatePhrase(monitors: MonitorStats[]): string {
	const status = aggregateStatus(monitors);
	if (status === 'operational') return 'All systems operational';
	if (status === 'unknown') return 'Status unavailable';

	const affected = monitors.filter((m) => m.status === status);
	const word = status === 'down' ? 'down' : 'degraded';
	if (affected.length === monitors.length && monitors.length > 1) return `All monitors ${word}`;
	if (affected.length === 1) return `${affected[0].name} is ${word}`;
	return `${affected.length} monitors ${word}`;
}

const rank: Record<PageStatus, number> = {
	unknown: 0,
	operational: 0,
	maintenance: 1,
	degraded: 2,
	down: 3
};

// How bad an incident is on the page status scale.
export function incidentLevel(incident: Incident): PageStatus {
	switch (incident.severity) {
		case 'maintenance':
			return 'maintenance';
		case 'minor':
			return 'degraded';
		default:
			return 'down';
	}
}

// A posted incident counts as much as a failing check, so the page never
// says "all good" while an outage is announced. The worst one sets the
// headline, monitors win ties because they are measured.
export function pageStatus(
	monitors: MonitorStats[],
	incidents: Incident[]
): { status: PageStatus; phrase: string } {
	const status = aggregateStatus(monitors);
	const worst = incidents
		.filter(isActiveIncident)
		.toSorted((a, b) => rank[incidentLevel(b)] - rank[incidentLevel(a)])[0];
	if (worst && rank[incidentLevel(worst)] > rank[status]) {
		return { status: incidentLevel(worst), phrase: worst.title };
	}
	return { status, phrase: aggregatePhrase(monitors) };
}

// Matches the backend: slower than the monitor's threshold is degraded.
export function latencyTextClass(ms: number | null | undefined, threshold: number): string {
	if (ms == null) return 'text-muted-foreground';
	return ms <= threshold ? 'text-chart-3' : 'text-chart-4';
}

export function uptimeTextClass(pct: number | null): string {
	if (pct === null) return 'text-muted-foreground';
	if (pct >= 99) return 'text-chart-3';
	if (pct >= 95) return 'text-chart-4';
	return 'text-chart-5';
}

// Mirrors checker.CertWarnDays on the backend.
export const certWarnDays = 30;

export function certTextClass(days: number | null | undefined): string {
	if (days == null) return 'text-muted-foreground';
	return days <= certWarnDays ? 'text-chart-4' : 'text-muted-foreground';
}

const relative = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' });

export function ago(date: Date): string {
	const seconds = Math.round((Date.now() - date.getTime()) / 1000);
	if (seconds < 120) return relative.format(-Math.max(seconds, 1), 'second');
	if (seconds < 7200) return relative.format(-Math.round(seconds / 60), 'minute');
	if (seconds < 172_800) return relative.format(-Math.round(seconds / 3600), 'hour');
	return relative.format(-Math.round(seconds / 86_400), 'day');
}

const typeLabels: Record<MonitorStats['type'], string> = {
	http: '',
	tcp: 'TCP',
	ssl: 'SSL',
	dns: 'DNS',
	ping: 'Ping',
	push: 'Push'
};

export function typeLabel(type: MonitorStats['type']): string {
	return typeLabels[type];
}

// What a row shows under the monitor name. Push monitors have no public url.
export function targetOf(monitor: MonitorStats): string {
	if (monitor.type === 'push')
		return `expects a ping every ${intervalText(monitor.check_interval)}`;
	try {
		const url = new URL(monitor.url);
		return url.host + (monitor.type === 'http' && url.pathname !== '/' ? url.pathname : '');
	} catch {
		return monitor.url;
	}
}

export function intervalText(seconds: number): string {
	if (seconds < 120) return `${seconds}s`;
	if (seconds < 7200) return `${Math.round(seconds / 60)}m`;
	if (seconds < 172_800) return `${Math.round(seconds / 3600)}h`;
	return `${Math.round(seconds / 86_400)}d`;
}

export interface MonitorGroup {
	name: string;
	monitors: MonitorStats[];
}

// Groups keep the order their first monitor has in the config, ungrouped
// monitors come first.
export function groupMonitors(monitors: MonitorStats[]): MonitorGroup[] {
	const groups = new Map<string, MonitorStats[]>([['', []]]);
	for (const m of monitors) {
		const group = groups.get(m.group) ?? [];
		group.push(m);
		groups.set(m.group, group);
	}
	return [...groups]
		.filter(([, list]) => list.length > 0)
		.map(([name, list]) => ({ name, monitors: list }));
}

// Tab title that reads like a status light, e.g. "🔴 2 down · Beacon".
export function statusTitle(monitors: MonitorStats[], brand: string): string {
	const down = monitors.filter((m) => m.status === 'down').length;
	if (down > 0) return `🔴 ${down} down · ${brand}`;
	const degraded = monitors.filter((m) => m.status === 'degraded').length;
	if (degraded > 0) return `🟡 ${degraded} degraded · ${brand}`;
	return brand;
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
		case 'scheduled':
			return { variant: 'outline', label: 'Scheduled', icon: CalendarClockIcon };
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

// Scheduled maintenance that has started is running, not scheduled anymore.
export function currentStatus(incident: Incident): IncidentBadge {
	if (incident.status === 'scheduled' && !isUpcoming(incident)) {
		return { variant: 'secondary', label: 'In progress', icon: WrenchIcon };
	}
	return incidentStatus(incident.status);
}

// Timeline dot color per update status.
export const updateDot: Record<Incident['status'], string> = {
	scheduled: 'bg-chart-2',
	investigating: 'bg-chart-5',
	identified: 'bg-chart-4',
	monitoring: 'bg-chart-2',
	resolved: 'bg-chart-3'
};

export function affectsMonitor(incident: Incident, monitorName: string): boolean {
	const affected = incident.affected_monitors;
	return !affected || affected.length === 0 || affected.includes(monitorName);
}

export function isActiveIncident(incident: Incident): boolean {
	return incident.status !== 'resolved' && !isUpcoming(incident);
}

export function isUpcoming(incident: Incident): boolean {
	return incident.status !== 'resolved' && Date.parse(incident.started_at) > Date.now();
}

// The span an incident covers, open ended ones run until now.
export function incidentWindow(incident: Incident): [number, number] {
	const start = Date.parse(incident.started_at);
	const end = incident.resolved_at ? Date.parse(incident.resolved_at) : Date.now();
	return [start, Math.max(start, end)];
}

export function durationText(startedAt: string, resolvedAt?: string | null): string {
	const end = resolvedAt ? new Date(resolvedAt) : new Date();
	const minutes = Math.max(0, Math.floor((end.getTime() - new Date(startedAt).getTime()) / 60000));
	if (minutes < 60) return `${minutes}m`;
	const hours = Math.floor(minutes / 60);
	if (hours < 48) return `${hours}h ${minutes % 60}m`;
	return `${Math.floor(hours / 24)}d ${hours % 24}h`;
}

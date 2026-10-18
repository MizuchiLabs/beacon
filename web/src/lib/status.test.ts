import type { Incident, MonitorStats } from '#lib/api/generated/types.gen.js';

import { describe, expect, it } from 'vitest';
import {
	aggregatePhrase,
	aggregateStatus,
	incidentLevel,
	isActiveIncident,
	isUpcoming,
	pageStatus
} from '#lib/status.js';

const hour = 3_600_000;

function monitor(name: string, status: MonitorStats['status']): MonitorStats {
	return {
		id: 1,
		name,
		status,
		type: 'http',
		url: 'https://example.com',
		group: '',
		check_interval: 60,
		degraded_threshold: 500,
		ignore_cert_expiry: false,
		avg_response_time: null,
		last_response_time: null,
		days_remaining: null,
		uptime_pct: null,
		data_points: null
	};
}

function incident(severity: Incident['severity'], extra: Partial<Incident> = {}): Incident {
	return {
		id: 'incident',
		title: 'Database outage',
		severity,
		status: 'investigating',
		started_at: new Date(Date.now() - hour).toISOString(),
		...extra
	};
}

const future = () => new Date(Date.now() + hour).toISOString();

describe('aggregateStatus', () => {
	it('is unknown without monitors', () => {
		expect(aggregateStatus([])).toBe('unknown');
	});

	it('lets down beat degraded', () => {
		const monitors = [monitor('a', 'operational'), monitor('b', 'degraded'), monitor('c', 'down')];
		expect(aggregateStatus(monitors)).toBe('down');
	});

	it('is degraded when nothing is down', () => {
		expect(aggregateStatus([monitor('a', 'operational'), monitor('b', 'degraded')])).toBe(
			'degraded'
		);
	});

	it('is operational when every monitor is', () => {
		expect(aggregateStatus([monitor('a', 'operational'), monitor('b', 'operational')])).toBe(
			'operational'
		);
	});

	it('is unknown when operational and unknown mix', () => {
		expect(aggregateStatus([monitor('a', 'operational'), monitor('b', 'unknown')])).toBe('unknown');
	});
});

describe('aggregatePhrase', () => {
	it('says all systems operational', () => {
		expect(aggregatePhrase([monitor('a', 'operational')])).toBe('All systems operational');
	});

	it('names a single broken monitor', () => {
		expect(aggregatePhrase([monitor('api', 'down'), monitor('web', 'operational')])).toBe(
			'api is down'
		);
		expect(aggregatePhrase([monitor('api', 'down')])).toBe('api is down');
	});

	it('says all monitors when every one is broken', () => {
		expect(aggregatePhrase([monitor('a', 'down'), monitor('b', 'down')])).toBe('All monitors down');
	});

	it('counts several broken monitors', () => {
		const monitors = [
			monitor('a', 'degraded'),
			monitor('b', 'degraded'),
			monitor('c', 'operational')
		];
		expect(aggregatePhrase(monitors)).toBe('2 monitors degraded');
	});

	it('only counts the worst status', () => {
		const monitors = [monitor('api', 'down'), monitor('b', 'degraded'), monitor('c', 'degraded')];
		expect(aggregatePhrase(monitors)).toBe('api is down');
	});
});

describe('incidentLevel', () => {
	it('maps severity to the page scale', () => {
		expect(incidentLevel(incident('maintenance'))).toBe('maintenance');
		expect(incidentLevel(incident('minor'))).toBe('degraded');
		expect(incidentLevel(incident('major'))).toBe('down');
		expect(incidentLevel(incident('critical'))).toBe('down');
	});
});

describe('isUpcoming and isActiveIncident', () => {
	it('treats a started incident as active', () => {
		const started = incident('major');
		expect(isUpcoming(started)).toBe(false);
		expect(isActiveIncident(started)).toBe(true);
	});

	it('treats a future start as upcoming, not active', () => {
		const planned = incident('maintenance', { status: 'scheduled', started_at: future() });
		expect(isUpcoming(planned)).toBe(true);
		expect(isActiveIncident(planned)).toBe(false);
	});

	it('never treats a resolved incident as upcoming or active', () => {
		const resolved = incident('maintenance', { status: 'resolved', started_at: future() });
		expect(isUpcoming(resolved)).toBe(false);
		expect(isActiveIncident(resolved)).toBe(false);
	});
});

describe('pageStatus', () => {
	const allGood = [monitor('api', 'operational'), monitor('web', 'operational')];

	it('lets a worse incident set the headline', () => {
		expect(pageStatus(allGood, [incident('critical')])).toEqual({
			status: 'down',
			phrase: 'Database outage'
		});
	});

	it('shows running maintenance over operational monitors', () => {
		expect(pageStatus(allGood, [incident('maintenance', { title: 'Upgrade' })])).toEqual({
			status: 'maintenance',
			phrase: 'Upgrade'
		});
	});

	it('picks the worst active incident', () => {
		const incidents = [
			incident('minor', { title: 'Slow' }),
			incident('critical', { title: 'Outage' })
		];
		expect(pageStatus(allGood, incidents)).toEqual({ status: 'down', phrase: 'Outage' });
	});

	it('keeps the monitors on a tie', () => {
		const monitors = [monitor('api', 'degraded'), monitor('web', 'operational')];
		expect(pageStatus(monitors, [incident('minor')])).toEqual({
			status: 'degraded',
			phrase: 'api is degraded'
		});
	});

	it('keeps the monitors when they are worse', () => {
		const monitors = [monitor('api', 'down'), monitor('web', 'operational')];
		expect(pageStatus(monitors, [incident('maintenance')])).toEqual({
			status: 'down',
			phrase: 'api is down'
		});
	});

	it('ignores resolved and upcoming incidents', () => {
		const incidents = [
			incident('critical', { status: 'resolved' }),
			incident('maintenance', { status: 'scheduled', started_at: future() })
		];
		expect(pageStatus(allGood, incidents)).toEqual({
			status: 'operational',
			phrase: 'All systems operational'
		});
	});
});

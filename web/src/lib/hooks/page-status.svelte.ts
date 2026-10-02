import { getIncidents, useMonitorStats } from '#lib/api/queries.js';
import { pageStatus, statusMeta, type PageStatus } from '#lib/status.js';
import { useInterval } from 'runed';

// Three missed polls.
const staleAfter = 90_000;

// A failed or paused refetch keeps the last answer around. Past a few missed
// polls that answer counts as unknown, so a dead connection never looks healthy.
export function usePageStatus() {
	const statsQuery = useMonitorStats();
	const incidentsQuery = getIncidents();
	const clock = useInterval(10_000);

	const monitors = $derived(statsQuery.data ?? []);
	const stale = $derived.by(() => {
		void clock.counter;
		if (!statsQuery.data) return false;
		if (statsQuery.isError) return true;
		// Placeholder data belongs to the previous range and has no age of its own.
		return !statsQuery.isPlaceholderData && Date.now() - statsQuery.dataUpdatedAt > staleAfter;
	});
	const page = $derived<{ status: PageStatus; phrase: string }>(
		stale
			? { status: 'unknown', phrase: 'Connection lost' }
			: pageStatus(monitors, incidentsQuery.data ?? [])
	);

	return {
		get monitors() {
			return monitors;
		},
		get stale() {
			return stale;
		},
		get status() {
			return page.status;
		},
		get phrase() {
			return page.phrase;
		},
		get meta() {
			return statusMeta[page.status];
		}
	};
}

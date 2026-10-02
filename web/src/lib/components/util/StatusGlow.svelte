<script lang="ts">
	import { getIncidents, useMonitorStats } from '#lib/api/queries.js';
	import { pageStatus, statusMeta } from '#lib/status.js';
	import { cn } from '#lib/utils.js';

	const statsQuery = useMonitorStats();
	const monitors = $derived(statsQuery.data ?? []);
	const incidentsQuery = getIncidents();
	const meta = $derived(statusMeta[pageStatus(monitors, incidentsQuery.data ?? []).status]);
</script>

<div
	aria-hidden="true"
	class={cn(
		'glow pointer-events-none fixed inset-x-0 top-0 z-0 h-[70vh] transition-[background-color,opacity] duration-1000 ease-out motion-reduce:transition-none',
		meta.dot,
		monitors.length > 0 ? 'opacity-15 dark:opacity-20' : 'opacity-0'
	)}
></div>

<style>
	.glow {
		mask-image: radial-gradient(ellipse 60% 70% at 50% 0%, black, transparent 100%);
	}
</style>

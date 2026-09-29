<script lang="ts">
	import { useMonitorStats } from '$lib/api/queries';
	import { aggregateStatus, statusMeta } from '$lib/status.js';
	import { cn } from '$lib/utils.js';

	const statsQuery = useMonitorStats();
	const monitors = $derived(statsQuery.data ?? []);
	const meta = $derived(statusMeta[aggregateStatus(monitors)]);
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

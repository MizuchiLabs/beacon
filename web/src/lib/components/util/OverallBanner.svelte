<script lang="ts">
	import type { Incident, MonitorStats } from '$lib/api/queries';
	import { timeRange } from '$lib/range.svelte';
	import { pageStatus, statusMeta } from '$lib/status.js';
	import { cn } from '$lib/utils.js';
	import { CheckIcon, CircleHelpIcon, TriangleAlertIcon, WrenchIcon, XIcon } from '@lucide/svelte';

	interface Props {
		monitors: MonitorStats[];
		incidents: Incident[];
		updatedAgo?: string | null;
		class?: string;
	}
	let { monitors, incidents, updatedAgo, class: className }: Props = $props();

	const page = $derived(pageStatus(monitors, incidents));
	const status = $derived(page.status);
	const meta = $derived(statusMeta[status]);
	const avgUptime = $derived.by(() => {
		const values = monitors.map((m) => m.uptime_pct).filter((v): v is number => v != null);
		if (values.length === 0) return null;
		return values.reduce((sum, v) => sum + v, 0) / values.length;
	});

	const icons = {
		operational: CheckIcon,
		degraded: TriangleAlertIcon,
		down: XIcon,
		maintenance: WrenchIcon,
		unknown: CircleHelpIcon
	};
	const rings = {
		operational: 'bg-chart-3/15',
		degraded: 'bg-chart-4/15',
		down: 'bg-chart-5/15',
		maintenance: 'bg-chart-2/15',
		unknown: 'bg-muted'
	};
	const Icon = $derived(icons[status]);
</script>

<div class={cn('flex items-center gap-4 px-1 py-2', className)} role="status">
	<span
		class={cn(
			'flex size-12 shrink-0 items-center justify-center rounded-full transition-colors duration-500',
			rings[status],
			meta.text,
			(status === 'degraded' || status === 'down') &&
				'animate-status-pulse animation-duration-[1.2s]'
		)}
	>
		<Icon class="size-6" strokeWidth={2.5} />
	</span>
	<div class="flex min-w-0 flex-col gap-0.5">
		<h1 class="truncate text-xl font-semibold tracking-tight sm:text-2xl">
			{page.phrase}
		</h1>
		<p class="text-sm text-muted-foreground">
			{monitors.length}
			{monitors.length === 1 ? 'monitor' : 'monitors'}
			{#if avgUptime !== null}
				· <span class="tabular-nums">{avgUptime.toFixed(2)}%</span> over {timeRange.entry.label}
			{/if}
			{#if updatedAgo}
				<span class="hidden sm:inline">· updated {updatedAgo}</span>
			{/if}
		</p>
	</div>
</div>

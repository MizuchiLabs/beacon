<script lang="ts">
	import type { MonitorStats } from '$lib/api/queries';
	import { aggregatePhrase, aggregateStatus, statusMeta } from '$lib/status.js';
	import { cn } from '$lib/utils.js';
	import { DotIcon } from '@lucide/svelte';

	interface Props {
		monitors: MonitorStats[];
		updatedAgo?: string | null;
		class?: string;
	}
	let { monitors, updatedAgo, class: className }: Props = $props();

	const status = $derived(aggregateStatus(monitors));
	const meta = $derived(statusMeta[status]);
	const operational = $derived(monitors.filter((m) => m.status === 'operational').length);
	const avgUptime = $derived.by(() => {
		const values = monitors.map((m) => m.uptime_pct).filter((v): v is number => v != null);
		if (values.length === 0) return null;
		return values.reduce((sum, v) => sum + v, 0) / values.length;
	});

	const tints = {
		operational: 'border-chart-3/25 bg-chart-3/8',
		degraded: 'border-chart-4/30 bg-chart-4/10',
		down: 'border-chart-5/30 bg-chart-5/10',
		unknown: 'border-border bg-muted/50'
	};
</script>

<div
	class={cn(
		'flex min-w-0 flex-1 items-center gap-3 rounded-2xl border px-4 py-3 transition-colors duration-500',
		tints[status],
		className
	)}
	role="status"
>
	<!-- The dot breathes while everything is fine and pulses faster when not. -->
	<span
		class={cn(
			'size-2.5 shrink-0 animate-status-pulse rounded-full',
			meta.dot,
			meta.text,
			status !== 'operational' && 'animation-duration-[1.2s]'
		)}
	></span>

	<span
		class={cn(
			'min-w-0 flex-1 truncate text-sm font-semibold',
			status !== 'operational' && meta.text
		)}
	>
		{aggregatePhrase(monitors)}
	</span>

	<div class="flex shrink-0 items-center gap-1 text-xs whitespace-nowrap text-muted-foreground">
		<span>{operational}/{monitors.length} up</span>
		{#if avgUptime !== null}
			<span class="hidden items-center sm:inline-flex">
				<DotIcon size={12} class="shrink-0" />
				{avgUptime.toFixed(2)}%
			</span>
		{/if}
		{#if updatedAgo}
			<span class="hidden items-center md:inline-flex" title="Last updated">
				<DotIcon size={12} class="shrink-0" />
				{updatedAgo}
			</span>
		{/if}
	</div>
</div>

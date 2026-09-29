<script lang="ts">
	import type { MonitorStats } from '$lib/api/queries';
	import * as Item from '$lib/components/ui/item';
	import { Separator } from '$lib/components/ui/separator';
	import { aggregatePhrase, aggregateStatus, statusMeta } from '$lib/status.js';
	import { cn } from '$lib/utils.js';

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
		'min-w-0 flex-1 rounded-2xl border transition-colors duration-500',
		tints[status],
		className
	)}
	role="status"
>
	<Item.Root size="sm" class="flex-nowrap">
		<Item.Media>
			<!-- The dot breathes while everything is fine and pulses faster when not. -->
			<span
				class={cn(
					'size-2.5 animate-status-pulse rounded-full',
					meta.dot,
					meta.text,
					status !== 'operational' && 'animation-duration-[1.2s]'
				)}
			></span>
		</Item.Media>
		<Item.Content class="min-w-0">
			<Item.Title>
				<span class={cn('font-semibold', status !== 'operational' && meta.text)}>
					{aggregatePhrase(monitors)}
				</span>
			</Item.Title>
		</Item.Content>
		<Item.Actions>
			<div class="flex items-center gap-2 text-xs whitespace-nowrap text-muted-foreground">
				<span>{operational}/{monitors.length} up</span>
				{#if avgUptime !== null}
					<Separator orientation="vertical" class="hidden h-3 sm:block" />
					<span class="hidden tabular-nums sm:inline">{avgUptime.toFixed(2)}%</span>
				{/if}
				{#if updatedAgo}
					<Separator orientation="vertical" class="hidden h-3 md:block" />
					<span class="hidden md:inline" title="Last updated">{updatedAgo}</span>
				{/if}
			</div>
		</Item.Actions>
	</Item.Root>
</div>

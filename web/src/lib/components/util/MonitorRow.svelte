<script lang="ts">
	import type { MonitorStats } from '#lib/api/queries.js';
	import HeartbeatBar from '#lib/components/chart/HeartbeatBar.svelte';
	import UptimeBar from '#lib/components/chart/UptimeBar.svelte';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as Item from '#lib/components/ui/item/index.js';
	import SubscribeBell from '#lib/components/util/SubscribeBell.svelte';
	import {
		durationText,
		formatMs,
		statusMeta,
		targetOf,
		typeLabel,
		uptimeTextClass
	} from '#lib/status.js';
	import { cn } from '#lib/utils.js';
	import { ChevronRightIcon } from '@lucide/svelte';

	interface Props {
		monitor: MonitorStats;
		flash?: boolean;
		onOpen: (id: number) => void;
	}
	let { monitor, flash = false, onOpen }: Props = $props();

	const meta = $derived(statusMeta[monitor.status]);
	const isPush = $derived(monitor.type === 'push');
	const badge = $derived.by(() => {
		if (monitor.status === 'down') {
			return monitor.down_since ? `Down for ${durationText(monitor.down_since)}` : 'Down';
		}
		if (monitor.status === 'degraded') return 'Slow';
		if (monitor.status === 'unknown') return meta.label;
		return null;
	});
</script>

<!-- The name button stretches over the whole row, so the row is one tab stop
while the bar and the bell stay usable on top of it. -->
<div
	role="listitem"
	class={cn(
		'group relative transition-colors hover:bg-muted/40',
		monitor.status === 'down' && 'bg-chart-5/5',
		flash && 'animate-status-flash'
	)}
>
	{#if monitor.status !== 'operational'}
		<span class={cn('absolute inset-y-0 left-0 w-0.5', meta.dot)}></span>
	{/if}
	<Item.Root class="md:flex-nowrap">
		<div class="flex min-w-0 flex-1 items-center md:w-48 md:flex-none">
			<Item.Content class="min-w-0">
				<Item.Title class="w-full min-w-0">
					<button
						type="button"
						class="truncate text-left outline-none after:absolute after:inset-0 focus-visible:after:ring-[3px] focus-visible:after:ring-ring/50 focus-visible:after:ring-inset"
						onclick={() => onOpen(monitor.id)}
						aria-label="{monitor.name}, {meta.label}. Show details"
					>
						{monitor.name}
					</button>
					{#if typeLabel(monitor.type)}
						<Badge variant="secondary">{typeLabel(monitor.type)}</Badge>
					{/if}
				</Item.Title>
				<Item.Description>
					<span class={cn('block truncate', !isPush && 'font-mono')}>
						{targetOf(monitor)}
					</span>
				</Item.Description>
				{#if badge}
					<Badge variant="outline" class={cn('mt-0.5', meta.badge)}>{badge}</Badge>
				{/if}
			</Item.Content>
		</div>

		<div class="order-last w-full md:order-0 md:w-auto md:min-w-0 md:flex-1">
			{#if isPush}
				<HeartbeatBar {monitor} class="relative" />
			{:else}
				<UptimeBar
					points={monitor.data_points ?? []}
					interval={monitor.check_interval}
					alert={monitor.status === 'down'}
					class="relative"
					onclick={() => onOpen(monitor.id)}
				/>
			{/if}
		</div>

		<Item.Actions class="shrink-0">
			<span class="hidden w-16 text-right text-sm font-medium tabular-nums sm:block">
				{#if !isPush && monitor.status !== 'down'}
					{formatMs(monitor.last_response_time)}
				{/if}
			</span>
			<span
				class={cn(
					'w-16 text-right text-sm font-medium tabular-nums',
					uptimeTextClass(monitor.uptime_pct)
				)}
			>
				{monitor.uptime_pct == null ? '-' : `${monitor.uptime_pct.toFixed(2)}%`}
			</span>
			<span
				class="relative opacity-100 md:opacity-0 md:group-hover:opacity-100 md:focus-within:opacity-100"
			>
				<SubscribeBell monitorId={monitor.id} />
			</span>
			<ChevronRightIcon
				class="hidden size-4 text-muted-foreground/50 transition-transform group-hover:translate-x-0.5 md:block"
			/>
		</Item.Actions>
	</Item.Root>
</div>

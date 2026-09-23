<script lang="ts">
	import type { MonitorStats } from '$lib/api/generated/types.gen';
	import { Badge } from '$lib/components/ui/badge';
	import * as HoverCard from '$lib/components/ui/hover-card';
	import * as Item from '$lib/components/ui/item/index.js';
	import SubscribeBell from '$lib/components/util/SubscribeBell.svelte';

	import {
		ago,
		certWarnDays,
		formatMs,
		latencyTextClass,
		statusMeta,
		uptimeTextClass
	} from '$lib/status.js';
	import { cn } from '$lib/utils.js';
	import { CalendarClockIcon, ChevronRightIcon } from '@lucide/svelte';
	import StatusChart from '../chart/StatusChart.svelte';

	interface Props {
		monitor: MonitorStats;
		onOpen: (monitor: MonitorStats) => void;
	}
	let { monitor: monitorProp, onOpen }: Props = $props();

	// The query wraps results in deep proxies. Reading chart fields through them
	// is slow, but an unconditional clone would hand the chart a new object on
	// every fetch notify. Tanstack keeps the prop identical while content is
	// unchanged, so only clone when the reference actually moves.
	let lastSource: MonitorStats | undefined;
	let lastPlain!: MonitorStats;
	const monitor = $derived.by(() => {
		if (monitorProp === lastSource) return lastPlain;
		lastSource = monitorProp;
		lastPlain = $state.snapshot(monitorProp) as MonitorStats;
		return lastPlain;
	});
	const meta = $derived(statusMeta[monitor.status]);
	const host = $derived.by(() => {
		try {
			return new URL(monitor.url).host;
		} catch {
			return monitor.url;
		}
	});

	function open() {
		onOpen(monitor);
	}
</script>

<Item.Root onclick={open} class="group">
	<Item.Content class="min-w-0 md:max-w-36">
		<Item.Title class="flex items-center">
			{monitor.name}
			{#if monitor.type !== 'http'}
				<span class="text-xs font-medium text-muted-foreground uppercase">{monitor.type}</span>
			{/if}
		</Item.Title>
		<Item.Description>
			<a href={monitor.url} target="_blank" rel="noreferrer" class="truncate no-underline!">
				{host}
			</a>
		</Item.Description>
	</Item.Content>
	<Item.Content
		class="order-last w-full flex-row items-center md:order-0 md:w-auto md:min-w-0 md:flex-1!"
	>
		<StatusChart {monitor} class="mr-6 h-9 w-full" />

		<HoverCard.Root openDelay={300}>
			<HoverCard.Trigger>
				{#if !monitor.ignore_cert_expiry && monitor.days_remaining != null && monitor.days_remaining <= certWarnDays}
					<Badge variant="outline" class="hidden md:inline-flex">
						<CalendarClockIcon class="size-3" />
						{monitor.days_remaining}d
					</Badge>
				{/if}
				<Badge variant="outline" class={cn('hidden md:inline-flex', meta.badge)}>
					{meta.label}
				</Badge>
			</HoverCard.Trigger>
			<HoverCard.Content align="center" side="left" sideOffset={16} class="flex w-56 flex-col">
				<div class="flex items-center gap-2">
					<span class="size-2 rounded-full {meta.dot}"></span>
					<span class="text-sm font-medium">{meta.label}</span>
					{#if monitor.last_checked_at}
						<span class="ml-auto text-xs text-muted-foreground">
							{ago(new Date(monitor.last_checked_at))}
						</span>
					{/if}
				</div>
				<div class="mt-3 grid grid-cols-2 gap-4">
					<div class="flex flex-col gap-0.5">
						<p class="text-xs text-muted-foreground">Uptime</p>
						<p class="text-sm font-semibold tabular-nums {uptimeTextClass(monitor.uptime_pct)}">
							{monitor.uptime_pct === null ? '-' : `${monitor.uptime_pct.toFixed(2)}%`}
						</p>
					</div>
					<div class="flex flex-col gap-0.5">
						<p class="text-xs text-muted-foreground">Avg response</p>
						<p
							class="text-sm font-semibold tabular-nums {latencyTextClass(
								monitor.avg_response_time
							)}"
						>
							{formatMs(monitor.avg_response_time)}
						</p>
					</div>
				</div>
			</HoverCard.Content>
		</HoverCard.Root>
	</Item.Content>
	<Item.Actions>
		<SubscribeBell
			monitorId={monitor.id}
			class="opacity-100 md:opacity-0 md:group-hover:opacity-100 md:focus-visible:opacity-100"
		/>
		<ChevronRightIcon class="hidden size-4 text-muted-foreground/50 md:block" />
	</Item.Actions>
</Item.Root>

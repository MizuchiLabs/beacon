<script lang="ts">
	import type { MonitorStats } from '$lib/api/generated/types.gen';
	import StatusChart from '$lib/components/chart/StatusChart.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import * as HoverCard from '$lib/components/ui/hover-card';
	import * as Item from '$lib/components/ui/item/index.js';
	import SubscribeBell from '$lib/components/util/SubscribeBell.svelte';

	import { ago, formatMs, latencyTextClass, statusMeta, uptimeTextClass } from '$lib/status.js';
	import { cn } from '$lib/utils.js';
	import { ChevronRightIcon } from '@lucide/svelte';

	interface Props {
		monitor: MonitorStats;
		onOpen: (monitor: MonitorStats) => void;
	}
	let { monitor, onOpen }: Props = $props();

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
	<Item.Content class="line-clamp-1 max-w-36">
		<Item.Title>{monitor.name}</Item.Title>
		<Item.Description class="flex items-center gap-1 text-xs">
			<a href={monitor.url} target="_blank" rel="noreferrer" class="no-underline!">
				{host}
			</a>
		</Item.Description>
	</Item.Content>
	<Item.Content class="w-full flex-row items-center gap-6 md:w-auto md:min-w-0 md:flex-1!">
		<StatusChart {monitor} class="h-9 w-full" />

		<HoverCard.Root openDelay={300}>
			<HoverCard.Trigger>
				<Badge variant="outline" class={cn('hidden gap-1.5 text-xs md:inline-flex', meta.badge)}>
					{meta.label}
				</Badge>
			</HoverCard.Trigger>
			<HoverCard.Content
				align="center"
				side="left"
				sideOffset={16}
				class="flex w-56 flex-col gap-3"
			>
				<div class="flex items-center gap-2">
					<span class="size-2 rounded-full {meta.dot}"></span>
					<span class="text-sm font-medium">{meta.label}</span>
					{#if monitor.last_checked_at}
						<span class="ml-auto text-xs text-muted-foreground">
							{ago(new Date(monitor.last_checked_at))}
						</span>
					{/if}
				</div>
				<div class="grid grid-cols-2 gap-4">
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
			class="opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
		/>
		<ChevronRightIcon class="hidden size-4 text-muted-foreground/50 md:block" />
	</Item.Actions>
</Item.Root>

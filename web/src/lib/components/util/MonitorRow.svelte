<script lang="ts">
	import type { MonitorStats } from '$lib/api/queries';
	import UptimeBar from '$lib/components/chart/UptimeBar.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import SubscribeBell from '$lib/components/util/SubscribeBell.svelte';
	import { certWarnDays, statusMeta, targetOf, typeLabel, uptimeTextClass } from '$lib/status.js';
	import { cn } from '$lib/utils.js';
	import { CalendarClockIcon, ChevronRightIcon } from '@lucide/svelte';

	interface Props {
		monitor: MonitorStats;
		flash?: boolean;
		onOpen: (id: number) => void;
	}
	let { monitor, flash = false, onOpen }: Props = $props();

	const meta = $derived(statusMeta[monitor.status]);
	const certSoon = $derived(
		!monitor.ignore_cert_expiry &&
			monitor.days_remaining != null &&
			monitor.days_remaining <= certWarnDays
	);
</script>

<!-- The name button stretches over the whole row, so the row is one tab stop
while the bar and the bell stay usable on top of it. -->
<div
	class={cn(
		'group relative flex flex-wrap items-center gap-x-4 gap-y-3 px-4 py-3.5 transition-colors hover:bg-muted/40 md:flex-nowrap',
		flash && 'animate-status-flash'
	)}
>
	<div class="flex min-w-0 flex-1 flex-col gap-0.5 md:w-44 md:flex-none">
		<div class="flex min-w-0 items-center gap-2 text-sm font-medium">
			<span
				class={cn(
					'size-2 shrink-0 rounded-full',
					meta.dot,
					meta.text,
					monitor.status === 'down' && 'animate-status-pulse'
				)}
			></span>
			<button
				type="button"
				class="truncate text-left outline-none after:absolute after:inset-0 focus-visible:after:ring-[3px] focus-visible:after:ring-ring/50 focus-visible:after:ring-inset"
				onclick={() => onOpen(monitor.id)}
				aria-label="{monitor.name}, {meta.label}. Show details"
			>
				{monitor.name}
			</button>
			{#if typeLabel(monitor.type)}
				<span
					class="shrink-0 text-[10px] font-medium tracking-wide text-muted-foreground uppercase"
				>
					{typeLabel(monitor.type)}
				</span>
			{/if}
		</div>
		<span class="truncate pl-4 text-xs text-muted-foreground">{targetOf(monitor)}</span>
	</div>

	<div class="order-last w-full md:order-0 md:w-auto md:min-w-0 md:flex-1">
		<UptimeBar
			points={monitor.data_points ?? []}
			class="relative"
			onclick={() => onOpen(monitor.id)}
		/>
	</div>

	<div class="flex shrink-0 items-center gap-2">
		{#if certSoon}
			<Badge variant="outline" class="hidden md:inline-flex" title="Certificate expires soon">
				<CalendarClockIcon class="size-3" />
				{monitor.days_remaining}d
			</Badge>
		{/if}
		<span
			class={cn(
				'w-16 text-right text-sm font-medium tabular-nums',
				uptimeTextClass(monitor.uptime_pct)
			)}
		>
			{monitor.uptime_pct == null ? '-' : `${monitor.uptime_pct.toFixed(2)}%`}
		</span>
		<SubscribeBell
			monitorId={monitor.id}
			class="relative opacity-100 md:opacity-0 md:group-hover:opacity-100 md:focus-visible:opacity-100"
		/>
		<ChevronRightIcon
			class="hidden size-4 text-muted-foreground/50 transition-transform group-hover:translate-x-0.5 md:block"
		/>
	</div>
</div>

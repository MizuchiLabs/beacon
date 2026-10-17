<script lang="ts">
	import { useMonitorChecks, type MonitorStats } from '#lib/api/queries.js';
	import { intervalText } from '#lib/status.js';
	import { cn } from '#lib/utils.js';
	import { useInterval } from 'runed';

	interface Props {
		monitor: MonitorStats;
		class?: string;
	}
	let { monitor, class: className }: Props = $props();

	const runs = 7;
	const checksQuery = useMonitorChecks(
		() => monitor.id,
		() => ({ failed: false, limit: runs })
	);
	// Oldest first, so the newest run sits next to the deadline.
	const recent = $derived((checksQuery.data ?? []).toReversed());

	const clock = useInterval(30_000);
	const now = $derived.by(() => {
		void clock.counter;
		return Date.now();
	});
	// A missed ping is stored as a failed check, so the latest check is not
	// always the latest ping.
	const lastPing = $derived.by(() => {
		const ping = checksQuery.data?.find((c) => c.is_up)?.checked_at ?? monitor.last_checked_at;
		return ping ? Date.parse(ping) : null;
	});
	const elapsed = $derived(lastPing === null ? null : now - lastPing);
	const interval = $derived(monitor.check_interval * 1000);
	// Same grace as the scheduler, a job a few seconds late is not overdue yet.
	const overdue = $derived(
		monitor.status === 'down' || elapsed === null || elapsed > interval * 1.5
	);
	const progress = $derived(elapsed === null ? 1 : Math.min(elapsed / interval, 1));

	const runFormat = new Intl.DateTimeFormat(undefined, {
		month: 'short',
		day: 'numeric',
		hour: 'numeric',
		minute: '2-digit'
	});
</script>

<div class={cn('flex flex-col gap-2', className)}>
	<div class="flex justify-between gap-3 text-xs text-muted-foreground">
		<span>
			{#if elapsed === null}
				No ping yet
			{:else}
				Last ping <span class="font-medium text-foreground"
					>{intervalText(Math.round(elapsed / 1000))} ago</span
				>
			{/if}
		</span>
		{#if elapsed !== null}
			<span class={cn('tabular-nums', overdue && 'font-medium text-chart-5')}>
				{#if overdue}
					overdue by {intervalText(Math.round((elapsed - interval) / 1000))}
				{:else if elapsed > interval}
					due now
				{:else}
					next due in {intervalText(Math.round((interval - elapsed) / 1000))}
				{/if}
			</span>
		{/if}
	</div>
	<div
		class="relative h-2 rounded-full bg-muted"
		role="progressbar"
		aria-label="Time until the next ping is due"
		aria-valuemin={0}
		aria-valuemax={100}
		aria-valuenow={Math.round(progress * 100)}
	>
		<div
			class={cn(
				'h-full rounded-full transition-[width] duration-500',
				overdue ? 'bg-chart-5' : 'bg-chart-3'
			)}
			style:width="{progress * 100}%"
		></div>
	</div>
	{#if recent.length > 0}
		<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
			Last {recent.length}
			{recent.length === 1 ? 'run' : 'runs'}
			<span class="flex gap-0.5">
				{#each recent as run (run.checked_at)}
					<span
						class={cn('size-2.5 rounded-[3px]', run.is_up ? 'bg-chart-3' : 'bg-chart-5')}
						title="{runFormat.format(new Date(run.checked_at))}, {run.is_up
							? 'ok'
							: (run.error ?? 'failed')}"
					></span>
				{/each}
			</span>
		</div>
	{/if}
</div>

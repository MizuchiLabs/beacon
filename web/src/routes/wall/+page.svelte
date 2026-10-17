<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { useConfig, useMonitorStats, type MonitorStats } from '#lib/api/queries.js';
	import HeartbeatBar from '#lib/components/chart/HeartbeatBar.svelte';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import { Kbd } from '#lib/components/ui/kbd/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { usePageStatus } from '#lib/hooks/page-status.svelte.js';
	import { certSoon, durationText, statusMeta, statusTitle, uptimeTextClass } from '#lib/status.js';
	import { cn } from '#lib/utils.js';
	import { useInterval } from 'runed';

	const page = usePageStatus();
	const statsQuery = useMonitorStats();
	const configQuery = useConfig();
	const brand = $derived(configQuery.data?.title ?? 'Beacon');

	const order: Record<MonitorStats['status'], number> = {
		down: 0,
		degraded: 1,
		unknown: 2,
		operational: 3
	};
	// Trouble floats to the top, everything else keeps the config order.
	const tiles = $derived(page.monitors.toSorted((a, b) => order[a.status] - order[b.status]));
	const counts = $derived({
		up: page.monitors.filter((m) => m.status === 'operational').length,
		degraded: page.monitors.filter((m) => m.status === 'degraded').length,
		down: page.monitors.filter((m) => m.status === 'down').length
	});

	const clock = useInterval(10_000);
	const timeFormat = new Intl.DateTimeFormat(undefined, { hour: 'numeric', minute: '2-digit' });
	const time = $derived.by(() => {
		void clock.counter;
		return timeFormat.format(new Date());
	});

	const sparkWidth = 120;
	const sparkHeight = 32;

	// Empty buckets are skipped, not drawn as gaps. A monitor that checks less
	// often than the bucket size would otherwise turn into scattered dots.
	function sparkline(m: MonitorStats): string {
		const points = m.data_points ?? [];
		const known = points.flatMap((p, i) => (p.avg_ms == null ? [] : [{ i, ms: p.avg_ms }]));
		if (known.length < 2) return '';
		const values = known.map((k) => k.ms);
		const lo = Math.min(...values);
		const span = Math.max(...values) - lo || 1;
		const step = sparkWidth / Math.max(points.length - 1, 1);
		return known
			.map(({ i, ms }, n) => {
				const y = sparkHeight - 2 - ((ms - lo) / span) * (sparkHeight - 4);
				return `${n === 0 ? 'M' : 'L'}${(i * step).toFixed(1)} ${y.toFixed(1)}`;
			})
			.join(' ');
	}

	const stroke: Record<MonitorStats['status'], string> = {
		operational: 'stroke-chart-3',
		degraded: 'stroke-chart-4',
		down: 'stroke-chart-5',
		unknown: 'stroke-muted-foreground'
	};

	function onkeydown(event: KeyboardEvent) {
		if (event.metaKey || event.ctrlKey || event.altKey || event.defaultPrevented) return;
		const target = event.target as HTMLElement;
		if (target.closest('input, textarea, select, [contenteditable="true"], [role="dialog"]'))
			return;
		if (event.key === 'w' || event.key === 'Escape') goto(resolve('/'));
	}
</script>

<svelte:window {onkeydown} />
<svelte:head>
	<title>{page.stale ? `offline · ${brand}` : statusTitle(page.monitors, brand)}</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-[1600px] flex-col gap-6 px-4 py-6 sm:px-6">
	<header class="flex flex-wrap items-center justify-between gap-4" role="status">
		<div class="flex min-w-0 items-center gap-4">
			<span
				class={cn(
					'size-4 shrink-0 rounded-full',
					page.meta.dot,
					page.meta.text,
					(page.status === 'down' || page.status === 'degraded') && 'animate-status-pulse'
				)}
			></span>
			<h1 class="truncate text-2xl font-semibold tracking-tight sm:text-3xl">{page.phrase}</h1>
		</div>
		<div class="flex items-center gap-5 text-sm text-muted-foreground tabular-nums">
			<span><span class="text-xl font-semibold text-chart-3">{counts.up}</span> up</span>
			<span><span class="text-xl font-semibold text-chart-4">{counts.degraded}</span> degraded</span
			>
			<span><span class="text-xl font-semibold text-chart-5">{counts.down}</span> down</span>
			<span class="border-l pl-5 text-xl font-semibold text-foreground">{time}</span>
		</div>
	</header>

	{#if !statsQuery.data}
		<div class="grid grid-cols-[repeat(auto-fill,minmax(16rem,1fr))] gap-3">
			{#each { length: 8 }, i (i)}
				<Skeleton class="h-36" />
			{/each}
		</div>
	{:else if statsQuery.data.length === 0}
		<Empty.Root>
			<Empty.Header>
				<Empty.Title>No monitors configured</Empty.Title>
				<Empty.Description>Add a monitor to your config file to start tracking.</Empty.Description>
			</Empty.Header>
		</Empty.Root>
	{:else}
		<div
			class={cn(
				'grid grid-cols-[repeat(auto-fill,minmax(16rem,1fr))] gap-3 transition-[opacity,filter] duration-500',
				page.stale && 'opacity-50 grayscale'
			)}
		>
			{#each tiles as m (m.id)}
				{@const meta = statusMeta[m.status]}
				{@const ms = m.last_response_time}
				<a
					href={resolve(`/?m=${m.id}`)}
					class={cn(
						'flex flex-col gap-3 rounded-2xl border bg-card p-4 transition-colors outline-none hover:bg-muted/40 focus-visible:ring-[3px] focus-visible:ring-ring/50',
						m.status === 'down' && 'border-chart-5/60 bg-chart-5/10 sm:col-span-2',
						m.status === 'degraded' && 'border-chart-4/50 bg-chart-4/5'
					)}
				>
					<div class="flex items-start justify-between gap-3">
						<div class="flex min-w-0 flex-col gap-0.5">
							{#if m.group}
								<span class="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
									{m.group}
								</span>
							{/if}
							<span class="truncate text-lg font-semibold tracking-tight">{m.name}</span>
						</div>
						<span
							class={cn(
								'shrink-0 rounded-full border px-2.5 py-0.5 text-xs font-semibold',
								meta.badge
							)}
						>
							{meta.label}
						</span>
					</div>

					{#if m.status === 'down'}
						<div class="flex flex-col gap-1">
							<span class="text-4xl font-bold tracking-tight text-chart-5 tabular-nums">
								{m.down_since ? durationText(m.down_since) : 'Down'}
							</span>
							{#if m.last_error}
								<span class="truncate font-mono text-sm text-muted-foreground">{m.last_error}</span>
							{/if}
						</div>
					{:else if m.type === 'push'}
						<HeartbeatBar monitor={m} />
					{:else}
						<div class="flex items-end justify-between gap-3">
							<span
								class={cn(
									'text-3xl font-semibold tracking-tight tabular-nums',
									m.status === 'degraded' && 'text-chart-4'
								)}
							>
								{#if ms == null}
									-
								{:else}
									{ms}<span class="text-sm font-medium text-muted-foreground"> ms</span>
								{/if}
							</span>
							<svg
								aria-hidden="true"
								viewBox="0 0 {sparkWidth} {sparkHeight}"
								preserveAspectRatio="none"
								class="h-8 w-28 shrink-0 overflow-visible"
							>
								<path
									d={sparkline(m)}
									fill="none"
									vector-effect="non-scaling-stroke"
									class={cn('stroke-2', stroke[m.status])}
									stroke-linejoin="round"
									stroke-linecap="round"
								/>
							</svg>
						</div>
					{/if}

					<div
						class="mt-auto flex justify-between gap-3 text-sm text-muted-foreground tabular-nums"
					>
						<span class={cn(certSoon(m) && 'text-chart-4')}>
							{#if certSoon(m)}
								cert expires in {m.days_remaining}d
							{:else if m.status === 'down' && m.down_since}
								since {timeFormat.format(new Date(m.down_since))}
							{:else if m.type !== 'push'}
								response now
							{/if}
						</span>
						<span class={cn('font-semibold', uptimeTextClass(m.uptime_pct))}>
							{m.uptime_pct == null ? '-' : `${m.uptime_pct.toFixed(2)}%`}
						</span>
					</div>
				</a>
			{/each}
		</div>
	{/if}

	<p class="flex items-center gap-2 text-xs text-muted-foreground">
		Press <Kbd>W</Kbd> to leave wall mode
	</p>
</div>

<script lang="ts">
	import type { DataPoint } from '#lib/api/generated/types.gen.js';
	import * as Tooltip from '#lib/components/ui/tooltip/index.js';
	import { formatMs } from '#lib/status.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		points: DataPoint[];
		// Check interval in seconds.
		interval: number;
		class?: string;
		alert?: boolean;
		marks?: { start: number; end: number; title: string; class: string }[];
		onclick?: () => void;
	}
	let { points, interval, class: className, alert = false, marks = [], onclick }: Props = $props();

	const step = $derived(
		points.length > 1 ? Date.parse(points[1].timestamp) - Date.parse(points[0].timestamp) : 0
	);

	// Marks are placed on the same time axis the pills cover.
	const placed = $derived.by(() => {
		if (points.length < 2 || marks.length === 0) return [];
		const first = Date.parse(points[0].timestamp);
		const span = Date.parse(points[points.length - 1].timestamp) + step - first;
		return marks
			.filter((m) => m.end > first && m.start < first + span)
			.map((m) => ({
				...m,
				left: (Math.max(m.start - first, 0) / span) * 100,
				width: ((Math.min(m.end, first + span) - Math.max(m.start, first)) / span) * 100
			}));
	});

	let hovered = $state<number | null>(null);
	let anchor = $state<HTMLElement | null>(null);
	const point = $derived(hovered === null ? null : points[hovered]);

	// Day sized buckets only need the date, finer ones need the time too.
	const format = $derived(
		new Intl.DateTimeFormat(
			undefined,
			step >= 86_400_000
				? { month: 'short', day: 'numeric' }
				: { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' }
		)
	);

	// A bucket shorter than the check interval is empty by design. The last
	// result holds until the next check is due, after that it is a real gap.
	const held = $derived.by(() => {
		const reach = step > 0 ? Math.ceil((interval * 1000) / step) : 0;
		let last = -1;
		return points.map((p, i) => {
			if (p.has_data) {
				last = i;
				return p;
			}
			return p.expected === 0 && last >= 0 && i - last <= reach ? points[last] : null;
		});
	});

	// Bars are scaled to the monitor's own normal, so a spike stands out even on
	// a 3ms TCP check. The median keeps one outlier from flattening the rest.
	const scale = $derived.by(() => {
		const ms = points
			.map((p) => p.avg_ms)
			.filter((v): v is number => v != null)
			.toSorted((a, b) => a - b);
		if (ms.length === 0) return 1;
		return Math.max(ms[Math.floor(ms.length / 2)] * 2.2, ms[ms.length - 1], 1);
	});

	// One slow check in a busy bucket does not tint the pill.
	function slow(shown: DataPoint) {
		return shown.degraded * 4 >= shown.total;
	}

	// The fill height is the response time, slow buckets fill up completely.
	function fillHeight(shown: DataPoint | null) {
		if (!shown || shown.avg_ms == null) return '0%';
		if (slow(shown)) return '100%';
		return `${Math.max(18, Math.min(100, (shown.avg_ms / scale) * 100))}%`;
	}

	// How much of the pill turns red from the top. A single failed check stays visible.
	function failShare(shown: DataPoint | null) {
		if (!shown || shown.down === 0) return '0%';
		if (shown.down === shown.total) return '100%';
		return `${Math.max(shown.down / shown.total, 0.25) * 100}%`;
	}

	function summary(p: DataPoint, shown: DataPoint | null) {
		if (!shown) return p.expected > 0 ? 'No checks recorded' : 'Nothing to check yet';
		if (shown !== p) {
			const last = shown.down > 0 ? 'down' : shown.degraded > 0 ? 'slow' : 'up';
			return `No check due, the last one was ${last}`;
		}
		const pct = ((p.up + p.degraded) / p.total) * 100;
		return `${pct.toFixed(pct === 100 ? 0 : 1)}% up`;
	}

	function track(event: PointerEvent) {
		const el = (event.target as HTMLElement).closest<HTMLElement>('[data-index]');
		if (!el) return;
		hovered = Number(el.dataset.index);
		anchor = el;
	}
</script>

<Tooltip.Root open={point !== null}>
	<!-- The pills are the anchor, the trigger only provides the tooltip context. -->
	<Tooltip.Trigger>
		{#snippet child()}
			<!-- Keyboard users open the details through the row's name button. -->
			<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
			<div
				class={cn('flex h-8 w-full items-stretch gap-0.5', onclick && 'cursor-pointer', className)}
				role="img"
				aria-label="Uptime history"
				onpointermove={track}
				onpointerleave={() => (hovered = null)}
				{onclick}
			>
				{#each points as p, i (p.timestamp)}
					{@const shown = held[i]}
					<span
						data-index={i}
						class={cn(
							'relative min-w-0.5 flex-1 overflow-hidden rounded-sm transition-[transform,opacity] duration-150',
							shown ? 'bg-muted' : p.expected > 0 ? 'bg-muted-foreground/25' : 'bg-muted/60',
							i === points.length - 1 && 'flex-[1.75]',
							i === points.length - 1 && alert && 'animate-status-pulse text-chart-5',
							hovered === i ? 'scale-y-110' : hovered !== null && 'opacity-60'
						)}
					>
						{#if shown}
							<span
								class={cn(
									'pointer-events-none absolute inset-x-0 bottom-0 rounded-sm',
									slow(shown) ? 'bg-chart-4' : 'bg-chart-3'
								)}
								style:height={fillHeight(shown)}
							></span>
							<span
								class="pointer-events-none absolute inset-x-0 top-0 rounded-sm bg-chart-5"
								style:height={failShare(shown)}
							></span>
						{/if}
					</span>
				{/each}
			</div>
		{/snippet}
	</Tooltip.Trigger>
	<Tooltip.Content customAnchor={anchor}>
		{#if point && hovered !== null}
			<div class="flex flex-col gap-0.5">
				<span class="font-medium">{format.format(new Date(point.timestamp))}</span>
				<span class="opacity-80">
					{summary(point, held[hovered])}
					{#if point.down > 0}
						· {point.down} failed
					{/if}
					{#if point.avg_ms != null}
						· {formatMs(point.avg_ms)} avg
					{/if}
				</span>
			</div>
		{/if}
	</Tooltip.Content>
</Tooltip.Root>
{#if placed.length > 0}
	<div class="relative mt-1.5 h-1" aria-label="Incidents">
		{#each placed as mark, i (i)}
			<span
				class={cn('absolute inset-y-0 min-w-1 rounded-full', mark.class)}
				style="left: {mark.left}%; width: {mark.width}%"
				title={mark.title}
			></span>
		{/each}
	</div>
{/if}

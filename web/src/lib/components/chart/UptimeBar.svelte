<script lang="ts">
	import type { DataPoint } from '$lib/api/generated/types.gen';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import { formatMs } from '$lib/status.js';
	import { cn } from '$lib/utils.js';

	interface Props {
		points: DataPoint[];
		class?: string;
		alert?: boolean;
		marks?: { start: number; end: number; title: string; class: string }[];
		onclick?: () => void;
	}
	let { points, class: className, alert = false, marks = [], onclick }: Props = $props();

	// Marks are placed on the same time axis the pills cover.
	const placed = $derived.by(() => {
		if (points.length < 2 || marks.length === 0) return [];
		const first = Date.parse(points[0].timestamp);
		const step = Date.parse(points[1].timestamp) - first;
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
	const format = $derived.by(() => {
		const step =
			points.length > 1 ? Date.parse(points[1].timestamp) - Date.parse(points[0].timestamp) : 0;
		return new Intl.DateTimeFormat(
			undefined,
			step >= 86_400_000
				? { month: 'short', day: 'numeric' }
				: { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' }
		);
	});

	// Any failure tints the pill, one slow check in a busy bucket does not.
	function pillClass(p: DataPoint) {
		if (!p.has_data) return p.expected > 0 ? 'bg-muted-foreground/25' : 'bg-muted';
		if (p.down * 2 >= p.total) return 'bg-chart-5';
		if (p.down > 0) return 'bg-chart-5/55';
		if (p.degraded * 4 >= p.total) return 'bg-chart-4';
		return 'bg-chart-3';
	}

	function summary(p: DataPoint) {
		if (!p.has_data) return p.expected > 0 ? 'No checks recorded' : 'Nothing to check yet';
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
					<span
						data-index={i}
						class={cn(
							'min-w-0.5 flex-1 rounded-full transition-[transform,opacity] duration-150',
							pillClass(p),
							i === points.length - 1 && 'flex-[1.75]',
							i === points.length - 1 && alert && 'animate-status-pulse text-chart-5',
							hovered === i ? 'scale-y-110' : hovered !== null && 'opacity-60'
						)}
					></span>
				{/each}
			</div>
		{/snippet}
	</Tooltip.Trigger>
	<Tooltip.Content customAnchor={anchor}>
		{#if point}
			<div class="flex flex-col gap-0.5">
				<span class="font-medium">{format.format(new Date(point.timestamp))}</span>
				<span class="opacity-80">
					{summary(point)}
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

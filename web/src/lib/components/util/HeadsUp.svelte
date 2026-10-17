<script lang="ts">
	import { resolve } from '$app/paths';
	import type { Incident, MonitorStats } from '#lib/api/queries.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as Item from '#lib/components/ui/item/index.js';
	import { Separator } from '#lib/components/ui/separator/index.js';
	import {
		certSoon,
		currentStatus,
		durationText,
		formatMs,
		incidentLevel,
		incidentSeverity,
		isActiveIncident,
		isUpcoming,
		statusMeta,
		targetOf,
		type PageStatus
	} from '#lib/status.js';
	import { cn } from '#lib/utils.js';
	import {
		CalendarClockIcon,
		ChevronRightIcon,
		ShieldAlertIcon,
		TriangleAlertIcon,
		XIcon
	} from '@lucide/svelte';
	import type { Component } from 'svelte';

	interface Props {
		monitors: MonitorStats[];
		incidents: Incident[];
		onOpen: (id: number) => void;
	}
	let { monitors, incidents, onOpen }: Props = $props();

	interface Entry {
		key: string;
		tone: PageStatus;
		icon: Component;
		title: string;
		detail: string;
		mono?: boolean;
		meta?: string;
		href?: string;
		monitorId?: number;
	}

	const tiles: Record<PageStatus, string> = {
		operational: 'bg-chart-3/15 text-chart-3',
		degraded: 'bg-chart-4/15 text-chart-4',
		down: 'bg-chart-5/15 text-chart-5',
		maintenance: 'bg-chart-2/15 text-chart-2',
		unknown: 'bg-muted text-muted-foreground'
	};

	const weekFormat = new Intl.DateTimeFormat(undefined, {
		weekday: 'short',
		hour: 'numeric',
		minute: '2-digit'
	});
	const laterFormat = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric' });

	// The weekday is enough within a week, further out the date reads better.
	function when(iso: string): string {
		const at = new Date(iso);
		return at.getTime() - Date.now() < 6 * 86_400_000
			? weekFormat.format(at)
			: laterFormat.format(at);
	}
	const dayFormat = new Intl.DateTimeFormat(undefined, {
		month: 'short',
		day: 'numeric',
		year: 'numeric'
	});

	function hostOf(url: string): string {
		try {
			return new URL(url).hostname;
		} catch {
			return url;
		}
	}

	// Problems first, then things to plan for. Each list keeps its own order.
	const entries = $derived.by(() => {
		const list: Entry[] = [];

		for (const incident of incidents.filter(isActiveIncident)) {
			const latest = incident.updates?.at(-1);
			list.push({
				key: `incident-${incident.id}`,
				tone: incidentLevel(incident),
				icon: incidentSeverity(incident.severity).icon,
				title: incident.title,
				detail: latest?.message ?? incident.description ?? '',
				meta: currentStatus(incident).label,
				href: resolve('/events/[id]', { id: incident.id })
			});
		}

		for (const m of monitors.filter((m) => m.status === 'down')) {
			list.push({
				key: `down-${m.id}`,
				tone: 'down',
				icon: XIcon,
				title: `${m.name} is down`,
				detail: m.last_error ?? targetOf(m),
				mono: !!m.last_error,
				meta: m.down_since ? durationText(m.down_since) : undefined,
				monitorId: m.id
			});
		}

		for (const m of monitors.filter((m) => m.status === 'degraded')) {
			list.push({
				key: `slow-${m.id}`,
				tone: 'degraded',
				icon: TriangleAlertIcon,
				title: `${m.name} is slow`,
				detail: `${formatMs(m.last_response_time)}, slower than the ${formatMs(m.degraded_threshold)} threshold`,
				monitorId: m.id
			});
		}

		// An https and an ssl monitor often watch the same host, one warning covers both.
		const certs = new Map<string, MonitorStats>();
		for (const m of monitors.filter(certSoon)) {
			const host = hostOf(m.url);
			const seen = certs.get(host);
			if (!seen || m.days_remaining! < seen.days_remaining!) certs.set(host, m);
		}
		for (const [host, m] of certs) {
			const days = m.days_remaining!;
			list.push({
				key: `cert-${host}`,
				tone: 'degraded',
				icon: ShieldAlertIcon,
				title: `Certificate for ${host} expires in ${days} ${days === 1 ? 'day' : 'days'}`,
				detail: `Valid until ${dayFormat.format(Date.now() + days * 86_400_000)}`,
				monitorId: m.id
			});
		}

		// The API sorts newest first, soonest first reads better here.
		for (const incident of incidents.filter(isUpcoming).toReversed()) {
			const affects = incident.affected_monitors?.length
				? ` · affects ${incident.affected_monitors.join(', ')}`
				: '';
			list.push({
				key: `upcoming-${incident.id}`,
				tone: 'maintenance',
				icon: CalendarClockIcon,
				title: incident.title,
				detail: `Scheduled${affects}`,
				meta: when(incident.started_at),
				href: resolve('/events/[id]', { id: incident.id })
			});
		}
		return list;
	});

	const urgent = $derived(entries.some((e) => e.tone === 'down' || e.tone === 'degraded'));
</script>

{#snippet body(entry: Entry)}
	<Item.Media variant="icon">
		<span class={cn('flex size-9 items-center justify-center rounded-xl', tiles[entry.tone])}>
			<entry.icon class="size-4" />
		</span>
	</Item.Media>
	<Item.Content class="min-w-0">
		<Item.Title class="w-full min-w-0">
			<span class="truncate">{entry.title}</span>
		</Item.Title>
		{#if entry.detail}
			<Item.Description>
				<span class={cn('block truncate', entry.mono && 'font-mono')}>{entry.detail}</span>
			</Item.Description>
		{/if}
	</Item.Content>
	<Item.Actions class="shrink-0">
		{#if entry.meta}
			<span class={cn('text-sm font-medium tabular-nums', statusMeta[entry.tone].text)}>
				{entry.meta}
			</span>
		{/if}
		<ChevronRightIcon
			class="size-4 text-muted-foreground/50 transition-transform group-hover/item:translate-x-0.5"
		/>
	</Item.Actions>
{/snippet}

{#if entries.length > 0}
	<section class="flex flex-col gap-2">
		<h2 class="flex items-center gap-2 px-1 text-sm font-medium text-muted-foreground">
			{urgent ? 'Needs attention' : 'Heads up'}
			<Badge variant="secondary">{entries.length}</Badge>
		</h2>
		<div class="flex flex-col overflow-hidden rounded-2xl border bg-card">
			{#each entries as entry, i (entry.key)}
				{#if i > 0}
					<Separator />
				{/if}
				<div class={cn(entry.tone === 'down' && 'bg-chart-5/5')}>
					<Item.Root>
						{#snippet child({ props })}
							{#if entry.href}
								<a href={entry.href} {...props}>{@render body(entry)}</a>
							{:else}
								<button
									type="button"
									{...props}
									class={cn(props.class as string, 'text-left hover:bg-muted')}
									onclick={() => entry.monitorId !== undefined && onOpen(entry.monitorId)}
								>
									{@render body(entry)}
								</button>
							{/if}
						{/snippet}
					</Item.Root>
				</div>
			{/each}
		</div>
	</section>
{/if}

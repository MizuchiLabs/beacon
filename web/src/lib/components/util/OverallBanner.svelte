<script lang="ts">
	import { cn } from '$lib/utils.js';
	import * as Item from '$lib/components/ui/item';
	import type { MonitorStats } from '$lib/api/generated/types.gen';
	import { aggregatePhrase, aggregateStatus } from '$lib/status.js';
	import {
		CircleAlertIcon,
		CircleCheckIcon,
		CircleQuestionMarkIcon,
		DotIcon,
		TriangleAlertIcon
	} from '@lucide/svelte';

	interface Props {
		monitors: MonitorStats[];
		updatedAgo?: string | null;
		class?: string;
	}
	let { monitors, updatedAgo, class: className }: Props = $props();
	const status = $derived(aggregateStatus(monitors));
	const operational = $derived(monitors.filter((m) => m.status === 'operational').length);
	const avgUptime = $derived.by(() => {
		const values = monitors.map((m) => m.uptime_pct).filter((v): v is number => v !== null);
		if (values.length === 0) return null;
		return values.reduce((sum, v) => sum + v, 0) / values.length;
	});

	const lookups = {
		operational: { icon: CircleCheckIcon, class: 'text-chart-3' },
		degraded: { icon: TriangleAlertIcon, class: 'text-chart-4' },
		down: { icon: CircleAlertIcon, class: 'text-chart-5' },
		unknown: { icon: CircleQuestionMarkIcon, class: 'text-muted-foreground' }
	};
	const look = $derived(lookups[status]);
	const troubled = $derived(status === 'down' || status === 'degraded');
</script>

<Item.Root class={cn('min-w-0 bg-card/75', className)}>
	<Item.Media class="shrink-0">
		<look.icon class="size-4 {look.class}" />
	</Item.Media>

	<Item.Content class="min-w-0 flex-1">
		<div class="flex w-full min-w-0 items-center justify-between gap-3">
			<span class="min-w-0 flex-1 truncate text-sm font-medium {troubled ? look.class : ''}">
				{aggregatePhrase(monitors)}
			</span>

			<div class="flex shrink-0 items-center gap-1 text-xs whitespace-nowrap text-muted-foreground">
				<span>{operational} of {monitors.length} operational</span>

				{#if avgUptime !== null}
					<span class="hidden items-center gap-1.5 sm:inline-flex">
						<DotIcon size={12} class="shrink-0" />
						<span>{avgUptime.toFixed(2)}% avg uptime</span>
					</span>
				{/if}

				{#if updatedAgo}
					<span class="hidden items-center gap-1.5 md:inline-flex">
						<DotIcon size={12} class="shrink-0" />
						<span title="Last updated">{updatedAgo}</span>
					</span>
				{/if}
			</div>
		</div>
	</Item.Content>
</Item.Root>

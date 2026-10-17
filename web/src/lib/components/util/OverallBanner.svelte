<script lang="ts">
	import { usePageStatus } from '#lib/hooks/page-status.svelte.js';
	import { durationText, statusMeta } from '#lib/status.js';
	import { cn } from '#lib/utils.js';
	import {
		CheckIcon,
		CircleHelpIcon,
		TriangleAlertIcon,
		WifiOffIcon,
		WrenchIcon,
		XIcon
	} from '@lucide/svelte';

	interface Props {
		updatedAgo?: string | null;
		onOpen?: (id: number) => void;
		class?: string;
	}
	let { updatedAgo, onOpen, class: className }: Props = $props();

	const page = usePageStatus();
	const monitors = $derived(page.monitors);
	const status = $derived(page.status);
	const counts = $derived({
		up: monitors.filter((m) => m.status === 'operational').length,
		degraded: monitors.filter((m) => m.status === 'degraded').length,
		down: monitors.filter((m) => m.status === 'down').length
	});
	const subline = $derived.by(() => {
		const total = monitors.length;
		if (counts.up === total) return `${total} ${total === 1 ? 'monitor' : 'monitors'}`;
		const healthy = `${counts.up} of ${total} operational`;
		const down = monitors.filter((m) => m.status === 'down');
		if (down.length === 1 && down[0].down_since) {
			return `Down for ${durationText(down[0].down_since)} · ${healthy}`;
		}
		return healthy;
	});

	const icons = {
		operational: CheckIcon,
		degraded: TriangleAlertIcon,
		down: XIcon,
		maintenance: WrenchIcon,
		unknown: CircleHelpIcon
	};
	const rings = {
		operational: 'bg-chart-3/15',
		degraded: 'bg-chart-4/15',
		down: 'bg-chart-5/15',
		maintenance: 'bg-chart-2/15',
		unknown: 'bg-muted'
	};
	const Icon = $derived(page.stale ? WifiOffIcon : icons[status]);
</script>

<div
	class={cn('flex flex-wrap items-end justify-between gap-x-6 gap-y-4 px-1 py-2', className)}
	role="status"
>
	<div class="flex min-w-0 items-center gap-4">
		<span
			class={cn(
				'flex size-12 shrink-0 items-center justify-center rounded-full transition-colors duration-500',
				rings[status],
				page.meta.text,
				(status === 'degraded' || status === 'down') &&
					'animate-status-pulse animation-duration-[1.2s]'
			)}
		>
			<Icon class="size-6" strokeWidth={2.5} />
		</span>
		<div class="flex min-w-0 flex-col gap-0.5">
			<h1 class="truncate text-xl font-semibold tracking-tight sm:text-2xl">
				{page.phrase}
			</h1>
			<p class="flex flex-wrap items-center gap-x-3 text-sm text-muted-foreground">
				{#if page.stale}
					Showing the last known state{updatedAgo ? `, updated ${updatedAgo}` : ''}
				{:else}
					<span>{subline}</span>
					{#if updatedAgo}
						<span class="hidden items-center gap-2 sm:flex">
							<span class="size-1.5 animate-status-pulse rounded-full bg-chart-3 text-chart-3"
							></span>
							updated {updatedAgo}
						</span>
					{/if}
				{/if}
			</p>
		</div>
	</div>

	{#if !page.stale}
		<div class="flex flex-col items-start gap-2 sm:items-end">
			<div class="flex max-w-72 flex-wrap gap-1" aria-label="Status of each monitor">
				{#each monitors as m (m.id)}
					<button
						type="button"
						class={cn(
							'h-5 w-3 rounded-[3px] transition-transform outline-none hover:scale-y-110 focus-visible:ring-[3px] focus-visible:ring-ring/50',
							statusMeta[m.status].dot
						)}
						title="{m.name}, {statusMeta[m.status].label}"
						aria-label="{m.name}, {statusMeta[m.status].label}. Show details"
						onclick={() => onOpen?.(m.id)}
					></button>
				{/each}
			</div>
			<p class="flex gap-3 text-xs text-muted-foreground tabular-nums">
				<span><span class="font-semibold text-foreground">{counts.up}</span> up</span>
				<span><span class="font-semibold text-foreground">{counts.degraded}</span> degraded</span>
				<span><span class="font-semibold text-foreground">{counts.down}</span> down</span>
			</p>
		</div>
	{/if}
</div>

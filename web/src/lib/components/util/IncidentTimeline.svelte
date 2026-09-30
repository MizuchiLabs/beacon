<script lang="ts">
	import type { IncidentUpdate } from '$lib/api/queries';
	import { incidentStatus, updateDot } from '$lib/status.js';
	import { cn } from '$lib/utils.js';

	interface Props {
		updates: IncidentUpdate[];
	}
	let { updates }: Props = $props();

	const newestFirst = $derived(updates.toReversed());
	const format = new Intl.DateTimeFormat(undefined, {
		month: 'short',
		day: 'numeric',
		hour: 'numeric',
		minute: '2-digit',
		timeZoneName: 'short'
	});
</script>

<ol
	class="relative flex flex-col gap-5 pl-6 before:absolute before:top-2 before:left-[0.4375rem] before:h-[calc(100%-1rem)] before:w-px before:bg-border"
>
	{#each newestFirst as update, i (update.created_at + update.status)}
		<li class="relative flex flex-col gap-1">
			<span
				class="absolute top-1 -left-6 flex size-4 items-center justify-center rounded-full bg-card"
			>
				<span class={cn('size-2 rounded-full', updateDot[update.status], i > 0 && 'opacity-60')}
				></span>
			</span>
			<div class="flex flex-wrap items-baseline gap-x-2">
				<span class="text-sm font-medium">{incidentStatus(update.status).label}</span>
				<time class="text-xs text-muted-foreground" datetime={update.created_at}>
					{format.format(new Date(update.created_at))}
				</time>
			</div>
			<p class={cn('text-sm', i > 0 && 'text-muted-foreground')}>{update.message}</p>
		</li>
	{/each}
</ol>

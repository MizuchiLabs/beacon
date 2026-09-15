<script lang="ts">
	import * as ToggleGroup from '$lib/components/ui/toggle-group';
	import { timeRange, timeRanges } from '$lib/range.svelte';
</script>

<ToggleGroup.Root
	variant="outline"
	type="single"
	bind:value={timeRange.current}
	aria-label="Time range"
>
	{#each timeRanges as t (t.value)}
		<ToggleGroup.Item
			value={t.value}
			title={t.title}
			aria-label={t.title}
			class="cursor-pointer select-none"
			onclick={(event) => {
				if (timeRange.current === t.value) event.preventDefault();
			}}
			onkeydown={(event) => {
				if (timeRange.current === t.value && (event.key === 'Enter' || event.key === ' ')) {
					event.preventDefault();
				}
			}}
		>
			{t.label}
		</ToggleGroup.Item>
	{/each}
</ToggleGroup.Root>

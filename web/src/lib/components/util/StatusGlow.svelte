<script lang="ts">
	import { usePageStatus } from '#lib/hooks/page-status.svelte.js';
	import { cn } from '#lib/utils.js';

	const page = usePageStatus();
</script>

<div
	aria-hidden="true"
	class={cn(
		'glow pointer-events-none fixed inset-x-0 top-0 z-0 h-[70vh] transition-[background-color,opacity] duration-1000 ease-out motion-reduce:transition-none',
		page.meta.dot,
		page.monitors.length > 0 ? 'opacity-15 dark:opacity-20' : 'opacity-0'
	)}
></div>

<style>
	.glow {
		mask-image: radial-gradient(ellipse 60% 70% at 50% 0%, black, transparent 100%);
	}
</style>

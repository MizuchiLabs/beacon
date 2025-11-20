<script lang="ts">
	import { useConfig } from '$lib/api/queries';
	import Button from '$lib/components/ui/button/button.svelte';
	import { DotIcon, Github, Moon, Sun } from '@lucide/svelte';
	import { mode, toggleMode } from 'mode-watcher';

	let configQuery = $derived(useConfig());
</script>

<footer class="fixed right-0 bottom-0 left-0 z-50 flex justify-center px-2 sm:px-4">
	<div
		class="flex min-h-12 w-full max-w-3xl items-center justify-between gap-2 rounded-t-xl border-x border-t px-3 py-2 text-sm backdrop-blur-md sm:px-4"
	>
		<div class="flex items-center gap-1.5 text-muted-foreground">
			<p class="text-xs sm:text-sm">
				<span class="hidden sm:inline">&copy; {new Date().getFullYear()}</span>
				<span class="font-medium">Mizuchi Labs</span>
			</p>

			{#if configQuery.isSuccess}
				<div class="hidden items-center gap-1.5 text-xs md:flex">
					<DotIcon />
					<span class="flex items-center gap-1.5">
						{configQuery.data?.timezone}
					</span>
				</div>
			{/if}
		</div>

		<div class="flex items-center gap-1">
			<Button
				variant="ghost"
				size="icon"
				href="https://github.com/mizuchilabs/beacon"
				rel="noopener noreferrer"
				target="_blank"
				class="h-9 w-9 rounded-full hover:text-primary"
				aria-label="View Beacon on GitHub"
			>
				<Github size={18} />
			</Button>

			<Button
				variant="ghost"
				size="icon"
				onclick={toggleMode}
				class="h-9 w-9 rounded-full hover:text-primary"
				aria-label="Toggle theme"
			>
				{#if mode.current === 'light'}
					<Moon size={18} />
				{:else}
					<Sun size={18} />
				{/if}
			</Button>
		</div>
	</div>
</footer>
